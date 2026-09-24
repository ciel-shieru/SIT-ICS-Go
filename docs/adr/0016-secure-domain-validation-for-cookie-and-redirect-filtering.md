# ADR-0016: Secure Domain Validation for Cookie and Redirect Filtering

## Status

Accepted

## Date

2026-09-20

## Context

During a security review (HIGH-2), a substring domain matching vulnerability was identified in the codebase. The original `isSingaporeTechDomain()` helper function used `strings.Contains(domain, "singaporetech.edu.sg")` for domain validation. This approach is vulnerable to substring attacks where an attacker-controlled domain like `singaporetech.edu.sg.evil.com` would incorrectly pass validation.

The vulnerable code was removed in commit `b98f40c` ("refactor: remove dead cookie extraction code from Rod browser"). However, the auth flow still validates redirect URLs after ADFS authentication, and cookie filtering must be equally strict to prevent session hijacking via domain spoofing.

## Decision Drivers

- **Prevent domain spoofing**: Domain validation must reject domains that merely contain the target string as a substring
- **Defend against SSRF/session hijacking**: Weak domain checks allow attackers to redirect authenticated sessions to attacker-controlled domains
- **Principle of least privilege**: Only explicitly whitelisted SIT domains should be accepted
- **Defense in depth**: Even though the vulnerable code was removed, the remaining auth flow needs equally strict validation

## Considered Options

### Option A: Exact match only (reject subpaths)

Only accept URLs that exactly match the allowed origins. Reject all subpaths.

**Benefits**: Simplest possible check.
**Costs**: Would break legitimate subpath navigation (e.g., `/psc/CSSISSTD/...`).

### Option B: Prefix match with delimiter enforcement (Selected)

Allow exact matches and prefix matches where the prefix is followed by `/` or `:` (port separator). This prevents `singaporetech.edu.sg.evil.com` from matching because `.evil.com` follows the domain without a `/` or `:` delimiter.

**Benefits**: Correctly allows legitimate subpaths and ports while rejecting domain spoofing.
**Costs**: Slightly more complex than exact-only matching.

### Option C: URL parsing with net/url

Parse URLs and validate the hostname component separately using `net/url` package.

**Benefits**: Most semantically correct approach.
**Costs**: Overhead of parsing, potential edge cases with URL normalization.

## Decision

We will choose **Option B: Prefix match with delimiter enforcement**.

The `isAllowedOrigin()` function in `internal/browser/browser.go` uses an allowlist of full origin strings (`https://in4sit.singaporetech.edu.sg`, `https://fs.singaporetech.edu.sg`, `https://xsite.singaporetech.edu.sg`) and validates incoming URLs by:

1. Checking for exact equality with an allowed origin
2. If no exact match, checking if the URL starts with an allowed origin AND the next character is `/` or `:`

```go
func isAllowedOrigin(url string) bool {
    for _, origin := range allowedOrigins {
        if url == origin || len(url) >= len(origin) && url[:len(origin)] == origin && (url[len(origin)] == '/' || url[len(origin)] == ':') {
            return true
        }
    }
    return false
}
```

This prevents substring attacks because `singaporetech.edu.sg.evil.com` does not match any allowed origin prefix (none of the origins start with `singaporetech.edu.sg` — they all have subdomain prefixes like `in4sit.`, `fs.`, `xsite.`).

## Security Properties

### What this prevents

| Attack Vector | Example | Blocked? |
|---------------|---------|----------|
| Substring suffix | `singaporetech.edu.sg.evil.com` | Yes — no allowed origin has this prefix |
| Substring with path | `in4sit.singaporetech.edu.sg.evil.com/path` | Yes — `.evil.com` follows without `/` or `:` |
| Prefix confusion | `notin4sit.singaporetech.edu.sg` | Yes — not a prefix of any allowed origin |
| TLD confusion | `in4sit.singaporetech.edu` | Yes — shorter than allowed origin |
| Scheme confusion | `http://in4sit.singaporetech.edu.sg` | Yes — scheme mismatch |
| Port attack | `in4sit.singaporetech.edu.sg:8443` | Allowed — port is legitimate navigation |

### What this allows

- Exact matches on all allowed origins
- Subpath navigation (e.g., `/psc/CSSISSTD/SA_LEARNER_SERVICES.SSR_SSENRL_LIST.GBL`)
- Port suffixes (e.g., `:8443`)
- Query strings and fragments (included in subpath)

## Implementation

### Current code (secure)

`internal/browser/browser.go:40-47`:
```go
func isAllowedOrigin(url string) bool {
    for _, origin := range allowedOrigins {
        if url == origin || len(url) >= len(origin) && url[:len(origin)] == origin && (url[len(origin)] == '/' || url[len(origin)] == ':') {
            return true
        }
    }
    return false
}
```

### Usage

`internal/browser/auth.go:33` — Used to validate the redirect URL after ADFS authentication completes, preventing SSRF via malicious redirect targets.

## Regression Tests

Added comprehensive tests in `internal/browser/browser_test.go`:

- `TestIsAllowedOrigin` — 25 test cases covering exact matches, subpaths, ports, and all known attack vectors
- `TestIsAllowedOriginNoSubstringVulnerability` — Dedicated regression test for HIGH-2 substring attacks

## Consequences

### Positive

- **Eliminates domain spoofing**: Substring attacks like `singaporetech.edu.sg.evil.com` are correctly rejected
- **Maintains functionality**: Legitimate subpath navigation and port suffixes continue to work
- **Testable**: Comprehensive regression tests prevent future regressions
- **Simple**: No external dependencies, no URL parsing overhead

### Negative

- **Hardcoded allowlist**: Adding new domains requires code changes (but this is intentional for security)
- **No DNS validation**: The check is string-based only (acceptable since the auth flow already establishes trust via browser interaction)

### Neutral / Operational

- The delimiter-based prefix check is slightly more complex than exact matching but necessary for correctness
- The allowlist is small (3 entries) and unlikely to change frequently

## Follow-Ups

- Related ADRs:
  - **ADR-0010**: Natural browser redirect for ADFS→PeopleSoft SAML exchange (introduces the auth flow that uses `isAllowedOrigin`)
  - **ADR-0005**: Browser abstraction layer (the `AuthBrowser` interface that uses this validation)

## References

- [CWE-601: URL Redirection to Untrusted Site ('Open Redirect')](https://cwe.mitre.org/data/definitions/601.html)
- [CWE-20: Improper Input Validation](https://cwe.mitre.org/data/definitions/20.html)
- [OWASP: Unvalidated Redirects and Forwards Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Unvalidated_Redirects_and_Forwards_Cheat_Sheet.html)
