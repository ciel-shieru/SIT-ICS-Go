package browser

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/ciel-shieru/sit-ics-go/internal/peoplesoft"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
)

type termPage interface {
	Context(ctx context.Context) termPage
	Element(selector string) (*rod.Element, error)
	Elements(selector string) (rod.Elements, error)
	WaitStable(timeout time.Duration) error
	Wait(fn *rod.EvalOptions) error
	HTML() (string, error)
	WaitNavigation(event proto.PageLifecycleEventName) func()
}

type pageAdapter struct {
	*rod.Page
}

func (a *pageAdapter) Context(ctx context.Context) termPage {
	return &pageAdapter{a.Page.Context(ctx)}
}

func (a *pageAdapter) HTML() (string, error) {
	return a.Page.HTML()
}

func (a *pageAdapter) WaitNavigation(event proto.PageLifecycleEventName) func() {
	return a.Page.WaitNavigation(event)
}

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

func fetchTimetable(ctx context.Context, page *rod.Page, cfg BrowserConfig) ([]peoplesoft.Entry, error) {
	if page == nil {
		return nil, fmt.Errorf("no active page: authenticate first")
	}
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("timetable fetch context already expired: %w", err)
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
		return nil, fmt.Errorf("navigate to timetable: %w", err)
	}

	navDone := make(chan struct{})
	go func() {
		page.WaitNavigation(proto.PageLifecycleEventNameNetworkAlmostIdle)()
		close(navDone)
	}()

	select {
	case <-fetchCtx.Done():
		return nil, fmt.Errorf("navigate to timetable: %w", fetchCtx.Err())
	case <-navDone:
	}

	stableDone := make(chan error)
	go func() {
		stableDone <- page.WaitStable(5000)
	}()

	select {
	case <-fetchCtx.Done():
		return nil, fmt.Errorf("wait for stable: %w", fetchCtx.Err())
	case err := <-stableDone:
		if err != nil {
			debug(cfg, "wait stable failed: %v", err)
		}
	}

	return handleMultiTermSelection(fetchCtx, &pageAdapter{page}, cfg)
}

func debug(cfg BrowserConfig, msg string, args ...any) {
	if cfg.Debug {
		log.Printf("browser: "+msg, args...)
	}
}

func handleMultiTermSelection(ctx context.Context, page termPage, cfg BrowserConfig) ([]peoplesoft.Entry, error) {
	debug(cfg, "checking for term selection screen")
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("term selection context already expired: %w", err)
	}
	if deadline, ok := ctx.Deadline(); ok {
		debug(cfg, "term selection context deadline: %s (remaining %s)", deadline.Format(time.RFC3339Nano), time.Until(deadline).Round(time.Millisecond))
	}

	var allEntries []peoplesoft.Entry

	if err := returnToTermSelection(ctx, page, cfg); err != nil {
		debug(cfg, "failed to navigate to term selection: %v — fetching from current page", err)
		return fetchFromCurrentPage(ctx, page, cfg)
	}

	var radioButtons []*rod.Element
	if err := Do(ctx, func() error {
		var err error
		radioButtons, err = page.Elements("input.PSRADIOBUTTON")
		return err
	}, cfg.MaxRetries, cfg.RetryInterval); err != nil {
		return nil, fmt.Errorf("query term selection radio buttons: %w", err)
	}

	if len(radioButtons) == 0 {
		debug(cfg, "no term selection radio buttons found, fetching from current page")
		return fetchFromCurrentPage(ctx, page, cfg)
	}

	debug(cfg, "found %d term selection radio button(s)", len(radioButtons))

	for i := 0; i < len(radioButtons); i++ {
		debug(cfg, "fetching term %d of %d", i+1, len(radioButtons))

		entries, err := navigateToTerm(ctx, i, page, cfg)
		if err != nil {
			debug(cfg, "failed to fetch term %d: %v — continuing to next term", i+1, err)
			continue
		}

		allEntries = append(allEntries, entries...)
		debug(cfg, "fetched %d entries from term %d", len(entries), i+1)

		if i < len(radioButtons)-1 {
			if err := returnToTermSelection(ctx, page, cfg); err != nil {
				debug(cfg, "failed to return to term selection after term %d: %v — stopping term iteration", i+1, err)
				break
			}

			if err := Do(ctx, func() error {
				_, err := page.Element("input.PSRADIOBUTTON")
				return err
			}, cfg.MaxRetries, cfg.RetryInterval); err != nil {
				debug(cfg, "term selection radio button not visible after returning: %v", err)
			}
		}
	}

	if len(allEntries) == 0 && len(radioButtons) > 0 {
		return nil, fmt.Errorf("no timetable entries fetched from any of %d term(s)", len(radioButtons))
	}

	debug(cfg, "multi-term fetch complete, total entries: %d", len(allEntries))
	return allEntries, nil
}

func navigateToTerm(ctx context.Context, index int, page termPage, cfg BrowserConfig) ([]peoplesoft.Entry, error) {
	radioID := fmt.Sprintf("SSR_DUMMY_RECV1\\$sels\\$%d\\$\\$0", index)
	cssSelector := fmt.Sprintf("#%s", radioID)

	debug(cfg, "clicking radio button: %s", cssSelector)
	if err := Do(ctx, func() error {
		el, err := page.Element(cssSelector)
		if err != nil {
			return fmt.Errorf("find radio button %s: %w", cssSelector, err)
		}
		return clickElement(el)
	}, cfg.MaxRetries, cfg.RetryInterval); err != nil {
		return nil, fmt.Errorf("click radio button %s: %w", cssSelector, err)
	}

	goBtnID := "DERIVED_SSS_SCT_SSR_PB_GO"
	debug(cfg, "clicking Go button: #%s", goBtnID)
	if err := Do(ctx, func() error {
		el, err := page.Element("#" + goBtnID)
		if err != nil {
			return fmt.Errorf("find go button: %w", err)
		}
		return clickElement(el)
	}, cfg.MaxRetries, cfg.RetryInterval); err != nil {
		return nil, fmt.Errorf("click go button: %w", err)
	}

	debug(cfg, "waiting for timetable data to load")
	termCtx, termCancel := context.WithTimeout(ctx, cfg.NavigationTimeout)
	defer termCancel()

	navigationPage := page.Context(termCtx)
	navDone := make(chan struct{})
	go func() {
		navigationPage.WaitNavigation(proto.PageLifecycleEventNameNetworkAlmostIdle)()
		close(navDone)
	}()

	select {
	case <-termCtx.Done():
		return nil, fmt.Errorf("navigate to term: %w", termCtx.Err())
	case <-navDone:
	}

	stableDone := make(chan error)
	go func() {
		stableDone <- navigationPage.WaitStable(5000)
	}()

	select {
	case <-termCtx.Done():
		return nil, fmt.Errorf("wait stable for term: %w", termCtx.Err())
	case err := <-stableDone:
		if err != nil {
			debug(cfg, "wait stable for term failed: %v", err)
		}
	}

	htmlStr, err := navigationPage.HTML()
	if err != nil {
		return nil, fmt.Errorf("get page HTML for term: %w", err)
	}

	debug(cfg, "timetable HTML extracted for term, length: %d", len(htmlStr))

	entries, err := peoplesoft.ParseTimetableHTML(htmlStr, 0, time.Local)
	if err != nil {
		return nil, fmt.Errorf("parse timetable HTML for term: %w", err)
	}

	return entries, nil
}

func returnToTermSelection(ctx context.Context, page termPage, cfg BrowserConfig) error {
	debug(cfg, "clicking change term button to return to term selection")
	if err := Do(ctx, func() error {
		el, err := page.Element("input[name='DERIVED_SSS_SCT_SSS_TERM_LINK']")
		if err != nil {
			return fmt.Errorf("find change term link: %w", err)
		}
		return clickElement(el)
	}, cfg.MaxRetries, cfg.RetryInterval); err != nil {
		return fmt.Errorf("click change term button: %w", err)
	}

	debug(cfg, "waiting for term selection screen to load after clicking change term")
	if err := Do(ctx, func() error {
		_, err := page.Element("input.PSRADIOBUTTON")
		return err
	}, cfg.MaxRetries, cfg.RetryInterval); err != nil {
		debug(cfg, "term selection radio button not found after change term click: %v", err)
	}

	stableDone := make(chan error)
	go func() {
		stableDone <- page.WaitStable(5000)
	}()

	select {
	case <-ctx.Done():
		return fmt.Errorf("wait stable for term selection return: %w", ctx.Err())
	case err := <-stableDone:
		if err != nil {
			debug(cfg, "wait stable for term selection return failed: %v", err)
		}
	}

	debug(cfg, "returned to term selection screen")
	return nil
}

func fetchFromCurrentPage(ctx context.Context, page termPage, cfg BrowserConfig) ([]peoplesoft.Entry, error) {
	debug(cfg, "fetching timetable from current page without term selection")

	pageCtx := page.Context(ctx)

	htmlStr, err := pageCtx.HTML()
	if err != nil {
		return nil, fmt.Errorf("get page HTML: %w", err)
	}

	debug(cfg, "timetable HTML extracted from current page, length: %d", len(htmlStr))

	entries, err := peoplesoft.ParseTimetableHTML(htmlStr, 0, time.Local)
	if err != nil {
		return nil, fmt.Errorf("parse timetable HTML from current page: %w", err)
	}

	return entries, nil
}

func clickElement(el *rod.Element) error {
	defer func() { recover() }()
	return el.Click(proto.InputMouseButtonLeft, 1)
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
