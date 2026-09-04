package browser

import (
	"context"
	"fmt"
	"log"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
)

type LocalBrowser struct {
	cfg         BrowserConfig
	launcherURL string
	browser     *rod.Browser
	incognito   *rod.Browser
	page        *rod.Page
}

func NewLocalBrowser(cfg BrowserConfig) (*LocalBrowser, error) {
	return &LocalBrowser{cfg: cfg}, nil
}

func (b *LocalBrowser) Authenticate(ctx context.Context, req AuthRequest) (AuthResult, error) {
	authCtx, authCancel := context.WithTimeout(ctx, b.cfg.AuthTimeout)
	defer authCancel()

	launcherURL, err := b.launchBrowser(authCtx)
	if err != nil {
		return AuthResult{}, fmt.Errorf("%w: %v", ErrBrowserLaunch, err)
	}
	b.launcherURL = launcherURL

	connectCtx, connectCancel := context.WithTimeout(authCtx, b.cfg.ConnectTimeout)
	defer connectCancel()

	b.browser = rod.New().ControlURL(launcherURL).Context(connectCtx)
	if err := b.browser.Connect(); err != nil {
		debug(b.cfg, "connect failed: %v", err)
		return AuthResult{}, fmt.Errorf("%w: %v", ErrBrowserConnect, err)
	}
	debug(b.cfg, "connected to browser")

	var incognito *rod.Browser

	if b.cfg.Incognito {
		debug(b.cfg, "creating incognito context")
		incognito, err = b.browser.Incognito()
		if err != nil {
			return AuthResult{}, fmt.Errorf("create incognito context: %w", err)
		}
		b.incognito = incognito
	} else {
		incognito = b.browser
	}

	page, err := navigateToAuthPage(authCtx, incognito, req, true, b.cfg)
	if err != nil {
		b.browser.Close()
		return AuthResult{}, err
	}
	b.page = page

	cookies, err := extractCookies(authCtx, func(msg string, args ...any) { debug(b.cfg, msg, args...) }, page)
	if err != nil {
		return AuthResult{}, err
	}

	return AuthResult{
		Cookies:     cookies,
		RedirectURL: getPageURL(page),
	}, nil
}

func (b *LocalBrowser) FetchTimetable(ctx context.Context, weekDate string) (string, error) {
	if b.browser == nil {
		return "", fmt.Errorf("%w: browser not initialized", ErrBrowserUnavailable)
	}
	return fetchTimetable(ctx, b.page, weekDate, b.cfg)
}

func (b *LocalBrowser) Close() {
	if b.incognito != nil {
		safeRod(func() {
			b.incognito.Close()
		})
		b.incognito = nil
	}
	if b.browser != nil {
		safeRod(func() {
			b.browser.Close()
		})
		b.browser = nil
		b.page = nil
	}
}

func (b *LocalBrowser) launchBrowser(ctx context.Context) (string, error) {
	debug(b.cfg, "launching browser")

	launcherInst := launcher.New()

	if b.cfg.Executable != "" {
		debug(b.cfg, "using executable: %s", b.cfg.Executable)
		launcherInst = launcherInst.Bin(b.cfg.Executable)
	}

	if b.cfg.ProxyURL != "" {
		debug(b.cfg, "using proxy: %s", b.cfg.ProxyURL)
		launcherInst = launcherInst.Proxy(b.cfg.ProxyURL)
	}

	launcherInst = launcherInst.Headless(b.cfg.Headless)
	debug(b.cfg, "headless: %t", b.cfg.Headless)
	launcherInst = launcherInst.NoSandbox(true)
	launcherInst = launcherInst.Set("disable-gpu", "true")
	launcherInst = launcherInst.Set("disable-dev-shm-usage", "true")
	launcherInst = launcherInst.Set("disable-setuid-sandbox", "true")

	url, err := launcherInst.Launch()
	if err != nil {
		debug(b.cfg, "launch failed: %v", err)
		return "", fmt.Errorf("launch browser: %w", err)
	}

	debug(b.cfg, "browser launched at %s", url)
	return url, nil
}

func (b *LocalBrowser) debug(msg string, args ...any) {
	if b.cfg.Debug {
		log.Printf("browser: "+msg, args...)
	}
}
