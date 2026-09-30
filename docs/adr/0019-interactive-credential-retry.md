# ADR-0019: Interactive credential retry on desktop when authentication fails

## Status
Accepted

## Date

2026-09-30

## Context
On desktop builds, users authenticate via ADFS using credentials stored in the OS keyring. When authentication fails due to invalid credentials or TOTP generation failure, the application previously logged the error and returned without any user interaction. This left users with no way to correct their credentials without restarting the application.

## Decision
On desktop builds (`!container`), when `runFetch()` encounters a credential-related authentication error (`ErrAuthentication`, `ErrCredentialExtraction`, or ADFS-specific error messages), it will:

1. Prompt the user with `[Y/n]` asking if they want to re-enter credentials
2. If the user answers Y/yes, call `credprompt.PromptIfNeeded()` to overwrite stored credentials in the keyring
3. Retry authentication once with the new credentials
4. If the retry succeeds, continue with timetable fetch; if it fails, log and return

This is implemented in a new `internal/credretry` package with desktop-only implementation and container stub.

## Consequences
- Users can recover from credential errors without restarting the application
- Single retry only — prevents infinite loops
- Desktop-only — container builds are no-op (env vars must be changed externally)
- Non-terminal stdin is handled gracefully (error returned, no prompt attempted)
- Network/browser errors do not trigger the prompt (only credential-related errors)
- The prompt treats empty input as "no" (decline), which differs from the conventional `[Y/n]` default but is safer for security
