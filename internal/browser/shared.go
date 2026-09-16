package browser

import (
	"context"
	"fmt"
	"log"
	"strings"

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

	fetchCtx, fetchCancel := context.WithTimeout(ctx, cfg.NavigationTimeout)
	defer fetchCancel()

	page = page.Context(fetchCtx)

	timetableURL := "https://in4sit.singaporetech.edu.sg/psc/CSSISSTD/EMPLOYEE/SA/c/SA_LEARNER_SERVICES.SSR_SSENRL_LIST.GBL"
	debug(cfg, "navigating to timetable endpoint: %s", timetableURL)
	if err := page.Navigate(timetableURL); err != nil {
		return "", fmt.Errorf("navigate to timetable: %w", err)
	}

	page.WaitNavigation(proto.PageLifecycleEventNameNetworkAlmostIdle)()
	if err := page.WaitStable(5000); err != nil {
		debug(cfg, "wait stable failed: %v", err)
	}

	if err := handleTermSelection(ctx, page, cfg); err != nil {
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

	// Check if term selection table exists by evaluating in JS
	var radioCount int
	var evalErr error
	func() {
		defer func() {
			if r := recover(); r != nil {
				evalErr = fmt.Errorf("panic during term selection check: %v", r)
			}
		}()
		_, evalErr = page.Eval(`() => {
			const radios = document.querySelectorAll('input[name^="SSR_DUMMY_RECV1$sels$"]');
			return radios.length;
		}`, &radioCount)
	}()
	if evalErr != nil {
		return fmt.Errorf("check term selection: %w", evalErr)
	}

	if radioCount == 0 {
		debug(cfg, "no term selection table detected, skipping")
		return nil
	}

	debug(cfg, "term selection detected with %d option(s)", radioCount)

	// Select the LAST radio button via JavaScript
	_, err := page.Eval(`() => {
		const radios = document.querySelectorAll('input[name^="SSR_DUMMY_RECV1$sels$"]');
		if (radios.length > 0) {
			radios[radios.length - 1].click();
		}
	}`)
	if err != nil {
		return fmt.Errorf("select term radio button: %w", err)
	}
	debug(cfg, "selected last radio button via JS")

	// Click Continue button via JavaScript
	_, err = page.Eval(`() => {
		const btn = document.querySelector('input[name="DERIVED_SSS_SCT_SSR_PB_GO"]');
		if (btn) btn.click();
	}`)
	if err != nil {
		return fmt.Errorf("click continue button: %w", err)
	}
	debug(cfg, "clicked continue button via JS")

	// Wait for navigation and stability after Continue
	page.WaitNavigation(proto.PageLifecycleEventNameNetworkAlmostIdle)()
	if err := page.WaitStable(5000); err != nil {
		debug(cfg, "wait stable after term selection: %v", err)
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
