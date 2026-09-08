package browser

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
)

// DecodeJSON navigates to a URL and parses the JSON response into v.
func DecodeJSON(page *rod.Page, url string, v interface{}, cfg BrowserConfig, ctx context.Context) error {
	fetchCtx, cancel := context.WithTimeout(ctx, cfg.NavigationTimeout)
	defer cancel()
	page = page.Context(fetchCtx)

	debug(cfg, "fetching %s", url)
	if err := page.Navigate(url); err != nil {
		return fmt.Errorf("navigate to %s: %w", url, err)
	}

	page.WaitNavigation(proto.PageLifecycleEventNameNetworkAlmostIdle)()
	if err := page.WaitStable(3 * time.Second); err != nil {
		debug(cfg, "wait stable failed for %s: %v", url, err)
	}

	text, err := extractPageText(page)
	if err != nil {
		return fmt.Errorf("extract response from %s: %w", url, err)
	}

	if text == "" {
		return nil
	}

	if err := json.Unmarshal([]byte(text), v); err != nil {
		return fmt.Errorf("parse JSON from %s: %w (status=%d, body=%q)", url, err, 200, text)
	}

	return nil
}

// DecodeJSONField navigates to a URL and extracts a specific JSON field as a string.
func DecodeJSONField(page *rod.Page, url, field string, cfg BrowserConfig, ctx context.Context) (string, error) {
	fetchCtx, cancel := context.WithTimeout(ctx, cfg.NavigationTimeout)
	defer cancel()
	page = page.Context(fetchCtx)

	debug(cfg, "fetching %s", url)
	if err := page.Navigate(url); err != nil {
		return "", fmt.Errorf("navigate to %s: %w", url, err)
	}

	page.WaitNavigation(proto.PageLifecycleEventNameNetworkAlmostIdle)()
	if err := page.WaitStable(3 * time.Second); err != nil {
		debug(cfg, "wait stable failed for %s: %v", url, err)
	}

	text, err := extractPageText(page)
	if err != nil {
		return "", fmt.Errorf("extract response from %s: %w", url, err)
	}

	if text == "" {
		return "", fmt.Errorf("empty response from %s", url)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal([]byte(text), &raw); err != nil {
		return "", fmt.Errorf("parse JSON from %s: %w", url, err)
	}

	val, ok := raw[field]
	if !ok {
		return "", fmt.Errorf("field %q not found in response from %s", field, url)
	}
	return fmt.Sprintf("%v", val), nil
}

// extractPageText extracts the text content of the page (works for JSON API responses).
func extractPageText(page *rod.Page) (string, error) {
	var result string
	safeRod(func() {
		val := page.MustEval("() => document.body ? document.body.innerText : document.documentElement.innerText")
		result = val.String()
	})
	return strings.TrimSpace(result), nil
}
