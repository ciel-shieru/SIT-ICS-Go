# ADR-0008: Three-Layer Testing Strategy with Mock AuthBrowser

## Status

Proposed

## Date

2026-09-02

## Context

Testing browser automation presents unique challenges:
- Browser operations require Chromium, which is slow to start (~2-5 seconds) and resource-intensive (~200-400 MB memory)
- Unit tests should not launch browsers (slow, flaky, resource-heavy)
- Integration tests need real browser but should be isolated and reproducible
- Authentication e2e tests need real browser + real ADFS (or mock ADFS endpoint)
- The `AuthBrowser` interface enables mocking, but mocks must stay in sync with the interface

Three distinct testing layers are needed, each with different requirements:

1. **Unit tests**: Fast, isolated, no browser required
2. **Browser integration tests**: Real browser, isolated, cleanup guaranteed
3. **Authentication e2e tests**: Real browser + real/mocked ADFS, full flow validation

## Decision Drivers

- **Speed**: Unit tests must run in <1 second per test; no browser overhead
- **Isolation**: Each test gets fresh state; no shared browser or cookies
- **Reliability**: Tests should not flake due to timing or network issues
- **Coverage**: All code paths tested; real browser tests catch integration issues
- **Maintainability**: Mock interface must stay in sync with real interface
- **CI/CD friendly**: Tests run in headless containers without display server

## Considered Options

### Option A: Three-Layer Testing (Selected)

Unit tests with `MockAuthBrowser`, browser integration tests with real Rod + local Chrome, auth e2e tests with real provider + real/mocked browser.

### Option B: Two-Layer Testing (Unit + E2E)

Combine browser integration and auth e2e into a single "integration" layer. Simpler structure but less granularity.

### Option C: Property-Based Testing

Use property-based testing frameworks to generate random inputs and verify invariants. Complements but doesn't replace structured testing layers.

## Decision

We will choose **Option A: Three-Layer Testing** with distinct unit, browser integration, and auth e2e layers.

## Rationale

### Why Three Layers

- **Speed**: Unit tests run instantly without browser; only integration/e2e tests incur browser overhead
- **Isolation**: Each layer has clear responsibilities; failures are easier to diagnose
- **Coverage**: Unit tests cover logic/errors; integration tests cover browser lifecycle; e2e tests cover full flow
- **CI/CD optimization**: Unit tests run on every commit; integration/e2e tests run on schedule or release

### Why Not Two Layers

Combining browser integration and auth e2e into one layer reduces granularity:
- Harder to isolate whether a failure is in browser lifecycle or auth logic
- Test setup/teardown becomes more complex
- CI/CD optimization less precise (can't run just browser tests without auth)

## Implementation Details

### Layer 1: Unit Tests (MockAuthBrowser)

**Purpose**: Test authentication logic, error handling, config parsing without browser overhead.

**Mock implementation**:
```go
type MockAuthBrowser struct {
    AuthenticateFunc func(ctx context.Context, req AuthRequest) (AuthResult, error)
}

func (m *MockAuthBrowser) Authenticate(ctx context.Context, req AuthRequest) (AuthResult, error) {
    return m.AuthenticateFunc(ctx, req)
}
```

**Usage**:
```go
func TestAuthTimeout(t *testing.T) {
    mock := &MockAuthBrowser{
        AuthenticateFunc: func(ctx context.Context, req AuthRequest) (AuthResult, error) {
            return AuthResult{}, fmt.Errorf("%w: %v", ErrAuthenticationTimeout, context.DeadlineExceeded)
        },
    }
    
    result, err := mock.Authenticate(ctx, req)
    if !errors.Is(err, ErrAuthenticationTimeout) {
        t.Errorf("expected ErrAuthenticationTimeout, got %v", err)
    }
    if result.Cookies != nil {
        t.Error("expected no cookies on timeout")
    }
}
```

**What to test**:
- Error handling and categorization (`errors.Is()` with sentinel errors)
- Config parsing and validation
- Credential extraction logic (non-browser code)
- Timeout behavior (context cancellation)
- Edge cases (empty responses, malformed data)

**What NOT to test**:
- Browser launch/connect (covered in Layer 2)
- DOM interaction (covered in Layer 3)
- Real ADFS flow (covered in Layer 3)

### Layer 2: Browser Integration Tests

**Purpose**: Test browser lifecycle, context isolation, timeouts with real Rod + local Chrome.

**Setup**:
```go
func TestMain(m *testing.M) {
    // Launch browser once for all integration tests
    launcher := launcher.New().Headless(true).NoSandbox(true)
    url, err := launcher.Launch()
    if err != nil {
        log.Fatalf("failed to launch browser: %v", err)
    }
    
    browser = rod.New().ControlURL(url)
    if err := browser.Connect(); err != nil {
        log.Fatalf("failed to connect to browser: %v", err)
    }
    
    exitCode := m.Run()
    
    // Cleanup
    browser.Close()
    launcher.Cleanup()
    os.Exit(exitCode)
}
```

**What to test**:
- Browser launch and connect (local and remote modes)
- Incognito context creation and isolation
- Context cleanup (leakless verification)
- Timeout behavior (connection timeout, navigation timeout)
- Error wrapping (Rod errors wrapped with application errors)
- Multiple concurrent contexts (no interference)

**Test isolation**:
- Each test gets fresh incognito context
- No shared state between tests
- Tests clean up after themselves (leakless handles zombie processes)

**Example**:
```go
func TestIncognitoIsolation(t *testing.T) {
    ctx := context.Background()
    
    // Create two incognito contexts
    incognito1, err := browser.Incognito()
    if err != nil {
        t.Fatalf("failed to create incognito context: %v", err)
    }
    defer incognito1.Close()
    
    incognito2, err := browser.Incognito()
    if err != nil {
        t.Fatalf("failed to create incognito context: %v", err)
    }
    defer incognito2.Close()
    
    // Set cookie in first context
    page1, err := incognito1.Page("https://example.com")
    if err != nil {
        t.Fatalf("failed to create page: %v", err)
    }
    err = page1.SetCookies([]http.Cookie{{Name: "test", Value: "value1"}})
    if err != nil {
        t.Fatalf("failed to set cookie: %v", err)
    }
    
    // Verify cookie not visible in second context
    page2, err := incognito2.Page("https://example.com")
    if err != nil {
        t.Fatalf("failed to create page: %v", err)
    }
    cookies, err := page2.Cookies([]string{"https://example.com"})
    if err != nil {
        t.Fatalf("failed to get cookies: %v", err)
    }
    if len(cookies) > 0 {
        t.Errorf("expected no cookies in isolated context, got %d", len(cookies))
    }
}
```

### Layer 3: Authentication E2E Tests

**Purpose**: Test full ADFS authentication flow with real browser + real/mocked ADFS.

**Setup**:
```go
// Requires test/staging ADFS environment or mocked ADFS endpoint
func TestADFSAuthentication(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping e2e test (use -run=E2E to enable)")
    }
    
    browser := setupTestBrowser(t)
    defer browser.Close()
    
    auth := NewADFSAuth(browser, ADFSConfig{
        Username: testUsername,
        Password: testPassword,
        TOTPSecret: testTOTPSecret,
    })
    
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
    defer cancel()
    
    result, err := auth.Authenticate(ctx, AuthRequest{
        URL: "https://in4sit.singaporetech.edu.sg/psc/CSSISSTD/",
    })
    if err != nil {
        t.Fatalf("authentication failed: %v", err)
    }
    
    if len(result.Cookies) == 0 {
        t.Error("expected cookies after successful auth")
    }
}
```

**What to test**:
- Full ADFS login flow (navigate → fill credentials → submit → MFA → extract cookies)
- MFA handling (TOTP code generation and submission)
- Error cases (invalid credentials, expired TOTP, network failure)
- Timeout behavior (5-minute overall deadline)
- Context isolation (multiple concurrent auth operations)

**Test environment**:
- **Preferred**: Test/staging ADFS environment with test credentials
- **Fallback**: Mocked ADFS endpoint (simulates ADFS responses without real authentication)
- **Not recommended**: Production ADFS (risk of locking accounts, rate limiting)

**Mocked ADFS approach** (if test environment unavailable):
```go
// Mock ADFS returns predictable responses for testing
type MockADFSServer struct {
    ts *httptest.Server
}

func (m *MockADFSServer) URL() string {
    return m.ts.URL
}

func (m *MockADFSServer) Close() {
    m.ts.Close()
}

// Setup mock server in tests
func setupMockADFS(t *testing.T) *MockADFSServer {
    server := &MockADFSServer{}
    server.ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Return mocked ADFS responses based on request path
        switch r.URL.Path {
        case "/adfs/ls/idpinitiatedsignon.asmx":
            w.Header().Set("Content-Type", "text/html")
            w.Write([]byte(mockADFSLoginPage))
        // ... other endpoints
        }
    }))
    t.Cleanup(server.Close)
    return server
}
```

### Test Organization

```
internal/auth/
    browser_auth_test.go       — Unit tests (MockAuthBrowser)
    browser_auth_e2e_test.go   — E2E tests (real browser + real/mocked ADFS)

internal/browser/
    local_test.go              — Browser integration tests (local Chrome)
    remote_test.go             — Browser integration tests (remote CDP)
    errors_test.go             — Error handling tests (mock errors)
```

**Test naming conventions**:
- `*_test.go`: Unit tests (run by default)
- `*_e2e_test.go`: E2E tests (run with `-run=E2E` or `-tags=e2e`)
- `*_integration_test.go`: Browser integration tests (run with `-tags=integration`)

**CI/CD strategy**:
```yaml
# Every commit
unit-tests:
  run: go test ./...

# Daily or on release
integration-tests:
  run: go test -tags=integration ./...
  env:
    BROWSER_MODE: rod

e2e-tests:
  run: go test -run=E2E -tags=e2e ./...
  env:
    ADFS_URL: https://test-adfs.sit.example.edu.sg
    TEST_CREDENTIALS: from secrets manager
```

### Leakless Verification

Verify no zombie processes after tests:

```go
func TestNoZombieProcesses(t *testing.T) {
    launcher := launcher.New().Headless(true).NoSandbox(true)
    url, err := launcher.Launch()
    if err != nil {
        t.Fatalf("failed to launch browser: %v", err)
    }
    
    browser := rod.New().ControlURL(url)
    if err := browser.Connect(); err != nil {
        t.Fatalf("failed to connect: %v", err)
    }
    
    browser.Close()
    launcher.Cleanup()
    
    // Wait for processes to terminate
    time.Sleep(1 * time.Second)
    
    // Check for zombie Chrome processes
    output, err := exec.Command("pgrep", "-f", "chrome").CombinedOutput()
    if err == nil && len(output) > 0 {
        t.Errorf("zombie Chrome processes found: %s", string(output))
    }
}
```

## Consequences

### Positive

- Fast unit tests (<1 second per test); no browser overhead
- Real browser tests catch integration issues (DOM changes, CDP protocol updates)
- Clear test boundaries; failures are easy to diagnose
- Mock errors in unit tests; real errors in integration/e2e tests
- CI/CD optimization: unit tests on every commit; integration/e2e on schedule

### Negative

- Three layers to maintain; more test files
- E2E tests require test/staging ADFS environment (or mocked endpoint)
- Browser integration tests are slower (~2-5 seconds per test)
- Mock interface must stay in sync with real `AuthBrowser` interface

### Neutral / Operational

- E2E tests skipped by default (`testing.Short()`); enable with `-run=E2E`
- Browser integration tests tagged (`-tags=integration`); opt-in
- Leakless integration prevents zombie processes; verification tests confirm cleanup
- Test credentials managed via secrets manager; never hardcoded

## Alternatives Considered

### Option B: Two-Layer Testing

**Summary**: Combine browser integration and auth e2e into a single "integration" layer.

**Benefits**: Simpler structure; fewer test files; less maintenance overhead.

**Costs**: Less granularity; harder to isolate failures; CI/CD optimization less precise.

**Reason rejected**: Three layers provide better isolation and faster feedback. Unit tests catch logic errors instantly; integration tests catch browser issues; e2e tests catch flow issues.

### Option C: Property-Based Testing

**Summary**: Use property-based testing (e.g., `github.com/thepudds/go-ppt`) to generate random inputs and verify invariants.

**Benefits**: Catches edge cases; exhaustive input coverage.

**Costs**: Complements but doesn't replace structured testing; harder to debug failures; not suited for browser interaction testing.

**Reason rejected**: Property-based testing is a valuable complement but doesn't address the core challenge of browser test isolation and speed. Three-layer structure is the foundation; property-based testing can be added later.

## Follow-Ups

- ADR-0005: Browser abstraction layer (`MockAuthBrowser` implements `AuthBrowser`)
- ADR-0006: Authentication flow (e2e tests validate full ADFS flow)
- ADR-0007: Error handling (unit tests verify error wrapping and categories)
- Implementation tracking lives outside this ADR

## References

- [Go testing package](https://pkg.go.dev/testing) — Go testing patterns
- [Testing with mocks](https://go.dev/doc/effective_go#interfaces-and-types) — Interface-based mocking in Go
- [leakless](https://github.com/ysmood/leakless) — Zombie process prevention for Chromium
- [CI/CD test optimization](https://docs.github.com/en/actions/use-cases-and-examples/pipeline/optimizing-pipeline-speed#run-tests-in-parallel) — Test parallelization and filtering
