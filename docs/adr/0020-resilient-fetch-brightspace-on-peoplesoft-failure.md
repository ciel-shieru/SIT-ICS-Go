# ADR-0020: Resilient Fetch — BrightSpace Still Runs on PeopleSoft Timetable Fetch Failure

## Status

Accepted

## Date

2026-10-03

## Context

The PeopleSoft timetable is fetched from the `SSR_SSENRL_LIST.GBL` page in a single browser navigation (ADR-0012). This page is the flakiest upstream in the fetch pipeline: it depends on PeopleSoft session state, server-side rendering, and is subject to load and maintenance windows. Its failure mode is intermittent and independent of the BrightSpace D2L API (ADR-0013), which runs on a different backend within the same browser session.

Prior to this ADR, a PeopleSoft timetable fetch failure in `runFetch()` returned early and aborted the entire fetch job. The consequences were:

1. **Starved xsite outputs** — `xsite-events.ics`, `xsite-dropbox.ics`, `xsite-quizzes.ics`, and `xsite.ics` (ADR-0014) were not updated at all during a degraded cycle, even though the BrightSpace backend was healthy and the authenticated browser session was still valid.
2. **Wasted fetch cycle** — the next data for every source (including BrightSpace) would not be attempted until the next cron tick.

The key insight is that a PeopleSoft page failure is a *data-source* failure, not a *session* failure. The browser session remains authenticated and usable for the BrightSpace fetches that follow. Only failures of the shared browser session itself (or the authentication that underpins it) invalidate the rest of the job.

## Decision

The PeopleSoft timetable fetch is classified as a **recoverable failure class** within `runFetch()`:

- **Auth failures and browser-lifecycle failures remain fatal.** A single authenticated browser session backs all fetch steps (PeopleSoft, BrightSpace calendar, BrightSpace quizzes, quiz submission pages). If that session is broken, every remaining step would fail identically, so the whole job aborts immediately.
- **On PeopleSoft fetch failure** (any non-fatal error), the fetch continues:
  1. PeopleSoft event construction is skipped (no entries → no new PeopleSoft events this cycle).
  2. Blocklist `RemoveWhere` cleanup still runs, but during a degraded cycle the predicate additionally skips non-BrightSpace events (source not prefixed `brightspace-`), so cached PeopleSoft events are never deleted by blocklist patterns.
  3. BrightSpace calendar/dropbox fetch, quiz API fetch, quiz dedup (`DedupQuizzes`), quiz attempt tracking, and quiz removal all proceed unchanged (verified safe with an empty PeopleSoft event list).
  4. `cache.Update()` runs with only the BrightSpace events. The non-destructive UID upsert semantics (ADR-0003) preserve existing PeopleSoft events untouched — they are absent from the new event list and therefore retained.
  5. `SaveOutputs()` runs for all 7 ICS files, so xsite outputs are up to date even on degraded cycles.

### Error Classification Rule

`isFatalPostAuth(err)` inspects the error's wrapped chain (`%w` / root sentinel) against the six browser sentinels defined in `internal/browser/session.go`:

| Sentinel | Meaning |
|----------|---------|
| `ErrBrowserUnavailable` | Browser cannot be used |
| `ErrBrowserLaunch` | Browser failed to launch |
| `ErrBrowserConnect` | Browser connection failed |
| `ErrAuthentication` | Authentication failed |
| `ErrAuthenticationTimeout` | Authentication timed out |
| `ErrCredentialExtraction` | Cookie/credential extraction failed |

Errors rooted at any of these sentinels are fatal and abort the fetch. **Everything else after a successful auth is recoverable** (navigation failures, page timeouts, parse errors, API errors), and the fetch continues to completion.

### Degraded-Cycle Smart Merge

Smart merge (`MergeEvents`) matches BrightSpace Zoom events into PeopleSoft events by module code, time overlap, and location conditions. With no fresh PeopleSoft events, there is nothing to merge into; forcing the merge would also risk dropping the BrightSpace events. Therefore, on a degraded cycle, smart merge is skipped entirely and BrightSpace events are appended standalone — the same path as when smart merge is disabled.

## Consequences

### Positive

- **xsite outputs stay fresh**: BrightSpace events, dropbox due dates, and quizzes are fetched and saved even when PeopleSoft is down.
- **No data loss**: Existing PeopleSoft events in the cache are preserved by non-destructive UID upsert (ADR-0003); the degraded cycle cannot delete them.
- **Simplicity**: The classification rule is a single sentinel check; no new retry machinery is introduced.

### Negative / Operational

- **Stale timetable data on degraded cycles**: `timetable.ics`, `timetable-online.ics`, and `timetable-campus.ics` may serve PeopleSoft events from a previous successful cycle. A warning is logged: `scheduler: warning: timetable events are stale (peoplesoft fetch failed this cycle)`.
- **No Zoom injection on degraded cycles**: Smart merge does not run, so BrightSpace Zoom meeting details are not injected into PeopleSoft event descriptions until the next successful PeopleSoft fetch.
- **Standalone BrightSpace events**: BrightSpace events that would normally be merged into PeopleSoft events appear as separate calendar entries instead, until the next successful PeopleSoft fetch.
- **Blocklist scope shrinks during degraded cycles**: On a degraded cycle, blocklist cleanup only applies to BrightSpace events. A blocklist pattern that matches a cached PeopleSoft event will not remove that PeopleSoft event until the next successful PeopleSoft cycle (this is intentional — PeopleSoft events are authoritative for their source and must not be pruned by BrightSpace-scoped filters).

## Alternatives Considered

### Option A: Authoritative per-source replace

Replace all cached events for a source when that source is re-fetched, treating an empty PeopleSoft response as "no events" and wiping cached PeopleSoft events.

**Reason rejected**: Breaks ADR-0003 non-destructive upsert semantics. A transient empty or erroring PeopleSoft page would silently delete the user's entire cached timetable. The upsert contract (events absent from new data are retained) is load-bearing for idempotency and must not be weakened per-source.

### Option B: Swallow all errors, including browser failures

Continue the fetch on any PeopleSoft error, including browser-lifecycle failures.

**Reason rejected**: If the shared browser session is broken, the BrightSpace fetches would fail identically (same session, same cookies), wasting the fetch cycle and producing misleading log noise. Browser/auth failures must abort the job immediately.

### Option C: Retry PeopleSoft before continuing

Retry the PeopleSoft fetch (with backoff) before deciding to continue degraded.

**Reason rejected**: Adds latency to every degraded cycle and still ends in the same degraded state. The next cron tick already provides a retry opportunity with a fresh browser session (fresh auth, fresh deadlines).

## Follow-Ups

- Consider surfacing the stale-timetable warning in the ICS output (e.g., a VTODO or alert property) so calendar clients can display it, if users request it.
- Monitor how often degraded cycles occur in production logging; if PeopleSoft failures become frequent, investigate upstream (page load, session lifetime) rather than further widening the recoverable class.
- Related ADRs:
  - **ADR-0003**: ICS generation and caching strategy (non-destructive UID upsert semantics)
  - **ADR-0007**: Application-facing error model with wrapped sentinel errors
  - **ADR-0012**: PeopleSoft timetable fetch to SSR_SSENRL_LIST endpoint (the flaky source)
  - **ADR-0013**: BrightSpace D2L event integration (the source protected by this ADR)
  - **ADR-0014**: xsite ICS HTTP endpoints (the outputs kept fresh by this ADR)
  - **ADR-0019**: Interactive credential retry on desktop (auth failure path remains fatal, unaffected by this ADR)

## References

- `internal/app/fetch.go` — `runFetch()` degraded-cycle control flow and `isFatalPostAuth()`
- `internal/browser/session.go` — sentinel error definitions
- `internal/calendar/cache.go` — `Update()` (non-destructive merge), `RemoveWhere()`
- `internal/smartmerge/smartmerge.go` — `MergeEvents()` (skipped on degraded cycles)
