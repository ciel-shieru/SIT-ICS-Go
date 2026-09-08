# ADR-0015: Codebase Restructuring for Production Readiness

## Status

Accepted

## Date

2026-09-08

## Context

The SIT ICS Go project had accumulated 14 Architecture Decision Records (ADR-0001 through ADR-0014) documenting incremental decisions over a two-week development period. Each ADR represented a focused change — authentication flow, browser abstraction, PeopleSoft endpoint selection, BrightSpace integration, and HTTP endpoint expansion.

By commit `faa5de11`, the codebase had grown to 14 packages with several structural problems:

1. **Misplaced responsibilities**: PeopleSoft date parsing logic lived in `main.go`. BrightSpace ICS coupling was implemented directly in the `ics` package rather than through the domain event model. The `ICSCache` type combined event state management, merging/filtering projections, ICS serialization, and filesystem persistence into a single monolithic type.

2. **Dual-transport duplication**: BrightSpace had two implementations — `internal/brightspace/client.go` used a direct `http.Client` with API key authentication, while `internal/browser/shared.go` contained browser-based retrieval logic. This violated the principle that authenticated fetching should follow a single transport path.

3. **`time.Local` mutation**: The application mutated `time.Local` at startup for timezone configuration. This is a global state mutation with well-known issues in concurrent Go programs — it affects all goroutines, cannot be changed per-operation, and makes testing harder.

4. **Misleading names**: Several functions and types did not reflect their actual responsibilities:
   - `navigateToAuthPage` — actually performed full ADFS authentication, not just navigation
   - `extractErrorMessage` — actually extracted ADFS login errors specifically
   - `fetchJSON` — actually decoded and validated JSON responses
   - `computeEntryDay` — actually computed entry dates (not just days)
   - `parseISOTime` — actually parsed BrightSpace API timestamps
   - `SaveAllWithXsiteFiles` — saved all output paths, not just xSite files
   - `GenerateWithTolerance` — generated events at a fixed offset, not with tolerance
   - `browserEntriesToICSEvents` — converted browser entries to domain events, not to ICS events
   - `GenerateUID` — renamed to `EventID` to clarify it is a domain function, not an ICS-specific one

5. **Source-to-output coupling**: The `entryToICSEvent` function in the `ics` package coupled PeopleSoft entries directly to ICS output format. BrightSpace events were similarly coupled through `browserEntriesToICSEvent`. This made it impossible to produce non-ICS outputs (e.g., JSON, CSV) from the same domain events.

The dependency direction was inverted for extensibility: instead of `source → ICS event → output`, the flow was `source → ICS event` with no intermediate domain model. Adding a new output format required duplicating the source-to-ICS conversion logic.

A comprehensive restructuring was executed across 17 sequential phases (commits `1d99502` through `dde4c38`) to reorganize the codebase without changing application behavior. Each phase was independently implemented and reviewed. The restructuring addressed all five structural problems above.

## Decision Drivers

- **Separation of concerns**: Each package should have a single, well-defined responsibility. Types should not combine state management, serialization, persistence, and projection logic.
- **Domain model decoupled from output formats**: The application's event model should be independent of any specific output format (ICS, JSON, CSV). Output rendering is a separate concern.
- **Single transport path**: All authenticated fetching (ADFS, PeopleSoft, BrightSpace) should use the same browser-backed transport, eliminating dual-transport patterns.
- **Explicit timezone usage**: Timezone should be passed explicitly as `*time.Location` to functions that need it, not derived from global `time.Local`.
- **Accurate naming**: Function and type names should reflect actual responsibilities to reduce cognitive load during code review and maintenance.
- **Testability**: Each responsibility should be independently testable without requiring browser automation or network access.
- **Production readiness**: The codebase should follow idiomatic Go patterns for concurrency, error handling, and package organization.

## Considered Options

### Option A: Comprehensive Restructuring (Selected)

Reorganize the entire codebase into focused packages with clear responsibility boundaries. Create a domain event model in `internal/calendar/` that decouples data sources from output rendering. Restructure the browser package around session mechanics. Consolidate BrightSpace to browser-backed transport. Split cache responsibilities into cache, render, persistence, and projection.

### Option B: Incremental Refactoring

Apply small, isolated refactors one at a time without a coordinated restructuring plan. Fix naming issues, move files to correct packages, and split monolithic types individually.

### Option C: Minimal Changes

Address only the most critical issues (dual-transport in BrightSpace, `time.Local` mutation) while leaving other structural problems for future work.

## Decision

We will choose **Option A: Comprehensive Restructuring**.

The 17 phases were executed sequentially, each building on the previous:

1. Behavioral invariant tests (retention, idempotency, deletion, determinism, persistence failures)
2. Move orchestration out of `main.go` → `internal/app`
3. Restructure browser package around authenticated session
4. Extract ADFS/SAML auth from browser → `internal/auth`
5. Make PeopleSoft exclusively browser-backed
6. Consolidate BrightSpace around browser-backed retrieval (delete direct HTTP client)
7. Create domain calendar/event layer, decouple from ICS
8. Split cache/rendering/projection/persistence responsibilities
9. Centralize deterministic ICS rendering
10. Centralize atomic calendar output persistence
11. Configuration cleanup and validation, remove `time.Local` mutation
12. Refactor server layer
13. Scheduler cleanup
14. Final naming pass
15. DRY and duplication audit
16. Go 1.26 idiomaticity and concurrency audit
17. Full end-to-end structural review

## Rationale

The comprehensive restructuring was selected over incremental refactoring because the structural problems were interconnected — fixing one in isolation would not resolve the underlying source-to-output coupling, and piecemeal changes would leave the codebase in an inconsistent state. The 17-phase approach provided a clear target architecture and ensured all problems were addressed in a single coordinated effort.

The domain event model in `calendar` was created to decouple data sources from output formats. Before this change, `ics.Generate()` was the only rendering function, and both PeopleSoft and BrightSpace conversions produced ICS-specific types. By introducing `calendar.Event` as a format-agnostic domain type, adding new output formats (JSON, CSV) requires only a new render function — the sources and domain model remain unchanged.

The browser package restructuring around session mechanics eliminated the dual-transport pattern. The direct `http.Client` in `brightspace/client.go` was removed because authenticated API calls are more reliably performed through the browser's CDP protocol (`DecodeJSON`), which handles session cookies and JavaScript-rendered authentication automatically. Consolidating all authenticated fetching into the `AuthBrowser` interface provides a single, testable abstraction.

Explicit timezone passing replaces `time.Local` mutation because `time.Local` is a global variable that affects all goroutines in a concurrent program. Passing `*time.Location` explicitly makes timezone handling predictable, testable, and safe for concurrent use — each operation uses the location passed to it rather than relying on global state.

The naming corrections reduce cognitive load by ensuring function and type names accurately reflect their responsibilities. Reviewers can understand what a function does from its name without reading the implementation, which speeds up code review and reduces maintenance errors.

## Target Architecture

```
cmd/sit-ics/main.go          = composition + process lifecycle only
internal/app                 = application orchestration (app.go, fetch.go, outputs.go)
internal/browser             = Rod/browser primitives + session mechanics (session.go, local.go, remote.go, cookies.go, auth.go, fetcher.go, shared.go, browser.go)
internal/auth                = ADFS/SAML/MFA authentication semantics (provider.go)
internal/peoplesoft          = PeopleSoft browser-backed fetching + parsing + mapping (peoplesoft.go, parser.go, datetime.go)
internal/brightspace         = BrightSpace browser-backed API fetching + mapping + filtering (fetch.go, models.go, filter.go, timestamp.go) — direct HTTP client.go removed
internal/calendar            = domain events + non-destructive cache + projections + deterministic ICS rendering + persistence (event.go, cache.go, render.go, persistence.go, projection.go)
internal/config              = configuration loading + validation (config.go, flags.go, validate.go, browser_mode.go)
internal/server              = HTTP serving (server.go, handlers.go)
internal/scheduler           = scheduling (scheduler.go)
internal/totp                = TOTP generation (totp.go)
```

### Dependency Graph

```
cmd/sit-ics → internal/app → internal/auth → internal/browser → internal/peoplesoft
                                         → internal/brightspace
internal/app → internal/calendar
internal/calendar → (no internal deps — domain package)
internal/config → (no internal deps)
internal/server → internal/calendar
internal/scheduler → (no internal deps)
internal/totp → (no internal deps)
```

The `calendar` package is the deepest internal dependency — it has no imports of other internal packages, making it a pure domain layer. All other packages depend on `calendar` or on external dependencies only.

### Key Architectural Decisions

#### 1. Browser-Backed Transport Only

All authenticated fetching (ADFS, PeopleSoft, BrightSpace) must go through the Rod browser session. The direct `http.Client` implementation in `internal/brightspace/client.go` was deleted. BrightSpace now uses `internal/browser.Fetcher` interface methods (`Navigate`, `DecodeJSON`) for API calls.

This eliminates the dual-transport pattern where BrightSpace had both HTTP and browser implementations. The `AuthBrowser` interface (`internal/browser/browser.go:16`) now defines all authenticated operations:

```go
type AuthBrowser interface {
    Authenticate(ctx context.Context, req AuthRequest) (AuthResult, error)
    FetchTimetable(ctx context.Context, weekDate string) (string, error)
    FetchBrightSpace(ctx context.Context, baseURL string) ([]brightspace.BrightSpaceStringEntry, error)
    Close()
}
```

The `ADFSProvider` (`internal/auth/provider.go`) wraps `AuthBrowser` and provides a higher-level interface that combines authentication with data fetching.

#### 2. Domain Event Model Decoupled from ICS

The `calendar.Event` type (`internal/calendar/event.go:5`) is the application's domain event, independent of any output format. PeopleSoft and BrightSpace both produce `calendar.Event` values. ICS serialization is a rendering concern in the calendar package, not a coupling between sources and ICS.

This changes the dependency direction from `source → ICS event` to `source → domain event → ICS renderer`. The `Render` function (`internal/calendar/render.go:40`) takes `[]Event` and `RenderOptions` and produces ICS bytes. Adding a non-ICS output format (e.g., JSON) would require adding a new render function — the domain events remain unchanged.

The `EventID` function (`internal/calendar/render.go:16`) computes deterministic UIDs via SHA-256, including the `Source` field when set (for BrightSpace events). This preserves idempotent upsert semantics documented in ADR-0003.

#### 3. Non-Destructive Cache Semantics

The `ICSCache` (`internal/calendar/cache.go:11`) intentionally retains events that disappear from upstream responses. Only explicit configuration-driven deletion (e.g., BrightSpace blocklist via `RemoveWhere`) removes events.

The `Update` method (`internal/calendar/cache.go:149`) merges new events into the cache with non-destructive semantics:
- Events in the new set are added or update matching existing events (by UID)
- Events absent from the new set are retained
- No duplicates are created

The `mergeEvents` method (`internal/calendar/cache.go:344`) documents this invariant explicitly:

```go
// mergeEvents merges newEvents into existing cache events with non-destructive semantics:
// - Events in newEvents are added or update matching existing events (by UID).
// - Events absent from newEvents are retained (non-destructive).
// - No duplicates are created.
// This does NOT implement ReplaceSource or authoritative source reconciliation.
```

This is tested and enforced by behavioral invariant tests in `internal/calendar/cache_test.go`.

#### 4. Centralized Atomic Persistence

The `writeAtomic` function (`internal/calendar/persistence.go:9`) implements the temp-file-then-rename algorithm centrally:

```go
func writeAtomic(path string, data []byte) error {
    tmp := path + ".tmp"
    if err := os.WriteFile(tmp, data, 0600); err != nil {
        return err
    }
    if err := os.Rename(tmp, path); err != nil {
        os.Remove(tmp)
        return err
    }
    return nil
}
```

`SaveOutputs` (`internal/calendar/cache.go:254`) handles all configured output paths (main, online, campus, BrightSpace events, BrightSpace dropbox). Rendering errors and persistence errors are propagated, never silently ignored. The `outputErrors` type (`internal/calendar/cache.go:25`) implements `error` with a formatted message listing all failed paths.

#### 5. Explicit Timezone Usage

`time.Local` mutation is removed. All components receive `*time.Location` explicitly:

- `LoadFromFile(path, loc)` — parses ICS dates in the given location
- `Render(events, RenderOptions{Timezone: loc})` — formats dates in the given location
- `peoplesoft.ParseTimetableHTML(html, year, loc)` — computes meeting dates in the given location
- `auth.ADFSProvider.FetchTimetable(ctx, weekDate, loc)` — delegates location to the parser

The `Get` and `GetFiltered` methods on `ICSCache` accept a timezone string and load the location internally (`internal/calendar/cache.go:83`), falling back to `time.UTC` on error.

#### 6. Naming Corrections

| Old Name | New Name | Rationale |
|----------|----------|-----------|
| `navigateToAuthPage` | `authenticateADFS` | Function performs full ADFS authentication, not just navigation |
| `extractErrorMessage` | `extractADFSLoginError` | Function extracts ADFS login errors specifically |
| `fetchJSON` | `DecodeJSON` | Function decodes and validates JSON responses |
| `computeEntryDay` | `computeEntryDate` | Function computes actual dates, not just day-of-week |
| `parseISOTime` | `parseAPITimestamp` | Function parses BrightSpace API timestamps (not generic ISO time) |
| `SaveAllWithXsiteFiles` | `SaveOutputs` | Method saves all output paths, not just xSite files |
| `GenerateWithTolerance` | `GenerateAtOffset` | Function generates events at a fixed offset, not with tolerance |
| `browserEntriesToICSEvents` | source/domain conversion | Converts browser entries to domain events, not to ICS events |
| `GenerateUID` | `EventID` | Domain function, not ICS-specific; clarifies responsibility |

## Consequences

### Positive

- **Single transport path**: BrightSpace no longer has a dual-transport pattern. All authenticated fetching goes through the browser session, reducing code duplication and maintenance burden.
- **Domain model extensibility**: The `calendar.Event` type is independent of ICS. Adding new output formats (JSON, CSV, API responses) requires only a new render function — no changes to the domain model or data sources.
- **Clear responsibility boundaries**: Each package has a single, well-defined responsibility. The `calendar` package owns events, caching, rendering, persistence, and projections. The `browser` package owns Rod primitives and session mechanics. The `auth` package owns ADFS/SAML/MFA authentication semantics.
- **Reduced cognitive load**: Naming corrections make it immediately clear what each function does. Reviewers no longer need to read implementation to understand responsibility.
- **Non-destructive cache**: The invariant that events are never silently deleted is documented and tested. This prevents data loss when upstream responses are incomplete.
- **Explicit timezone**: Components receive `*time.Location` explicitly, eliminating global state mutation and making timezone handling testable and predictable.
- **Centralized persistence**: The `writeAtomic` algorithm is defined once in the calendar package. All output paths use the same atomic write logic. Errors are collected and propagated.
- **Testability**: The domain layer (`calendar.Event`, `ICSCache`) is independently testable without browser automation. Mock `AuthBrowser` supports unit testing of higher-level orchestration.
- **Idiomatic Go**: The restructuring aligns with Go package organization conventions — small, focused packages with clear boundaries. Concurrency patterns (RWMutex, goroutine-safe cache) follow established idioms.

### Negative

- **Same package count, different layout**: The codebase retained 14 internal packages but redistributed responsibilities — the `ics` package was replaced by `calendar`, which absorbed responsibilities from multiple sources. New developers must understand the new package layout.
- **Cache complexity**: The `ICSCache` type is larger than the original `ICSCache` because it now owns rendering, persistence, and projection logic in addition to state management. However, these responsibilities are clearly separated into distinct files (`render.go`, `persistence.go`, `projection.go`).
- **Breaking change risk**: The restructuring changed function signatures (e.g., `LoadFromFile(path)` → `LoadFromFile(path, loc)`, `Write(events, tz, refreshInterval)` → `Render(events, RenderOptions{...})`). Any external consumers of the package would need updates.
- **BrightSpace HTTP client removal**: Users who relied on the direct HTTP client for BrightSpace (e.g., for custom API key authentication) must now use the browser-backed path. This is intentional but reduces flexibility for advanced use cases.

### Neutral / Operational

- **Same binary, different internals**: The restructuring did not change application behavior, configuration, or output format. Users experience identical functionality.
- **Test impact**: Existing tests were migrated to the new package structure. `internal/ics/cache_test.go` became `internal/calendar/cache_test.go`. `internal/ics/ics_test.go` was deleted (ICS rendering is now tested through `Render` in the calendar package).
- **Commit history**: The 17 phases are visible in git log. The atomic nature of each commit makes it possible to bisect issues to specific restructuring phases.
- **ADR references**: This ADR supersedes or extends ADR-0003, ADR-0005, ADR-0012, and ADR-0013 by documenting the architectural decisions that enabled the restructuring.

## Alternatives Considered

### Option B: Incremental Refactoring

**Summary**: Apply small, isolated refactors one at a time without a coordinated restructuring plan. Fix naming issues, move files to correct packages, and split monolithic types individually.

**Benefits**: Lower risk per commit; easier to revert individual changes; no large reorganization that could introduce subtle bugs.

**Costs**: Does not address the fundamental issue of source-to-output coupling. Dual-transport pattern in BrightSpace persists until explicitly addressed. `time.Local` mutation remains until someone notices it. The codebase would continue to accumulate structural debt with no clear endpoint.

**Reason rejected**: Incremental refactoring without a target architecture leads to inconsistent boundaries. The 17-phase approach provides a clear target state and ensures all structural problems are addressed in a single coordinated effort.

### Option C: Minimal Changes

**Summary**: Address only the most critical issues (dual-transport in BrightSpace, `time.Local` mutation) while leaving other structural problems for future work.

**Benefits**: Faster to implement; lower immediate risk; focuses effort on the highest-priority issues.

**Costs**: Leaves source-to-output coupling, misleading names, monolithic cache, and misplaced responsibilities unaddressed. Future refactoring work would need to undo the partial restructuring.

**Reason rejected**: The partial approach would leave the codebase in an inconsistent state — some packages reorganized, others not. The comprehensive approach ensures all packages follow the same responsibility boundaries.

## Follow-Ups

- Monitor whether the non-destructive cache semantics cause unbounded growth over time. If the event set grows beyond practical limits, consider implementing a time-based retention policy (e.g., drop events older than N months).
- Consider adding structured logging to the `calendar` package for cache operations (merge, save, load) to aid debugging in production.
- Evaluate whether the `AuthBrowser` interface should expose additional capabilities (e.g., cookie inspection for debugging, session health checks) without exposing Rod internals.
- Related ADRs:
  - **ADR-0003**: ICS Generation and Caching Strategy — extended by domain event separation; cache responsibilities split into cache, render, persistence, projection
  - **ADR-0005**: Browser Abstraction Layer — extended by session abstraction and elimination of direct HTTP transport for authenticated sources
  - **ADR-0012**: PeopleSoft Timetable Fetch to List Endpoint — PeopleSoft made exclusively browser-backed
  - **ADR-0013**: BrightSpace D2L Event Integration — BrightSpace consolidated to single browser-backed implementation; `entryToICSEvent` coupling removed

## References

- `internal/calendar/event.go` — Domain event type, independent of output format
- `internal/calendar/cache.go` — Non-destructive cache with merge, save, and projection logic
- `internal/calendar/render.go` — Deterministic ICS rendering via `Render` function
- `internal/calendar/persistence.go` — Centralized `writeAtomic` algorithm
- `internal/calendar/projection.go` — Filter predicates (`IsOnline`, `IsCampus`)
- `internal/browser/browser.go:16` — `AuthBrowser` interface defining all authenticated operations
- `internal/auth/provider.go` — ADFS/SAML/MFA authentication semantics, wrapping `AuthBrowser`
- `internal/peoplesoft/` — PeopleSoft browser-backed fetching + parsing + mapping
- `internal/brightspace/` — BrightSpace browser-backed API fetching + mapping + filtering
- `internal/app/` — Application orchestration, composition root
- `internal/calendar/cache_test.go` — Behavioral invariant tests for cache semantics
- Commits `1d99502` through `dde4c38` — 17-phase restructuring implementation
