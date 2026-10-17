# ADR-0023: Browser Egress Control via Per-Page Request Interception and a Domain Whitelist

## Status

Accepted

## Date

2026-10-05

## Context

The automated browser holds live ADFS credentials, MFA TOTP, and PeopleSoft/BrightSpace session cookies while navigating the ADFS (`fs.`), PeopleSoft (`in4sit.`), BrightSpace (`xsite.`) and Microsoft login/CDN hosts. ADR-0016 validates only the final landing URL after the auth redirect (`isAllowedOrigin`). Nothing constrained the hosts the browser may actually *talk to*: a malicious redirect chain, a compromised page, or a malicious SAML response rendering attacker HTML could exfiltrate credentials/cookies to any external host via subresource or XHR requests.

The requirement is to intercept every request issued by the authenticated page's CDP session, check the hostname, and block everything off-whitelist; the whitelist includes `*.singaporetech.edu.sg` and is extensible via configuration. This per-page design does not cover out-of-process iframes using separate CDP sessions.

## Decision Drivers

- **Fail closed**: every error path and every unknown input must result in a block, never a silent allow
- **go-rod only**: no new dependencies; the interception mechanism must be provided by the existing Rod dependency
- **Login flow requires Microsoft hosts**: the ADFS/Entra ID login flow legitimately requires a specific set of Microsoft CDN/login hosts (verified live against the login pages)
- **Extensibility via config**: operators must be able to add domain suffixes without a code change
- **Per-request granularity**: the constraint must apply to every request (subresource, XHR, navigation), not just per-navigation
- **Testable without a browser**: the policy must be a pure function that can be unit-tested without Chromium

## Considered Options

### Option 1: Per-page `HijackRequests` (Selected)

Install a `HijackRouter` on the single authenticated page via `page.HijackRequests()`. The router is session-scoped to that page's CDP session, is Rod-native, and has a clean lifecycle (start in `AuthenticateADFS`, `Stop()` in `Close()`).

**Benefits**: Session-scoped, Rod-native, clean lifecycle, no new dependencies.
**Costs**: Does not cover out-of-page iframes (OOPIFs) in other sessions (see Consequences).

### Option 2: Browser-level `browser.HijackRequests`

Install the router at the browser level so it covers OOPIF sessions too.

**Benefits**: Would cover OOPIF sessions.
**Costs**: Intercepts every target in the process; coarser scope than needed for this flow. Kept as a documented upgrade path if OOPIF coverage becomes necessary.

### Option 3: Raw CDP `Fetch.enable` + `Fetch.requestPaused`

Handle the Fetch domain events directly.

**Benefits**: Same effect.
**Costs**: More boilerplate, no benefit over the Rod abstraction.

### Option 4: `Page.SetBlockedURLs`-style API

A URL-blocking API on the Page domain.

**Costs**: No such CDP domain exists; rejected.

### Option 5: Keep ADR-0016 check only

Rely solely on the post-redirect landing-URL validation.

**Costs**: Insufficient — landing-URL validation does not constrain in-flight subresource/XHR egress.

## Decision

We adopt **Option 1: per-page `HijackRequests`** with the all-URLs pattern (`""`, empty resource type) so every request type on the page's CDP session is intercepted.

- A **pure `DecideURL` policy** function (no I/O, no rod imports) maps a request URL to a decision:
  - **Enforced schemes** (`http`, `https`, `ws`, `wss`): the host must match the whitelist or the request is blocked.
  - **Exempt schemes** (`data`, `blob`, `about`, `chrome-extension`, `chrome`, `devtools`, `file`, `filesystem`): no network egress, allowed without a host check.
  - **Unknown schemes**: fail closed when they carry a host; allowed when they carry no host.
- A **suffix-based `HostWhitelist`** matches a host safely: `host == s || strings.HasSuffix(host, "."+s)`. This rejects suffix spoofing (`singaporetech.edu.sg.evil.com`) and evil-prefix hosts (`evilsingaporetech.edu.sg`) while allowing the apex and all subdomains.
- The **default whitelist** = the SIT institutional domain (`singaporetech.edu.sg`) plus 11 documented Microsoft Entra ID / ADFS login-flow domains.
- **`BROWSER_ALLOWED_HOSTS`** is additive (default + operator entries) and startup-validated (fail closed on any invalid entry).
- **`isAllowedOrigin`** is retained as the separate post-redirect tier — two-tier defense in depth.
- **Fail closed** on every error path (whitelist build, `Fetch.enable`, malformed/nil URLs).
- **Block events are always logged** (not debug-gated) since they are security-relevant.

## Security Properties

### What this prevents

| Attack Vector | Example | Blocked? |
|---------------|---------|----------|
| Suffix spoofing | `singaporetech.edu.sg.evil.com` | Yes — no suffix match (`.`-delimited) |
| Evil-prefix host | `evilsingaporetech.edu.sg` | Yes — not the apex, no `.singaporetech.edu.sg` suffix |
| Off-whitelist XHR/subresource | `https://evil.com/steal` | Yes — host not whitelisted |
| ws/wss exfil | `wss://evil.com/` | Yes — ws/wss are enforced schemes |
| Nil/malformed URL | `https://` (empty host) | Yes — fail closed |
| Unknown scheme with host | `foo://example.com/` | Yes — fail closed |

### What this allows

- Exact and subdomain matches on all whitelisted SIT and Microsoft login-flow hosts
- Port suffixes (e.g. `:8443`)
- Exempt no-egress schemes (`data:`, `blob:`, `about:blank`, `file:`, …)

## Consequences

### Positive

- **Per-page egress control**: requests issued in the authenticated page's CDP session are checked, and off-whitelist requests in that session are blocked. Out-of-process iframes that use separate CDP sessions are outside this boundary.
- **Extensible**: operators add domain suffixes via `BROWSER_ALLOWED_HOSTS` without a code change.
- **Pure policy fully unit-tested**: `DecideURL` and `HostWhitelist` are tested without a browser.
- **Defense in depth**: complements (does not replace) the ADR-0016 post-redirect `isAllowedOrigin` tier.

### Negative / Operational

- **OOPIF residual risk**: out-of-page iframes in other CDP sessions are not covered by the per-page router. Argued acceptable: an attacker-controlled OOPIF document request would itself be intercepted at the browser level if Option 2 were adopted; the app's three page families use no third-party OOPIFs in this flow.
- **Blocked navigations can surface as auth timeouts** on misconfiguration (e.g. an operator narrow whitelist that drops a legitimate Microsoft host).
- **Operator-added wide suffixes are the operator's responsibility** (a broad suffix widens the egress surface).

### Neutral

- Browser page caching is disabled during interception (Rod's hijack behavior).
- `isAllowedOrigin` tests are unchanged (separate tier).

## Follow-Ups

- **ADR-0016**: two-tier design (post-redirect `isAllowedOrigin` + per-request egress control).
- **ADR-0005**: `AuthBrowser` abstraction.
- **ADR-0010**: auth flow.
- **ADR-0020**: error classification — `ErrAuthentication` keeps interception failures fatal/non-retriable.

## References

- go-rod `HijackRequests` docs
- CDP Fetch domain
- [CWE-20: Improper Input Validation](https://cwe.mitre.org/data/definitions/20.html)
- [CWE-601: URL Redirection to Untrusted Site](https://cwe.mitre.org/data/definitions/601.html)
- [OWASP: Unvalidated Redirects and URL Manipulation](https://cheatsheetseries.owasp.org/cheatsheets/Unvalidated_Redirects_and_Forwards_Cheat_Sheet.html)
