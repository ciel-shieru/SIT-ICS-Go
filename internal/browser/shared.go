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

func getPageURL(ctx context.Context, page *rod.Page, cfg BrowserConfig) string {
	var url string
	_ = Do(ctx, func() error {
		result, err := page.Eval("() => window.location.href")
		if err != nil {
			return err
		}
		url = result.Value.Str()
		return nil
	}, cfg.MaxRetries, cfg.RetryInterval)
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

	timetableURL := "https://in4sit.singaporetech.edu.sg/psc/CSSISSTD/EMPLOYEE/SA/c/SA_LEARNER_SERVICES.SSR_SSENRL_LIST.GBL"
	debug(cfg, "navigating to timetable endpoint: %s", timetableURL)
	if err := Do(fetchCtx, func() error {
		return page.Navigate(timetableURL)
	}, cfg.MaxRetries, cfg.RetryInterval); err != nil {
		return "", fmt.Errorf("navigate to timetable: %w", err)
	}

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
	termCtx, termCancel := context.WithTimeout(ctx, 5*time.Second)
	defer termCancel()
	termPage := page.Context(termCtx)

	if err := Do(termCtx, func() error {
		_, err := termPage.Element("input.PSRADIOBUTTON")
		return err
	}, cfg.MaxRetries, cfg.RetryInterval); err != nil {
		return fmt.Errorf("term selection radio button not found: %w", err)
	}

	var radioButtons []*rod.Element
	if err := Do(termCtx, func() error {
		var err error
		radioButtons, err = termPage.Elements("input.PSRADIOBUTTON")
		return err
	}, cfg.MaxRetries, cfg.RetryInterval); err != nil {
		return fmt.Errorf("query term selection radio buttons: %w", err)
	}

	if len(radioButtons) == 0 {
		debug(cfg, "no term selection radio buttons found, skipping")
		return nil
	}

	debug(cfg, "found %d term selection radio button(s)", len(radioButtons))

	lastRadio := radioButtons[len(radioButtons)-1]
	debug(cfg, "selecting last radio button (index %d of %d)", len(radioButtons)-1, len(radioButtons))
	if err := Do(termCtx, func() error {
		return lastRadio.Click(proto.InputMouseButtonLeft, 1)
	}, cfg.MaxRetries, cfg.RetryInterval); err != nil {
		return fmt.Errorf("click term selection radio button: %w", err)
	}

	var continueEl *rod.Element
	if err := Do(ctx, func() error {
		var err error
		continueEl, err = page.Element("input[name='DERIVED_SSS_SCT_SSR_PB_GO']")
		return err
	}, cfg.MaxRetries, cfg.RetryInterval); err != nil {
		return fmt.Errorf("find continue button: %w", err)
	}

	debug(cfg, "clicking continue button")
	if err := Do(ctx, func() error {
		return continueEl.Click(proto.InputMouseButtonLeft, 1)
	}, cfg.MaxRetries, cfg.RetryInterval); err != nil {
		return fmt.Errorf("click continue button: %w", err)
	}

	debug(cfg, "waiting for timetable data to load after term selection")

	if err := Do(ctx, func() error {
		return page.Wait(rod.Eval("() => document.querySelectorAll('table.PSGROUPBOXWBO').length > 0"))
	}, cfg.MaxRetries, cfg.RetryInterval); err != nil {
		return fmt.Errorf("wait for timetable data: %w", err)
	}

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

func extractPageText(ctx context.Context, page *rod.Page, maxRetries int, interval time.Duration) (result string, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			result = ""
			err = fmt.Errorf("browser evaluation failed: %v", recovered)
		}
	}()

	var value string
	if err := Do(ctx, func() error {
		r, err := page.Eval(`() => document.body ? document.body.innerText : ""`)
		if err != nil {
			return err
		}
		value = r.Value.Str()
		return nil
	}, maxRetries, interval); err != nil {
		return "", fmt.Errorf("eval: %w", err)
	}
	return strings.TrimSpace(value), nil
}

func safeRod(fn func()) {
	defer func() { recover() }()
	fn()
}
