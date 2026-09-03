# ADR-0009: Complete PeopleSoft Authentication via SAMLResponse Submission

## Status

Proposed

## Date

2026-09-03

## Context

The application authenticates against Singapore Institute of Technology's (SIT) Microsoft ADFS at `https://fs.singaporetech.edu.sg/` to access the Oracle PeopleSoft timetable system at `https://in4sit.singaporetech.edu.sg/`. The current implementation (ADR-0005, ADR-0006) performs browser automation through ADFS credentials and MFA, extracts cookies, and then attempts to fetch the timetable via HTTP.

**However, the timetable fetch always fails with a redirect to ADFS.** This indicates that PeopleSoft rejects unauthenticated requests and redirects them back to the identity provider.

A HAR capture of a successful in-browser authentication flow (recorded 2026-09-03 via Chrome DevTools) reveals that the current implementation is **missing a critical intermediate step** between ADFS authentication and PeopleSoft HTTP requests: the SAMLResponse submission to PeopleSoft.

### The Complete Flow (from HAR capture)

The successful flow has three distinct phases:

**Phase 1 — ADFS Authentication (current code does this):**
1. Navigate to `https://in4sit.singaporetech.edu.sg` → 302 redirect to ADFS `idpinitiatedsignon.asmx`
2. Fill username, password, submit
3. Fill TOTP code, submit (`AuthMethod=AzureMfaAuthentication&Context=<base64>&VerificationCode=<6-digit>`)
4. ADFS returns HTML containing a hidden `<input name="SAMLResponse">` form field
5. ADFS sets cookies: `MSISAuth`, `MSISAuth1`, `SamlSession`, `MSISAuthenticated`, `MSISLoopDetectionCookie` (all on `fs.singaporetech.edu.sg`)

**Phase 2 — SAMLResponse POST to PeopleSoft (current code skips this):**
6. Browser POSTs to `https://in4sit.singaporetech.edu.sg/psc/CSSISSTD/EMPLOYEE/SA/c/NUI_FRAMEWORK.PT_LANDINGPAGE.GBL`
7. POST body: `SAMLResponse=<base64-encoded-SAML-2.0-assertion>`
8. PeopleSoft validates the SAML assertion, extracts the user identity (`1234567@sit.singaporetech.edu.sg`), and establishes a session
9. PeopleSoft sets **13+ cookies** on `.singaporetech.edu.sg`:
   - `AWSSISWEBPRD02-8002-PORTAL-PSJSESSIONID` — AWS Application Load Balancer session cookie (HttpOnly)
   - `PS_TOKEN` — PeopleSoft CSRF/session token (HttpOnly, Secure)
   - `PS_TOKENEXPIRE` — Token expiration timestamp
   - `PS_LASTSITE`, `ExpirePage`, `PS_TokenSite`, `PS_LOGINLIST` — Navigation/session state
   - `ps_theme` — UI theme configuration
   - `AWSALB`, `AWSALBTG`, `AWSALBCORS`, `AWSALBTGCORS` — AWS ALB stickiness cookies
   - `PS_DEVICEFEATURES` — Browser capability detection

**Phase 3 — Timetable Fetch (current code attempts this but fails):**
10. GET/POST to `/psc/CSSISSTD/EMPLOYEE/SA/c/SA_LEARNER_SERVICES.SSR_SSENRL_SCHD_W.GBL`
11. Requires all cookies from Phase 2 to be present in the request
12. Returns XML with embedded HTML timetable data

### Current Implementation Gap

The current code flow:
```
Browser auth (ADFS) → Extract cookies → PeopleSoft HTTP fetch (no session cookies) → Redirect to ADFS
```

What it should do:
```
Browser auth (ADFS) → Extract SAMLResponse → POST SAMLResponse to PeopleSoft → Extract PeopleSoft cookies → PeopleSoft HTTP fetch (with session cookies) → XML timetable data
```

The `PeopleSoftClient.FetchTimetable()` method in `internal/peoplesoft/peoplesoft.go` creates an `http.Client` with **no cookie jar**. Even if the browser correctly extracts PeopleSoft cookies, they are never passed to the HTTP client. The `extractPS_TOKEN()` function in `internal/auth/auth.go` only extracts the `PS_TOKEN` cookie and discards all other cookies.

### Evidence from HAR Capture

The HAR file shows that after the SAMLResponse POST (entry 3), the PeopleSoft response sets a complete set of session cookies. Subsequent requests (entries 4-7) include all these cookies in their `Cookie` header. The timetable AJAX request (entry 7) sends 18 cookies including `PS_TOKEN`, `AWSSISWEBPRD02-8002-PORTAL-PSJSESSIONID`, `AWSALB*`, `PS_DEVICEFEATURES`, `psback`, and others. Without this full cookie suite, PeopleSoft cannot identify the session and redirects to ADFS.

## Decision Drivers

- **Complete auth flow**: Every step from ADFS login through PeopleSoft session establishment must be implemented
- **Cookie persistence**: The HTTP client must automatically persist and send cookies across all PeopleSoft requests
- **SAMLResponse extraction**: The browser must parse the ADFS response to extract the SAML assertion for submission to PeopleSoft
- **All-session cookies**: Extract and use all PeopleSoft session cookies, not just PS_TOKEN
- **Backwards compatibility**: The change should not break existing functionality; the ICS generation and caching pipeline remains unchanged
- **Minimal browser dependency**: SAMLResponse extraction should be done via HTML parsing in Go, not additional browser dependencies

## Considered Options

### Option A: Browser POSTs SAMLResponse to PeopleSoft, HTTP Client Uses Cookie Jar (Selected)

After ADFS authentication completes, the browser automation code parses the response HTML to extract the `SAMLResponse` hidden form field value. It then automatically POSTs this value to the PeopleSoft landing page URL, which triggers PeopleSoft to establish a session and set all required cookies. The browser extracts all `.singaporetech.edu.sg` cookies after this POST. The `PeopleSoftClient` is updated to use `net/http/cookiejar` so that all subsequent HTTP requests automatically include the session cookies.

### Option B: Browser-Only Timetable Fetch

Perform the entire timetable fetch inside the browser (navigate to the timetable page, extract HTML, parse it). This eliminates the need for a separate HTTP client and cookie jar entirely.

### Option C: Reuse Existing Browser Cookies Without SAML POST

Extract cookies from the ADFS domain and hope that PeopleSoft accepts them. This is the current (broken) approach.

## Decision

We will choose **Option A: Browser POSTs SAMLResponse to PeopleSoft, HTTP Client Uses Cookie Jar**.

## Rationale

### Why Option A over Option B

- **Separation of concerns**: The current architecture separates browser automation (auth) from data fetching (HTTP client). Option A preserves this separation. Option B would merge them, requiring HTML parsing logic inside the browser package and changing the existing timetable fetch pipeline.
- **Existing parsing logic**: The `peoplesoft.parseTimetableHTML()` function already correctly parses PeopleSoft HTML. Moving the fetch into the browser would require duplicating or refactoring this logic.
- **Testability**: The HTTP client can be tested independently with mock servers. A browser-based fetch would require browser mocks for timetable parsing tests.
- **Performance**: Browser navigation to the timetable page is slower (~1-2 seconds) than an HTTP POST (~0.5 seconds). The cookie jar approach is faster.
- **Flexibility**: If PeopleSoft ever changes its HTML structure but keeps the XML API, only the parsing logic needs updating. With browser-based fetching, both the navigation and parsing would need changes.

### Why Option A over Option C

Option C is the current implementation and it **does not work**. PeopleSoft rejects requests that lack a valid session (no `AWSSISWEBPRD02-8002-PORTAL-PSJSESSIONID` and `PS_TOKEN` cookies), returning a 302 redirect to ADFS. This has been verified through both the HAR capture and the application's actual behavior.

### Why Cookie Jar Is Necessary

PeopleSoft uses multiple cookies for session management:
- `AWSSISWEBPRD02-8002-PORTAL-PSJSESSIONID` is the primary session identifier
- `PS_TOKEN` is used for CSRF protection on every POST
- `PS_TOKENEXPIRE` tracks token validity
- AWS ALB cookies (`AWSALB*`) provide load balancer stickiness

Go's `net/http/cookiejar` automatically handles:
- Cookie storage and retrieval across requests
- Domain and path matching
- Expiration and security attributes (Secure, HttpOnly)
- Cookie updates from `Set-Cookie` headers on each response

Manual cookie management would be error-prone and fragile — any new cookie added by PeopleSoft would require code changes. The cookie jar handles this transparently.

### Why Extract All Cookies, Not Just PS_TOKEN

The HAR capture shows that PeopleSoft sets 13+ cookies during the SAML POST. Only extracting `PS_TOKEN` discards the session cookie (`AWSSISWEBPRD02-8002-PORTAL-PSJSESSIONID`) which is the primary session identifier. Without it, PeopleSoft cannot associate the HTTP request with the authenticated session.

### Why Parse SAMLResponse in Go Rather Than Using a SAML Library

The SAMLResponse is already available as a hidden form field value in the ADFS response HTML. Extracting it requires a simple DOM parse (find `<input name="SAMLResponse">`, read its `value` attribute). This is far simpler than integrating a full SAML library like `crewjam/saml`, which:
- Is designed for SP-initiated flows, not idP-initiated
- Cannot handle the intermediate FormsAuthentication and AzureMfaAuthentication steps
- Adds a large dependency tree
- Would not help with the actual problem (which is session establishment, not SAML parsing)

## Implementation Changes

### 1. `internal/browser/browser.go` — Extend `AuthResult`

Add a `SAMLResponse string` field to `browser.AuthResult` to carry the extracted SAML assertion from ADFS back to the caller.

### 2. `internal/browser/local.go` and `internal/browser/remote.go` — Two-Phase Auth

After the MFA sign-in completes:
- Parse the current page HTML to extract the `SAMLResponse` hidden form field value
- POST the SAMLResponse to `https://in4sit.singaporetech.edu.sg/psc/CSSISSTD/EMPLOYEE/SA/c/NUI_FRAMEWORK.PT_LANDINGPAGE.GBL` with body `SAMLResponse=<value>`
- Wait for the PeopleSoft landing page to load (navigation complete + stable)
- Extract all cookies for `.singaporetech.edu.sg` domain (not just ADFS cookies)
- Return the SAMLResponse in `AuthResult.SAMLResponse`

### 3. `internal/peoplesoft/peoplesoft.go` — Add Cookie Jar

Add a `*http.cookiejar` to the `Client` struct. Initialize it with `cookiejar.New(nil)` in `NewClient()`. Set `httpClient.Jar = jar`. This ensures all cookies are automatically persisted and sent with requests.

### 4. `internal/auth/auth.go` — Thread SAMLResponse Through

Pass the `SAMLResponse` from browser auth through to the caller in `AuthResult`. The `PeopleSoftURL` field on `AuthRequest` (currently dead code) is now used to determine the PeopleSoft landing page URL for the SAML POST.

### 5. Tests

Update `auth_test.go` to verify SAMLResponse extraction and PeopleSoft cookie handling. Update `peoplesoft_test.go` to verify the cookie jar is used in requests.

## Consequences

### Positive
- **Fixes the core bug**: PeopleSoft requests will now include a valid session, eliminating the ADFS redirect
- **Complete session**: All PeopleSoft session cookies are captured and used, not just PS_TOKEN
- **Automatic cookie management**: The cookie jar handles domain/path matching, expiration, and security attributes
- **Preserves existing architecture**: Browser automation and HTTP fetching remain separate concerns
- **No new dependencies**: Uses `net/http/cookiejar` (stdlib) and simple HTML parsing

### Negative
- **Longer auth flow**: Adding the SAML POST step adds ~1-2 seconds to the authentication process
- **More brittle HTML parsing**: Extracting SAMLResponse from HTML depends on the ADFS response structure remaining unchanged
- **Cookie jar overhead**: Minimal (~KB of memory), but non-zero
- **Two-phase auth complexity**: The browser now performs two distinct navigation cycles (ADFS → PeopleSoft landing), which adds code paths that need testing

### Neutral / Operational
- The SAMLResponse is transient — only used for the POST, never persisted or logged
- Cookie jar state is per-`PeopleSoftClient` instance; each fetch operation uses the same client with its cookie store
- The `PS_TOKENEXPIRE` cookie is updated on each response, so the cookie jar automatically manages token freshness
- AWS ALB cookies are rotated by the load balancer; the cookie jar handles this transparently

## Alternatives Considered

### Option B: Browser-Only Timetable Fetch

**Summary**: Navigate to the timetable page inside the browser after SAML POST, extract the HTML response, and parse it in the browser package.

**Benefits**:
- No separate HTTP client needed for timetable fetching
- All state (cookies, session) is naturally available in the browser
- Eliminates cookie jar complexity

**Costs**:
- Merges auth and data-fetching concerns — violates the existing separation of concerns
- Requires duplicating or refactoring `peoplesoft.parseTimetableHTML()` into the browser package
- Slower: browser navigation (~1-2s) vs HTTP POST (~0.5s)
- Harder to test: requires browser mocks for timetable parsing
- Less flexible: if PeopleSoft changes HTML but keeps the XML API, both navigation and parsing need changes

**Reason rejected**: The existing architecture cleanly separates browser automation (auth) from data fetching (HTTP client with parsing). Option A preserves this separation while fixing the authentication gap. Option B would require significant refactoring of the parsing pipeline.

### Option C: Maintain Status Quo

**Summary**: Keep the current implementation as-is.

**Benefits**:
- No code changes required
- No new code paths to test

**Costs**:
- Timetable fetch always fails (302 redirect to ADFS)
- Application is non-functional for its primary purpose
- Users must manually use a browser to access timetables

**Reason rejected**: The application is unusable in its current state. The HAR capture confirms the missing step and provides the exact fix.

## Follow-Ups

- Implementation tracking lives outside this ADR
- Related ADRs:
  - **ADR-0005**: Browser abstraction layer (unchanged; SAML POST is an extension of the existing auth flow)
  - **ADR-0006**: Authentication flow (superseded in part — the flow now includes SAML POST to PeopleSoft after ADFS auth)
  - **ADR-0001**: Original ADFS auth design (already superseded; this further extends the auth flow)

## References

- HAR capture: `in4sit.singaporetech.edu.sg-2.har` — Chrome DevTools recording of successful in-browser auth flow (2026-09-03)
- [Oracle PeopleSoft Cookie Documentation](https://docs.oracle.com/cd/E72013_01/ptools11pbr1/eng/pt/tpccsd/RPCOOKIE-150039.html) — PeopleSoft cookie names and purposes
- [RFC 7230 Section 5.4](https://datatracker.ietf.org/doc/html/rfc7230#section-5.4) — HTTP Cookie header
- [Go net/http/cookiejar](https://pkg.go.dev/net/http/cookiejar) — Standard library cookie jar implementation
- [SAML 2.0 Web SSO](https://docs.oasis-open.org/security/saml/v2.0/saml-profiles-2.0-os.html#web-sso) — SAML Web Single Sign-On profile
