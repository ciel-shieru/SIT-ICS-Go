# ADR-0013: BrightSpace D2L Event Integration

## Status

Accepted

## Date

2026-09-08

## Context

The SIT ICS Go project already fetched PeopleSoft timetable data via browser automation using the Rod library. The authentication flow navigated to PeopleSoft (`in4sit.singaporetech.edu.sg`), handled ADFS SAML authentication with MFA, and extracted cookies for both PeopleSoft API calls and direct HTML parsing of the `SSR_SSENRL_LIST.GBL` endpoint (ADR-0012).

Singapore Tech students also use BrightSpace D2L, the institution's learning management system, which contains two categories of time-sensitive data not available in PeopleSoft:

1. **Calendar events** — instructor-created events, exams, and announcements with specific dates and times
2. **Dropbox due dates** — assignment submission deadlines for each course

These events are valuable additions to the unified ICS calendar but require a separate authentication and data-fetching flow. BrightSpace uses D2L's proprietary SAML-based authentication at `/d2l/lp/auth/saml/login`, which is distinct from the PeopleSoft ADFS flow.

The key architectural question was how to integrate BrightSpace into the existing browser automation architecture without launching a second browser instance, duplicating the auth flow, or introducing unnecessary complexity.

## Decision Drivers

- **Single browser session**: Reuse the existing Rod browser session created for PeopleSoft auth — do not launch a second browser
- **Leverage existing SAML pattern**: The browser already handles ADFS→PeopleSoft SAML redirects naturally (ADR-0010); BrightSpace SAML follows a similar pattern
- **Configurable filtering**: Users need to exclude specific courses, events, or locations from BrightSpace data to avoid cluttering their calendar
- **Source tracking**: Events from BrightSpace must be distinguishable from PeopleSoft events for filtering and debugging
- **Separate ICS outputs**: BrightSpace calendar events and dropbox due dates should have dedicated ICS files alongside the existing three (main, online, campus)
- **Campus projection integrity**: The campus ICS file should exclude BrightSpace events (they are xSite/D2L content, not campus content)
- **Stale event cleanup**: If a user changes their blocklist to include previously-unblocked events, old instances of those events must be removed from the cache

## Considered Options

### Option A: Reuse Existing Rod Browser Session for BrightSpace Auth

After authenticating to PeopleSoft, the same open browser session navigates to BrightSpace's SAML login page (`/d2l/lp/auth/saml/login`). The browser follows the SAML redirect naturally (per ADR-0010) to `/d2l/home`, then uses Rod's `page.DecodeJSON()` to call BrightSpace D2L API endpoints (`/d2l/api/le/{version}/{orgUnitId}/calendar/events/` and `/d2l/api/le/{version}/{orgUnitId}/dropbox/folders/`).

### Option B: Separate Browser Instance for BrightSpace

Launch a second Rod browser instance (or connect to a second remote browser) specifically for BrightSpace authentication and data fetching. This would require a second set of credentials or cookie sharing between browser contexts.

### Option C: HTTP Client with Cookie Sharing

After PeopleSoft auth, extract BrightSpace cookies from the same browser session, then use an HTTP client (similar to the Peoplesoft client pattern) to call BrightSpace APIs. This would require understanding BrightSpace's API authentication model.

## Decision

We will choose **Option A: Reuse Existing Rod Browser Session for BrightSpace Auth**.

## Rationale

### Why Option A over Option B

Launching a second browser instance doubles the resource overhead (memory, CPU, Chromium processes) and adds complexity around coordinating two browser lifecycles. Since the user is already authenticated in the first browser session, reusing it avoids any credential sharing or cookie-passing between browsers. The SAML flow for BrightSpace is well-understood from the PeopleSoft integration (ADR-0006, ADR-0010) — the browser navigates to the login page, waits for the SAML redirect, and arrives at the home page.

### Why Option A over Option C

BrightSpace D2L's API requires authentication via session cookies set after SAML login. While extracting these cookies and using an HTTP client is technically feasible, the existing `Fetcher` interface (`Navigate` + `DecodeJSON`) already provides a clean abstraction over Rod's page operations. The `DecodeJSON` method uses Rod's CDP protocol to navigate and parse JSON responses, which is more reliable than managing cookies manually across multiple HTTP requests. Additionally, the browser-based approach ensures that any JavaScript-rendered content or dynamic authentication checks are handled automatically.

### How It Works

The BrightSpace integration extends the existing `AuthBrowser` interface:

1. **`FetchBrightSpace(ctx, baseURL)`** is added to the `AuthBrowser` interface, implemented by both `LocalBrowser` and `RemoteBrowser`
2. The method navigates to `/d2l/lp/auth/saml/login` on the BrightSpace base URL and waits for the SAML redirect to complete by watching for `PageFrameNavigated` events targeting `/d2l/home`
3. A simpler pattern was chosen over the initial event-listener approach: `page.Navigate()` → `page.WaitNavigation(NetworkAlmostIdle)` → `page.WaitStable(5000)` (commit 6888165)
4. Once at `/d2l/home`, the browser calls `page.DecodeJSON()` to hit the BrightSpace API endpoints
5. The `brightspace.Client` owns endpoint paths and URL construction, while the `Fetcher` interface abstracts the browser page operations
6. Courses are fetched via `/d2l/le/manageCourses/api/mycourses`, then calendar events and dropbox folders are fetched per-course
7. A `Blocklist` filters out courses and events by name, ID, title, or location patterns
8. Events are converted to `calendar.Event` values with source tracking (`"brightspace-calendar"` or `"brightspace-dropbox"`)
9. Blocked events are deleted from the cache BEFORE new events are fetched (commit 6872707), preventing stale blocked events from persisting across refresh cycles

### SAML Auth Simplification

The initial implementation used a complex `navigateAndWaitForURL` function that registered a CDP `PageFrameNavigated` event listener via `EachEvent` and used a `select` loop to wait for the target URL. This was simplified (commit 6888165) to the standard pattern:

```
page.Navigate(url) → page.WaitNavigation(NetworkAlmostIdle) → page.WaitStable(5000)
```

This eliminates the event-listener boilerplate and aligns with the established waiting primitives from ADR-0006. The `https://xsite.singaporetech.edu.sg` origin was added to `allowedOrigins` to support the BrightSpace SAML redirect.

### Blocklist Design

The `Blocklist` struct (in `internal/brightspace/filter.go`) provides four filtering dimensions:

- **Course name patterns** — case-insensitive substring match against course names
- **Course IDs** — exact match against `OrgUnitId`
- **Event title patterns** — case-insensitive substring match against event titles
- **Event location patterns** — case-insensitive substring match against event locations

All patterns are configurable via comma-separated environment variables and CLI flags. The `Matches()` method combines all checks for a single predicate used by `ICSCache.RemoveWhere()`.

### ICS Output Expansion

The cache now writes five files instead of three:

| File | Content | Filter |
|------|---------|--------|
| `timetable.ics` | All events | None |
| `timetable-online.ics` | Online events only | `IsOnline` |
| `timetable-campus.ics` | Campus events only | `IsCampus` (non-online AND not BrightSpace) |
| `xsite-events.ics` | BrightSpace calendar events | `Source == "brightspace-calendar"` |
| `xsite-dropbox.ics` | BrightSpace dropbox due dates | `Source == "brightspace-dropbox"` |

The `IsCampus()` filter explicitly excludes BrightSpace events via `isBrightSpaceEvent()`, which checks `strings.HasPrefix(event.Source, "brightspace-")`. This ensures campus ICS files contain only PeopleSoft-sourced events.

### Source Tracking and UID Computation

Events carry a `Source` field set to `"brightspace-calendar"` or `"brightspace-dropbox"`. The `EventID()` function (previously `GenerateUID()`) includes the source in the SHA-256 UID computation when the source is set, ensuring that BrightSpace events have distinct UIDs from any PeopleSoft events with the same summary/location/time.

The `parseICS()` function reads the `X-SOURCE` field during file loading for round-trip support, preserving source information across cache reloads.

### All-Day Event Fix

An important correctness fix was applied to all-day event handling: only events where `entry.IsAllDay` is explicitly `true` extend the end time by 24 hours. Previously, the condition `entry.IsAllDay || dtStart.Equal(dtEnd)` treated zero-duration events as all-day, which incorrectly extended their end time. The corrected condition is `entry.IsAllDay` alone (commit 37da839).

## Consequences

### Positive

- **No second browser**: Reusing the existing Rod session avoids launching a second Chromium instance, saving memory and CPU
- **Natural SAML flow**: BrightSpace SAML authentication follows the same pattern as PeopleSoft ADFS (ADR-0010), requiring no custom SAML parsing or cookie manipulation
- **Configurable filtering**: Users can exclude specific courses or events via environment variables without code changes
- **Source distinguishability**: The `X-SOURCE` field in ICS output allows downstream tools to identify event origins
- **Campus projection integrity**: BrightSpace events are excluded from the campus ICS file, respecting the semantic boundary between campus and xSite content
- **Stale event cleanup**: Delete-before-fetch ensures that blocklist changes take effect immediately without manual cache clearing
- **Separate xSite endpoints**: HTTP endpoints `/xsite-events.ics` and `/xsite-dropbox.ics` serve filtered BrightSpace events directly from the server

### Negative

- **Tighter coupling to BrightSpace API**: The `Fetcher` interface depends on Rod's `DecodeJSON()` method, which uses CDP protocol navigation. If BrightSpace changes their API structure or authentication model, the integration may break
- **Five output files instead of three**: Increases disk I/O on each save and adds complexity to the cache's `SaveOutputs()` method
- **API response fragility**: The `MyCoursesResponse` struct wraps the courses array in a `Courses` field (commit 5566223). If BrightSpace changes the JSON structure, parsing will fail silently or with an error
- **Timestamp parsing complexity**: BrightSpace returns ISO 8601 timestamps that may include fractional seconds (`2006-01-02T15:04:05.000Z`) or full RFC 3339 nano precision. The `ParseTimestamp()` function tries two formats with fallback, adding parsing overhead
- **HTML-to-text stripping**: The `htmlToPlainText()` function does basic tag stripping but does not handle all HTML entities or nested tags, potentially producing incomplete descriptions

### Neutral / Operational

- **Same browser session lifetime**: BrightSpace auth happens within the existing browser lifecycle — if the browser crashes or is closed, BrightSpace data is unavailable until the next auth cycle
- **Per-course API calls**: Calendar events and dropbox folders are fetched per-course, so the number of API calls scales with the number of enrolled courses
- **Blocklist maintenance**: Users must manually maintain blocklist patterns via environment variables; there is no persistent blocklist storage
- **Server endpoint behavior**: `/xsite-events.ics` and `/xsite-dropbox.ics` return `StatusNoContent` (204) when no events exist, which some ICS clients may not handle gracefully
- **Timezone handling**: BrightSpace timestamps are parsed in the configured timezone (from `TZ` env var), consistent with PeopleSoft events

## Alternatives Considered

### Option B: Separate Browser Instance for BrightSpace

**Summary**: Launch a second Rod browser instance (or connect to a second remote browser) for BrightSpace authentication and data fetching. This would maintain complete isolation between the PeopleSoft and BrightSpace sessions.

**Benefits**: Complete session isolation; a crash in one browser does not affect the other; different browser profiles or contexts could be used for different purposes.

**Costs**: Doubles Chromium resource usage (memory, CPU, file descriptors). Requires coordinating two browser lifecycles in the shutdown sequence. If using remote browser mode, requires two separate remote browser connections with potentially different discovery URLs. No authentication benefit — the user is already logged in to both systems in the first browser.

**Reason rejected**: The existing browser session already has valid credentials for both PeopleSoft and BrightSpace. There is no security or isolation benefit to a second browser, and the resource cost is significant. Session isolation is not a requirement for this use case.

### Option C: HTTP Client with Cookie Sharing

**Summary**: After PeopleSoft auth, extract BrightSpace session cookies from the browser, then use an HTTP client (similar to the Peoplesoft client in `internal/peoplesoft/`) to call BrightSpace D2L API endpoints.

**Benefits**: Decouples data fetching from browser automation; HTTP client calls are faster and more testable than browser-based navigation; easier to add retries and timeouts.

**Costs**: Requires understanding BrightSpace's cookie-based authentication model and ensuring all required cookies are extracted and set correctly. BrightSpace may use additional authentication mechanisms (e.g., JavaScript-based token generation) that an HTTP client cannot replicate. The existing `Fetcher` interface already provides a clean abstraction. Adding a separate HTTP client for BrightSpace would duplicate the cookie extraction and injection logic.

**Reason rejected**: The `Fetcher` interface with `DecodeJSON()` already provides reliable browser-backed HTTP calls through Rod's CDP protocol. The additional complexity of managing cookies, handling authentication edge cases, and maintaining a separate HTTP client outweighs the marginal performance benefit. The browser-based approach is more robust against BrightSpace authentication changes.

### Option D: No BrightSpace Integration

**Summary**: Maintain the current PeopleSoft-only integration without adding BrightSpace support.

**Benefits**: Simpler codebase; fewer dependencies; no new configuration options or output files to maintain.

**Costs**: Students miss out on calendar events and assignment deadlines from BrightSpace, which are valuable additions to their unified calendar. The project would only cover half of the relevant academic data sources at Singapore Tech.

**Reason rejected**: BrightSpace is a primary data source for Singapore Tech students. Excluding it from the ICS output would significantly reduce the utility of the application for a large portion of the user base.

## Follow-Ups

- Monitor BrightSpace API response structure for changes that may break parsing (especially the `MyCoursesResponse` and `CalendarEventAPI` JSON structures)
- Track whether BrightSpace introduces pagination or rate limiting for the `/api/le/{version}/{orgUnitId}/calendar/events/` endpoint, which would require handling `Bookmark` fields and request throttling
- Consider adding a `--dry-run` or preview mode for blocklist changes so users can see which events would be filtered before applying them
- The `htmlToPlainText()` function is a best-effort HTML stripper; if description content becomes important for ICS consumers, consider integrating a proper HTML-to-text library
- Related ADRs:
  - **ADR-0005**: Browser abstraction layer (extended by `FetchBrightSpace` method on `AuthBrowser`)
  - **ADR-0006**: Authentication flow (BrightSpace SAML follows the same natural redirect pattern)
  - **ADR-0010**: Natural browser redirect for SAML exchange (applies to BrightSpace SAML at `/d2l/lp/auth/saml/login`)
  - **ADR-0012**: PeopleSoft list endpoint (complementary data source; both feed into the same `calendar.Event` model)

## References

- [BrightSpace D2L API Documentation](https://community.brightspace.com/wiki/display/public/API/Welcome+to+the+D2L+Learning+Management+System+API) — D2L Learning Management System API
- **ADR-0005**: Browser abstraction layer with Rod implementation (`AuthBrowser` interface extended)
- **ADR-0006**: Authentication flow and context isolation (SAML redirect pattern reused)
- **ADR-0010**: Natural browser redirect for ADFS→PeopleSoft SAML exchange (pattern applied to BrightSpace)
- **ADR-0012**: PeopleSoft timetable fetch to SSR_SSENRL_LIST endpoint (complementary data source)
- `internal/brightspace/` — New package containing models, client, filter, and timestamp conversion logic
- `internal/browser/browser.go:19` — `FetchBrightSpace` method on `AuthBrowser` interface
- `internal/calendar/projection.go:37` — `IsCampus()` filter excludes BrightSpace events
- `internal/calendar/cache.go:254` — `SaveOutputs()` writes five files instead of three
