package browser

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ciel-shieru/sit-ics-go/internal/totp"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"golang.org/x/net/html"
)

type LocalBrowser struct {
	cfg BrowserConfig
}

func NewLocalBrowser(cfg BrowserConfig) (*LocalBrowser, error) {
	return &LocalBrowser{cfg: cfg}, nil
}

func (b *LocalBrowser) debug(msg string, args ...any) {
	if b.cfg.Debug {
		log.Printf("browser: "+msg, args...)
	}
}

func (b *LocalBrowser) Authenticate(ctx context.Context, req AuthRequest) (AuthResult, error) {
	authCtx, authCancel := context.WithTimeout(ctx, b.cfg.AuthTimeout)
	defer authCancel()

	launcherURL, err := b.launchBrowser(authCtx)
	if err != nil {
		return AuthResult{}, fmt.Errorf("%w: %v", ErrBrowserLaunch, err)
	}

	connectCtx, connectCancel := context.WithTimeout(authCtx, b.cfg.ConnectTimeout)
	defer connectCancel()

	browser := rod.New().ControlURL(launcherURL).Context(connectCtx)
	if err := browser.Connect(); err != nil {
		b.debug("connect failed: %v", err)
		return AuthResult{}, fmt.Errorf("%w: %v", ErrBrowserConnect, err)
	}
	b.debug("connected to browser")
	defer browser.Close()

	var incognito *rod.Browser

	if b.cfg.Incognito {
		b.debug("creating incognito context")
		incognito, err = browser.Incognito()
		if err != nil {
			return AuthResult{}, fmt.Errorf("create incognito context: %w", err)
		}
		defer incognito.Close()
	} else {
		incognito = browser
	}

	page, err := b.navigateAndAuth(authCtx, incognito, req)
	if err != nil {
		return AuthResult{}, err
	}

	cookies, err := b.extractCookies(authCtx, page, "https://in4sit.singaporetech.edu.sg/")
	if err != nil {
		return AuthResult{}, err
	}

	return AuthResult{
		Cookies:     cookies,
		RedirectURL: page.MustInfo().URL,
	}, nil
}

func (b *LocalBrowser) launchBrowser(ctx context.Context) (string, error) {
	b.debug("launching browser")

	launcherInst := launcher.New()

	if b.cfg.Executable != "" {
		b.debug("using executable: %s", b.cfg.Executable)
		launcherInst = launcherInst.Bin(b.cfg.Executable)
	}

	if b.cfg.ProxyURL != "" {
		b.debug("using proxy: %s", b.cfg.ProxyURL)
		launcherInst = launcherInst.Proxy(b.cfg.ProxyURL)
	}

	launcherInst = launcherInst.Headless(b.cfg.Headless)
	b.debug("headless: %t", b.cfg.Headless)
	launcherInst = launcherInst.NoSandbox(true)
	launcherInst = launcherInst.Set("disable-gpu", "true")
	launcherInst = launcherInst.Set("disable-dev-shm-usage", "true")
	launcherInst = launcherInst.Set("disable-setuid-sandbox", "true")

	url, err := launcherInst.Launch()
	if err != nil {
		b.debug("launch failed: %v", err)
		return "", fmt.Errorf("launch browser: %w", err)
	}

	b.debug("browser launched at %s", url)
	return url, nil
}

func (b *LocalBrowser) navigateAndAuth(ctx context.Context, incognito *rod.Browser, req AuthRequest) (*rod.Page, error) {
	navigateCtx, navigateCancel := context.WithTimeout(ctx, b.cfg.NavigationTimeout)
	defer navigateCancel()

	initialURL := req.URL
	b.debug("navigating to %s", initialURL)
	page := incognito.MustPage(initialURL).Context(navigateCtx)

	if err := page.WaitStable(3000); err != nil {
		b.debug("wait stable failed: %v", err)
	}

	finalURL := page.MustInfo().URL
	b.debug("navigated to %s", finalURL)
	if !isAllowedOrigin(finalURL) {
		return nil, fmt.Errorf("%w: redirect to disallowed origin %s (started from %s)", ErrAuthentication, finalURL, initialURL)
	}

	page.WaitNavigation(proto.PageLifecycleEventNameNetworkAlmostIdle)()

	b.debug("finding username field")
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

	b.debug("filling username")
	if err := el.Input(req.Username); err != nil {
		return nil, fmt.Errorf("%w: failed to fill username: %v", ErrCredentialExtraction, err)
	}

	b.debug("finding password field")
	passEl, err := page.Element("#passwordInput")
	if err != nil {
		return nil, fmt.Errorf("%w: password field not found: %v", ErrAuthentication, err)
	}
	b.debug("filling password")
	if err := passEl.Input(req.Password); err != nil {
		return nil, fmt.Errorf("%w: failed to fill password: %v", ErrCredentialExtraction, err)
	}

	b.debug("finding submit button")
	submitEl, err := page.Element("#submitButton")
	if err != nil {
		return nil, fmt.Errorf("%w: submit button not found: %v", ErrAuthentication, err)
	}
	b.debug("clicking submit button")
	if err := submitEl.Click(proto.InputMouseButtonLeft, 1); err != nil {
		return nil, fmt.Errorf("%w: failed to submit credentials: %v", ErrAuthentication, err)
	}

	page.WaitNavigation(proto.PageLifecycleEventNameNetworkAlmostIdle)()
	if err := page.WaitStable(3000); err != nil {
		b.debug("wait stable after submit failed: %v", err)
	}

	b.debug("checking for MFA field")
	mfaEl, err := page.Element("#verificationCodeInput")
	mfaVisible := err == nil

	if mfaVisible {
		b.debug("MFA detected, generating TOTP code")
		totpCode, err := totp.Generate(req.TOTPSecret, time.Now())
		if err != nil {
			return nil, fmt.Errorf("%w: failed to generate TOTP: %v", ErrAuthentication, err)
		}
		b.debug("filling MFA code")
		if err := mfaEl.Input(totpCode); err != nil {
			return nil, fmt.Errorf("%w: failed to fill MFA code: %v", ErrCredentialExtraction, err)
		}
		b.debug("finding sign-in button")
		signInEl, err := page.Element("#signInButton")
		if err != nil {
			return nil, fmt.Errorf("%w: sign-in button not found: %v", ErrAuthentication, err)
		}
		b.debug("clicking sign-in button")
		if err := signInEl.Click(proto.InputMouseButtonLeft, 1); err != nil {
			return nil, fmt.Errorf("%w: failed to submit MFA code: %v", ErrAuthentication, err)
		}
		if err := page.WaitLoad(); err != nil {
			b.debug("wait load after MFA failed: %v", err)
		}
		if err := page.WaitStable(5000); err != nil {
			b.debug("wait stable after MFA failed: %v", err)
		}
		page.WaitNavigation(proto.PageLifecycleEventNameNetworkAlmostIdle)()
		if err := page.WaitStable(5000); err != nil {
			b.debug("wait stable after MFA redirect failed: %v", err)
		}
	} else {
		b.debug("no MFA field detected, waiting for redirect")
		page.WaitNavigation(proto.PageLifecycleEventNameNetworkAlmostIdle)()
		if err := page.WaitStable(5000); err != nil {
			b.debug("wait stable after submit redirect failed: %v", err)
		}
	}

	b.debug("extracting SAMLResponse from ADFS page")
	samlResponse, err := extractSAMLResponse(page)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to extract SAMLResponse: %v", ErrAuthentication, err)
	}
	if samlResponse == "" {
		return nil, fmt.Errorf("%w: SAMLResponse is empty", ErrAuthentication)
	}
	b.debug("SAMLResponse extracted (%d bytes)", len(samlResponse))

	b.debug("POSTing SAMLResponse to PeopleSoft landing page")
	samlPostCtx, samlPostCancel := context.WithTimeout(ctx, 30*time.Second)
	defer samlPostCancel()

	samlPostURL := "https://in4sit.singaporetech.edu.sg/psc/CSSISSTD/EMPLOYEE/SA/c/NUI_FRAMEWORK.PT_LANDINGPAGE.GBL"
	samlBody := fmt.Sprintf("SAMLResponse=%s", url.QueryEscape(samlResponse))
	samlReq, err := http.NewRequestWithContext(samlPostCtx, "POST", samlPostURL, strings.NewReader(samlBody))
	if err != nil {
		return nil, fmt.Errorf("create SAML POST request: %w", err)
	}
	samlReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	samlPage := incognito.MustPage(samlPostURL).Context(samlPostCtx)

	if err := samlPage.Navigate(samlReq.URL.String()); err != nil {
		return nil, fmt.Errorf("navigate to PeopleSoft with SAMLResponse: %w", err)
	}

	if err := samlPage.WaitLoad(); err != nil {
		b.debug("wait load after SAML POST failed: %v", err)
	}
	if err := samlPage.WaitStable(5000); err != nil {
		b.debug("wait stable after SAML POST failed: %v", err)
	}
	samlPage.WaitNavigation(proto.PageLifecycleEventNameNetworkAlmostIdle)()
	if err := samlPage.WaitStable(5000); err != nil {
		b.debug("wait stable after SAML POST redirect failed: %v", err)
	}

	b.debug("SAMLResponse POST complete, final URL: %s", samlPage.MustInfo().URL)

	return samlPage, nil
}

func extractSAMLResponse(page *rod.Page) (string, error) {
	htmlStr, err := page.HTML()
	if err != nil {
		return "", fmt.Errorf("get page HTML: %w", err)
	}

	doc, err := html.Parse(strings.NewReader(htmlStr))
	if err != nil {
		return "", fmt.Errorf("parse HTML: %w", err)
	}

	var findInput func(*html.Node) *html.Node
	findInput = func(n *html.Node) *html.Node {
		if n.Type == html.ElementNode && n.Data == "input" {
			for _, a := range n.Attr {
				if a.Key == "name" && a.Val == "SAMLResponse" {
					return n
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if result := findInput(c); result != nil {
				return result
			}
		}
		return nil
	}

	input := findInput(doc)
	if input == nil {
		return "", fmt.Errorf("SAMLResponse input not found in page")
	}

	for _, a := range input.Attr {
		if a.Key == "value" {
			return a.Val, nil
		}
	}

	var sb strings.Builder
	for c := input.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.TextNode {
			sb.WriteString(c.Data)
		}
	}
	val := strings.TrimSpace(sb.String())
	if val == "" {
		return "", fmt.Errorf("SAMLResponse value is empty")
	}
	return val, nil
}

func (b *LocalBrowser) extractCookies(ctx context.Context, page *rod.Page, targetURL string) ([]Cookie, error) {
	cookieCtx, cookieCancel := context.WithTimeout(ctx, 10*time.Second)
	defer cookieCancel()

	page = page.Context(cookieCtx)

	var rodCookies []*proto.NetworkCookie
	func() {
		defer func() { recover() }()
		rodCookies = page.MustCookies(targetURL)
	}()

	if rodCookies == nil {
		return nil, fmt.Errorf("%w: page became invalid during cookie extraction (session closed or navigated away)", ErrAuthentication)
	}

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
