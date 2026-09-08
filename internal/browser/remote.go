package browser

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
)

type RemoteBrowser struct {
	cfg           BrowserConfig
	resolvedHost  string
	browser       *rod.Browser
	incognito     *rod.Browser
	page          *rod.Page
	pageTargetID  proto.TargetTargetID
}

// resolveHost resolves an FQDN to its IP address for use in HTTP requests
// and Rod connections. If the host is already an IP or localhost, it is
// returned unchanged. This avoids Chromium's /json/version/ 500 error when
// the Host header contains a non-IP hostname.
func resolveHost(host string) (string, error) {
	if net.ParseIP(host) != nil || host == "localhost" || host == "127.0.0.1" || host == "::1" {
		return host, nil
	}

	addrs, err := net.DefaultResolver.LookupIPAddr(context.Background(), host)
	if err != nil {
		return "", fmt.Errorf("resolve %s: %w", host, err)
	}
	if len(addrs) == 0 {
		return "", fmt.Errorf("resolve %s: no addresses found", host)
	}

	return addrs[0].IP.String(), nil
}

// versionInfo represents the /json/version/ endpoint response.
type versionInfo struct {
	WebSocketDebuggerUrl string `json:"webSocketDebuggerUrl"`
}

// discoverWebSocketURL fetches http://host:port/json/version/ and returns the
// webSocketDebuggerUrl. This must be called fresh for every fetch cycle because
// the URL contains a per-session UUID that changes when Chromium restarts.
func discoverWebSocketURL(ctx context.Context, host string, port int, timeout time.Duration) (string, error) {
	url := fmt.Sprintf("http://%s:%d/json/version/", host, port)

	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("discover: create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("discover: fetch %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return "", fmt.Errorf("discover: %s returned status %d: %s", url, resp.StatusCode, string(body))
	}

	var info versionInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return "", fmt.Errorf("discover: decode /json/version/: %w", err)
	}

	if info.WebSocketDebuggerUrl == "" {
		return "", fmt.Errorf("discover: %s returned empty webSocketDebuggerUrl", url)
	}

	return info.WebSocketDebuggerUrl, nil
}

func NewRemoteBrowser(cfg BrowserConfig) (*RemoteBrowser, error) {
	if cfg.RemoteHost == "" {
		return nil, fmt.Errorf("remote browser requires RemoteHost")
	}
	if cfg.RemotePort <= 0 {
		return nil, fmt.Errorf("remote browser requires positive RemotePort")
	}

	resolvedHost, err := resolveHost(cfg.RemoteHost)
	if err != nil {
		return nil, fmt.Errorf("resolve remote host: %w", err)
	}

	return &RemoteBrowser{
		cfg:          cfg,
		resolvedHost: resolvedHost,
	}, nil
}

func (b *RemoteBrowser) Authenticate(ctx context.Context, req AuthRequest) (AuthResult, error) {
	authCtx, authCancel := context.WithTimeout(ctx, b.cfg.AuthTimeout)
	defer authCancel()

	connectCtx, connectCancel := context.WithTimeout(authCtx, b.cfg.ConnectTimeout)
	defer connectCancel()

	// Discover the WebSocket URL fresh on every Authenticate() call.
	// The webSocketDebuggerUrl contains a per-session UUID that may change
	// when Chromium restarts or new sessions are created.
	// Use the resolved IP address to avoid Chromium's 500 error on FQDN hosts.
	wsURL, err := discoverWebSocketURL(connectCtx, b.resolvedHost, b.cfg.RemotePort, b.cfg.ConnectTimeout)
	if err != nil {
		return AuthResult{}, fmt.Errorf("%w: %v", ErrBrowserConnect, err)
	}

	b.browser = rod.New().ControlURL(wsURL).Context(connectCtx)
	if err := b.browser.Connect(); err != nil {
		debug(b.cfg, "connect failed: %v", err)
		return AuthResult{}, fmt.Errorf("%w: %v", ErrBrowserConnect, err)
	}
	debug(b.cfg, "connected to remote browser via %s", wsURL)

	var incognito *rod.Browser

	if b.cfg.Incognito {
		debug(b.cfg, "creating incognito context")
		var err error
		incognito, err = b.browser.Incognito()
		if err != nil {
			return AuthResult{}, fmt.Errorf("create incognito context: %w", err)
		}
		b.incognito = incognito
	} else {
		incognito = b.browser
	}

	page, err := navigateToAuthPage(authCtx, incognito, req, false, b.cfg)
	if err != nil {
		b.browser.Close()
		return AuthResult{}, err
	}
	b.page = page
	b.pageTargetID = page.TargetID

	cookies, err := extractCookiesForPage(page, authCtx)
	if err != nil {
		return AuthResult{}, err
	}

	return AuthResult{
		Cookies:     cookies,
		RedirectURL: getPageURL(page),
	}, nil
}

func (b *RemoteBrowser) FetchTimetable(ctx context.Context, weekDate string) (string, error) {
	if b.browser == nil {
		return "", fmt.Errorf("%w: browser not initialized", ErrBrowserUnavailable)
	}
	return fetchTimetable(ctx, b.page, weekDate, b.cfg)
}

func (b *RemoteBrowser) FetchBrightSpace(ctx context.Context, baseURL string) ([]BrightSpaceEntry, error) {
	if b.browser == nil {
		return nil, fmt.Errorf("%w: browser not initialized", ErrBrowserUnavailable)
	}
	return fetchBrightSpace(ctx, b.page, baseURL, b.cfg)
}

func (b *RemoteBrowser) Close() {
	if b.incognito != nil && b.incognito != b.browser {
		var pageIDs rod.Pages
		safeRod(func() {
			var err error
			pageIDs, err = b.incognito.Pages()
			if err != nil {
				pageIDs = nil
			}
		})

		if pageIDs == nil || len(pageIDs) <= 1 {
			safeRod(func() {
				b.incognito.Close()
			})
			b.incognito = nil
		} else {
			safeRod(func() {
				proto.TargetCloseTarget{TargetID: b.pageTargetID}.Call(b.browser)
			})
			b.page = nil
		}
	}
	if b.browser != nil {
		safeRod(func() {
			b.browser.Close()
		})
		b.browser = nil
	}
}

func (b *RemoteBrowser) debug(msg string, args ...any) {
	if b.cfg.Debug {
		log.Printf("browser: "+msg, args...)
	}
}
