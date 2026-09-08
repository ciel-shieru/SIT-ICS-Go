package browser

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/ciel-shieru/sit-ics-go/internal/brightspace"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
)

// RodFetcher implements brightspace.Fetcher using a rod.Page for browser-based API calls.
type RodFetcher struct {
	page *rod.Page
	cfg  BrowserConfig
	ctx  context.Context
}

// NewRodFetcher creates a brightspace.Fetcher that uses the given rod page.
func NewRodFetcher(page *rod.Page, cfg BrowserConfig) *RodFetcher {
	return &RodFetcher{
		page: page,
		cfg:  cfg,
		ctx:  context.Background(),
	}
}

// Navigate navigates the page to the given URL.
func (f *RodFetcher) Navigate(url string) error {
	fetchCtx, cancel := context.WithTimeout(f.ctx, f.cfg.NavigationTimeout)
	defer cancel()
	page := f.page.Context(fetchCtx)

	debug(f.cfg, "brightspace fetching %s", url)
	if err := page.Navigate(url); err != nil {
		return fmt.Errorf("navigate to %s: %w", url, err)
	}

	page.WaitNavigation(proto.PageLifecycleEventNameNetworkAlmostIdle)()
	if err := page.WaitStable(3 * time.Second); err != nil {
		debug(f.cfg, "brightspace wait stable failed for %s: %v", url, err)
	}
	return nil
}

// DecodeJSON navigates to a URL and parses the JSON response into v.
func (f *RodFetcher) DecodeJSON(url string, v interface{}) error {
	if err := f.Navigate(url); err != nil {
		return err
	}

	text, err := extractPageText(f.page)
	if err != nil {
		return fmt.Errorf("extract response from %s: %w", url, err)
	}

	if text == "" {
		return nil
	}

	if err := json.Unmarshal([]byte(text), v); err != nil {
		return fmt.Errorf("parse JSON from %s: %w (body=%q)", url, err, text)
	}

	return nil
}

// FetchBrightSpace fetches BrightSpace entries using the authenticated browser session.
func FetchBrightSpace(ctx context.Context, page *rod.Page, baseURL string, cfg BrowserConfig) ([]BrightSpaceEntry, error) {
	if page == nil {
		return nil, fmt.Errorf("%w: no active page: authenticate first", ErrAuthentication)
	}

	baseURL = strings.TrimRight(baseURL, "/")

	// Authenticate to BrightSpace via SAML using the existing ADFS session.
	if err := authBrightSpace(ctx, page, baseURL, cfg); err != nil {
		return nil, fmt.Errorf("brightspace SAML auth: %w", err)
	}

	fetcher := NewRodFetcher(page, cfg)
	client := brightspace.NewClient(baseURL, fetcher)

	events, folders, err := brightspace.Fetch(ctx, client)
	if err != nil {
		return nil, fmt.Errorf("brightspace fetch: %w", err)
	}

	// Convert API objects to BrightSpaceEntry (string-based timestamps)
	entries := make([]BrightSpaceEntry, 0, len(events)+len(folders))
	for _, ev := range events {
		se := brightspace.APIToStringEntry(ev, "brightspace-calendar")
		entries = append(entries, BrightSpaceEntry{
			Title:       se.Title,
			OrgUnitId:   se.OrgUnitId,
			OrgUnitName: se.OrgUnitName,
			OrgUnitCode: se.OrgUnitCode,
			Location:    se.Location,
			Description: se.Description,
			DTStart:     se.DTStart,
			DTEnd:       se.DTEnd,
			IsAllDay:    se.IsAllDay,
			Source:      se.Source,
		})
	}
	for _, folder := range folders {
		se := brightspace.FolderToStringEntry(folder)
		entries = append(entries, BrightSpaceEntry{
			Title:       se.Title,
			OrgUnitId:   se.OrgUnitId,
			OrgUnitName: se.OrgUnitName,
			OrgUnitCode: se.OrgUnitCode,
			Location:    se.Location,
			Description: se.Description,
			DTStart:     se.DTStart,
			DTEnd:       se.DTEnd,
			IsAllDay:    se.IsAllDay,
			Source:      se.Source,
		})
	}

	return entries, nil
}

// authBrightSpace navigates to the BrightSpace SAML login endpoint and waits
// for the ADFS redirect to complete, landing on /d2l/home.
func authBrightSpace(ctx context.Context, page *rod.Page, baseURL string, cfg BrowserConfig) error {
	samlURL := fmt.Sprintf("%s/d2l/lp/auth/saml/login", baseURL)
	debug(cfg, "brightspace: initiating SAML auth via %s", samlURL)

	freshCtx, freshCancel := context.WithTimeout(ctx, 5*time.Minute)
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
