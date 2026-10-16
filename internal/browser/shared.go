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
	Navigate(url string) error
	CurrentURL() (string, error)
	Click(selector string, index int) error
	Elements(selector string) (rod.Elements, error)
	WaitStable(timeout time.Duration) error
	HTML() (string, error)
}

type pageAdapter struct {
	*rod.Page
}

func (a *pageAdapter) Context(ctx context.Context) termPage {
	return &pageAdapter{a.Page.Context(ctx)}
}

func (a *pageAdapter) Navigate(url string) error {
	return a.Page.Navigate(url)
}

func (a *pageAdapter) CurrentURL() (string, error) {
	result, err := a.Page.Eval("() => window.location.href")
	if err != nil {
		return "", err
	}
	return result.Value.Str(), nil
}

func (a *pageAdapter) Click(selector string, index int) error {
	elements, err := a.Page.Elements(selector)
	if err != nil {
		return err
	}
	if index < 0 || index >= len(elements) {
		return fmt.Errorf("element %q at index %d not found (found %d)", selector, index, len(elements))
	}
	return clickElement(elements[index])
}

func (a *pageAdapter) HTML() (string, error) {
	return a.Page.HTML()
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
	return fetchTimetablePage(ctx, &pageAdapter{page}, cfg)
}

// fetchTimetablePage keeps the initial endpoint navigation timeout separate
// from term selection and the subsequent per-term navigations.
func fetchTimetablePage(ctx context.Context, page termPage, cfg BrowserConfig) ([]peoplesoft.Entry, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("timetable fetch context already expired: %w", err)
	}

	// Keep page operations bound to the overall timetable context. The initial
	// navigation gets a nested timeout without shortening this page's lifetime.
	page = page.Context(ctx)
	if err := navigateToTimetable(ctx, page, cfg); err != nil {
		return nil, err
	}

	return handleMultiTermSelection(ctx, page, cfg)
}

const timetableEndpointURL = "https://in4sit.singaporetech.edu.sg/psc/CSSISSTD/EMPLOYEE/SA/c/SA_LEARNER_SERVICES.SSR_SSENRL_LIST.GBL"

// urlWithoutQuery removes the query string from a URL for endpoint comparison.
func urlWithoutQuery(rawURL string) string {
	if query := strings.IndexByte(rawURL, '?'); query >= 0 {
		return rawURL[:query]
	}
	return rawURL
}

// navigateToTimetable retries the complete navigation cycle. Each cycle gets
// its own navigation timeout, then waits for stability and verifies that the
// browser actually landed on the timetable endpoint before allowing the fetch
// to continue.
func navigateToTimetable(ctx context.Context, page termPage, cfg BrowserConfig) error {
	return Do(ctx, func() error {
		navigationCtx, navigationCancel := context.WithTimeout(ctx, cfg.NavigationTimeout)
		navigationPage := page.Context(navigationCtx)
		if deadline, ok := navigationCtx.Deadline(); ok {
			debug(cfg, "timetable navigation deadline: %s (remaining %s)", deadline.Format(time.RFC3339Nano), time.Until(deadline).Round(time.Millisecond))
		}

		debug(cfg, "navigating to timetable endpoint: %s", timetableEndpointURL)
		err := Do(navigationCtx, func() error {
			return navigationPage.Navigate(timetableEndpointURL)
		}, 0, cfg.RetryInterval)
		navigationCancel()
		if err != nil {
			return fmt.Errorf("navigate to timetable: %w", err)
		}

		if err := page.WaitStable(5000); err != nil {
			if ctx.Err() != nil {
				return fmt.Errorf("wait for timetable page stability: %w", ctx.Err())
			}
			return fmt.Errorf("wait for timetable page stability: %w", err)
		}

		currentURL, err := page.CurrentURL()
		if err != nil {
			return fmt.Errorf("check URL after timetable navigation: %w", err)
		}
		if urlWithoutQuery(currentURL) == timetableEndpointURL {
			debug(cfg, "timetable endpoint URL verified: %s", currentURL)
			return nil
		}
		debug(cfg, "URL after timetable navigation does not match endpoint (got %q, want %q); waiting one more second", currentURL, timetableEndpointURL)

		if err := page.WaitStable(time.Second); err != nil {
			if ctx.Err() != nil {
				return fmt.Errorf("wait for timetable page stability after URL mismatch: %w", ctx.Err())
			}
			return fmt.Errorf("wait for timetable page stability after URL mismatch: %w", err)
		}

		currentURL, err = page.CurrentURL()
		if err != nil {
			return fmt.Errorf("recheck URL after timetable navigation: %w", err)
		}
		if urlWithoutQuery(currentURL) != timetableEndpointURL {
			return fmt.Errorf("timetable endpoint URL mismatch after additional stability wait: got %q, want %q", urlWithoutQuery(currentURL), timetableEndpointURL)
		}

		debug(cfg, "timetable endpoint URL verified after additional stability wait: %s", currentURL)
		return nil
	}, cfg.MaxRetries, cfg.RetryInterval)
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
		entries, fallbackErr := fetchFromCurrentPage(ctx, page, cfg)
		if fallbackErr != nil {
			return nil, fmt.Errorf("term selection failed (%v); current-page fallback failed: %w", err, fallbackErr)
		}
		return entries, nil
	}

	radioButtons, err := queryTermRadioButtons(ctx, page, cfg)
	if err != nil {
		return nil, fmt.Errorf("query term selection radio buttons: %w", err)
	}
	if len(radioButtons) == 0 {
		debug(cfg, "no term selection radio buttons found, fetching from current page")
		return fetchFromCurrentPage(ctx, page, cfg)
	}

	initialTermCount := len(radioButtons)
	var firstTermErr error

	for i := 0; i < len(radioButtons); i++ {
		debug(cfg, "fetching term %d of %d", i+1, len(radioButtons))

		entries, termErr := navigateToTerm(ctx, i, page, cfg)
		if termErr != nil {
			debug(cfg, "failed to fetch term %d: %v — attempting to restore term selection", i+1, termErr)
			if ctx.Err() != nil {
				return nil, fmt.Errorf("timetable fetch interrupted after term %d: %w", i+1, ctx.Err())
			}
			if firstTermErr == nil {
				firstTermErr = fmt.Errorf("fetch term %d: %w", i+1, termErr)
			}
			if i == len(radioButtons)-1 {
				continue
			}

			// A failed term attempt can leave the browser on its timetable page.
			// Restore the selector to finish diagnosing this attempt, but never
			// report success if any term failed; runFetch must retry the full run.
			restoredButtons, restoreErr := restoreTermSelection(ctx, page, cfg)
			if restoreErr != nil {
				debug(cfg, "failed to restore term selection after term %d: %v — stopping term iteration", i+1, restoreErr)
				break
			}
			radioButtons = restoredButtons
			if i+1 >= len(radioButtons) {
				debug(cfg, "no further term radio button available after term %d", i+1)
				break
			}
			continue
		}

		allEntries = append(allEntries, entries...)
		debug(cfg, "fetched %d entries from term %d", len(entries), i+1)

		if i < len(radioButtons)-1 {
			if err := returnToTermSelection(ctx, page, cfg); err != nil {
				if firstTermErr == nil {
					firstTermErr = fmt.Errorf("return to term selection after term %d: %w", i+1, err)
				}
				debug(cfg, "failed to return to term selection after term %d: %v — stopping term iteration", i+1, err)
				break
			}
			nextRadioButtons, err := queryTermRadioButtons(ctx, page, cfg)
			if err != nil {
				if firstTermErr == nil {
					firstTermErr = fmt.Errorf("query term selection after term %d: %w", i+1, err)
				}
				debug(cfg, "failed to query term selection after term %d: %v — stopping term iteration", i+1, err)
				break
			}
			if i+1 >= len(nextRadioButtons) {
				if firstTermErr == nil {
					firstTermErr = fmt.Errorf("term selection no longer contains term %d after fetching term %d", i+2, i+1)
				}
				debug(cfg, "no further term radio button available after term %d", i+1)
				break
			}
			radioButtons = nextRadioButtons
		}
	}

	if firstTermErr != nil {
		return nil, fmt.Errorf("incomplete PeopleSoft timetable fetch: %w", firstTermErr)
	}
	if len(allEntries) == 0 {
		return nil, fmt.Errorf("no timetable entries fetched from any of %d term(s)", initialTermCount)
	}

	debug(cfg, "multi-term fetch complete, total entries: %d", len(allEntries))
	return allEntries, nil
}

// queryTermRadioButtons is a one-shot DOM query. Element polls until a match is
// found, which can block until the overall fetch deadline if the page is not on
// the term selection screen.
func queryTermRadioButtons(ctx context.Context, page termPage, cfg BrowserConfig) (rod.Elements, error) {
	var radioButtons rod.Elements
	err := Do(ctx, func() error {
		var err error
		radioButtons, err = page.Elements("input.PSRADIOBUTTON")
		return err
	}, cfg.MaxRetries, cfg.RetryInterval)
	return radioButtons, err
}

// restoreTermSelection recovers after a failed term navigation. Hidden/stale
// radio buttons can remain in the timetable DOM, so the selector is considered
// active only when its Go button is present. Otherwise explicitly click Change
// Term before selecting the next term.
func restoreTermSelection(ctx context.Context, page termPage, cfg BrowserConfig) (rod.Elements, error) {
	goButtons, err := queryTermGoButtons(ctx, page, cfg)
	if err != nil {
		return nil, fmt.Errorf("query term selection Go button: %w", err)
	}
	if len(goButtons) == 0 {
		if err := returnToTermSelection(ctx, page, cfg); err != nil {
			return nil, err
		}
	}

	radioButtons, err := queryTermRadioButtons(ctx, page, cfg)
	if err != nil {
		return nil, err
	}
	if len(radioButtons) == 0 {
		return nil, fmt.Errorf("term selection screen has no radio buttons after recovery")
	}
	return radioButtons, nil
}

func queryTermGoButtons(ctx context.Context, page termPage, cfg BrowserConfig) (rod.Elements, error) {
	var goButtons rod.Elements
	err := Do(ctx, func() error {
		var err error
		goButtons, err = page.Elements("input[name='DERIVED_SSS_SCT_SSR_PB_GO']")
		return err
	}, cfg.MaxRetries, cfg.RetryInterval)
	return goButtons, err
}

func navigateToTerm(ctx context.Context, index int, page termPage, cfg BrowserConfig) ([]peoplesoft.Entry, error) {
	radioID := fmt.Sprintf("SSR_DUMMY_RECV1\\$sels\\$%d\\$\\$0", index)
	cssSelector := fmt.Sprintf("#%s", radioID)

	debug(cfg, "clicking radio button: %s", cssSelector)
	if err := Do(ctx, func() error {
		return page.Click("input.PSRADIOBUTTON", index)
	}, cfg.MaxRetries, cfg.RetryInterval); err != nil {
		return nil, fmt.Errorf("click radio button %s: %w", cssSelector, err)
	}

	goBtnID := "DERIVED_SSS_SCT_SSR_PB_GO"
	debug(cfg, "clicking Go button: #%s", goBtnID)
	if err := Do(ctx, func() error {
		return page.Click("input[name='DERIVED_SSS_SCT_SSR_PB_GO']", 0)
	}, cfg.MaxRetries, cfg.RetryInterval); err != nil {
		return nil, fmt.Errorf("click go button: %w", err)
	}

	debug(cfg, "waiting for timetable data to load")
	if err := page.WaitStable(2 * time.Second); err != nil {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("wait for timetable stability: %w", ctx.Err())
		}
		return nil, fmt.Errorf("wait for timetable stability: %w", err)
	}

	htmlStr, err := page.HTML()
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
		return page.Click("input[name='DERIVED_SSS_SCT_SSS_TERM_LINK']", 0)
	}, cfg.MaxRetries, cfg.RetryInterval); err != nil {
		return fmt.Errorf("click change term button: %w", err)
	}

	debug(cfg, "waiting for term selection screen to load after clicking change term")
	if err := page.WaitStable(5000); err != nil {
		if ctx.Err() != nil {
			return fmt.Errorf("wait stable for term selection return: %w", ctx.Err())
		}
		return fmt.Errorf("wait stable for term selection return: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("return to term selection: %w", err)
	}

	goButtons, err := queryTermGoButtons(ctx, page, cfg)
	if err != nil {
		return fmt.Errorf("query Go button after returning to term selection: %w", err)
	}
	if len(goButtons) == 0 {
		return fmt.Errorf("term selection Go button not present after returning")
	}
	radioButtons, err := queryTermRadioButtons(ctx, page, cfg)
	if err != nil {
		return fmt.Errorf("query term selection after returning: %w", err)
	}
	if len(radioButtons) == 0 {
		return fmt.Errorf("term selection radio buttons not present after returning")
	}

	debug(cfg, "returned to term selection screen")
	return nil
}

func fetchFromCurrentPage(ctx context.Context, page termPage, cfg BrowserConfig) ([]peoplesoft.Entry, error) {
	debug(cfg, "fetching timetable from current page without term selection")
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("fetch timetable from current page: %w", err)
	}

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
	debug(cfg, "parsed %d timetable entries from current page", len(entries))
	if len(entries) == 0 {
		return nil, fmt.Errorf("no timetable entries parsed from current page")
	}

	return entries, nil
}

func clickElement(el *rod.Element) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("click element panicked: %v", recovered)
		}
	}()
	if el == nil {
		return fmt.Errorf("cannot click a nil element")
	}
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
