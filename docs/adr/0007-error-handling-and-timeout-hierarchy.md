# ADR-0007: Application-Facing Error Model with Wrapped Rod Errors

## Status

Proposed

## Date

2026-09-02

## Context

Browser operations can fail at many levels:
- Browser launch (Chromium binary not found, port in use)
- CDP connection (remote browser unreachable, network timeout)
- Navigation (page load failed, redirect loop, DNS failure)
- Interaction (element not found, element not visible, JS error)
- Authentication (invalid credentials, MFA failure, session expired)
- Credential extraction (cookies not set, token missing)

Callers should not depend on Rod's internal error types or behavior. Errors must be actionable—providing clear category and underlying cause—so that:
- Calling code can make decisions based on error category (retry vs. fail fast)
- Logging captures sufficient context without exposing credentials
- Testing uses mocks without needing real Rod errors
- Future browser implementation swaps don't break error handling

## Decision Drivers

- **Abstraction**: Callers depend on application-facing errors, not Rod errors
- **Actionability**: Error categories enable decision-making (retry, fail, timeout)
- **Preserve context**: Underlying errors preserved via `%w` wrapping for debugging
- **Logging boundaries**: Sensitive data (passwords, cookies, tokens) never logged
- **Testability**: Mock errors in unit tests; real errors in integration tests

## Considered Options

### Option A: Application-Facing Errors with %w Wrapping (Selected)

Define application-facing sentinel errors. Wrap Rod errors using `fmt.Errorf("%w: %v", appErr, err)`. Callers check categories with `errors.Is()`, debug with `%+` formatting.

### Option B: Error Types with Structs

Define error types as structs with fields (category, message, underlying error). More structured but more boilerplate.

### Option C: Error Codes with Strings

Use string error codes (`"BROWSER_LAUNCH_FAILED"`) for machine parsing. Less idiomatic Go; harder to chain errors.

## Decision

We will choose **Option A: Application-Facing Errors with %w Wrapping**.

## Rationale

### Why Option A over Option B

- **Idiomatic Go**: `errors.Is()` and `errors.As()` with `%w` wrapping is the Go standard pattern (since Go 1.13)
- **Less boilerplate**: No error struct definitions; just sentinel errors and wrapping
- **Standard library**: No custom error handling code; relies on proven `errors` package patterns

### Why Option A over Option C

- **Idiomatic Go**: Sentinel errors with `errors.Is()` are more idiomatic than string codes
- **Type safety**: Compile-time checking with `errors.Is()` vs. runtime string comparison
- **Chaining**: `%w` wrapping preserves error chain; string codes lose context

### Why %w Wrapping

```go
return fmt.Errorf("%w: %v", ErrBrowserConnect, err)
```

- Preserves underlying error for debugging (`errors.Is()` works through layers)
- Provides context (category + underlying message)
- Standard Go pattern; familiar to all Go developers

## Implementation Details

### Sentinel Errors

Define application-facing errors in `internal/browser/errors.go`:

```go
var (
    ErrBrowserUnavailable       = errors.New("browser unavailable")
    ErrBrowserLaunch            = errors.New("browser launch failed")
    ErrBrowserConnect           = errors.New("browser connection failed")
    ErrNavigation               = errors.New("navigation failed")
    ErrAuthentication           = errors.New("authentication failed")
    ErrAuthenticationTimeout    = errors.New("authentication timed out")
    ErrCredentialExtraction     = errors.New("credential extraction failed")
)
```

### Error Wrapping

Wrap Rod errors at package boundaries:

```go
// In internal/browser/local.go
func (b *LocalBrowser) Launch() error {
    url, err := launcher.New().Launch()
    if err != nil {
        return fmt.Errorf("%w: %v", ErrBrowserLaunch, err)
    }
    // ...
}

// In internal/browser/remote.go
func (b *RemoteBrowser) Connect() error {
    browser := rod.New().ControlURL(b.cfg.ControlURL)
    if err := browser.Connect(); err != nil {
        return fmt.Errorf("%w: %v", ErrBrowserConnect, err)
    }
    // ...
}

// In internal/auth/browser_auth.go
func (a *ADFSAuth) Authenticate(ctx context.Context, req AuthRequest) (AuthResult, error) {
    page, err := incognito.Page(req.URL)
    if err != nil {
        return AuthResult{}, fmt.Errorf("%w: %v", ErrNavigation, err)
    }
    // ...
    
    // Wait for credential form
    _, err = page.MustQuery("#userNameInput").Visible().Context(ctx).Do()
    if err != nil {
        return AuthResult{}, fmt.Errorf("%w: credential form not rendered: %w", ErrAuthentication, err)
    }
    // ...
}
```

### Error Checking

Callers check error categories with `errors.Is()`:

```go
result, err := authBrowser.Authenticate(ctx, req)
if err != nil {
    if errors.Is(err, ErrAuthenticationTimeout) {
        // Timeout: retry with fresh context
        log.Warn("authentication timed out, retrying")
        return retryAuth(ctx, req)
    }
    
    if errors.Is(err, ErrBrowserUnavailable) {
        // Browser not available: fail fast
        log.Error("browser unavailable, cannot authenticate")
        return ErrServiceUnavailable
    }
    
    // Unknown error: log and fail
    log.Error("authentication failed", "error", err)
    return err
}
```

### Logging Boundaries

**Browser package logs**:
```
browser selected: chrome
browser version: 151.0.7922.34
local vs remote: local
launch duration: 2.3s
connect duration: 0.8s
browser restart: true
navigation duration: 1.2s
```

**Auth provider logs**:
```
provider: adfs
authentication outcome: success
duration: 4.2s
failure category: timeout
```

**Neither logs**:
```
passwords
cookies
tokens
Authorization headers
localStorage
sessionStorage
```

Logging example:
```go
// GOOD: Log category, not credentials
log.Error("authentication failed", "provider", "adfs", "category", "timeout")

// BAD: Log raw error that may contain sensitive data
log.Error("authentication failed", "error", err) // err may contain cookies
```

### Error Context Addition

Add context at failure points, not at origin:

```go
// In browser package (low level)
if err != nil {
    return fmt.Errorf("%w: %v", ErrBrowserConnect, err)
}

// In auth package (high level)
if err != nil {
    // Add context about what we were doing
    return fmt.Errorf("adfs authenticate: %w", err)
}
```

This preserves the error chain while adding context at each layer.

### Timeout Errors

Timeout errors are a special case—often transient, often retryable:

```go
func isTimeout(err error) bool {
    return errors.Is(err, context.DeadlineExceeded) ||
           errors.Is(err, ErrAuthenticationTimeout)
}

// Usage
if err := authBrowser.Authenticate(ctx, req); err != nil {
    if isTimeout(err) {
        log.Warn("authentication timed out, may retry")
        return handleTimeout(err)
    }
    log.Error("authentication failed", "error", err)
    return handleError(err)
}
```

### Testing with Mock Errors

Unit tests use mock errors, not real Rod errors:

```go
type MockAuthBrowser struct {
    AuthenticateFunc func(ctx context.Context, req AuthRequest) (AuthResult, error)
}

func (m *MockAuthBrowser) Authenticate(ctx context.Context, req AuthRequest) (AuthResult, error) {
    return m.AuthenticateFunc(ctx, req)
}

// Test timeout handling
func TestAuthTimeout(t *testing.T) {
    mock := &MockAuthBrowser{
        AuthenticateFunc: func(ctx context.Context, req AuthRequest) (AuthResult, error) {
            return AuthResult{}, fmt.Errorf("%w: %v", ErrAuthenticationTimeout, ctx.Err())
        },
    }
    
    // Test that caller handles timeout correctly
    err := mock.Authenticate(ctx, req)
    if !errors.Is(err, ErrAuthenticationTimeout) {
        t.Errorf("expected ErrAuthenticationTimeout, got %v", err)
    }
}
```

## Consequences

### Positive

- Callers independent of Rod; easy to swap browser implementations
- Error categories enable decision-making (retry vs. fail fast)
- Underlying errors preserved for debugging via `%w` wrapping
- Clear logging boundaries prevent credential leakage
- Mock errors in unit tests; real errors in integration tests

### Negative

- Additional error definition/mapping code
- Error chain can become deep (harder to read with `fmt.Printf("%v")`)
- Must remember to wrap errors at package boundaries

### Neutral / Operational

- `errors.Is()` works through `%w` wrapping layers; `errors.As()` for type assertions
- Deep error chains readable with `fmt.Printf("%+v", err)` (stacktrace-style)
- Logging policy enforced by code review; no automated enforcement

## Alternatives Considered

### Option B: Error Types with Structs

**Summary**: Define error types as structs with fields (category, message, underlying error).

**Benefits**: More structured; easy to extract category programmatically; explicit fields.

**Costs**: More boilerplate; custom error handling code; less idiomatic Go; harder to chain errors.

**Reason rejected**: `%w` wrapping with sentinel errors is the idiomatic Go pattern. Struct errors add complexity without meaningful benefit for this use case.

### Option C: Error Codes with Strings

**Summary**: Use string error codes (`"BROWSER_LAUNCH_FAILED"`) for machine parsing.

**Benefits**: Easy to parse programmatically; language-agnostic.

**Costs**: Less idiomatic Go; no compile-time checking; runtime string comparison; loses error chain context.

**Reason rejected**: Sentinel errors with `errors.Is()` provide compile-time safety and idiomatic Go patterns. String codes are better for external APIs, not internal Go code.

## Follow-Ups

- ADR-0005: Browser abstraction layer (error definitions in `internal/browser/errors.go`)
- ADR-0006: Authentication flow (error wrapping at auth step boundaries)
- ADR-0008: Testing strategy (mock errors for unit tests)
- Implementation tracking lives outside this ADR

## References

- [Go errors package](https://pkg.go.dev/errors) — Error handling patterns
- [Go error wrapping](https://go.dev/blog/go1.13-errors) — `%w` wrapping introduced in Go 1.13
- [errors.Is and errors.As](https://pkg.go.dev/errors#Is) — Error checking patterns
- [Sentry error grouping](https://docs.sentry.io/platforms/go/configuration/filtering/) — Error categorization best practices
