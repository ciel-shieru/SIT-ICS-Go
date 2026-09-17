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

	// The caller owns the overall timetable-fetch deadline. NavigationTimeout is
	// not an appropriate lifetime for this whole operation because PeopleSoft
	// can return the page before the timetable DOM has finished rendering.
	fetchCtx, fetchCancel := context.WithCancel(ctx)
	defer fetchCancel()
	page = page.Context(fetchCtx)
	if deadline, ok := fetchCtx.Deadline(); ok {
		debug(cfg, "timetable navigation deadline: %s (remaining %s)", deadline.Format(time.RFC3339Nano), time.Until(deadline).Round(time.Millisecond))
	}

	timetableURL := "https://in4sit.singaporetech.edu.sg/psc/CSSISSTD/EMPLOYEE/SA/c/SA_LEARNER_SERVICES.SSR_SSENRL_LIST.GBL"
	debug(cfg, "navigating to timetable endpoint: %s", timetableURL)
	if err := page.Navigate(timetableURL); err != nil {
		return "", fmt.Errorf("navigate to timetable: %w", err)
	}

	// Do not wait for NetworkAlmostIdle. Rod's Navigate returns after the HTTP
	// response headers, while PeopleSoft can continue rendering asynchronously;
	// it can also keep background network activity alive. Wait for a concrete
	// DOM state that the rest of this fetch actually needs instead.
	if err := waitForTermSelectionOrTimetable(fetchCtx, page); err != nil {
		return "", fmt.Errorf("wait for timetable page: %w", err)
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

func waitForTermSelectionOrTimetable(ctx context.Context, page *rod.Page) error {
	return waitForDOMCondition(ctx, page, `() => {
		if (document.querySelector('input.PSRADIOBUTTON')) {
			return true
		}
		return Array.from(document.querySelectorAll('table')).some((table) =>
			(table.textContent || '').includes('Class Nbr')
		);
	}`)
}

func waitForTimetableTable(ctx context.Context, page *rod.Page) error {
	return waitForDOMCondition(ctx, page, `() => Array.from(document.querySelectorAll('table')).some((table) =>
		(table.textContent || '').includes('Class Nbr')
	)`)
}

func waitForDOMCondition(ctx context.Context, page *rod.Page, script string) error {
	const pollInterval = 200 * time.Millisecond
	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		var ready bool
		if _, err := page.Context(ctx).Eval(script, &ready); err == nil && ready {
			return nil
		}

		timer := time.NewTimer(pollInterval)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return ctx.Err()
		case <-timer.C:
		}
	}
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
	// Use the overall timetable context for the term-selection interaction. The
	// page may immediately begin a long PeopleSoft transition after a click, so
	// a short child context here can cancel an action that has already succeeded
	// in Chromium while Rod is finishing the interaction.
	termPage := page.Context(ctx)
	radioButtons, err := termPage.Elements("input.PSRADIOBUTTON")
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

	// Start waiting before Continue is clicked. Waiting for a lifecycle event
	// after the click is inherently racy: the navigation may already have begun
	// or the event may be missed. More importantly, NetworkAlmostIdle is not a
	// reliable completion signal for this PeopleSoft page.
	tableCtx, tableCancel := context.WithCancel(ctx)
	tableDone := make(chan error, 1)
	go func() {
		tableDone <- waitForTimetableTable(tableCtx, page)
	}()
	if err := continueEl.Click(proto.InputMouseButtonLeft, 1); err != nil {
		tableCancel()
		return fmt.Errorf("click continue button: %w", err)
	}

	debug(cfg, "waiting for timetable DOM after term selection")
	if err := <-tableDone; err != nil {
		tableCancel()
		return fmt.Errorf("wait for timetable after term selection: %w", err)
	}
	tableCancel()

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
