package browser

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/ciel-shieru/sit-ics-go/internal/brightspace"
	"github.com/ciel-shieru/sit-ics-go/internal/environment"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
)

type LocalBrowser struct {
	cfg           BrowserConfig
	launcherURL   string
	browser       *rod.Browser
	incognito     *rod.Browser
	page          *rod.Page
	browserCancel context.CancelFunc
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

	// The Rod browser/page object must outlive the authentication operation.
	// AuthenticateADFS derives a short-lived page context from this browser,
	// while Rod's page root remains tied to the browser's session context.
	// Therefore the connection context must not inherit authCtx or expire after
	// ConnectTimeout; otherwise later Click()/Hover()/WaitStableRAF() calls can
	// observe context.Canceled through the page root.
	browserCtx, browserCancel := context.WithCancel(context.Background())
	connectCtx, connectCancel := context.WithCancel(browserCtx)
	var connectTimer *time.Timer
	if b.cfg.ConnectTimeout > 0 {
		connectTimer = time.AfterFunc(b.cfg.ConnectTimeout, connectCancel)
	}

	b.browser = rod.New().ControlURL(launcherURL).Context(connectCtx)
	if err := b.browser.Connect(); err != nil {
		if connectTimer != nil {
			connectTimer.Stop()
		}
		connectCancel()
		browserCancel()
		debug(b.cfg, "connect failed: %v", err)
		return AuthResult{}, fmt.Errorf("%w: %v", ErrBrowserConnect, err)
	}
	if connectTimer != nil {
		connectTimer.Stop()
	}
	b.browserCancel = browserCancel
	_ = connectCancel
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

	page, err := AuthenticateADFS(authCtx, incognito, req, true, b.cfg, isAllowedOrigin)
	if err != nil {
		b.browser.Close()
		browserCancel()
		b.browser = nil
		return AuthResult{}, err
	}
	b.page = page

	return AuthResult{
		RedirectURL: getPageURL(page),
	}, nil
}

func (b *LocalBrowser) FetchTimetable(ctx context.Context, weekDate string) (string, error) {
	if b.browser == nil {
		return "", fmt.Errorf("%w: browser not initialized", ErrBrowserUnavailable)
	}
	return fetchTimetable(ctx, b.page, b.cfg)
}

func (b *LocalBrowser) FetchBrightSpace(ctx context.Context, baseURL string) ([]brightspace.BrightSpaceStringEntry, error) {
	if b.browser == nil {
		return nil, fmt.Errorf("%w: browser not initialized", ErrBrowserUnavailable)
	}
	return FetchBrightSpace(ctx, b.page, baseURL, b.cfg)
}

func (b *LocalBrowser) FetchBrightSpaceQuizzes(ctx context.Context, baseURL string) ([]brightspace.BrightSpaceStringEntry, error) {
	if b.browser == nil {
		return nil, fmt.Errorf("%w: browser not initialized", ErrBrowserUnavailable)
	}
	return FetchBrightSpaceQuizzes(ctx, b.page, baseURL, b.cfg)
}

func (b *LocalBrowser) FetchBrightSpaceQuizzesAPI(ctx context.Context, baseURL string) ([]brightspace.QuizAPI, error) {
	if b.browser == nil {
		return nil, fmt.Errorf("%w: browser not initialized", ErrBrowserUnavailable)
	}
	fetcher := NewRodFetcher(b.page, b.cfg)
	return brightspace.FetchQuizzes(ctx, brightspace.NewClient(baseURL, fetcher))
}

func (b *LocalBrowser) FetchQuizSubmissionPage(ctx context.Context, quizURL string) (string, error) {
	if b.page == nil {
		return "", fmt.Errorf("%w: browser not initialized", ErrBrowserUnavailable)
	}
	return FetchQuizSubmissionPage(ctx, b.page, quizURL, b.cfg)
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
	if b.browserCancel != nil {
		b.browserCancel()
		b.browserCancel = nil
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
	launcherInst = launcherInst.NoSandbox(environment.IsContainerized())
	launcherInst = launcherInst.Set("disable-gpu", "true")
	launcherInst = launcherInst.Set("disable-dev-shm-usage", "true")
	launcherInst = launcherInst.Set("disable-setuid-sandbox", strconv.FormatBool(environment.IsContainerized()))

	url, err := launcherInst.Launch()
	if err != nil {
		debug(b.cfg, "launch failed: %v", err)
		return "", fmt.Errorf("launch browser: %w", err)
	}

	debug(b.cfg, "browser launched at %s", url)
	return url, nil
}

// GetPage returns the active page for use by brightspace package.
func (b *LocalBrowser) GetPage() *rod.Page {
	return b.page
}
