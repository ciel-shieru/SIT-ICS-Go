# ADR-0022: Multi-Term PeopleSoft Timetable Fetch

## Status

Proposed

## Date

2026-10-16

## Context

The PeopleSoft timetable endpoint `SSR_SSENRL_LIST.GBL` (ADR-0012) displays the class schedule for a single selected term. When a student is enrolled in multiple terms (e.g., summer + fall, or two concurrent semesters), the page shows a "Change Term" button (`DERIVED_SSS_SCT_SSS_TERM_LINK`) that opens a term selection screen with radio buttons — one per available term.

The current implementation (`fetchTimetable` in `internal/browser/shared.go`) navigates to the endpoint, clicks the "Change Term" button, selects the **last** radio button (the most recently-used term), clicks "Go", and returns the timetable HTML for that single term. Students enrolled in multiple terms only receive timetable events from one term; events from all other terms are silently omitted.

This is not a transient issue. A student with enrollments across three terms will always receive only one term's schedule, regardless of how many times the fetch runs. The consequence is incomplete calendar data — missed classes, missing exam dates, and no awareness of cross-term scheduling conflicts.

The current term-selection logic in `handleTermSelectionWithPage()` (shared.go:126-219) already handles the navigation primitives needed to iterate across terms: clicking a radio button, clicking "Go", waiting for timetable data, and returning to term selection. The decision is whether to use this machinery once (current behavior) or loop through all available terms.

## Decision Drivers

- **Complete data**: All enrolled terms' timetables must be fetched in a single fetch cycle
- **Per-term error resilience**: A failure on one term (e.g., empty enrollment, navigation timeout) must not abort the entire multi-term fetch; remaining terms should still be processed (consistent with ADR-0020's degraded-fetch philosophy)
- **Minimal interface churn**: The `AuthBrowser.FetchTimetable` signature and the auth provider's `FetchTimetable` method must return parsed entries directly, since the browser layer now owns both fetching and parsing
- **Context deadline discipline**: The per-term loop shares a single context deadline; each iteration must complete within the remaining time budget

## Considered Options

### Option A: Loop Through All Term Radio Buttons in `fetchTimetable`

Navigate to `SSR_SSENRL_LIST.GBL`, detect the term selection screen, count radio buttons dynamically (`page.Elements("input.PSRADIOBUTTON")`), then for each radio button: click it, click "Go", wait for timetable data, extract and parse HTML, accumulate entries. After processing all but the last term, click "Change Term" to return to the term selection screen for the next iteration. Return `[]peoplesoft.Entry` directly from the browser layer.

### Option B: Fetch HTML Per-Term, Parse Upstream

Keep `FetchTimetable` returning raw HTML strings (one per term, as a slice), and move the parsing loop into the auth provider or app layer. The browser layer iterates terms but does not parse; the caller accumulates and parses entries.

### Option C: Fetch All Terms in App Layer via New Browser Method

Add a new browser method (e.g., `FetchAllTermsHTML`) that returns a slice of HTML strings (one per term). The app layer iterates the slice, calls `peoplesoft.ParseTimetableHTML` for each, and accumulates entries. This preserves the existing `FetchTimetable` signature and adds a parallel method.

### Option D: Keep Status Quo (Single Term)

Do not change the fetch behavior. Students with multiple terms continue to receive only one term's schedule.

## Decision

We will choose **Option A: Loop Through All Term Radio Buttons in `fetchTimetable`**.

## Rationale

### Why Option A over Option B

Option B keeps the browser layer as a raw HTML extractor but pushes the per-term parsing loop into the auth provider. This splits the responsibility: the browser knows how to navigate between terms, but the caller knows how to parse entries. The consequence is that `FetchTimetable` returns `[]string` (raw HTML) and the auth provider must loop over them, calling `peoplesoft.ParseTimetableHTML` for each. This adds an extra layer of indirection without clear benefit — the browser layer already has access to `peoplesoft.ParseTimetableHTML` in the current codebase (via the import chain), and moving parsing upstream complicates the interface contract ( callers must now handle empty-HTML entries, per-term parse errors, and accumulation logic).

Option A is simpler: the browser layer returns `[]peoplesoft.Entry` directly, and callers (auth provider → app layer) receive a single flat list of entries from all terms. This matches the existing pattern where `FetchBrightSpace` returns parsed `[]brightspace.BrightSpaceStringEntry` rather than raw HTML.

### Why Option A over Option C

Option C introduces a new browser method (`FetchAllTermsHTML`) alongside the existing `FetchTimetable`, creating two code paths for the same data source. The existing method would remain unchanged (still single-term), and callers would need to decide which method to use. This doubles maintenance burden and creates confusion about which method is authoritative. Option A replaces the existing method with a single, correct implementation that handles all terms.

### Why Option A over Option D

Option D preserves the status quo but leaves students with multi-term enrollments with incomplete data. This is not a viable long-term option — the feature is required by the user base and is technically straightforward to implement given the existing term-selection machinery in `handleTermSelectionWithPage()`.

### How It Works

The `fetchTimetable` function changes from:

```
Navigate → HandleTermSelection (selects last term) → Extract HTML → return string
```

To:

```
Navigate → Detect term selection screen → Count radio buttons → For each radio button:
    1. Click radio button (by XPath index: `//*[@id="SSR_DUMMY_RECV1$sels$0$i$0"]`)
    2. Click "Go" button (`#DERIVED_SSS_SCT_SSR_PB_GO`)
    3. Wait for timetable data element (`#DERIVED_CLS_DTL_CLASS_NBR$0`)
    4. Extract HTML, parse to []peoplesoft.Entry
    5. Append to accumulator
    6. If not last term, click "Change Term" to return to term selection screen
Return accumulated entries
```

The `FetchTimetable` signature changes from:

```go
FetchTimetable(ctx context.Context, weekDate string) (string, error)
```

To:

```go
FetchTimetable(ctx context.Context, weekDate string) ([]peoplesoft.Entry, error)
```

The `weekDate` parameter is retained for interface compatibility (even though it is unused in the current codebase per ADR-0012) and may be repurposed in future ADRs.

The auth provider's `FetchTimetable` method simplifies: instead of calling `browser.FetchTimetable()` → `peoplesoft.ParseTimetableHTML()` (two steps), it now calls `browser.FetchTimetable()` directly and returns the result. The parsing step is eliminated because the browser layer already returns parsed entries.

The app layer's `runFetch()` gains no new loop logic — it already receives `[]peoplesoft.Entry` from the auth provider and iterates over them. The multi-term accumulation is transparent to callers.

### Edge Case: Zero Radio Buttons

If the term selection screen is not detected (no `input.PSRADIOBUTTON` elements), the function falls back to extracting the current page's HTML and parsing it — the same behavior as the current single-term path. This ensures backward compatibility when a student has only one term (no term selection screen appears).

### Edge Case: Per-Term Failure

If parsing fails for one term's HTML (e.g., unexpected DOM structure, empty response), that term's entries are skipped with a warning log, and processing continues to the next term. This is consistent with ADR-0020's philosophy: individual data-source failures are recoverable, and the fetch continues with whatever data was successfully collected.

## Consequences

### Positive

- **Complete timetable data**: All enrolled terms' schedules are fetched in a single cycle, eliminating silent data loss for multi-term students.
- **Per-term error resilience**: A failure on one term does not abort the entire fetch. Remaining terms are still processed, and at least partial timetable data is cached.
- **Simplified auth provider**: The auth layer no longer performs HTML parsing; it delegates to the browser layer which returns parsed entries directly. This reduces coupling between auth and peoplesoft packages.
- **Transparent to callers**: The app layer (`runFetch`) already iterates over `[]peoplesoft.Entry`; the multi-term accumulation is invisible to existing code.
- **Leverages existing machinery**: The term-selection navigation primitives already exist in `handleTermSelectionWithPage()`; the change extends rather than replaces them.

### Negative

- **More browser navigation time**: Each additional term adds one or two navigations (click radio + click Go, plus click Change Term to return). A student with 5 terms adds ~10 navigations. The shared context deadline must accommodate all iterations.
- **Interface signature change**: `AuthBrowser.FetchTimetable` returns `[]peoplesoft.Entry` instead of `string`. This propagates to `MockAuthBrowser`, `ADFSProvider.FetchTimetable`, and all callers. Tests using `MockAuthBrowser` must be updated to return `[]peoplesoft.Entry` instead of `string`.
- **Parsing moves to browser layer**: The `peoplesoft.ParseTimetableHTML` function is now called from `internal/browser/shared.go` rather than `internal/auth/provider.go`. This is a slight architectural shift — the browser layer now depends on the peoplesoft parsing package.
- **Radio button XPath fragility**: The per-term radio button selector (`//*[@id="SSR_DUMMY_RECV1$sels$0$i$0"]`) uses a PeopleSoft-generated ID pattern. If PeopleSoft changes this pattern, the iteration loop breaks. The existing single-term path (click last radio button by element reference) is also fragile but has worked in practice.

### Neutral / Operational

- **Debug logging**: The `BROWSER_DEBUG` flag already covers navigation and HTML extraction; the new per-term iteration adds a log line per term (e.g., `browser: processing term 2 of 5`).
- **Timezone handling**: Parsing is unchanged — each term's entries are parsed using the same `loc` parameter passed from the app layer.
- **Cache semantics**: The non-destructive UID upsert (ADR-0003) is unaffected — events from all terms receive distinct UIDs based on their start/end times, so multi-term events do not collide with single-term cached events.

## Alternatives Considered

### Option B: Fetch HTML Per-Term, Parse Upstream

**Summary**: `FetchTimetable` returns `[]string` (raw HTML per term). The auth provider loops over the strings, parses each with `peoplesoft.ParseTimetableHTML`, and accumulates entries.

**Benefits**: Keeps parsing in the auth/peoplesoft layer, preserving the existing dependency direction (browser → auth → peoplesoft).

**Costs**: Adds an extra slice level (`[]string` → `[]Entry`). Callers must handle per-term parse errors during accumulation. The interface contract becomes less clean (callers receive raw HTML instead of domain objects).

**Reason rejected**: The auth provider's role is to orchestrate, not parse. The browser layer already has access to parsing logic and returning domain objects (`[]peoplesoft.Entry`) is more idiomatic — it matches how `FetchBrightSpace` returns parsed entries rather than raw HTML.

### Option C: New Browser Method for All-Terms Fetch

**Summary**: Keep `FetchTimetable` unchanged (single-term). Add `FetchAllTermsHTML() ([]string, error)` as a new method. App layer iterates and parses.

**Benefits**: No breaking changes to existing interfaces. The old method remains as a fallback or for single-term use cases.

**Costs**: Two methods for the same data source. Callers must choose which to use. Doubles test coverage requirements. The old method becomes dead code over time.

**Reason rejected**: Introducing a parallel method creates maintenance overhead and confusion. The existing `FetchTimetable` is the canonical timetable fetch method and should be updated, not duplicated.

### Option D: Keep Status Quo (Single Term)

**Summary**: Do not change the fetch behavior. Continue selecting the last-used term only.

**Benefits**: No code changes, no interface churn, no risk of new bugs.

**Costs**: Students with multi-term enrollments continue to receive incomplete timetable data. This is a data correctness issue, not a technical limitation.

**Reason rejected**: The feature is required by the user base. The technical implementation is straightforward and builds on existing term-selection machinery.

## Follow-Ups

- Monitor navigation latency for users with many terms (5+). If the shared context deadline is consistently exhausted, consider increasing `NavigationTimeout` or adding per-term timeout budgeting.
- Consider adding a `--term-filter` CLI flag (or `TIMETABLE_TERM_FILTER` env var) to allow users to fetch only specific terms by index or name, if users request selective term fetching.
- Add integration tests for the multi-term iteration path (requires staging PeopleSoft environment with multi-term enrollment).
- Related ADRs:
  - **ADR-0012**: PeopleSoft timetable fetch to SSR_SSENRL_LIST endpoint (the endpoint this ADR extends)
  - **ADR-0020**: Resilient fetch — BrightSpace still runs on PeopleSoft failure (per-term error resilience aligns with degraded-fetch philosophy)
  - **ADR-0003**: ICS generation and caching strategy (multi-term events receive distinct UIDs, no collision risk)

## References

- `internal/browser/shared.go` — `fetchTimetable()`, `handleTermSelectionWithPage()` (affected)
- `internal/browser/browser.go` — `AuthBrowser` interface, `MockAuthBrowser` (affected)
- `internal/auth/provider.go` — `ADFSProvider.FetchTimetable()` (simplified, parsing removed)
- `internal/app/fetch.go` — `runFetch()` (unchanged — receives `[]peoplesoft.Entry` from provider)
- `internal/peoplesoft/parser.go` — `ParseTimetableHTML()` (now called from browser layer)
