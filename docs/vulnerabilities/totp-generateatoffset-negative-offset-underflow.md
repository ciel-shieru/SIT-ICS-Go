# VULN-2026-001: TOTP GenerateAtOffset negative-offset underflow (Low)

- **Status**: Open (not yet fixed)
- **Reported**: 2026-11-27
- **Component**: internal/totp (GenerateAtOffset)
- **Consumer**: ADFS MFA authentication (internal/browser/auth.go)
- **Affected versions**: all released versions (v0.0.21-v0.0.31) and main (as of a4bfb89)
- **CVSS v3.1**: AV:L/AC:H/PR:N/UI:N/S:U/C:N/I:N/A:L — base score 0.1 — **Low** (borderline "None")
- **CWE**: CWE-681 (Incorrect Conversion between Numeric Types), related CWE-194

## Description

GenerateAtOffset generates TOTP candidate codes across +/-tolerancePeriods time
steps to absorb clock skew between code generation and submission. For negative
offsets it computes:

~~~go
step := currentStep + uint64(offset)
~~~

Converting a negative int offset to uint64 wraps around
(uint64(-1) = 2^64 - 1), so a negative offset does not step back one period —
it produces a huge, nonsensical time step. The subsequent guard:

~~~go
if step == 0 && offset < 0 {
    continue
}
~~~

only catches the wrapped sum landing exactly on zero, so the underflowed step is
normally passed straight to totp.GenerateCode.

## Evidence / reproduction

Deterministic, no network or browser required:

~~~
$ go test -count=1 ./internal/totp/
--- FAIL: TestGenerateAtOffset (0.00s)
    --- FAIL: TestGenerateAtOffset/-1_period (0.00s)
        totp_test.go:132: GenerateAtOffset() = 755224, want 094451
FAIL
~~~

With the pinned pquerna/otp v1.5.0, the underflowed (negative) timestamp is
rejected and the loop falls through to offset 0 — the current step — which is
why MFA authentication still succeeds today and the observed value is the
RFC 4226 vector for counter 0 (755224). The test expectation 094451 matches
the buggy wrapped-step path under an older dependency behavior, and the
+1_period expectation (287082 for now=89s) matches the buggy path that wraps
to step 1. No correct implementation of the documented +/-1 tolerance produces
both expectations: the RFC 6238 vectors for the pinned RFC test secret are
step 0 -> 755224, step 1 -> 287082, step 2 -> 359152.

## Impact

- **No confidentiality or integrity impact.** The helper only generates the
  candidate code the user submits; the ADFS server independently validates
  codes. There is no path by which an invalid code could be accepted.
- **Limited availability / robustness impact.** The advertised backward
  clock-skew tolerance (-1 period) is non-functional: every call spends its
  first iteration on a wrapped step. With the currently pinned
  pquerna/otp v1.5.0 that iteration errors and the current-step code is
  returned (correct behavior by fallback). If the dependency's behavior for
  negative timestamps were to change, the function could return a code for a
  wrapped time step first, causing MFA submissions to fail where they should
  have succeeded (availability-only impact).
- **Engineering impact (immediate).** go test ./internal/totp/ is red on every
  build; the repository's stated test gate (go test -v -count=1 -race ./...)
  fails on main and on all released versions.

## Scoring rationale (CVSS v3.1)

- **Attack Vector: Local** — the defect only affects the local authentication
  helper; there is no remote attack surface.
- **Attack Complexity: High** — the bounded availability impact requires a
  specific dependency behavior change or a clock-skew edge case beyond one
  period.
- **Availability: Low** — worst case an MFA submission fails and the user
  retries; no denial of service.

Base score 0.1, classified **Low** (borderline "None"). Documented per
repository policy (AGENTS.md security rule 8) because the defect is present in
code that has been pushed to the public remote and included in released
versions.

## Recommended remediation (not yet implemented)

1. Use signed arithmetic for candidate steps and skip negative steps instead of
   letting them wrap:

~~~go
for offset := -tolerancePeriods; offset <= tolerancePeriods; offset++ {
    step := currentStep + int64(offset)
    if step < 0 {
        continue
    }
    // ...
}
~~~

2. Correct the TestGenerateAtOffset expectations to the RFC 6238 vectors for
   the pinned secret (step 0 -> 755224, step 1 -> 287082, step 2 -> 359152),
   or restructure the test to assert the tolerated code for a fixed timestamp.

## References

- RFC 4226 Appendix D (HOTP test vectors); RFC 6238 (TOTP test vectors)
- internal/totp/totp.go — GenerateAtOffset
- internal/totp/totp_test.go — TestGenerateAtOffset
- Call site: internal/browser/auth.go (ADFS MFA code generation)
