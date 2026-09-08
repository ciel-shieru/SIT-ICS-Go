package browser

import (
	"context"
	"fmt"
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
		fmt.Printf("browser: "+msg+"\n", args...)
	}
}
