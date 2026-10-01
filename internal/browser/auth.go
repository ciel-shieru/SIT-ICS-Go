package browser

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ciel-shieru/sit-ics-go/internal/totp"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
	"golang.org/x/net/html"
)

// pageHTML is a minimal interface for extracting page HTML, enabling testability.
type pageHTML interface {
	HTML() (string, error)
}

func AuthenticateADFS(ctx context.Context, incognito *rod.Browser, req AuthRequest, waitNavigation bool, cfg BrowserConfig, isAllowedOrigin func(string) bool) (*rod.Page, error) {
	initialURL := req.URL
	debug(cfg, "navigating to %s", initialURL)

	var page *rod.Page
	if err := Do(ctx, func() error {
		p, err := incognito.Page(proto.TargetCreateTarget{URL: initialURL})
		if err != nil {
			return err
		}
		page = p
		return nil
	}, cfg.MaxRetries, cfg.RetryInterval); err != nil {
		return nil, fmt.Errorf("create page: %w", err)
	}
	page = page.Context(ctx)

	if err := Do(ctx, func() error {
		return page.WaitStable(3000)
	}, cfg.MaxRetries, cfg.RetryInterval); err != nil {
		debug(cfg, "wait stable failed: %v", err)
	}

	var info *proto.TargetTargetInfo
	if err := Do(ctx, func() error {
		i, err := page.Info()
		if err != nil {
			return err
		}
		info = i
		return nil
	}, cfg.MaxRetries, cfg.RetryInterval); err != nil {
		return nil, fmt.Errorf("get page info: %w", err)
	}
	finalURL := info.URL
	debug(cfg, "navigated to %s", finalURL)
	if !isAllowedOrigin(finalURL) {
		return nil, fmt.Errorf("%w: redirect to disallowed origin %s (started from %s)", ErrAuthentication, finalURL, initialURL)
	}

	if waitNavigation {
		page.WaitNavigation(proto.PageLifecycleEventNameNetworkAlmostIdle)()
	}

	debug(cfg, "finding username field")
	var el *rod.Element
	if err := Do(ctx, func() error {
		var err error
		el, err = page.Element("#userNameInput")
		return err
	}, cfg.MaxRetries, cfg.RetryInterval); err != nil {
		return nil, fmt.Errorf("%w: credential form not rendered: %v", ErrAuthentication, err)
	}

	visible, err := el.Visible()
	if err != nil {
		return nil, fmt.Errorf("%w: credential form not visible: %v", ErrAuthentication, err)
	}
	if !visible {
		return nil, fmt.Errorf("%w: credential form not rendered: element not visible", ErrAuthentication)
	}

	debug(cfg, "filling username")
	if err := Do(ctx, func() error {
		return el.Input(req.Username)
	}, cfg.MaxRetries, cfg.RetryInterval); err != nil {
		return nil, fmt.Errorf("%w: failed to fill username: %v", ErrCredentialExtraction, err)
	}

	debug(cfg, "finding password field")
	var passEl *rod.Element
	if err := Do(ctx, func() error {
		var err error
		passEl, err = page.Element("#passwordInput")
		return err
	}, cfg.MaxRetries, cfg.RetryInterval); err != nil {
		return nil, fmt.Errorf("%w: password field not found: %v", ErrAuthentication, err)
	}
	debug(cfg, "filling password")
	if err := Do(ctx, func() error {
		return passEl.Input(req.Password)
	}, cfg.MaxRetries, cfg.RetryInterval); err != nil {
		return nil, fmt.Errorf("%w: failed to fill password: %v", ErrCredentialExtraction, err)
	}

	debug(cfg, "finding submit button")
	var submitEl *rod.Element
	if err := Do(ctx, func() error {
		var err error
		submitEl, err = page.Element("#submitButton")
		return err
	}, cfg.MaxRetries, cfg.RetryInterval); err != nil {
		return nil, fmt.Errorf("%w: submit button not found: %v", ErrAuthentication, err)
	}
	debug(cfg, "clicking submit button")
	if err := Do(ctx, func() error {
		return submitEl.Click(proto.InputMouseButtonLeft, 1)
	}, cfg.MaxRetries, cfg.RetryInterval); err != nil {
		return nil, fmt.Errorf("%w: failed to submit credentials: %v", ErrAuthentication, err)
	}

	if waitNavigation {
		page.WaitNavigation(proto.PageLifecycleEventNameNetworkAlmostIdle)()
	}
	if err := Do(ctx, func() error {
		return page.WaitStable(5000)
	}, cfg.MaxRetries, cfg.RetryInterval); err != nil {
		debug(cfg, "wait stable after submit failed: %v", err)
	}

	debug(cfg, "checking for pre-MFA credential error")
	isCredErr, err := IsADFSCredentialError(page)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to check for credential error: %v", ErrAuthentication, err)
	}
	if isCredErr {
		return nil, fmt.Errorf("%w: incorrect user ID or password", ErrAuthentication)
	}

	debug(cfg, "checking for MFA field")
	var mfaEl *rod.Element
	mfaVisible := false
	if err := Do(ctx, func() error {
		var err error
		mfaEl, err = page.Element("#verificationCodeInput")
		if err != nil {
			return nil
		}
		mfaVisible = true
		return nil
	}, cfg.MaxRetries, cfg.RetryInterval); err != nil {
		debug(cfg, "MFA element check error (non-fatal): %v", err)
	}

	if mfaVisible {
		debug(cfg, "MFA detected, generating TOTP code")
		totpCode, err := totp.GenerateAtOffset(req.TOTPSecret, time.Now().UTC(), 1)
		if err != nil {
			return nil, fmt.Errorf("%w: failed to generate TOTP: %v", ErrAuthentication, err)
		}
		debug(cfg, "filling MFA code")
		if err := Do(ctx, func() error {
			return mfaEl.Input(totpCode)
		}, cfg.MaxRetries, cfg.RetryInterval); err != nil {
			return nil, fmt.Errorf("%w: failed to fill MFA code: %v", ErrCredentialExtraction, err)
		}
		debug(cfg, "finding sign-in button")
		var signInEl *rod.Element
		if err := Do(ctx, func() error {
			var err error
			signInEl, err = page.Element("#signInButton")
			return err
		}, cfg.MaxRetries, cfg.RetryInterval); err != nil {
			return nil, fmt.Errorf("%w: sign-in button not found: %v", ErrAuthentication, err)
		}
		debug(cfg, "clicking sign-in button")
		if err := Do(ctx, func() error {
			return signInEl.Click(proto.InputMouseButtonLeft, 1)
		}, cfg.MaxRetries, cfg.RetryInterval); err != nil {
			return nil, fmt.Errorf("%w: failed to submit MFA code: %v", ErrAuthentication, err)
		}
		debug(cfg, "waiting for SAML redirect after MFA")
		page.WaitNavigation(proto.PageLifecycleEventNameNetworkAlmostIdle)()
		if err := Do(ctx, func() error {
			return page.WaitStable(5000)
		}, cfg.MaxRetries, cfg.RetryInterval); err != nil {
			debug(cfg, "wait stable after MFA redirect failed: %v", err)
		}

		debug(cfg, "checking for MFA error message")
		errorMsg, err := ExtractADFSLoginError(page)
		if err != nil {
			return nil, fmt.Errorf("%w: failed to check for error message: %v", ErrAuthentication, err)
		}
		if errorMsg != "" {
			return nil, fmt.Errorf("ADFS auth error: %s", errorMsg)
		}
	} else {
		debug(cfg, "no MFA field detected, waiting for redirect")
		page.WaitNavigation(proto.PageLifecycleEventNameNetworkAlmostIdle)()
		if err := Do(ctx, func() error {
			return page.WaitStable(5000)
		}, cfg.MaxRetries, cfg.RetryInterval); err != nil {
			debug(cfg, "wait stable after submit redirect failed: %v", err)
		}
	}

	debug(cfg, "waiting for ADFS redirect to complete")
	page.WaitNavigation(proto.PageLifecycleEventNameNetworkAlmostIdle)()
	if err := Do(ctx, func() error {
		return page.WaitStable(5000)
	}, cfg.MaxRetries, cfg.RetryInterval); err != nil {
		debug(cfg, "wait stable after redirect failed: %v", err)
	}

	finalURL = getPageURL(ctx, page, cfg)
	debug(cfg, "auth complete, final URL: %s", finalURL)

	return page.Context(context.Background()), nil
}

func ExtractADFSLoginError(p pageHTML) (string, error) {
	htmlStr, err := p.HTML()
	if err != nil {
		return "", fmt.Errorf("get page HTML: %w", err)
	}

	doc, err := html.Parse(strings.NewReader(htmlStr))
	if err != nil {
		return "", fmt.Errorf("parse HTML: %w", err)
	}

	var findErrorText func(*html.Node) string
	findErrorText = func(n *html.Node) string {
		if n.Type == html.ElementNode {
			for _, a := range n.Attr {
				if a.Key == "id" && a.Val == "errorText" {
					return strings.TrimSpace(nodeText(n))
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if result := findErrorText(c); result != "" {
				return result
			}
		}
		return ""
	}

	return findErrorText(doc), nil
}

// nodeText extracts all text content from a node and its children.
func nodeText(n *html.Node) string {
	var text string
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.TextNode {
			text += node.Data
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return text
}

// hasADFSCredentialError checks if the HTML contains a credential error message.
func hasADFSCredentialError(htmlStr string) bool {
	doc, err := html.Parse(strings.NewReader(htmlStr))
	if err != nil {
		return false
	}

	var errorText string
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if errorText != "" {
			return
		}
		for _, attr := range node.Attr {
			if attr.Key == "id" && attr.Val == "errorText" {
				errorText = strings.TrimSpace(nodeText(node))
				return
			}
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)

	if errorText == "" {
		return false
	}

	return strings.Contains(strings.ToLower(errorText), "incorrect user id or password")
}

// IsADFSCredentialError checks if the ADFS login page shows a credential error
// (incorrect username or password) before the MFA stage.
func IsADFSCredentialError(p pageHTML) (bool, error) {
	htmlStr, err := p.HTML()
	if err != nil {
		return false, fmt.Errorf("%w: failed to extract page HTML: %v", ErrAuthentication, err)
	}
	return hasADFSCredentialError(htmlStr), nil
}
