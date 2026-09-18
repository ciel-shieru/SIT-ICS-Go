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

	// Keep the timetable operation alive for the caller-provided timetable
	// deadline. The individual initial navigation below gets its own shorter
	// NavigationTimeout; PeopleSoft's post-term-selection rendering can take
	// substantially longer than that.
	fetchCtx, fetchCancel := context.WithCancel(ctx)
	defer fetchCancel()
	page = page.Context(fetchCtx)
	timetableURL := "https://in4sit.singaporetech.edu.sg/psc/CSSISSTD/EMPLOYEE/SA/c/SA_LEARNER_SERVICES.SSR_SSENRL_LIST.GBL"
	debug(cfg, "navigating to timetable endpoint: %s", timetableURL)

	initialNavCtx, initialNavCancel := context.WithTimeout(fetchCtx, cfg.NavigationTimeout)
	initialNavPage := page.Context(initialNavCtx)
	if deadline, ok := initialNavCtx.Deadline(); ok {
		debug(cfg, "initial timetable navigation deadline: %s (remaining %s)", deadline.Format(time.RFC3339Nano), time.Until(deadline).Round(time.Millisecond))
	}
	if err := initialNavPage.Navigate(timetableURL); err != nil {
		initialNavCancel()
		return "", fmt.Errorf("navigate to timetable: %w", err)
	}

	// Wait for navigation with context awareness
	navDone := make(chan struct{})
	go func() {
		initialNavPage.WaitNavigation(proto.PageLifecycleEventNameNetworkAlmostIdle)()
		close(navDone)
	}()
	select {
	case <-initialNavCtx.Done():
		initialNavCancel()
		return "", fmt.Errorf("navigate to timetable: %w", initialNavCtx.Err())
	case <-navDone:
	}

	// Wait for stable with context awareness
	stableDone := make(chan error)
	go func() {
		initialNavPage.WaitStable(5000)
	}()

	select {
	case <-initialNavCtx.Done():
		initialNavCancel()
		return "", fmt.Errorf("wait for stable: %w", initialNavCtx.Err())
	case err := <-stableDone:
		if err != nil {
			debug(cfg, "wait stable failed: %v", err)
		}
	}
	initialNavCancel()

	if err := handleTermSelection(fetchCtx, page, cfg); err != nil {
		return "", fmt.Errorf("handle term selection: %w", err)
	}

	if err := waitForTimetableDOM(fetchCtx, page, cfg); err != nil {
		return "", fmt.Errorf("wait for timetable DOM: %w", err)
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

func waitForTimetableDOM(ctx context.Context, page *rod.Page, cfg BrowserConfig) error {
	debug(cfg, "waiting for timetable table to render")
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()

	for {
		var ready bool
		obj, err := page.Eval(`() => Array.from(document.querySelectorAll('table')).some((table) => {
			const firstRow = table.querySelector('tr');
			if (!firstRow) return false;
			const cells = firstRow.querySelectorAll('td, th');
			return cells.length >= 7 && (cells[0].textContent || '').includes('Class Nbr');
		})`)
		if err == nil {
			err = obj.Value.Unmarshal(&ready)
		}
		if err == nil && ready {
			debug(cfg, "timetable table rendered")
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
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

	// Do not wait for PeopleSoft network idle here. Continue.Click has already
	// performed the requested browser interaction successfully. The resulting
	// timetable page is awaited by fetchTimetable using the actual DOM condition
	// consumed by the parser.
	debug(cfg, "term selection submitted successfully")
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
