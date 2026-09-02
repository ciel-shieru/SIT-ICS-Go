# ADR-0005: Browser Abstraction Layer via AuthBrowser Interface with Rod Implementation

## Status

Proposed

## Date

2026-09-02

## Context

The application must authenticate against Singapore Institute of Technology's (SIT) Microsoft ADFS at `https://fs.singaporetech.edu.sg/` to access the Oracle PeopleSoft timetable system. Verification of the ADFS flow revealed that the `idpinitiatedsignon.asmx` endpoint requires JavaScript execution to render the credential form (`userNameInput`, `passwordInput`, `submitButton`). Raw HTTP POST requests cannot bypass this requirement—every POST returns the same options page with a JavaScript `SelectOption()` function.

This necessitates headless browser automation for authentication. The application must support two deployment modes:

1. **Local (desktop)**: Automatic Chrome/Edge/Chromium discovery, with Rod-managed Chromium as fallback
2. **Remote (container/Kubernetes)**: Connect to externally managed Chromium or `chrome-headless-shell` via Chrome DevTools Protocol (CDP)

The authentication code must not depend on browser implementation details. It should work identically whether the browser is local or remote, Chrome or Chromium, Rod-managed or system-installed.

## Decision Drivers

- **Abstraction**: Authentication code uses an `AuthBrowser` interface and never imports `github.com/go-rod/rod`
- **Local/Remote flexibility**: Same auth code works with local Chrome or remote headless-shell
- **Security**: Security-sensitive Chromium arguments (`--no-sandbox`, `--disable-setuid-sandbox`, etc.) are implementation-owned, not exposed as configuration knobs
- **Isolation**: Each authentication operation uses an incognito browser context to prevent cookie/state leakage
- **Ephemeral state**: No persistent browser profiles by default; authentication state is destroyed after use
- **Rootless container compatibility**: Must run in unprivileged Docker/Podman/Kubernetes containers without special capabilities
- **Minimal dependencies**: Single direct browser dependency; avoid heavy multi-browser frameworks
- **Go 1.26 compatibility**: Libraries must support the target Go version

## Considered Options

### Option A: Rod with AuthBrowser Interface (Selected)

Use `github.com/go-rod/rod` as the sole browser dependency, wrapped behind an internal `AuthBrowser` interface. Rod provides auto container detection, auto Chromium download, leakless process management, and a native Go API with chained context design.

### Option B: chromedp

Use `github.com/chromedp/chromedp`, the most popular Go CDP automation library (13.3k stars). Requires manual `NoSandbox()` configuration and system-installed Chrome; no auto browser download.

### Option C: playwright-go

Use `github.com/mxschmitt/playwright-go`, a Go port of Microsoft Playwright. Supports Chromium, Firefox, and WebKit via a single API. Includes Node.js bridge (~60 MB binary), larger dependency tree.

### Option D: Direct CDP (mafredri/cdp)

Use `github.com/mafredri/cdp` for low-level CDP bindings. Provides type-safe protocol access but requires manual browser lifecycle management, process spawning, and flag configuration.

## Decision

We will choose **Option A: Rod with AuthBrowser Interface**.

## Rationale

### Why Rod over chromedp

- **Auto container detection**: Rod's launcher automatically checks for `/.dockerenv`, `/.containerenv`, and `KUBERNETES_SERVICE_HOST` to detect container environments and add `--no-sandbox` automatically. chromedp requires manual `NoSandbox()` configuration.
- **Auto browser download**: Rod downloads and caches Chromium automatically at `~/.cache/rod/`. chromedp requires system-installed Chrome or manual binary management.
- **Leakless integration**: Rod integrates `github.com/ysmood/leakless` to prevent zombie browser processes after the Go process exits. chromedp only force-kills on Linux and has edge cases.
- **Smaller dependency tree**: Rod has 6 direct dependencies (all by the same author, ensuring consistency). chromedp has 5 direct dependencies but requires more manual configuration.
- **Headless mode**: Rod supports `--headless=new` (Chrome's newer, more authentic headless mode). chromedp uses legacy `--headless`.

### Why Rod over playwright-go

- **Smaller binary**: Rod produces a ~10-15 MB Go binary. playwright-go includes a Node.js bridge (~60 MB).
- **Fewer dependencies**: Rod has 6 direct dependencies. playwright-go has 9 direct dependencies plus requires `playwright install` step.
- **Simpler mental model**: Rod is Chromium-only (sufficient for our needs). playwright-go's multi-browser support adds complexity we don't require.
- **Native Go API**: Rod's chained context API is idiomatic Go. playwright-go wraps Node.js Playwright, adding an indirection layer.

### Why Rod over direct CDP

- **Browser lifecycle**: Rod handles browser discovery, launching, and cleanup. Direct CDP requires manual process management.
- **Waiting primitives**: Rod provides high-level waiting (element visible, navigation complete, URL changed). Direct CDP requires low-level event handling.
- **Less boilerplate**: Rod's high-level API reduces code compared to raw CDP domain objects.

### Why Rod over Status Quo (HTTP client only)

Verified ADFS behavior shows the credential form is only rendered after JavaScript execution. HTTP client alone cannot authenticate. Browser automation is required.

## Implementation Details

### AuthBrowser Interface

```go
type AuthBrowser interface {
    Authenticate(
        ctx context.Context,
        req AuthRequest,
    ) (AuthResult, error)
}
```

Authentication code uses this interface and never imports Rod types. This keeps the rest of the application independent of browser implementation.

### Browser Modes

```go
type BrowserMode string

const (
    BrowserAuto   BrowserMode = "auto"     // Try system browser, fall back to Rod-managed
    BrowserSystem BrowserMode = "system"   // Require installed Chrome/Edge/Chromium
    BrowserRod    BrowserMode = "rod"      // Use Rod-managed Chromium (auto-download)
    BrowserRemote BrowserMode = "remote"   // Connect to externally managed browser via CDP
)
```

### Configuration

```go
type BrowserConfig struct {
    Mode              BrowserMode
    Executable        string            // Optional explicit browser path
    ControlURL        string            // Used in remote mode
    Headless          bool              // Default: true
    Incognito         bool              // Default: true (isolated contexts)
    ConnectTimeout    time.Duration     // Default: 10s
    NavigationTimeout time.Duration     // Default: 30s
    AuthTimeout       time.Duration     // Default: 5m
}
```

### Directory Structure

```
internal/browser/
    browser.go      — AuthBrowser interface, AuthRequest, AuthResult types
    local.go        — Rod launcher for local Chrome/Chromium
    remote.go       — Rod CDP connection for remote browser
    options.go      — BrowserConfig, BrowserOptions
    errors.go       — Application-facing errors wrapping Rod errors

internal/auth/
    browser_auth.go — ADFS authentication provider (uses AuthBrowser interface)
```

### Local Browser (Desktop)

```go
launcher := launcher.New()

if cfg.Executable != "" {
    launcher = launcher.Bin(cfg.Executable)
}

if cfg.Headless {
    launcher = launcher.Headless(true)
}

url, err := launcher.Launch()
if err != nil {
    return err
}

browser := rod.New().ControlURL(url)
if err := browser.Connect(); err != nil {
    return err
}
```

Resolution order: explicit browser path → system browser → Rod-managed Chromium.

### Remote Browser (Container/Kubernetes)

```go
browser := rod.New().
    Context(ctx).
    ControlURL(cfg.ControlURL)

if err := browser.Connect(); err != nil {
    return err
}
```

The application only owns the connection/context, not the browser process. Closing the Rod object does not terminate Chromium.

### Context Isolation

Each authentication operation uses an incognito context:

```go
incognito, err := browser.Incognito()
if err != nil {
    return err
}
defer incognito.Close()

page, err := incognito.Page(req.URL)
```

This ensures:
- Cookies don't leak between auth operations
- Storage is isolated per operation
- Failed auth doesn't pollute subsequent attempts

### Security-Sensitive Arguments

The following Chromium arguments are **implementation-owned**, not configuration knobs:

```
--no-sandbox
--disable-setuid-sandbox
--disable-seccomp-filter-sandbox
```

The browser abstraction owns these. Application-level configuration exposes only:

```
headless
window size
browser executable
remote endpoint
timeouts
```

### Result Extraction

The browser package returns application-level data, not browser objects:

```go
type AuthResult struct {
    Cookies     []Cookie
    RedirectURL string
    Token       string
}
```

Returning `*rod.Page` or similar would leak the browser implementation throughout the application.

## Consequences

### Positive

- Authentication code is browser-agnostic; easy to swap implementations or add providers
- Works identically in desktop (local Chrome) and container (remote headless-shell) environments
- Auto container detection eliminates manual `--no-sandbox` configuration
- Leakless integration prevents zombie browser processes
- Incognito contexts ensure authentication state isolation
- No persistent browser profiles by default; ephemeral auth state

### Negative

- Adds Chromium binary dependency (~280 MB downloaded, cached)
- Runtime memory footprint: ~150-400 MB per browser instance
- Slower startup than HTTP client (~2-5 seconds vs <1 second)
- Additional abstraction layer adds code complexity

### Neutral / Operational

- Rod is the only direct browser dependency; rest of codebase never imports `github.com/go-rod/rod`
- Browser package owns browser lifecycle; auth package owns authentication workflow
- Security args are implementation-owned, not exposed to application configuration
- Remote mode enables K8s deployments with sidecar or external browser pod

## Alternatives Considered

### Option B: chromedp

**Summary**: Use `github.com/chromedp/chromedp`, the most popular Go CDP automation library.

**Benefits**: Largest community (13.3k stars), most examples, Go 1.26 compatibility.

**Costs**: No auto container detection (manual `NoSandbox()` required), no auto browser download, legacy headless mode, requires system Chrome.

**Reason rejected**: Rod's auto container detection and auto browser download reduce operational complexity. chromedp's larger community is less valuable for a greenfield project with clear requirements.

### Option C: playwright-go

**Summary**: Use `github.com/mxschmitt/playwright-go`, a Go port of Microsoft Playwright.

**Benefits**: Multi-browser support (Chromium, Firefox, WebKit), trace viewer, network interception, rich feature set.

**Costs**: ~60 MB binary (Node.js bridge), 9 direct dependencies, requires `playwright install` step, overkill for Chromium-only auth flow.

**Reason rejected**: Multi-browser support is unnecessary for authentication. The larger binary and dependency count conflict with the minimal-dependency driver.

### Option D: Direct CDP

**Summary**: Use `github.com/mafredri/cdp` for low-level CDP bindings.

**Benefits**: Full protocol control, no abstraction overhead.

**Costs**: Manual browser lifecycle management, no waiting primitives, significantly more boilerplate, error-prone process management.

**Reason rejected**: Rod's high-level API reduces code and complexity. Direct CDP is better suited for building custom libraries, not application-level authentication.

## Follow-Ups

- ADR-0006: Authentication flow with isolated contexts and Rod waiting primitives
- ADR-0007: Error handling and timeout hierarchy
- ADR-0008: Three-layer testing strategy with mock AuthBrowser
- Implementation tracking lives outside this ADR

## References

- [rod](https://github.com/go-rod/rod) — Go-native Chrome DevTools Protocol driver with auto container detection and leakless integration
- [Chrome DevTools Protocol](https://chromedevtools.github.io/devtools-protocol/) — Browser automation protocol
- [leakless](https://github.com/ysmood/leakless) — Zombie process prevention for Chromium
- [chromedp](https://github.com/chromedp/chromedp) — Alternative Go CDP library (13.3k stars)
- [playwright-go](https://github.com/mxschmitt/playwright-go) — Go port of Microsoft Playwright
- [headless-shell](https://hub.docker.com/r/chromedp/headless-shell/) — Minimal headless Chrome build for containers
