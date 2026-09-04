package browser

import (
	"context"
	"fmt"
	"log"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
)

type RemoteBrowser struct {
	cfg           BrowserConfig
	launcherURL   string
	browser       *rod.Browser
	incognito     *rod.Browser
	page          *rod.Page
	pageTargetID  proto.TargetTargetID
}

func NewRemoteBrowser(cfg BrowserConfig) (*RemoteBrowser, error) {
	if cfg.ControlURL == "" {
		return nil, fmt.Errorf("remote browser requires ControlURL")
	}
	return &RemoteBrowser{cfg: cfg}, nil
}

func (b *RemoteBrowser) Authenticate(ctx context.Context, req AuthRequest) (AuthResult, error) {
	authCtx, authCancel := context.WithTimeout(ctx, b.cfg.AuthTimeout)
	defer authCancel()

	connectCtx, connectCancel := context.WithTimeout(authCtx, b.cfg.ConnectTimeout)
	defer connectCancel()

	b.browser = rod.New().ControlURL(b.cfg.ControlURL).Context(connectCtx)
	if err := b.browser.Connect(); err != nil {
		debug(b.cfg, "connect failed: %v", err)
		return AuthResult{}, fmt.Errorf("%w: %v", ErrBrowserConnect, err)
	}
	debug(b.cfg, "connected to remote browser")

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

	cookies, err := extractCookies(authCtx, func(msg string, args ...any) { debug(b.cfg, msg, args...) }, page)
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
