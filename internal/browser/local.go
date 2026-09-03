package browser

import (
	"context"
	"fmt"
	"log"
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

	cookies, err := b.extractCookies(authCtx, page)
	if err != nil {
		return AuthResult{}, err
	}

	return AuthResult{
		Cookies:     cookies,
		RedirectURL: getPageURL(page),
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
		totpCode, err := totp.GenerateWithTolerance(req.TOTPSecret, time.Now(), 1)
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
		b.debug("waiting for SAML redirect after MFA")
		page.WaitNavigation(proto.PageLifecycleEventNameNetworkAlmostIdle)()
		if err := page.WaitStable(5000); err != nil {
			b.debug("wait stable after MFA redirect failed: %v", err)
		}

		b.debug("checking for MFA error message")
		errorMsg, err := extractErrorMessage(page)
		if err != nil {
			return nil, fmt.Errorf("%w: failed to check for error message: %v", ErrAuthentication, err)
		}
		if errorMsg != "" {
			return nil, fmt.Errorf("ADFS auth error: %s", errorMsg)
		}
	} else {
		b.debug("no MFA field detected, waiting for redirect")
		page.WaitNavigation(proto.PageLifecycleEventNameNetworkAlmostIdle)()
		if err := page.WaitStable(5000); err != nil {
			b.debug("wait stable after submit redirect failed: %v", err)
		}
	}

	b.debug("waiting for ADFS redirect to complete")
	page.WaitNavigation(proto.PageLifecycleEventNameNetworkAlmostIdle)()
	if err := page.WaitStable(5000); err != nil {
		b.debug("wait stable after redirect failed: %v", err)
	}

	finalURL = getPageURL(page)
	b.debug("auth complete, final URL: %s", finalURL)

	b.debug("waiting for PeopleSoft landing page")
	landingPageURL := "https://in4sit.singaporetech.edu.sg/psc/CSSISSTD/EMPLOYEE/SA/c/NUI_FRAMEWORK.PT_LANDINGPAGE.GBL"
	landingCtx, landingCancel := context.WithTimeout(ctx, b.cfg.NavigationTimeout)
	defer landingCancel()
	page = page.Context(landingCtx)

	landingWait := page.WaitNavigation(proto.PageLifecycleEventNameNetworkAlmostIdle)
	navigationDone := make(chan struct{})
	go func() {
		for {
			currentURL := getPageURL(page)
			if currentURL == landingPageURL || (len(currentURL) >= len(landingPageURL) && currentURL[:len(landingPageURL)] == landingPageURL && (currentURL[len(landingPageURL)] == '?' || currentURL[len(landingPageURL)] == '#')) {
				navigationDone <- struct{}{}
				return
			}
			select {
			case <-landingCtx.Done():
				return
			default:
			}
			time.Sleep(500 * time.Millisecond)
		}
	}()

	landingWait()
	<-navigationDone
	if err := page.WaitStable(3000); err != nil {
		b.debug("wait stable on landing page failed: %v", err)
	}

	finalURL = getPageURL(page)
	b.debug("landed on PeopleSoft URL: %s", finalURL)

	return page, nil
}

func extractErrorMessage(page *rod.Page) (string, error) {
	htmlStr, err := page.HTML()
	if err != nil {
		return "", fmt.Errorf("get page HTML: %w", err)
	}

	doc, err := html.Parse(strings.NewReader(htmlStr))
	if err != nil {
		return "", fmt.Errorf("parse HTML: %w", err)
	}

	var findErrorText func(*html.Node) string
	findErrorText = func(n *html.Node) string {
		if n.Type == html.ElementNode && n.Data == "p" {
			for _, a := range n.Attr {
				if a.Key == "id" && a.Val == "errorText" {
					var sb strings.Builder
					for c := n.FirstChild; c != nil; c = c.NextSibling {
						if c.Type == html.TextNode {
							sb.WriteString(c.Data)
						}
					}
					return strings.TrimSpace(sb.String())
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if result := findErrorText(c); result != "" {
				return result
			}
		}
		return ""
	}

	return findErrorText(doc), nil
}

func getPageURL(page *rod.Page) string {
	var url string
	func() {
		defer func() { recover() }()
		url = page.MustInfo().URL
	}()
	return url
}

func (b *LocalBrowser) extractCookies(ctx context.Context, page *rod.Page) ([]Cookie, error) {
	cookieCtx, cookieCancel := context.WithTimeout(ctx, 10*time.Second)
	defer cookieCancel()
	ctx = cookieCtx

	page = page.Context(cookieCtx)

	var rodCookies []*proto.NetworkCookie
	func() {
		defer func() { recover() }()
		rodCookies = page.MustCookies(page.MustInfo().URL)
	}()

	if rodCookies == nil {
		b.debug("no cookies from current page URL, trying to get all browser cookies")
		pageURL := getPageURL(page)
		var domains []string
		if strings.Contains(pageURL, "in4sit.singaporetech.edu.sg") {
			domains = append(domains, "https://in4sit.singaporetech.edu.sg/")
		}
		if strings.Contains(pageURL, "fs.singaporetech.edu.sg") {
			domains = append(domains, "https://fs.singaporetech.edu.sg/")
		}
		if !strings.Contains(pageURL, "in4sit.singaporetech.edu.sg") {
			domains = append(domains, "https://in4sit.singaporetech.edu.sg/")
		}
		if !strings.Contains(pageURL, "fs.singaporetech.edu.sg") {
			domains = append(domains, "https://fs.singaporetech.edu.sg/")
		}
		for _, domainURL := range domains {
			func() {
				defer func() { recover() }()
				extra := page.Context(cookieCtx).MustCookies(domainURL)
				rodCookies = append(rodCookies, extra...)
			}()
		}
	}

	if rodCookies == nil {
		return nil, fmt.Errorf("%w: page became invalid during cookie extraction (session closed or navigated away)", ErrAuthentication)
	}

	cookieMap := make(map[string]Cookie)
	for _, c := range rodCookies {
		if !isSingaporeTechDomain(c.Domain) {
			continue
		}
		expiry := int64(c.Expires)
		existing, exists := cookieMap[c.Name]
		if !exists || expiry > existing.Expiry {
			cookieMap[c.Name] = Cookie{
				Name:   c.Name,
				Value:  c.Value,
				Domain: c.Domain,
				Path:   c.Path,
				Expiry: expiry,
			}
		}
	}

	cookies := make([]Cookie, 0, len(cookieMap))
	for _, c := range cookieMap {
		cookies = append(cookies, c)
	}

	return cookies, nil
}

func isSingaporeTechDomain(domain string) bool {
	return strings.Contains(domain, "singaporetech.edu.sg")
}
