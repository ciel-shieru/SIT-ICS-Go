# ADR-0001: ADFS Authentication and PeopleSoft Timetable Retrieval via HTTP Client

## Status

Proposed

## Date

2026-09-01

## Context

The application must fetch a Singapore Institute of Technology (SIT) student's weekly class timetable from the Oracle PeopleSoft instance at `https://in4sit.singaporetech.edu.sg/`. This PeopleSoft instance is protected by Microsoft ADFS at `https://fs.singaporetech.edu.sg/` as the identity provider, which in turn requires Azure MFA (TOTP) for authentication.

The authentication flow involves multiple steps:
1. An initial request to a PeopleSoft page triggers a 302 redirect to ADFS `idpinitiatedsignon.asmx` with a SAMLRequest parameter.
2. A `POST` with `AuthMethod=FormsAuthentication`, `UserName`, and `Password` to the ADFS login endpoint.
3. If MFA is enabled (which it is for SIT), ADFS returns an HTML form requiring `AuthMethod=AzureMfaAuthentication`, a `VerificationCode` (TOTP generated from a secret), `__EVENTTARGET` (blank), `SignIn=Sign in`, and `Context`.
4. ADFS validates the TOTP and posts a `SAMLResponse` back to the PeopleSoft Service Provider, setting session cookies (`PS_TOKEN`, `PSJSESSIONID`, etc.).
5. The authenticated session can then make AJAX `POST` requests to the PeopleSoft timetable endpoint (`SA_LEARNER_SERVICES.SSR_SSENRL_SCHD_W.GBL`) with form data specifying the week date, and receive an XML response containing HTML with the schedule data in table cells.

The application must support fetching timetables for every week from `START_DATE` to `END_DATE`, running on a cron schedule (default: daily at 1 AM), and operating entirely offline once deployed (no browser binaries or GUI dependencies).

## Decision Drivers

- **Security-first**: Credentials (username, password, TOTP secret) must never be logged, cached, or transmitted outside the auth flow.
- **Minimal dependencies**: Reduce third-party libraries where possible; only use well-maintained, Go 1.26-compatible packages.
- **Lightweight runtime**: No headless browser binaries (Chromium, Firefox) required.
- **Idempotent**: Repeated runs produce the same result; no duplicate ICS events.
- **Reliable authentication**: Must handle ADFS MFA flow correctly, including cookie persistence across requests.
- **Maintainable**: Clear separation between auth, data fetching, parsing, and ICS generation.

## Considered Options

### Option A: HTTP Client with Cookie Jar (Selected)

Use Go's `net/http` with a custom `CookieJar` to manage session cookies across the multi-step ADFS flow. Implement TOTP generation in-app using a minimal RFC 6238 implementation. Parse the PeopleSoft XML/HTML response using `encoding/xml` and `golang.org/x/net/html`.

### Option B: Headless Browser Automation

Use a Go binding for Playwright or Puppeteer (e.g., `mafredri/chromedp` or similar browser automation) to drive a headless Chromium instance through the login and scraping flow.

### Option C: Third-Party Authentication Library

Use an existing ADFS/SAML library (e.g., `crewjam/saml`) to handle the SAML flow, combined with an HTTP client for PeopleSoft requests.

## Decision

We will choose **Option A: HTTP Client with Cookie Jar**.

## Rationale

### Why Option A over Option B

- **No browser binary dependency**: Browser automation tools require a locally installed Chromium/Chrome binary, increasing deployment complexity and attack surface. The HTTP client approach uses only Go stdlib plus a few focused libraries.
- **Lower resource usage**: No full browser process means significantly less memory and CPU overhead, important for a background service running on a cron schedule.
- **Deterministic behavior**: Browser automation is susceptible to timing issues, DOM rendering differences, and browser version mismatches. HTTP client requests are deterministic and easier to test.
- **Reference precedent**: The `in4sit.el` Emacs package successfully implements the same ADFS flow using Emacs's `url-retrieve` (an HTTP client), proving the approach is viable.

### Why Option A over Option C

- **ADFS uses SAML 2.0 idP-initiated flow with custom form authentication**: The SIT ADFS instance uses a custom ASP.NET login form (`AuthMethod=FormsAuthentication`) rather than standard SAML Web SSO. Libraries like `crewjam/saml` expect a standard SAML redirect/POST flow and would not handle the intermediate FormsAuthentication → AzureMfaAuthentication steps.
- **TOTP is handled client-side**: The TOTP secret is provided by the user, and codes are generated on-demand. This is outside the scope of SAML libraries and must be implemented independently.
- **Over-engineering risk**: Introducing a full SAML library adds complexity for a flow that is essentially two HTTP POSTs and a cookie exchange.

### Why Option A over Status Quo (no solution)

This is a greenfield project with no existing implementation. Option A establishes a clear, minimal, and secure foundation.

## Rationale for Implementation Details

### TOTP Generation

Implement TOTP (RFC 6238) in-app rather than using a third-party library. The algorithm is well-defined and the implementation is short (~50 lines). This avoids a dependency whose maintenance status may be uncertain. The TOTP secret is read from the `TOTP_SECRET` env var and used only to generate time-based codes at login time.

### Cookie Management

Use Go's `net/http/cookiejar` package (stdlib) to automatically handle cookie persistence across requests. This ensures that session cookies set by ADFS's SAML callback are retained for subsequent PeopleSoft AJAX requests.

### PeopleSoft Response Parsing

The timetable endpoint returns an XML document (`Content-Type: text/xml; CHARSET=UTF-8`) with HTML content embedded in `<![CDATA[...]]>` sections within `<FIELD>` elements. Parse the outer XML using `encoding/xml` to extract the CDATA fields, then parse the HTML table using `golang.org/x/net/html` (a well-maintained, stdlib-adjacent package from the Go team) to extract schedule entries.

Each schedule entry is in a table row with:
- Time slot (e.g., "9:00AM")
- Day cells (Monday through Sunday) containing event blocks with course name, type, time range, and location
- Events span multiple rows via `rowspan` attributes

### ICS Generation

Implement a minimal ICS writer in-app. Since each fetch produces a static snapshot of weekly events (no RRULE recurrence), the ICS output is straightforward: one `VEVENT` per class meeting with `DTSTART`, `DTEND`, `SUMMARY`, and `LOCATION`. This keeps dependencies minimal and avoids RFC compliance edge cases in third-party libraries.

### Credential Handling

- Username, password, and TOTP secret are read from environment variables (`USERNAME`, `PASSWORD`, `TOTP_SECRET`) only.
- Credentials are never written to disk, logs, or memory beyond the duration of the auth request.
- The TOTP secret is used to generate a code, then the code is passed to the auth POST body and the secret is not retained.

## Consequences

### Positive
- Lightweight deployment: single binary, no browser dependencies, minimal system requirements.
- Secure: credentials are transient, TOTP is generated client-side, no sensitive data persisted.
- Testable: each step (ADFS login, MFA, PeopleSoft fetch, HTML parsing, ICS generation) can be unit tested in isolation with mock HTTP servers.
- Predictable: no browser timing issues or rendering differences.

### Negative
- Fragile to upstream changes: if SIT changes the PeopleSoft HTML structure or ADFS form fields, the parser/auth handler must be updated. This is inherent to any scraping approach.
- No automatic JavaScript execution: if PeopleSoft ever requires JS-rendered content, the HTTP client approach would need to be supplemented (but current data is server-rendered HTML in XML CDATA).
- TOTP implementation must be maintained: while simple, any bug in time-window handling could cause auth failures.

### Neutral / Operational
- Requires `TZ` environment variable (default `Asia/Singapore`) for all date/time operations.
- Cron schedule for periodic fetches requires a scheduler library (e.g., `github.com/robfig/cron/v3` — minimal, well-maintained, Go 1.26-compatible).
- In-memory ICS cache on startup provides fast responses; async background fetch keeps data fresh.

## Alternatives Considered

### Option B: Headless Browser Automation

**Summary**: Drive a headless Chromium instance through the ADFS login, MFA, and PeopleSoft scraping flow using `chromedp` or similar.

**Benefits**:
- Resilient to HTML structure changes (can use XPath/CSS selectors robustly).
- Handles any JavaScript rendering automatically.
- Proven approach: `timetable-grabber-sit` (Electron + Puppeteer) uses this successfully.

**Costs**:
- Requires Chromium/Chrome binary installation on the target system.
- High memory footprint (~200-400 MB per browser instance).
- Slower execution (browser startup + navigation).
- Harder to test deterministically (timing, race conditions).
- Larger attack surface (browser vulnerabilities).

**Reason rejected**: Overkill for the current requirements. The PeopleSoft data is server-rendered HTML in XML CDATA — no client-side JS rendering is needed. The security and operational costs outweigh the benefits.

### Option C: Third-Party SAML Library

**Summary**: Use `crewjam/saml` or similar to handle the ADFS SAML flow, combined with an HTTP client for PeopleSoft requests.

**Benefits**:
- Standard SAML handling reduces custom code.
- Well-tested SAML implementation.

**Costs**:
- SIT's ADFS uses a non-standard flow (FormsAuthentication → AzureMfaAuthentication) that SAML libraries do not support.
- Adds a large dependency with its own dependency tree.
- SAML libraries are designed for SP-initiated flows; SIT uses idP-initiated sign-on.

**Reason rejected**: The SIT ADFS flow is not a standard SAML Web SSO flow. The library cannot handle the intermediate form authentication and MFA steps. Custom HTTP client implementation is simpler and more appropriate.

## Follow-Ups

- ADR-0002: Configuration approach (env vars primary, CLI flags fallback via `caarlos0/env` and `spf13/pflag`).
- ADR-0003: ICS generation and caching strategy (in-memory cache with disk fallback, upsert semantics).
- ADR-0004: Cron scheduling and timezone handling.
- Implementation tracking lives outside this ADR.

## References

- [in4sit.el](https://github.com/achrinza/in4sit.el) — Emacs Lisp package implementing the same ADFS + PeopleSoft flow (active, 2025). Uses Emacs `url-retrieve` for HTTP requests, `auth-source` for credentials, and `libxml` for DOM parsing. Does not handle MFA or generate ICS.
- [timetable-grabber-sit](https://github.com/JustBrandonLim/timetable-grabber-sit) — Electron + Puppeteer application (archived Mar 2025). Uses headless browser automation, `ics` npm package for ICS generation. Handles basic login but not MFA.
- [Oracle PeopleSoft URL Format](https://docs.oracle.com/cd/E92519_02/pt856pbr3/eng/pt/tprt/concept_PortalURLFormats-c071f6.html)
- [RFC 6238](https://tools.ietf.org/html/rfc6238) — Time-Based One-Time Password (TOTP)
- [RFC 5545](https://tools.ietf.org/html/rfc5545) — Internet Calendaring and Scheduling Core Object Specification (iCalendar)
