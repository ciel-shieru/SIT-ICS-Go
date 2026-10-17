package browser

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
)

// Decision is the outcome of applying the host whitelist to a request URL.
// DecisionBlock is the zero value so that any defaulting is fail-closed.
type Decision int

const (
	DecisionBlock Decision = iota // request must be aborted
	DecisionAllow                 // request may proceed
	DecisionExempt                // no network egress (data:, blob:, ...); allowed without host check
)

// HostWhitelist is an immutable set of domain-suffix rules. Each rule
// matches the apex host exactly and every subdomain of it.
type HostWhitelist struct {
	suffixes []string // lowercased, validated, de-duplicated
}

// egressSchemes are the URL schemes that carry real network egress and whose
// host must be checked against the whitelist.
var egressSchemes = map[string]bool{"http": true, "https": true, "ws": true, "wss": true}

// exemptSchemes are URL schemes that perform no network egress and are allowed
// without a host check.
var exemptSchemes = map[string]bool{"data": true, "blob": true, "about": true, "chrome-extension": true, "chrome": true, "devtools": true, "file": true, "filesystem": true}

// NewHostWhitelist builds a whitelist from raw entries. Each entry must pass
// ValidHostEntry. Entries are lowercased and de-duplicated (order preserved).
// nil/empty input yields an empty (deny-all) whitelist, not an error.
func NewHostWhitelist(entries []string) (*HostWhitelist, error) {
	w := &HostWhitelist{}
	seen := make(map[string]bool)
	for _, entry := range entries {
		e := strings.ToLower(entry)
		if err := ValidHostEntry(e); err != nil {
			return nil, fmt.Errorf("invalid whitelist entry %q: %w", entry, err)
		}
		if seen[e] {
			continue
		}
		seen[e] = true
		w.suffixes = append(w.suffixes, e)
	}
	return w, nil
}

// ValidHostEntry reports whether s is a syntactically valid whitelist entry:
// an IPv4/IPv6 literal (net.ParseIP) or a DNS hostname of 1-253 chars made of
// labels of 1-63 chars from [a-zA-Z0-9-] with no leading/trailing hyphen per
// label, no empty labels, no trailing dot, and no characters from ":/\\?# ".
// (Ports, paths, and URLs are rejected: they belong in request URLs, not in
// whitelist entries.)
func ValidHostEntry(s string) error {
	if s == "" {
		return errors.New("empty entry")
	}
	if net.ParseIP(s) != nil {
		return nil
	}
	if len(s) > 253 {
		return fmt.Errorf("host too long (%d chars, max 253)", len(s))
	}
	if strings.ContainsAny(s, ":/\\?# ") {
		return fmt.Errorf("entry contains disallowed character (ports, paths, and URLs are not valid entries)")
	}
	if strings.HasSuffix(s, ".") {
		return fmt.Errorf("entry has trailing dot")
	}
	for _, label := range strings.Split(s, ".") {
		if label == "" {
			return fmt.Errorf("entry has empty label")
		}
		if len(label) > 63 {
			return fmt.Errorf("label %q too long (%d chars, max 63)", label, len(label))
		}
		if label[0] == '-' || label[len(label)-1] == '-' {
			return fmt.Errorf("label %q has leading/trailing hyphen", label)
		}
		for _, r := range label {
			if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-') {
				return fmt.Errorf("label %q contains invalid character %q", label, string(r))
			}
		}
	}
	return nil
}

// SplitHostEntries splits a comma-separated value into trimmed, non-empty
// entries. Used for the BROWSER_ALLOWED_HOSTS env var / CLI flag.
func SplitHostEntries(s string) []string {
	var entries []string
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		entries = append(entries, part)
	}
	return entries
}

// defaultAllowedHosts is the built-in browser egress whitelist.
//
// Group 1: Singapore Institute of Technology (institutional).
var defaultAllowedHosts = []string{
	"singaporetech.edu.sg", // apex + ALL subdomains: in4sit (PeopleSoft), fs (ADFS), xsite (BrightSpace), www, and any future SIT subdomain

	// Group 2: Microsoft Entra ID / ADFS login flow. The ADFS login page is
	// served from fs.singaporetech.edu.sg (Group 1) but its JavaScript, CSS,
	// telemetry, MFA orchestration, and asset CDN traffic go to the following
	// Microsoft-owned hosts. Verified against the live login pages
	// (login.microsoftonline.com page config: urlCdn, cdnRoots, desktop-SSO
	// probe URLs; fs.singaporetech.edu.sg ADFS portal: MFA proofup URL).
	"login.microsoftonline.com",                // Entra ID core: login page logic, userrealm APIs, token endpoints, MFA orchestration
	"login.windows.net",                        // alias endpoint of login.microsoftonline.com (redirects)
	"aadcdn.msauth.net",                        // login-page asset CDN root #1 (verified in page cdnRoots)
	"aadcdn.msftauth.net",                      // login-page asset CDN root #2, current primary (verified: urlCdn, script srcs)
	"msauthimages.msauth.net",                  // tenant branding images on the login page
	"secure.aadcdn.microsoftonline-pub.com",   // current public CDN for login-page assets
	"secure.aadcdn.microsoft.com",             // legacy CDN variant (hedge against CDN rotation)
	"clienttls.microsoft.com",                 // client TLS negotiation/telemetry for the login page
	"sts.windows.net",                          // legacy SAML / STS token endpoint
	"autologon.microsoftazuread-sso.com",       // desktop SSO (IWA/Edge) probes referenced by login page config
	"account.activedirectory.windowsazure.com", // Azure MFA registration/proofup (referenced by SIT ADFS portal custom MFA JS)
}

// DefaultHostWhitelist returns the built-in whitelist: the SIT institutional
// domain plus the Microsoft Entra ID / ADFS login-flow domain group.
func DefaultHostWhitelist() *HostWhitelist {
	w, err := NewHostWhitelist(defaultAllowedHosts)
	if err != nil {
		// These literals are valid by construction; a failure here is a
		// programming error.
		panic(fmt.Sprintf("internal error: default whitelist is invalid: %v", err))
	}
	return w
}

// BuildHostWhitelist returns the effective whitelist for a session: the
// default whitelist plus cfg.AllowedHosts (additive). Returns an error if any
// extra entry fails ValidHostEntry (fail closed).
func BuildHostWhitelist(cfg BrowserConfig) (*HostWhitelist, error) {
	w, err := NewHostWhitelist(append(append([]string{}, defaultAllowedHosts...), cfg.AllowedHosts...))
	if err != nil {
		return nil, err
	}
	return w, nil
}

// IsHostAllowed reports whether host matches any rule. Nil receiver → false.
// Lowercases and strips a single trailing dot before matching.
func (w *HostWhitelist) IsHostAllowed(host string) bool {
	if w == nil {
		return false
	}
	h := strings.ToLower(host)
	h = strings.TrimSuffix(h, ".")
	for _, s := range w.suffixes {
		if h == s || strings.HasSuffix(h, "."+s) {
			return true
		}
	}
	return false
}

// DecideURL is the pure policy function. No I/O, no rod imports.
func DecideURL(w *HostWhitelist, u *url.URL) Decision {
	if u == nil {
		return DecisionBlock
	}
	scheme := strings.ToLower(u.Scheme)
	if exemptSchemes[scheme] {
		return DecisionExempt
	}
	if !egressSchemes[scheme] {
		// Unknown scheme: fail closed when it carries a host.
		if u.Hostname() == "" {
			return DecisionExempt
		}
		return DecisionBlock
	}
	host := u.Hostname() // lowercased by url.Parse, port stripped
	if host == "" {
		return DecisionBlock // egress scheme with no host: malformed, fail closed
	}
	if w != nil && w.IsHostAllowed(host) {
		return DecisionAllow
	}
	return DecisionBlock
}
