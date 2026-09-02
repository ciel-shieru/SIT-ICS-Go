package browser

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/ciel-shieru/sit-ics-go/internal/totp"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
)

type RemoteBrowser struct {
	cfg BrowserConfig
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

	page, err := b.connectAndAuth(authCtx, req)
	if err != nil {
		return AuthResult{}, err
	}

	cookies, err := b.extractCookies(authCtx, page, req.URL)
	if err != nil {
		return AuthResult{}, err
	}

	return AuthResult{
		Cookies:     cookies,
		RedirectURL: page.MustInfo().URL,
	}, nil
}

func (b *RemoteBrowser) connectAndAuth(ctx context.Context, req AuthRequest) (*rod.Page, error) {
	connectCtx, connectCancel := context.WithTimeout(ctx, b.cfg.ConnectTimeout)
	defer connectCancel()

	browser := rod.New().ControlURL(b.cfg.ControlURL).Context(connectCtx)
	if err := browser.Connect(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrBrowserConnect, err)
	}
	defer browser.Close()

	var incognito *rod.Browser
	var err error

	if b.cfg.Incognito {
		incognito, err = browser.Incognito()
		if err != nil {
			return nil, fmt.Errorf("create incognito context: %w", err)
		}
		defer incognito.Close()
	} else {
		incognito = browser
	}

	navigateCtx, navigateCancel := context.WithTimeout(ctx, b.cfg.NavigationTimeout)
	defer navigateCancel()

	initialURL := req.URL
	page := incognito.MustPage(initialURL).Context(navigateCtx)

	if err := page.WaitStable(3000); err != nil {
		log.Printf("browser: wait stable failed: %v", err)
	}

	finalURL := page.MustInfo().URL
	if !isAllowedOrigin(finalURL) {
		return nil, fmt.Errorf("%w: redirect to disallowed origin %s (started from %s)", ErrAuthentication, finalURL, initialURL)
	}

	el, err := page.Element("#userNameInput")
	if err != nil {
		return nil, fmt.Errorf("%w: credential form not rendered: %v", ErrAuthentication, err)
	}

	visible, err := el.Visible()
	if err != nil {
		return nil, fmt.Errorf("%w: credential form not visible: %v", ErrAuthentication, err)
	}
	if !visible {
		return nil, fmt.Errorf("%w: credential form not rendered: element not visible", ErrAuthentication)
	}

	if err := el.Input(req.Username); err != nil {
		return nil, fmt.Errorf("%w: failed to fill username: %v", ErrCredentialExtraction, err)
	}

	passEl, err := page.Element("#passwordInput")
	if err != nil {
		return nil, fmt.Errorf("%w: password field not found: %v", ErrAuthentication, err)
	}
	if err := passEl.Input(req.Password); err != nil {
		return nil, fmt.Errorf("%w: failed to fill password: %v", ErrCredentialExtraction, err)
	}

	submitEl, err := page.Element("#submitButton")
	if err != nil {
		return nil, fmt.Errorf("%w: submit button not found: %v", ErrAuthentication, err)
	}
	if err := submitEl.Click(proto.InputMouseButtonLeft, 1); err != nil {
		return nil, fmt.Errorf("%w: failed to submit credentials: %v", ErrAuthentication, err)
	}

	if err := page.WaitStable(5000); err != nil {
		log.Printf("browser: wait stable after submit failed: %v", err)
	}

	mfaEl, err := page.Element("#VerificationCode")
	mfaVisible := err == nil

	if mfaVisible {
		totpCode, err := totp.Generate(req.TOTPSecret, time.Now())
		if err != nil {
			return nil, fmt.Errorf("%w: failed to generate TOTP: %v", ErrAuthentication, err)
		}
		if err := mfaEl.Input(totpCode); err != nil {
			return nil, fmt.Errorf("%w: failed to fill MFA code: %v", ErrCredentialExtraction, err)
		}
		signInEl, err := page.Element("#SignIn")
		if err != nil {
			return nil, fmt.Errorf("%w: sign-in button not found: %v", ErrAuthentication, err)
		}
		if err := signInEl.Click(proto.InputMouseButtonLeft, 1); err != nil {
			return nil, fmt.Errorf("%w: failed to submit MFA code: %v", ErrAuthentication, err)
		}
		if err := page.WaitStable(5000); err != nil {
			log.Printf("browser: wait stable after MFA failed: %v", err)
		}
	}

	return page, nil
}

func (b *RemoteBrowser) extractCookies(ctx context.Context, page *rod.Page, targetURL string) ([]Cookie, error) {
	cookieCtx, cookieCancel := context.WithTimeout(ctx, 10*time.Second)
	defer cookieCancel()

	page = page.Context(cookieCtx)
	rodCookies := page.MustCookies(targetURL)

	cookies := make([]Cookie, 0, len(rodCookies))
	for _, c := range rodCookies {
		cookies = append(cookies, Cookie{
			Name:   c.Name,
			Value:  c.Value,
			Domain: c.Domain,
			Path:   c.Path,
		})
	}

	return cookies, nil
}
