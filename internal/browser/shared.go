package browser

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/ciel-shieru/sit-ics-go/internal/totp"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
	"golang.org/x/net/html"
)

func safeRod(f func()) {
	defer func() { recover() }()
	f()
}

func getPageURL(page *rod.Page) string {
	var url string
	safeRod(func() {
		url = page.MustInfo().URL
	})
	return url
}

func findActivePage(browser *rod.Browser) (*rod.Page, error) {
	var pages rod.Pages
	safeRod(func() {
		var err error
		pages, err = browser.Pages()
		if err != nil {
			return
		}
	})

	if pages == nil || len(pages) == 0 {
		return nil, fmt.Errorf("no pages found")
	}

	for _, p := range pages {
		var pageURL string
		safeRod(func() {
			pageURL = p.MustInfo().URL
		})
		if strings.Contains(pageURL, "singaporetech.edu.sg") {
			return p, nil
		}
	}

	return pages[0], nil
}

func extractCookies(ctx context.Context, debugFn func(string, ...any), page *rod.Page) ([]Cookie, error) {
	cookieCtx, cookieCancel := context.WithTimeout(ctx, 10*time.Second)
	defer cookieCancel()
	ctx = cookieCtx

	page = page.Context(cookieCtx)

	var rodCookies []*proto.NetworkCookie
	safeRod(func() {
		rodCookies = page.MustCookies(page.MustInfo().URL)
	})

	if rodCookies == nil {
		debugFn("no cookies from current page URL, trying to get all browser cookies")
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
			safeRod(func() {
				extra := page.Context(cookieCtx).MustCookies(domainURL)
				rodCookies = append(rodCookies, extra...)
			})
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

func navigateToAuthPage(ctx context.Context, incognito *rod.Browser, req AuthRequest, waitNavigation bool, cfg BrowserConfig) (*rod.Page, error) {
	navigateCtx, navigateCancel := context.WithTimeout(ctx, cfg.NavigationTimeout)
	defer navigateCancel()

	initialURL := req.URL
	debug(cfg, "navigating to %s", initialURL)
	page := incognito.MustPage(initialURL).Context(navigateCtx)

	if err := page.WaitStable(3000); err != nil {
		debug(cfg, "wait stable failed: %v", err)
	}

	finalURL := page.MustInfo().URL
	debug(cfg, "navigated to %s", finalURL)
	if !isAllowedOrigin(finalURL) {
		return nil, fmt.Errorf("%w: redirect to disallowed origin %s (started from %s)", ErrAuthentication, finalURL, initialURL)
	}

	if waitNavigation {
		page.WaitNavigation(proto.PageLifecycleEventNameNetworkAlmostIdle)()
	}

	debug(cfg, "finding username field")
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

	debug(cfg, "filling username")
	if err := el.Input(req.Username); err != nil {
		return nil, fmt.Errorf("%w: failed to fill username: %v", ErrCredentialExtraction, err)
	}

	debug(cfg, "finding password field")
	passEl, err := page.Element("#passwordInput")
	if err != nil {
		return nil, fmt.Errorf("%w: password field not found: %v", ErrAuthentication, err)
	}
	debug(cfg, "filling password")
	if err := passEl.Input(req.Password); err != nil {
		return nil, fmt.Errorf("%w: failed to fill password: %v", ErrCredentialExtraction, err)
	}

	debug(cfg, "finding submit button")
	submitEl, err := page.Element("#submitButton")
	if err != nil {
		return nil, fmt.Errorf("%w: submit button not found: %v", ErrAuthentication, err)
	}
	debug(cfg, "clicking submit button")
	if err := submitEl.Click(proto.InputMouseButtonLeft, 1); err != nil {
		return nil, fmt.Errorf("%w: failed to submit credentials: %v", ErrAuthentication, err)
	}

	if waitNavigation {
		page.WaitNavigation(proto.PageLifecycleEventNameNetworkAlmostIdle)()
	}
	if err := page.WaitStable(5000); err != nil {
		debug(cfg, "wait stable after submit failed: %v", err)
	}

	debug(cfg, "checking for MFA field")
	mfaEl, err := page.Element("#verificationCodeInput")
	mfaVisible := err == nil

	if mfaVisible {
		debug(cfg, "MFA detected, generating TOTP code")
		totpCode, err := totp.GenerateWithTolerance(req.TOTPSecret, time.Now(), 1)
		if err != nil {
			return nil, fmt.Errorf("%w: failed to generate TOTP: %v", ErrAuthentication, err)
		}
		debug(cfg, "filling MFA code")
		if err := mfaEl.Input(totpCode); err != nil {
			return nil, fmt.Errorf("%w: failed to fill MFA code: %v", ErrCredentialExtraction, err)
		}
		debug(cfg, "finding sign-in button")
		signInEl, err := page.Element("#signInButton")
		if err != nil {
			return nil, fmt.Errorf("%w: sign-in button not found: %v", ErrAuthentication, err)
		}
		debug(cfg, "clicking sign-in button")
		if err := signInEl.Click(proto.InputMouseButtonLeft, 1); err != nil {
			return nil, fmt.Errorf("%w: failed to submit MFA code: %v", ErrAuthentication, err)
		}
		debug(cfg, "waiting for SAML redirect after MFA")
		page.WaitNavigation(proto.PageLifecycleEventNameNetworkAlmostIdle)()
		if err := page.WaitStable(5000); err != nil {
			debug(cfg, "wait stable after MFA redirect failed: %v", err)
		}

		debug(cfg, "checking for MFA error message")
		errorMsg, err := extractErrorMessage(page)
		if err != nil {
			return nil, fmt.Errorf("%w: failed to check for error message: %v", ErrAuthentication, err)
		}
		if errorMsg != "" {
			return nil, fmt.Errorf("ADFS auth error: %s", errorMsg)
		}
	} else {
		debug(cfg, "no MFA field detected, waiting for redirect")
		page.WaitNavigation(proto.PageLifecycleEventNameNetworkAlmostIdle)()
		if err := page.WaitStable(5000); err != nil {
			debug(cfg, "wait stable after submit redirect failed: %v", err)
		}
	}

	debug(cfg, "waiting for ADFS redirect to complete")
	page.WaitNavigation(proto.PageLifecycleEventNameNetworkAlmostIdle)()
	if err := page.WaitStable(5000); err != nil {
		debug(cfg, "wait stable after redirect failed: %v", err)
	}

	finalURL = getPageURL(page)
	debug(cfg, "auth complete, final URL: %s", finalURL)

	return page, nil
}

func fetchTimetable(ctx context.Context, page *rod.Page, weekDate string, cfg BrowserConfig) (string, error) {
	if page == nil {
		return "", fmt.Errorf("no active page: authenticate first")
	}

	fetchCtx, fetchCancel := context.WithTimeout(ctx, cfg.NavigationTimeout)
	defer fetchCancel()

	page = page.Context(fetchCtx)

	timetableURL := "https://in4sit.singaporetech.edu.sg/psc/CSSISSTD/EMPLOYEE/SA/c/SA_LEARNER_SERVICES.SSR_SSENRL_LIST.GBL"
	debug(cfg, "navigating to timetable endpoint: %s", timetableURL)
	if err := page.Navigate(timetableURL); err != nil {
		return "", fmt.Errorf("navigate to timetable: %w", err)
	}

	page.WaitNavigation(proto.PageLifecycleEventNameNetworkAlmostIdle)()
	if err := page.WaitStable(5000); err != nil {
		debug(cfg, "wait stable failed: %v", err)
	}

	htmlStr, err := page.HTML()
	if err != nil {
		return "", fmt.Errorf("get page HTML: %w", err)
	}

	debug(cfg, "timetable HTML extracted, length: %d", len(htmlStr))
	return htmlStr, nil
}

func debug(cfg BrowserConfig, msg string, args ...any) {
	if cfg.Debug {
		log.Printf("browser: "+msg, args...)
	}
}
