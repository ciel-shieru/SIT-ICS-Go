# ADR-0006: Authentication Flow with Isolated Browser Contexts and Rod Waiting Primitives

## Status

Proposed

## Date

2026-09-02

## Context

The ADFS authentication flow involves multiple steps that depend on JavaScript execution and dynamic DOM state:

1. Navigate to PeopleSoft URL → 302 redirect to ADFS `idpinitiatedsignon.asmx`
2. ADFS page loads with JavaScript `SelectOption()` function
3. JavaScript renders credential form (`userNameInput`, `passwordInput`, `submitButton`)
4. Fill credentials and submit
5. If MFA enabled, ADFS returns AzureMFA form (`VerificationCode` input)
6. Enter TOTP code and submit
7. ADFS posts `SAMLResponse` back to PeopleSoft
8. Extract session cookies (`PS_TOKEN`, `PSJSESSIONID`, etc.)

Fixed sleeps (e.g., `time.Sleep(5 * time.Second)`) produce fragile behavior on JS-heavy authentication pages. Network timing varies, DOM rendering differs across environments, and ADFS response times fluctuate.

Each authentication operation must use an isolated browser context to prevent cookie/state leakage between users or sessions. Authentication state must be ephemeral—destroyed after use—with no persistent browser profiles.

## Decision Drivers

- **No sleeps**: Use Rod's waiting primitives (element visible, navigation complete, URL changed, DOM state) instead of time-based sleeps
- **Context isolation**: Each auth operation uses an incognito browser context
- **Ephemeral state**: Browser context destroyed after auth; no persistent profiles
- **Timeout hierarchy**: Overall authentication deadline plus narrower operation limits
- **Context propagation**: All browser operations accept `context.Context` for cancellation and timeouts
- **Deterministic waits**: Wait for application/JS state, not arbitrary time intervals

## Considered Options

### Option A: Rod Waiting Primitives with Incognito Contexts (Selected)

Use Rod's built-in waiting methods (`WaitURL`, `WaitIdle`, `WaitStable`, element visibility checks) combined with incognito browser contexts for isolation.

### Option B: Polling with Backoff

Implement custom polling with exponential backoff to wait for DOM conditions. More control but reinvents Rod's existing primitives.

### Option C: Event Listeners with Channels

Use CDP event listeners to detect state changes. Maximum precision but significantly more complex and error-prone.

## Decision

We will choose **Option A: Rod Waiting Primitives with Incognito Contexts**.

## Rationale

### Why Option A over Option B

- **Built-in reliability**: Rod's waiting primitives are battle-tested, handle edge cases, and integrate with context cancellation
- **Less code**: No custom polling logic; Rod provides `WaitURL`, `WaitIdle`, `WaitStable`, `element.Visible()`
- **Standard patterns**: Rod's waiting API is well-documented with examples; custom polling requires maintenance

### Why Option A over Option C

- **Simplicity**: Rod's high-level API abstracts CDP event complexity
- **Context integration**: Rod's waiting methods respect `context.Context` cancellation; custom event listeners require manual channel management
- **Maintainability**: Rod handles protocol version changes; custom CDP code breaks with protocol updates

### Why Incognito Contexts

Each authentication operation gets its own incognito context:
- Cookies don't leak between auth operations
- Storage (localStorage, sessionStorage) is isolated
- Failed auth doesn't pollute subsequent attempts
- Multiple concurrent auth operations don't interfere

### Why Timeout Hierarchy

Authentication involves multiple sequential operations, each with different timeout requirements:
- Browser connect: fast (~10 seconds)
- Navigation: moderate (~30 seconds, ADFS can be slow)
- Overall auth: generous (~5 minutes, accounts for MFA delays, network issues)

Narrower timeouts per operation provide faster failure detection and clearer error messages.

## Implementation Details

### Authentication Workflow

```
Authenticate()
    ├── acquire browser (from AuthBrowser interface)
    ├── create isolated incognito context
    ├── create page
    ├── navigate to ADFS URL
    ├── wait for application/JS state (not time-based sleeps)
    ├── perform required interaction (fill form, click submit)
    ├── wait for authenticated state (SAMLResponse or MFA form)
    ├── extract cookies/tokens
    └── destroy context
```

### Rod Waiting Primitives

**Wait for navigation**:
```go
err := page.WaitURL("https://fs.singaporetech.edu.sg/adfs/ls/")
```

**Wait for JS/network idle**:
```go
err := page.WaitIdle()
```

**Wait for element to be visible**:
```go
_, err := page.MustQuery("#userNameInput").Visible()
```

**Wait for page to stabilize**:
```go
err := page.WaitStable(3000) // 3 seconds of stability
```

**Wait for URL change**:
```go
err := page.WaitURL("https://in4sit.singaporetech.edu.sg/psc/")
```

### Timeout Hierarchy

```
Authentication
├── overall:        5 min   (total auth deadline)
├── browser connect: 10 sec  (CDP connection)
├── navigation:      30 sec  (page load + JS execution)
└── individual waits: condition-dependent
```

Implementation:
```go
const (
    authTimeout          = 5 * time.Minute
    connectTimeout       = 10 * time.Second
    navigationTimeout    = 30 * time.Second
)

ctx, cancel := context.WithTimeout(parentCtx, authTimeout)
defer cancel()

// Connect with narrower timeout
connectCtx, connectCancel := context.WithTimeout(ctx, connectTimeout)
defer connectCancel()
```

### Context Isolation

```go
// Create incognito context
incognito, err := browser.Incognito()
if err != nil {
    return AuthResult{}, fmt.Errorf("%w: %v", ErrBrowserConnect, err)
}
defer incognito.Close() // Ensures cleanup even on error

// Create page in isolated context
page, err := incognito.Page(req.URL)
if err != nil {
    return AuthResult{}, fmt.Errorf("%w: %v", ErrNavigation, err)
}
```

Each authentication operation gets its own incognito context. The `defer incognito.Close()` ensures cleanup even if errors occur mid-authentication.

### Context Propagation

Every browser operation accepts `context.Context`:

```go
func (b *RodAuthBrowser) Authenticate(
    ctx context.Context,
    req AuthRequest,
) (AuthResult, error) {
    // All Rod operations use ctx
    page, err := incognito.Page(req.URL).WaitURL("").Context(ctx)
    // ...
}
```

Context is used for:
- Application shutdown (parent context cancelled)
- Request cancellation (user aborts)
- Authentication timeout (deadline exceeded)
- Browser connection cancellation
- Navigation cancellation

No detached background contexts are created unless deliberately required.

### ADFS-Specific Flow

**Step 1: Navigate to PeopleSoft**
```go
err := page.Navigate(req.URL).Context(ctx)
if err != nil {
    return handleErr(ErrNavigation, err)
}
```

**Step 2: Wait for ADFS redirect (JS-rendered credential form)**
```go
// Wait for ADFS page to load and JS to render credential form
_, err := page.MustQuery("#userNameInput").Visible().Context(ctx).Do()
if err != nil {
    return handleErr(ErrAuthentication, fmt.Errorf("credential form not rendered: %w", err))
}
```

**Step 3: Fill credentials**
```go
err = page.MustElement("#userNameInput").Type(username).Context(ctx).Do()
if err != nil {
    return handleErr(ErrCredentialExtraction, err)
}
err = page.MustElement("#passwordInput").Type(password).Context(ctx).Do()
if err != nil {
    return handleErr(ErrCredentialExtraction, err)
}
```

**Step 4: Submit and wait for MFA or success**
```go
err = page.MustElement("#submitButton").Click().Context(ctx).Do()
if err != nil {
    return handleErr(ErrAuthentication, err)
}

// Wait for either MFA form or SAML response
mfaVisible := page.MustQuery("#VerificationCode").Visible().Context(ctx)
samlVisible := page.MustQuery("input[name='SAMLResponse']").Visible().Context(ctx)

// Use WaitStable to let JS complete
err = page.WaitStable(5000).Context(ctx).Do()
```

**Step 5: Handle MFA if present**
```go
if mfaVisible {
    totpCode := generateTOTP(totpSecret)
    err = page.MustElement("#VerificationCode").Type(totpCode).Context(ctx).Do()
    err = page.MustElement("#SignIn").Click().Context(ctx).Do()
}
```

**Step 6: Extract cookies**
```go
cookies, err := page.Cookies([]string{req.URL}).Context(ctx).Do()
if err != nil {
    return handleErr(ErrCredentialExtraction, err)
}
```

### No Sleeps Policy

Avoid:
```go
// BAD: Fixed sleep
time.Sleep(5 * time.Second)
```

Prefer:
```go
// GOOD: Wait for condition
_, err := page.MustQuery("#userNameInput").Visible().Context(ctx).Do()
```

Conditions to wait for:
- Element exists (`page.MustQuery()`)
- Element visible (`element.Visible()`)
- Element becomes enabled (`element.Enabled()`)
- Navigation completes (`page.WaitURL()`)
- URL changes (`page.WaitURL()`)
- Specific DOM state appears (`page.MustQuery()`)
- Specific JS/application state (`page.Eval()`)

## Consequences

### Positive

- Robust against timing variations; no fragile sleeps
- Clean state isolation between auth operations
- Fast failure detection via timeout hierarchy
- Clear error messages (which step failed, why)
- Context cancellation propagates through entire auth flow
- Incognito contexts prevent cookie leakage

### Negative

- Slightly more complex wait logic than simple sleeps
- Timeout values require tuning for different network conditions
- Incognito contexts add minor overhead (~10-20 ms per context creation)

### Neutral / Operational

- Wait primitives handle most edge cases; custom waits only for unusual ADFS behavior
- Timeout hierarchy can be adjusted via configuration if needed
- Incognito context isolation is the default; no opt-out to prevent accidental state leakage

## Alternatives Considered

### Option B: Polling with Backoff

**Summary**: Custom polling loop with exponential backoff to wait for DOM conditions.

**Benefits**: Fine-grained control over polling interval and retry logic.

**Costs**: Reinvents Rod's existing primitives; more code to maintain; error-prone edge cases; doesn't integrate with context cancellation as cleanly.

**Reason rejected**: Rod's waiting primitives are well-tested and handle context cancellation. Custom polling adds complexity without meaningful benefit.

### Option C: Event Listeners with Channels

**Summary**: Use CDP event listeners (`Page.loadEventFired`, `DOM.contentFrameLoaded`) with Go channels to detect state changes.

**Benefits**: Maximum precision; instant reaction to state changes.

**Costs**: Significantly more complex; manual channel management; protocol version sensitivity; hard to test; easy to leak goroutines.

**Reason rejected**: Rod's high-level API abstracts CDP event complexity. Event listeners are better suited for building custom libraries, not application-level authentication.

## Follow-Ups

- ADR-0005: Browser abstraction layer (AuthBrowser interface, Rod implementation)
- ADR-0007: Error handling and timeout hierarchy (error wrapping, logging boundaries)
- ADR-0008: Testing strategy (mock AuthBrowser for unit tests)
- Implementation tracking lives outside this ADR

## References

- [rod waiting primitives](https://go-rod.github.io/#/getting-started.md) — WaitURL, WaitIdle, WaitStable, element visibility
- [Chrome DevTools Protocol events](https://chromedevtools.github.io/devtools-protocol/tot/Page/#event-loadEventFired) — CDP event system
- [context package](https://pkg.go.dev/context) — Go context for cancellation and timeouts
- [incognito contexts](https://go-rod.github.io/#/browser_context.md) — Rod browser context isolation
