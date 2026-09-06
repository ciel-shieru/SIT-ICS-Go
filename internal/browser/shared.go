package browser

import (
	"context"
	"encoding/json"
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
	_, err := page.Eval("() => window.location.href", &url)
	if err != nil {
		return ""
	}
	return url
}

// getPageURLFromPageInfo returns the URL from Chromium's CDP target info.
// This works on cross-origin pages where JS eval is blocked.
func getPageURLFromPageInfo(page *rod.Page) string {
	info, err := page.Info()
	if err != nil {
		return ""
	}
	return info.URL
}

// waitForPageURL polls the page URL using multiple methods until a non-empty
// URL is obtained or the timeout expires. This handles cross-origin navigation
// where Chromium's target URL may not be immediately available after redirect.
func waitForPageURL(page *rod.Page, timeout time.Duration) string {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		// Try CDP target info first (works on cross-origin pages).
		if url := getPageURLFromPageInfo(page); url != "" {
			return url
		}
		// Fall back to JS eval (works on same-origin pages).
		if url := getPageURL(page); url != "" {
			return url
		}
		time.Sleep(500 * time.Millisecond)
	}
	// Return whatever we can get on the final attempt.
	if url := getPageURLFromPageInfo(page); url != "" {
		return url
	}
	return getPageURL(page)
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

func htmlToPlainText(htmlStr string) string {
	htmlStr = strings.ReplaceAll(htmlStr, "<br>", "\n")
	htmlStr = strings.ReplaceAll(htmlStr, "<br/>", "\n")
	htmlStr = strings.ReplaceAll(htmlStr, "<br />", "\n")
	htmlStr = strings.ReplaceAll(htmlStr, "</p>", "\n")
	htmlStr = strings.ReplaceAll(htmlStr, "<p>", "")

	var result strings.Builder
	inTag := false
	for _, ch := range htmlStr {
		switch ch {
		case '<':
			inTag = true
		case '>':
			inTag = false
		default:
			if !inTag {
				result.WriteRune(ch)
			}
		}
	}

	text := result.String()
	text = strings.ReplaceAll(text, "&nbsp;", " ")
	text = strings.ReplaceAll(text, "&amp;", "&")
	text = strings.ReplaceAll(text, "&lt;", "<")
	text = strings.ReplaceAll(text, "&gt;", ">")
	return strings.TrimSpace(text)
}

// authBrightSpace navigates to the BrightSpace SAML login endpoint and waits
// for the ADFS redirect to complete, landing on /d2l/home.
func authBrightSpace(ctx context.Context, page *rod.Page, baseURL string, cfg BrowserConfig) error {
	samlURL := fmt.Sprintf("%s/d2l/lp/auth/saml/login", baseURL)
	debug(cfg, "brightspace: initiating SAML auth via %s", samlURL)

	// Reattach a fresh context to the page - the original context may be canceled.
	freshCtx, freshCancel := context.WithTimeout(ctx, cfg.AuthTimeout)
	defer freshCancel()
	page = page.Context(freshCtx)

	debug(cfg, "brightspace: navigating to SAML login")
	if err := page.Navigate(samlURL); err != nil {
		return fmt.Errorf("navigate to SAML login: %w", err)
	}

	page.WaitNavigation(proto.PageLifecycleEventNameNetworkAlmostIdle)()
	if err := page.WaitStable(5000); err != nil {
		debug(cfg, "brightspace: wait stable after SAML auth failed: %v", err)
	}

	finalURL := waitForPageURL(page, 5*time.Second)
	debug(cfg, "brightspace: SAML auth complete, landed on %s", finalURL)

	if !isAllowedOrigin(finalURL) {
		return fmt.Errorf("%w: BrightSpace SAML redirect to disallowed origin %q (expected %s)", ErrAuthentication, finalURL, baseURL)
	}

	return nil
}

func fetchBrightSpace(ctx context.Context, page *rod.Page, baseURL string, cfg BrowserConfig) ([]BrightSpaceEntry, error) {
	if page == nil {
		return nil, fmt.Errorf("%w: no active page: authenticate first", ErrAuthentication)
	}

	baseURL = strings.TrimRight(baseURL, "/")

	// Authenticate to BrightSpace via SAML using the existing ADFS session.
	if err := authBrightSpace(ctx, page, baseURL, cfg); err != nil {
		return nil, fmt.Errorf("brightspace SAML auth: %w", err)
	}

	// Step 1: Check API version
	version, err := fetchJSONField(ctx, page, fmt.Sprintf("%s/d2l/api/le/versions/", baseURL), "LatestVersion", cfg)
	if err != nil {
		return nil, fmt.Errorf("check version: %w", err)
	}
	debug(cfg, "brightspace API version: %s", version)

	// Step 2: Fetch courses
	type courseItem struct {
		OrgUnitId string `json:"OrgUnitId"`
		Name      string `json:"Name"`
		Code      string `json:"Code"`
	}
	type courseResponse struct {
		Courses []courseItem `json:"Courses"`
	}
	var coursesResp courseResponse
	if err := fetchJSON(ctx, page, fmt.Sprintf("%s/d2l/le/manageCourses/api/mycourses", baseURL), &coursesResp, cfg); err != nil {
		return nil, fmt.Errorf("fetch courses: %w", err)
	}
	courses := coursesResp.Courses
	debug(cfg, "brightspace: found %d courses", len(courses))

	// Step 3: For each course, fetch calendar events and dropbox folders
	var entries []BrightSpaceEntry

	for _, course := range courses {
		orgUnitID := course.OrgUnitId

		// Fetch calendar events
		var calendarEvents []CalendarEventAPI
		if err := fetchJSON(ctx, page, fmt.Sprintf("%s/d2l/api/le/%s/%s/calendar/events/", baseURL, version, orgUnitID), &calendarEvents, cfg); err != nil {
			debug(cfg, "brightspace: failed to fetch calendar events for %s: %v", course.Name, err)
			continue
		}
		for _, ev := range calendarEvents {
			title := ev.Title
			if title == "" {
				title = ev.OrgUnitName
			}

			descParts := []string{}
			if ev.Description != "" {
				plainDesc := htmlToPlainText(ev.Description)
				if plainDesc != "" {
					descParts = append(descParts, plainDesc)
				}
			}
			if ev.LocationName != "" {
				descParts = append(descParts, "Location: "+ev.LocationName)
			}

			entries = append(entries, BrightSpaceEntry{
				Title:       title,
				OrgUnitId:   orgUnitID,
				OrgUnitName: course.Name,
				OrgUnitCode: course.Code,
				Location:    ev.LocationName,
				Description: strings.Join(descParts, "\n"),
				DTStart:     ev.StartDateTime,
				DTEnd:       ev.EndDateTime,
				IsAllDay:    ev.IsAllDayEvent,
				Source:      "brightspace-calendar",
			})
		}

		// Fetch dropbox folders
		type dropboxFolderAPI struct {
			Id      int    `json:"Id"`
			Name    string `json:"Name"`
			DueDate string `json:"DueDate"`
		}
		var folders []dropboxFolderAPI
		if err := fetchJSON(ctx, page, fmt.Sprintf("%s/d2l/api/le/%s/%s/dropbox/folders/", baseURL, version, orgUnitID), &folders, cfg); err != nil {
			debug(cfg, "brightspace: failed to fetch dropbox folders for %s: %v", course.Name, err)
			continue
		}
		for _, folder := range folders {
			entries = append(entries, BrightSpaceEntry{
				Title:       fmt.Sprintf("[Submission Due] %s", folder.Name),
				OrgUnitId:   orgUnitID,
				OrgUnitName: course.Name,
				OrgUnitCode: course.Code,
				Description: "Dropbox: " + folder.Name,
				DTStart:     folder.DueDate,
				DTEnd:       folder.DueDate,
				IsAllDay:    false,
				Source:      "brightspace-dropbox",
			})
		}
	}

	debug(cfg, "brightspace: extracted %d entries", len(entries))
	return entries, nil
}

// CalendarEventAPI mirrors the BrightSpace calendar event JSON structure.
type CalendarEventAPI struct {
	CalendarEventId int       `json:"CalendarEventId"`
	OrgUnitId       int       `json:"OrgUnitId"`
	Title           string    `json:"Title"`
	Description     string    `json:"Description"`
	IsAllDayEvent   bool      `json:"IsAllDayEvent"`
	StartDateTime   string    `json:"StartDateTime"`
	EndDateTime     string    `json:"EndDateTime"`
	IsRecurring     bool      `json:"IsRecurring"`
	LocationName    string    `json:"LocationName"`
	OrgUnitName     string    `json:"OrgUnitName"`
	OrgUnitCode     string    `json:"OrgUnitCode"`
	EventType       int       `json:"EventType"`
}

// fetchJSON navigates to a URL and parses the JSON response into the given value.
func fetchJSON(ctx context.Context, page *rod.Page, url string, result interface{}, cfg BrowserConfig) error {
	fetchCtx, fetchCancel := context.WithTimeout(ctx, cfg.NavigationTimeout)
	defer fetchCancel()

	page = page.Context(fetchCtx)

	debug(cfg, "brightspace: fetching %s", url)
	if err := page.Navigate(url); err != nil {
		return fmt.Errorf("navigate to %s: %w", url, err)
	}

	page.WaitNavigation(proto.PageLifecycleEventNameNetworkAlmostIdle)()
	if err := page.WaitStable(3000); err != nil {
		debug(cfg, "brightspace: wait stable failed for %s: %v", url, err)
	}

	text, err := extractPageText(page)
	if err != nil {
		return fmt.Errorf("extract response from %s: %w", url, err)
	}

	if text == "" {
		return nil
	}

	if err := json.Unmarshal([]byte(text), result); err != nil {
		return fmt.Errorf("parse JSON from %s: %w (status=%d, body=%q)", url, err, 200, text)
	}

	return nil
}

// fetchJSONField navigates to a URL and extracts a specific JSON field as a string.
func fetchJSONField(ctx context.Context, page *rod.Page, url, field string, cfg BrowserConfig) (string, error) {
	type wrapper struct {
		Data map[string]interface{} `json:"d2l-api-response"`
	}

	fetchCtx, fetchCancel := context.WithTimeout(ctx, cfg.NavigationTimeout)
	defer fetchCancel()

	page = page.Context(fetchCtx)

	debug(cfg, "brightspace: fetching %s", url)
	if err := page.Navigate(url); err != nil {
		return "", fmt.Errorf("navigate to %s: %w", url, err)
	}

	page.WaitNavigation(proto.PageLifecycleEventNameNetworkAlmostIdle)()
	if err := page.WaitStable(3000); err != nil {
		debug(cfg, "brightspace: wait stable failed for %s: %v", url, err)
	}

	text, err := extractPageText(page)
	if err != nil {
		return "", fmt.Errorf("extract response from %s: %w", url, err)
	}

	if text == "" {
		return "", fmt.Errorf("empty response from %s", url)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal([]byte(text), &raw); err != nil {
		return "", fmt.Errorf("parse JSON from %s: %w", url, err)
	}

	val, ok := raw[field]
	if !ok {
		return "", fmt.Errorf("field %q not found in response from %s", field, url)
	}

	return fmt.Sprintf("%v", val), nil
}

// extractPageText extracts the text content of the page (works for JSON API responses).
func extractPageText(page *rod.Page) (string, error) {
	var result string
	safeRod(func() {
		val := page.MustEval("() => document.body ? document.body.innerText : document.documentElement.innerText")
		result = val.String()
	})
	return strings.TrimSpace(result), nil
}


