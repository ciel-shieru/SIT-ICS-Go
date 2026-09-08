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

// AuthenticateADFS performs the complete ADFS login workflow:
// 1. Navigate to login page
// 2. Fill username and password
// 3. Submit credentials
// 4. Handle MFA if present (TOTP)
// 5. Wait for SAML redirect
// 6. Verify authenticated destination
func AuthenticateADFS(ctx context.Context, incognito *rod.Browser, req AuthRequest, waitNavigation bool, cfg BrowserConfig, isAllowedOrigin func(string) bool) (*rod.Page, error) {
	navigateCtx, navigateCancel := context.WithTimeout(ctx, cfg.NavigationTimeout)
	defer navigateCancel()

	initialURL := req.URL
	debug(cfg, "navigating to %s", initialURL)
	page := incognito.MustPage(initialURL).Context(navigateCtx)

	if err := page.WaitStable(3 * time.Second); err != nil {
		debug(cfg, "wait stable failed: %v", err)
	}

	finalURL := page.MustInfo().URL
	debug(cfg, "navigated to %s", finalURL)
	if !isAllowedOrigin(finalURL) {
		return nil, fmt.Errorf("%w: redirect to disallowed origin %s (started from %s)", ErrAuthentication, finalURL, initialURL)
	}

	if waitNavigation {
		page.WaitNavigation(proto.PageLifecycleEventNameNetworkAlmostIdle)()
	}

	debug(cfg, "finding username field")
	el, err := page.Element("#userNameInput")
	if err != nil {
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
	if err := el.Input(req.Username); err != nil {
		return nil, fmt.Errorf("%w: failed to fill username: %v", ErrCredentialExtraction, err)
	}

	debug(cfg, "finding password field")
	passEl, err := page.Element("#passwordInput")
	if err != nil {
		return nil, fmt.Errorf("%w: password field not found: %v", ErrAuthentication, err)
	}
	debug(cfg, "filling password")
	if err := passEl.Input(req.Password); err != nil {
		return nil, fmt.Errorf("%w: failed to fill password: %v", ErrCredentialExtraction, err)
	}

	debug(cfg, "finding submit button")
	submitEl, err := page.Element("#submitButton")
	if err != nil {
		return nil, fmt.Errorf("%w: submit button not found: %v", ErrAuthentication, err)
	}
	debug(cfg, "clicking submit button")
	if err := submitEl.Click(proto.InputMouseButtonLeft, 1); err != nil {
		return nil, fmt.Errorf("%w: failed to submit credentials: %v", ErrAuthentication, err)
	}

	if waitNavigation {
		page.WaitNavigation(proto.PageLifecycleEventNameNetworkAlmostIdle)()
	}
	if err := page.WaitStable(5 * time.Second); err != nil {
		debug(cfg, "wait stable after submit failed: %v", err)
	}

	debug(cfg, "checking for MFA field")
	mfaEl, err := page.Element("#verificationCodeInput")
	mfaVisible := err == nil

	if mfaVisible {
		debug(cfg, "MFA detected, generating TOTP code")
		totpCode, err := totp.GenerateAtOffset(req.TOTPSecret, time.Now().UTC(), 1)
		if err != nil {
			return nil, fmt.Errorf("%w: failed to generate TOTP: %v", ErrAuthentication, err)
		}
		debug(cfg, "filling MFA code")
		if err := mfaEl.Input(totpCode); err != nil {
			return nil, fmt.Errorf("%w: failed to fill MFA code: %v", ErrCredentialExtraction, err)
		}
		debug(cfg, "finding sign-in button")
		signInEl, err := page.Element("#signInButton")
		if err != nil {
			return nil, fmt.Errorf("%w: sign-in button not found: %v", ErrAuthentication, err)
		}
		debug(cfg, "clicking sign-in button")
		if err := signInEl.Click(proto.InputMouseButtonLeft, 1); err != nil {
			return nil, fmt.Errorf("%w: failed to submit MFA code: %v", ErrAuthentication, err)
		}
		debug(cfg, "waiting for SAML redirect after MFA")
		page.WaitNavigation(proto.PageLifecycleEventNameNetworkAlmostIdle)()
		if err := page.WaitStable(5 * time.Second); err != nil {
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
		if err := page.WaitStable(5 * time.Second); err != nil {
			debug(cfg, "wait stable after submit redirect failed: %v", err)
		}
	}

	debug(cfg, "waiting for ADFS redirect to complete")
	page.WaitNavigation(proto.PageLifecycleEventNameNetworkAlmostIdle)()
	if err := page.WaitStable(5 * time.Second); err != nil {
		debug(cfg, "wait stable after redirect failed: %v", err)
	}

	finalURL = getPageURL(page)
	debug(cfg, "auth complete, final URL: %s", finalURL)

	return page, nil
}

// ExtractADFSLoginError extracts the error message from an ADFS login page.
func ExtractADFSLoginError(page *rod.Page) (string, error) {
	htmlStr, err := page.HTML()
	if err != nil {
		return "", fmt.Errorf("get page HTML: %w", err)
	}

	doc, err := html.Parse(strings.NewReader(htmlStr))
	if err != nil {
		return "", fmt.Errorf("parse HTML: %w", err)
	}

	var findErrorText func(*html.Node) string
	findErrorText = func(n *html.Node) string {
		if n.Type == html.ElementNode && n.Data == "p" {
			for _, a := range n.Attr {
				if a.Key == "id" && a.Val == "errorText" {
					var sb strings.Builder
					for c := n.FirstChild; c != nil; c = c.NextSibling {
						if c.Type == html.TextNode {
							sb.WriteString(c.Data)
						}
					}
					return strings.TrimSpace(sb.String())
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
