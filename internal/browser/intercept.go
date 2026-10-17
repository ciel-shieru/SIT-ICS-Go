package browser

import (
	"fmt"
	"log"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
)

// enableRequestInterception installs a per-page HijackRouter that aborts
// every request whose URL does not pass the whitelist. The all-URLs pattern
// "" with empty resource type intercepts all request types on this page's
// CDP session. Returns the router so the caller can Stop() it on cleanup.
// Any CDP failure here is returned to the caller (fail closed).
func enableRequestInterception(page *rod.Page, wl *HostWhitelist, cfg BrowserConfig) (*rod.HijackRouter, error) {
	router := page.HijackRequests()
	if err := router.Add("", proto.NetworkResourceType(""), makeHijackHandler(wl, cfg)); err != nil {
		return nil, fmt.Errorf("enable fetch interception: %w", err)
	}
	router.Run()
	debug(cfg, "request interception enabled (%d whitelist entries)", len(wl.suffixes))
	return router, nil
}

// makeHijackHandler maps whitelist decisions to CDP fetch actions.
func makeHijackHandler(wl *HostWhitelist, cfg BrowserConfig) func(*rod.Hijack) {
	return func(h *rod.Hijack) {
		u := h.Request.URL()
		d := DecideURL(wl, u)
		raw := ""
		if u != nil {
			raw = u.String()
		}
		if d == DecisionBlock {
			// Security-relevant: always logged, not debug-gated.
			log.Printf("browser: SECURITY: blocked request to non-whitelisted host %q (type %s)", raw, h.Request.Type())
			h.Response.Fail(proto.NetworkErrorReasonBlockedByClient)
			return
		}
		debug(cfg, "request allowed: %s (type %s)", raw, h.Request.Type())
		// Explicit continue is REQUIRED: without it the router fulfills the
		// request with an empty 200 response.
		h.ContinueRequest(&proto.FetchContinueRequest{})
	}
}
