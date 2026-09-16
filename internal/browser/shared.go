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



func getPageURL(page *rod.Page) string {
	var url string
	page.Eval("() => window.location.href", &url)
	return url
}

func fetchTimetable(ctx context.Context, page *rod.Page, cfg BrowserConfig) (string, error) {
	if page == nil {
		return "", fmt.Errorf("no active page: authenticate first")
	}
	if err := ctx.Err(); err != nil {
		return "", fmt.Errorf("timetable fetch context already expired: %w", err)
	}

	fetchCtx, fetchCancel := context.WithTimeout(ctx, cfg.NavigationTimeout)
	defer fetchCancel()
	page = page.Context(fetchCtx)
	if deadline, ok := fetchCtx.Deadline(); ok {
		debug(cfg, "timetable navigation deadline: %s (remaining %s)", deadline.Format(time.RFC3339Nano), time.Until(deadline).Round(time.Millisecond))
	}
	defer fetchCancel()

	page = page.Context(fetchCtx)

	timetableURL := "https://in4sit.singaporetech.edu.sg/psc/CSSISSTD/EMPLOYEE/SA/c/SA_LEARNER_SERVICES.SSR_SSENRL_LIST.GBL"
	debug(cfg, "navigating to timetable endpoint: %s", timetableURL)
	if err := page.Navigate(timetableURL); err != nil {
		return "", fmt.Errorf("navigate to timetable: %w", err)
	}

	// Wait for navigation with context awareness
	navDone := make(chan struct{})
	go func() {
		page.WaitNavigation(proto.PageLifecycleEventNameNetworkAlmostIdle)()
		close(navDone)
	}()

	select {
	case <-fetchCtx.Done():
		return "", fmt.Errorf("navigate to timetable: %w", fetchCtx.Err())
	case <-navDone:
	}

	// Wait for stable with context awareness
	stableDone := make(chan error)
	go func() {
		stableDone <- page.WaitStable(5000)
	}()

	select {
	case <-fetchCtx.Done():
		return "", fmt.Errorf("wait for stable: %w", fetchCtx.Err())
	case err := <-stableDone:
		if err != nil {
			debug(cfg, "wait stable failed: %v", err)
		}
	}

	if err := handleTermSelection(fetchCtx, page, cfg); err != nil {
		return "", fmt.Errorf("handle term selection: %w", err)
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

func handleTermSelection(ctx context.Context, page *rod.Page, cfg BrowserConfig) error {
	debug(cfg, "checking for term selection screen")
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("term selection context already expired: %w", err)
	}
	if deadline, ok := ctx.Deadline(); ok {
		debug(cfg, "term selection context deadline: %s (remaining %s)", deadline.Format(time.RFC3339Nano), time.Until(deadline).Round(time.Millisecond))
	}
	// Wait for term selection radio buttons to appear with timeout
	_, err := page.Timeout(5 * time.Second).Element("input.PSRADIOBUTTON")
	if err != nil {
		return fmt.Errorf("term selection radio button not found: %w", err)
	}

	radioButtons, err := page.Elements("input.PSRADIOBUTTON")
	if err != nil {
		return fmt.Errorf("query term selection radio buttons: %w", err)
	}

	if len(radioButtons) == 0 {
		debug(cfg, "no term selection radio buttons found, skipping")
		return nil
	}

	debug(cfg, "found %d term selection radio button(s)", len(radioButtons))

	lastRadio := radioButtons[len(radioButtons)-1]
	debug(cfg, "selecting last radio button (index %d of %d)", len(radioButtons)-1, len(radioButtons))
	if err := lastRadio.Click(proto.InputMouseButtonLeft, 1); err != nil {
		return fmt.Errorf("click term selection radio button: %w", err)
	}

	continueEl, err := page.Element("input[name='DERIVED_SSS_SCT_SSR_PB_GO']")
	if err != nil {
		return fmt.Errorf("find continue button: %w", err)
	}

	debug(cfg, "clicking continue button")
	if err := continueEl.Click(proto.InputMouseButtonLeft, 1); err != nil {
		return fmt.Errorf("click continue button: %w", err)
	}

	debug(cfg, "waiting for navigation after term selection")

	// Wait for navigation after term selection with context awareness
	navDone := make(chan struct{})
	go func() {
		page.WaitNavigation(proto.PageLifecycleEventNameNetworkAlmostIdle)()
		close(navDone)
	}()

	select {
	case <-ctx.Done():
		return fmt.Errorf("wait navigation after term selection: %w", ctx.Err())
	case <-navDone:
	}

	// Wait for stable with context awareness
	stableDone := make(chan error)
	go func() {
		stableDone <- page.WaitStable(5000)
	}()

	select {
	case <-ctx.Done():
		return fmt.Errorf("wait stable after term selection: %w", ctx.Err())
	case err := <-stableDone:
		if err != nil {
			debug(cfg, "wait stable after term selection failed: %v", err)
		}
	}

	debug(cfg, "term selection handled successfully")
	return nil
}

func extractPageText(page *rod.Page) (result string, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			result = ""
			err = fmt.Errorf("browser evaluation failed: %v", recovered)
		}
	}()

	value := page.MustEval(`() => document.body ? document.body.innerText : ""`)
	return strings.TrimSpace(value.String()), nil
}

func safeRod(fn func()) {
	defer func() { recover() }()
	fn()
}
