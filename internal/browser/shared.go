package browser

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
)

func safeRod(f func()) {
	defer func() { recover() }()
	f()
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

func fetchTimetable(ctx context.Context, page *rod.Page, cfg BrowserConfig) (string, error) {
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
	if err := page.WaitStable(5 * time.Second); err != nil {
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
	if err := page.WaitStable(5 * time.Second); err != nil {
		debug(cfg, "brightspace: wait stable after SAML auth failed: %v", err)
	}

	debug(cfg, "brightspace: SAML auth complete")
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
	version, err := DecodeJSONField(page, fmt.Sprintf("%s/d2l/api/le/versions/", baseURL), "LatestVersion", cfg, ctx)
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
	if err := DecodeJSON(page, fmt.Sprintf("%s/d2l/le/manageCourses/api/mycourses", baseURL), &coursesResp, cfg, ctx); err != nil {
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
		if err := DecodeJSON(page, fmt.Sprintf("%s/d2l/api/le/%s/%s/calendar/events/", baseURL, version, orgUnitID), &calendarEvents, cfg, ctx); err != nil {
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
		if err := DecodeJSON(page, fmt.Sprintf("%s/d2l/api/le/%s/%s/dropbox/folders/", baseURL, version, orgUnitID), &folders, cfg, ctx); err != nil {
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
