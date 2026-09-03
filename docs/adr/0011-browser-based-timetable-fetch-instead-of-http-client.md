# ADR-0011: Browser-Based Timetable Fetch Instead of HTTP Client with Cookie Injection

## Status

Proposed

## Date

2026-09-03

## Context

ADR-0010 established that after ADFS authentication, the Rod browser extracts all `*.singaporetech.edu.sg` cookies and injects them into the `peoplesoft.Client` Go HTTP client via `SetCookies()`. The HTTP client then makes POST requests to the PeopleSoft timetable endpoint (`SA_LEARNER_SERVICES.SSR_SSENRL_SCHD_W.GBL`) with an attached `cookiejar`, receiving XML responses containing HTML schedule data.

In practice, this cookie injection bridge has proven unreliable: the PeopleSoft session established by the Rod browser consistently gets invalidated when the Go HTTP client reuses those cookies. The `cookiejar` and the browser maintain separate cookie stores, and PeopleSoft's session validation detects the mismatch — the HTTP client's `PSJSESSIONID` or `PS_TOKEN` does not correspond to an active server-side session that the browser authenticated.

The auth flow already uses a headless Chromium browser via Rod. The timetable fetch requires the same authenticated session. Maintaining two separate cookie stores (browser + Go `cookiejar`) and trying to synchronize them has led to increasingly fragile workarounds without resolving the root cause: PeopleSoft session state lives in the browser, not in the Go HTTP client.

## Decision Drivers

- **Single source of truth for session**: The authenticated session exists in the browser; the HTTP client cannot reliably mirror it
- **Simplify the data fetch path**: Eliminate the cookie injection bridge between browser and HTTP client
- **Reuse existing browser instance**: The Rod browser is already running after auth; use it for timetable fetch
- **Extract data from browser DOM**: The timetable endpoint response (XML with embedded HTML) can be extracted from the browser's network response or page content
- **Preserve existing parsing logic**: The `peoplesoft.parseTimetableHTML()` function remains unchanged; only the data source changes

## Considered Options

### Option A: Extend AuthBrowser with FetchTimetable Method

Extend the `AuthBrowser` interface with a `FetchTimetable(ctx, weekDate) (string, error)` method. After authentication, the browser navigates to the timetable endpoint, submits the week-date POST form, waits for the response, and extracts the HTML content from the page's DOM or response body. The auth flow calls this method directly.

### Option B: Create Separate TimetableBrowser

Create a new `TimetableBrowser` interface that handles timetable fetching independently. After auth, a new browser instance (or the same browser in a new incognito context) navigates through ADFS auth again and fetches the timetable. This keeps auth and data-fetching concerns separate.

### Option C: Fix Cookie Injection

Continue investing in the HTTP client approach by improving cookie synchronization — e.g., extracting cookies after each PeopleSoft response and re-injecting them, or using a shared cookie store between the browser and HTTP client.

### Option D: Browser-Only Application

Replace the entire HTTP client stack with browser-based fetching for all PeopleSoft interactions, including future endpoints beyond timetables.

## Decision

We will choose **Option A: Extend AuthBrowser with FetchTimetable Method**.

## Rationale

### Why Option A over Option B

Option B introduces a second authentication pass (or requires keeping the incognito context open across multiple fetches), which adds complexity and latency. Option A reuses the browser state established during auth — the same incognito context, same cookies, same session — making the timetable fetch a natural continuation of the auth flow. The AuthBrowser interface already owns the browser lifecycle; extending it with a data-fetching method is a minimal change that keeps all browser operations in one place.

### Why Option A over Option C

Option C addresses the symptom (cookie mismatch) rather than the root cause (PeopleSoft session lives in the browser). Every fix to cookie injection is a workaround — PeopleSoft's session validation may detect inconsistencies between the browser's session state and the HTTP client's cookie jar at any time. Option A eliminates the problem entirely by removing the HTTP client from the data fetch path.

### Why Option A over Option D

Option D is a broader architectural shift that would require replacing all HTTP-based interactions with browser automation. Option A is scoped to the timetable fetch only, which is the immediate pain point. Future endpoints can be evaluated individually.

### How It Works

The auth flow changes from:

```
AuthBrowser.Authenticate() → cookies → peoplesoft.Client.SetCookies() → peoplesoft.Client.FetchTimetable() (HTTP POST) → XML parse → HTML parse → entries
```

To:

```
AuthBrowser.Authenticate() → AuthBrowser.FetchTimetable(weekDate) → navigate to timetable endpoint, submit form, extract HTML from response → HTML parse → entries
```

The `peoplesoft.Client` is no longer used for fetching. It remains only for parsing (`parseTimetableHTML()`), which is unaffected. The `peoplesoft.Entry` type and all parsing logic stay the same.

The `FetchTimetable` method:
1. Navigates to the timetable endpoint URL with query parameters for the week date
2. Submits the POST form (`_PANEL_MODE=VIEW`, `_PROCESS=SSR_SSENRL_SCHD_W`, etc.)
3. Waits for the response to load (the timetable HTML appears in the page DOM)
4. Extracts the HTML content from the page
5. Returns the HTML string for downstream parsing

## Implementation Changes

### 1. `internal/browser/browser.go` — Extend AuthBrowser Interface

Add `FetchTimetable` method to `AuthBrowser`:

```go
type AuthBrowser interface {
    Authenticate(ctx context.Context, req AuthRequest) (AuthResult, error)
    FetchTimetable(ctx context.Context, weekDate string) (string, error)
}
```

Update `MockAuthBrowser` to include `FetchTimetableFunc`.

### 2. `internal/browser/local.go` — Implement FetchTimetable

Add `FetchTimetable` method to `LocalBrowser`:

```go
func (b *LocalBrowser) FetchTimetable(ctx context.Context, weekDate string) (string, error) {
    // Navigate to timetable endpoint with week date query params
    // Submit POST form with panel parameters
    // Wait for response HTML to appear in DOM
    // Extract and return HTML content
}
```

This method operates on the same browser instance used for auth. It navigates, submits the form, and extracts the timetable HTML using Rod's page APIs (`page.Element()`, `page.HTML()`).

### 3. `internal/browser/remote.go` — Implement FetchTimetable

Add equivalent `FetchTimetable` method to `RemoteBrowser`, following the same pattern as `LocalBrowser`.

### 4. `internal/auth/auth.go` — Add FetchTimetable to PeoplesoftClient Interface

Update the `PeoplesoftClient` interface and `ADFSProvider`:

```go
type PeoplesoftClient interface {
    FetchTimetable(ctx context.Context, weekDate string) ([]peoplesoft.Entry, error)
}

func (a *ADFSProvider) FetchTimetable(ctx context.Context, weekDate string) ([]peoplesoft.Entry, error) {
    html, err := a.browser.FetchTimetable(ctx, weekDate)
    if err != nil {
        return nil, fmt.Errorf("browser fetch timetable: %w", err)
    }
    return parseTimetableHTML(html)
}
```

Add a `parseTimetableHTML` helper in the auth package (or import from peoplesoft) to convert HTML to `[]peoplesoft.Entry`.

### 5. `internal/peoplesoft/peoplesoft.go` — Remove FetchTimetable HTTP Logic

Remove `FetchTimetable`, `SetCookies`, `httpClient`, `jar`, and all HTTP-related code from the `Client` struct. The package retains only:
- `Entry` struct (shared type)
- `parseTimetableHTML()` function (used by auth provider)

The `Client` struct can be removed entirely or reduced to a no-op placeholder if any external code references it.

### 6. `cmd/sit-ics/main.go` — Simplify Fetch Flow

Remove `peoplesoft.NewClient()`, `ps.SetCookies()`, and `ps.FetchTimetable()`. Replace with:

```go
entries, err := provider.FetchTimetable(ctx, weekDate)
```

### 7. `internal/server/server.go` — No Changes

The server serves the ICS file from cache. No changes needed — it already depends only on `ics.ICSCache`, not on the data fetch mechanism.

## Consequences

### Positive
- **Eliminates session invalidation**: The timetable fetch uses the same browser session that was authenticated, so there is no cookie mismatch
- **Simpler code**: Removes the entire cookie injection bridge (`SetCookies`, `cookiejar` management, multi-domain cookie extraction in peoplesoft package)
- **Fewer dependencies in peoplesoft package**: Removes `net/http`, `net/http/cookiejar`, `net/proxy`, and related imports from the peoplesoft package
- **Single browser lifecycle**: All PeopleSoft interactions go through the browser; no need to maintain parallel session state
- **More robust**: The browser handles all navigation, form submission, and response waiting natively — no manual HTTP request construction

### Negative
- **Slower fetch**: Browser navigation and DOM extraction is slower than an HTTP request (~1-2 seconds vs ~0.5 seconds per week date)
- **AuthBrowser interface grows**: The interface now covers both auth and data fetching, which is a slight violation of single responsibility
- **Harder to test in isolation**: Timetable fetching requires the full browser stack; unit tests for the fetch path are more complex
- **Browser dependency in auth package**: The auth provider now directly depends on browser capabilities beyond authentication

### Neutral / Operational
- The `peoplesoft` package becomes a parsing-only utility; its `Client` struct is removed
- The `PROXY_URL` config still applies to the browser (set via `BrowserConfig.ProxyURL` in `launchBrowser`)
- Debug logging (`BROWSER_DEBUG`) covers both auth and timetable fetch operations
- The `FetchTimetable` method reuses the browser's proxy configuration if set

## Alternatives Considered

### Option B: Separate TimetableBrowser

**Summary**: Create a new `TimetableBrowser` interface with its own browser instance for timetable fetching.

**Benefits**: Preserves separation of concerns; auth and data-fetching are independent.

**Costs**: Requires a second authentication pass (or keeping incognito context open); adds another abstraction layer; potential session inconsistency between the two browsers.

**Reason rejected**: The second auth pass adds latency and complexity. Keeping the incognito context open across multiple timetable fetches defeats its purpose (isolation). Option A is simpler and more reliable.

### Option C: Fix Cookie Injection

**Summary**: Improve the HTTP client approach by synchronizing cookies between browser and `cookiejar` — e.g., periodic re-injection, shared cookie store, or session validation before each request.

**Benefits**: Preserves the existing architecture; HTTP client remains faster.

**Costs**: Each fix is a workaround; PeopleSoft's session validation may detect inconsistencies at any time; the problem is structural (two separate cookie stores), not a bug that can be fixed.

**Reason rejected**: The fundamental issue is that PeopleSoft session state lives in the browser, not in a Go HTTP client. No amount of cookie synchronization can guarantee consistency.

### Option D: Browser-Only Application

**Summary**: Replace all HTTP-based interactions with browser automation across the entire application.

**Benefits**: Unified architecture; no mixed browser/HTTP patterns.

**Costs**: Significant architectural shift; affects all data fetching, not just timetables; larger impact on testing and deployment.

**Reason rejected**: Scoped to the immediate problem (timetable fetch). Future endpoints can be evaluated individually.

## Follow-Ups

- Implementation of `FetchTimetable` in `LocalBrowser` and `RemoteBrowser`
- Removal of HTTP client code from `peoplesoft` package
- Migration of `parseTimetableHTML` to `internal/auth` or kept as a standalone utility
- Related ADRs:
  - **ADR-0010**: Natural browser redirect for ADFS→PeopleSoft SAML exchange (superseded in part — cookie injection bridge removed)
  - **ADR-0005**: Browser abstraction layer (extended with `FetchTimetable`)
  - **ADR-0006**: Authentication flow (timetable fetch now part of browser operations)

## References

- [rod page APIs](https://go-rod.dev/) — `page.Element()`, `page.HTML()`, `page.WaitNavigation()`
- [Oracle PeopleSoft Timetable Endpoint](https://docs.oracle.com/cd/E92519_02/pt856pbr3/eng/pt/tprt/concept_PortalURLFormats-c071f6.html)
- **ADR-0010**: Cookie injection bridge that this ADR replaces
- **ADR-0005**: Browser abstraction layer (extended by this ADR)
