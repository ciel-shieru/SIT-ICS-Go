# ADR-0010: Natural Browser Redirect for ADFS→PeopleSoft SAML Exchange

## Status

Accepted

## Date

2026-09-03

## Context

ADR-0009 proposed extracting the `SAMLResponse` hidden form field from the ADFS response HTML and manually POSTing it to the PeopleSoft landing page. This approach was implemented but revealed a critical flaw: after MFA/TOTP authentication completes, ADFS triggers a JavaScript-based redirect to PeopleSoft rather than rendering a static HTML form with `SAMLResponse`.

The implementation attempted to extract `SAMLResponse` by parsing the page HTML after MFA submission, but the page had already been destroyed by the redirect:

```
2026/09/03 13:13:43 browser: wait load after MFA failed: {-32000 Execution context was destroyed. }
2026/09/03 13:13:45 browser: checking for MFA error message
2026/09/03 13:13:45 browser: extracting SAMLResponse from ADFS page
2026/09/03 13:13:45 scheduler: auth failed: adfs authenticate: authentication failed: failed to extract SAMLResponse: SAMLResponse input not found in page
```

The `-32000 Execution context was destroyed` error indicates that Rod's page context was invalidated when ADFS navigated away via JavaScript. The subsequent `SAMLResponse input not found in page` error confirms that by the time HTML parsing was attempted, the page had already been replaced by the redirect target.

## Decision Drivers

- **Browser handles SAML naturally**: The browser already follows ADFS redirects automatically — no manual SAMLResponse extraction needed
- **Eliminate brittle HTML parsing**: Removing SAMLResponse extraction removes a fragile code path that depends on ADFS response structure
- **Simpler auth flow**: Fewer code paths = fewer bugs = easier to maintain
- **All cookies captured**: The browser naturally receives all PeopleSoft session cookies after the redirect
- **Cookie bridge**: Browser-extracted cookies must be injected into the Peoplesoft HTTP client

## Considered Options

### Option A: Let Browser Follow ADFS Redirect Naturally (Selected)

After MFA/TOTP authentication completes, wait for the browser to naturally follow the ADFS redirect to PeopleSoft. Extract all cookies from `*.singaporetech.edu.sg` domains (both `in4sit.singaporetech.edu.sg` and `fs.singaporetech.edu.sg`) after the redirect completes. Inject these cookies into the Peoplesoft HTTP client via `SetCookies()`.

### Option B: Fix SAMLResponse Extraction Timing

Attempt to extract SAMLResponse before the redirect occurs, or use CDP protocol events to intercept the SAML POST. This would require deeper Rod/CDP integration and careful timing.

### Option C: Browser-Only Timetable Fetch

Perform the entire timetable fetch inside the browser after ADFS auth, eliminating the need for cookie injection between browser and HTTP client.

## Decision

We will choose **Option A: Let Browser Follow ADFS Redirect Naturally**.

## Rationale

### Why Option A over Option B

Option B requires intercepting ADFS's JavaScript-driven redirect, which is fragile and depends on ADFS implementation details. The redirect may use `window.location`, `form.submit()`, or other mechanisms that are difficult to intercept reliably. Option A leverages the browser's natural redirect behavior, which is robust and doesn't depend on timing or ADFS internals.

### Why Option A over Option C

Option C would merge auth and data-fetching concerns, violating the existing separation of concerns in the architecture. The Peoplesoft HTTP client has existing parsing logic (`parseTimetableHTML()`) that would need to be duplicated or refactored. Option A preserves this separation while fixing the authentication gap.

### How It Works

The auth flow is now:

1. **Navigate to PeopleSoft** (`https://in4sit.singaporetech.edu.sg`) → 302 redirect to ADFS
2. **Fill credentials** (username, password, submit)
3. **Handle MFA** if present (fill TOTP code, submit)
4. **Wait for ADFS redirect** — browser naturally follows the SAML redirect to PeopleSoft
5. **Wait for PeopleSoft landing page** to load and stabilize
6. **Extract all cookies** from `*.singaporetech.edu.sg` domains
7. **Return cookies** in `AuthResult.Cookies`
8. **Inject cookies** into Peoplesoft client via `SetCookies()` before fetching timetables

### Cookie Extraction Strategy

The `extractCookies()` method in both `LocalBrowser` and `RemoteBrowser`:

1. First attempts to extract cookies from the current page URL (PeopleSoft landing page)
2. If no cookies found, queries both `https://in4sit.singaporetech.edu.sg/` and `https://fs.singaporetech.edu.sg/`
3. Filters cookies to only include those with `singaporetech.edu.sg` in the domain
4. Returns all matching cookies

The Peoplesoft client's `SetCookies()` method:

1. Sets cookies on the Peoplesoft base URL
2. Additionally sets cookies on both `in4sit.singaporetech.edu.sg` and `fs.singaporetech.edu.sg` domains to ensure ADFS cookies (like `MSISAuth`) are available for subsequent requests

## Implementation Changes

### 1. `internal/browser/browser.go` — Remove SAMLResponse

Removed `SAMLResponse string` field from `AuthResult`. The browser no longer carries SAMLResponse data.

### 2. `internal/browser/local.go` and `internal/browser/remote.go` — Simplify Auth Flow

- Removed `extractSAMLResponse()` function entirely
- Removed SAMLResponse extraction and POST logic from `navigateAndAuth()` / `connectAndAuth()`
- After MFA submission, wait for the natural ADFS redirect to complete (`WaitLoad`, `WaitStable`, `WaitNavigation`)
- Rewrote `extractCookies()` to extract from all `*.singaporetech.edu.sg` domains instead of a single hardcoded URL
- Added `isSingaporeTechDomain()` helper function

### 3. `internal/peoplesoft/peoplesoft.go` — Add Cookie Injection

Added `SetCookies([]browser.Cookie)` method to the `Client` struct. This method:
- Converts browser cookies to `*http.Cookie`
- Sets them on the Peoplesoft base URL via the cookie jar
- Also sets ADFS-domain cookies (`fs.singaporetech.edu.sg`) and Peoplesoft-domain cookies (`in4sit.singaporetech.edu.sg`) separately

### 4. `cmd/sit-ics/main.go` — Inject Cookies After Auth

The main function now captures `AuthResult` from `provider.Authenticate()` and calls `ps.SetCookies(authResult.Cookies)` before fetching timetables.

### 5. Tests

Updated `browser_test.go` and `auth_test.go` to remove all `SAMLResponse` references from mock browser results and test assertions.

## Consequences

### Positive

- **Fixes the core bug**: The "Execution context was destroyed" error is eliminated because we no longer try to parse a page that's being redirected away
- **Simpler code**: Removing SAMLResponse extraction removes ~50 lines of HTML parsing code and multiple error paths
- **More robust**: The browser naturally handles all redirect types (JavaScript, meta refresh, form POST) without code changes
- **All cookies captured**: Both ADFS cookies (`MSISAuth`, etc.) and PeopleSoft session cookies are extracted and used
- **Clear separation**: Auth (browser) and data fetching (HTTP client) remain separate, with cookies as the bridge

### Negative

- **Less explicit control**: We no longer have direct access to the SAMLResponse value (though we never needed to inspect it)
- **Cookie extraction timing**: Cookies must be extracted after the redirect completes, which requires waiting for page stability

### Neutral / Operational

- The auth flow is slightly simpler but still requires waiting for redirects, so total auth time is similar
- Cookie extraction from multiple domains adds a small amount of overhead (~100ms per domain query)

## Alternatives Considered

### Option B: Fix SAMLResponse Extraction Timing

**Summary**: Use CDP protocol events or Rod's navigation event system to intercept the SAML POST before it completes, extract the SAMLResponse from the pending request body.

**Benefits**: Would preserve the explicit SAML flow described in the HAR capture.

**Costs**: Requires deep CDP integration, is fragile to ADFS changes, and adds complexity to an already timing-sensitive code path.

**Reason rejected**: The browser already handles this automatically. There's no benefit to manually intercepting what the browser does naturally.

### Option C: Browser-Only Timetable Fetch

**Summary**: After ADFS auth completes, navigate the browser to the timetable page, extract the HTML, and parse it.

**Benefits**: Eliminates cookie injection complexity entirely.

**Costs**: Merges auth and data-fetching concerns, duplicates parsing logic, slower (~1-2s vs ~0.5s), harder to test.

**Reason rejected**: Violates the existing separation of concerns. The HTTP client approach is faster and more testable.

## Follow-Ups

- Related ADRs:
  - **ADR-0009**: SAMLResponse submission approach (superseded by this ADR)
  - **ADR-0005**: Browser abstraction layer (unchanged; auth flow simplified)
  - **ADR-0006**: Authentication flow (superseded in part — no longer extracts SAMLResponse)

## References

- Chrome DevTools Protocol error `-32000`: Execution context was destroyed
- [Rod WaitLoad documentation](https://go-rod.dev/#/wait?id=waitload) — Rod page waiting primitives
- [Oracle PeopleSoft Cookie Documentation](https://docs.oracle.com/cd/E72013_01/ptools11pbr1/eng/pt/tpccsd/RPCOOKIE-150039.html) — PeopleSoft cookie names and purposes
