package browser

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"strings"
	"time"

	"github.com/ciel-shieru/sit-ics-go/internal/totp"
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

func (b *RemoteBrowser) debug(msg string, args ...any) {
	if b.cfg.Debug {
		log.Printf("browser: "+msg, args...)
	}
}

func (b *RemoteBrowser) Authenticate(ctx context.Context, req AuthRequest) (AuthResult, error) {
	authCtx, authCancel := context.WithTimeout(ctx, b.cfg.AuthTimeout)
	defer authCancel()

	connectCtx, connectCancel := context.WithTimeout(authCtx, b.cfg.ConnectTimeout)
	defer connectCancel()

	b.browser = rod.New().ControlURL(b.cfg.ControlURL).Context(connectCtx)
	if err := b.browser.Connect(); err != nil {
		b.debug("connect failed: %v", err)
		return AuthResult{}, fmt.Errorf("%w: %v", ErrBrowserConnect, err)
	}
	b.debug("connected to remote browser")

	var incognito *rod.Browser

	if b.cfg.Incognito {
		b.debug("creating incognito context")
		var err error
		incognito, err = b.browser.Incognito()
		if err != nil {
			return AuthResult{}, fmt.Errorf("create incognito context: %w", err)
		}
		b.incognito = incognito
	} else {
		incognito = b.browser
	}

	page, err := b.connectAndAuth(authCtx, incognito, req)
	if err != nil {
		b.browser.Close()
		return AuthResult{}, err
	}
	b.page = page
	b.pageTargetID = page.TargetID

	cookies, err := b.extractCookies(authCtx, page)
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

	fetchCtx, fetchCancel := context.WithTimeout(ctx, b.cfg.NavigationTimeout)
	defer fetchCancel()

	page := b.page
	if page == nil {
		return "", fmt.Errorf("no active page: authenticate first")
	}
	page = page.Context(fetchCtx)

	// Navigate to the PeopleSoft start page first to initialize session state.
	// Without this initial request, subsequent timetable data will not be returned.
	startURL := "https://in4sit.singaporetech.edu.sg/psc/CSSISSTD/EMPLOYEE/SA/c/SA_LEARNER_SERVICES.SSR_SSENRL_SCHD_W.GBL?ICAGTarget=start&ICAJAX=1"
	b.debug("navigating to PeopleSoft start page")
	if err := page.Navigate(startURL); err != nil {
		b.debug("navigating to PeopleSoft start page failed: %v", err)
	}

	b.debug("navigating to timetable endpoint")
	encodedDate := url.QueryEscape(weekDate)
	timetableURL := fmt.Sprintf(
		"https://in4sit.singaporetech.edu.sg/psc/CSSISSTD/EMPLOYEE/SA/c/SA_LEARNER_SERVICES.SSR_SSENRL_SCHD_W.GBL?ICAJAX=1&ICAction=DERIVED_CLASS_S_SSR_REFRESH_CAL$8$&DERIVED_CLASS_S_START_DT=%s",
		encodedDate,
	)

	if err := page.Navigate(timetableURL); err != nil {
		b.debug("navigating to timetable endpoint failed: %v", err)
	}

	body, err := page.Element("body")
	if err != nil {
		return "", fmt.Errorf("get page HTML: %w", err)
	}

	xml, err := body.Text()
	if err != nil {
		return "", fmt.Errorf("read XML document text: %w", err)
	}

	b.debug("timetable HTML extracted, length: %d", len(xml))
	return xml, nil
}

func (b *RemoteBrowser) findActivePage(browser *rod.Browser) (*rod.Page, error) {
	var pages rod.Pages
	func() {
		defer func() { recover() }()
		var err error
		pages, err = browser.Pages()
		if err != nil {
			return
		}
	}()

	if pages == nil || len(pages) == 0 {
		return nil, fmt.Errorf("no pages found")
	}

	for _, p := range pages {
		var pageURL string
		func() {
			defer func() { recover() }()
			pageURL = p.MustInfo().URL
		}()
		if strings.Contains(pageURL, "singaporetech.edu.sg") {
			return p, nil
		}
	}

	return pages[0], nil
}

func (b *RemoteBrowser) Close() {
	if b.incognito != nil && b.incognito != b.browser {
		var pageIDs rod.Pages
		func() {
			defer func() { recover() }()
			var err error
			pageIDs, err = b.incognito.Pages()
			if err != nil {
				pageIDs = nil
			}
		}()

		if pageIDs == nil || len(pageIDs) <= 1 {
			func() {
				defer func() { recover() }()
				b.incognito.Close()
			}()
			b.incognito = nil
		} else {
			func() {
				defer func() { recover() }()
				proto.TargetCloseTarget{TargetID: b.pageTargetID}.Call(b.browser)
			}()
			b.page = nil
		}
	}
	if b.browser != nil {
		func() {
			defer func() { recover() }()
			b.browser.Close()
		}()
		b.browser = nil
	}
}

func (b *RemoteBrowser) connectAndAuth(ctx context.Context, incognito *rod.Browser, req AuthRequest) (*rod.Page, error) {
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

	if err := page.WaitStable(5000); err != nil {
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

	return page, nil
}

func (b *RemoteBrowser) extractCookies(ctx context.Context, page *rod.Page) ([]Cookie, error) {
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
