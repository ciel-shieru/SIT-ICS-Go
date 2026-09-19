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

	// Keep the caller's full timetable deadline for the whole operation.
	// Only the initial PeopleSoft navigation gets the shorter browser navigation
	// timeout. Applying NavigationTimeout to the entire fetch also limits the
	// subsequent term-selection/timetable rendering to 30 seconds.
	fetchPage := page.Context(ctx)
	timetableURL := "https://in4sit.singaporetech.edu.sg/psc/CSSISSTD/EMPLOYEE/SA/c/SA_LEARNER_SERVICES.SSR_SSENRL_LIST.GBL"
	debug(cfg, "navigating to timetable endpoint: %s", timetableURL)
	navCtx, navCancel := context.WithTimeout(ctx, cfg.NavigationTimeout)
	defer navCancel()
	navPage := fetchPage.Context(navCtx)
	if deadline, ok := navCtx.Deadline(); ok {
		debug(cfg, "initial timetable navigation deadline: %s (remaining %s)", deadline.Format(time.RFC3339Nano), time.Until(deadline).Round(time.Millisecond))
	}
	if err := navPage.Navigate(timetableURL); err != nil {
		return "", fmt.Errorf("navigate to timetable: %w", err)
	}

	// Wait for navigation with context awareness
	navDone := make(chan struct{})
	go func() {
		navPage.WaitNavigation(proto.PageLifecycleEventNameNetworkAlmostIdle)()
		close(navDone)
	}()

	select {
	case <-navCtx.Done():
		return "", fmt.Errorf("navigate to timetable: %w", navCtx.Err())
	case <-navDone:
	}

	if err := handleTermSelection(ctx, fetchPage, cfg); err != nil {
		return "", fmt.Errorf("handle term selection: %w", err)
	}

	if err := waitForTimetableDOM(ctx, fetchPage, cfg); err != nil {
		return "", fmt.Errorf("wait for timetable DOM: %w", err)
	}

	htmlStr, err := fetchPage.HTML()
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
	termCtx, termCancel := context.WithTimeout(ctx, 5*time.Second)
	defer termCancel()
	termPage := page.Context(termCtx)

	_, err := termPage.Element("input.PSRADIOBUTTON")
	if err != nil {
		return fmt.Errorf("term selection radio button not found: %w", err)
	}

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
	if err := continueEl.Click(proto.InputMouseButtonLeft, 1); err != nil {
		return fmt.Errorf("click continue button: %w", err)
	}

	debug(cfg, "term selection submitted successfully")
	return nil
}

func waitForTimetableDOM(ctx context.Context, page *rod.Page, cfg BrowserConfig) error {
	const pollInterval = 250 * time.Millisecond
	for {
		var ready bool
		if _, err := page.Eval(`() => Array.from(document.querySelectorAll('table')).some(table => {
			const row = table.querySelector('tr');
			if (!row) return false;
			const cells = row.querySelectorAll('td, th');
			if (cells.length < 7) return false;
			return (cells[0].textContent || '').includes('Class Nbr');
		})`, &ready); err != nil {
			if err := ctx.Err(); err != nil {
				return err
			}
			return fmt.Errorf("check timetable DOM: %w", err)
		}
		if ready {
			debug(cfg, "timetable DOM is ready")
			return nil
		}

		timer := time.NewTimer(pollInterval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
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
