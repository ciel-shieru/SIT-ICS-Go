package browser

import (
	"net/url"
	"strings"
	"testing"
)

func TestValidHostEntry(t *testing.T) {
	// 64-char label + ".com"
	longLabel := strings.Repeat("a", 64) + ".com"
	// 254-char host (a 249-char first label + ".com" = 254)
	longHost := strings.Repeat("a", 249) + ".com"

	tests := []struct {
		name    string
		entry   string
		wantErr bool
	}{
		{"sit_apex", "singaporetech.edu.sg", false},
		{"ms_login", "login.microsoftonline.com", false},
		{"hyphen_label", "a-b.com", false},
		{"digit_label", "a1.com", false},
		{"ipv4", "192.168.1.100", false},
		{"ipv6_loopback", "::1", false},
		{"ipv6", "2001:db8::1", false},

		{"empty", "", true},
		{"space", "example .com", true},
		{"empty_label", "example..com", true},
		{"leading_hyphen", "-example.com", true},
		{"trailing_hyphen", "example-.com", true},
		{"single_dot", ".com", true},
		{"trailing_dot", "example.com.", true},
		{"port", "example.com:443", true},
		{"path", "example.com/path", true},
		{"wildcard", "*", true},
		{"label_too_long", longLabel, true},
		{"host_too_long", longHost, true},
		{"underscore", "exa_mple.com", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidHostEntry(tt.entry)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidHostEntry(%q) error = %v, wantErr %v", tt.entry, err, tt.wantErr)
			}
		})
	}
}

func TestSplitHostEntries(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{"empty", "", nil},
		{"two", "a.com, b.com", []string{"a.com", "b.com"}},
		{"extra_commas", "a.com,,b.com,", []string{"a.com", "b.com"}},
		{"single", "a.com", []string{"a.com"}},
		{"whitespace_only_dropped", " , ,  , a.com", []string{"a.com"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SplitHostEntries(tt.input)
			if len(got) != len(tt.want) {
				t.Fatalf("SplitHostEntries(%q) = %v (len %d), want %v (len %d)", tt.input, got, len(got), tt.want, len(tt.want))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("SplitHostEntries(%q)[%d] = %q, want %q", tt.input, i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestNewHostWhitelist(t *testing.T) {
	t.Run("valid_mix_builds", func(t *testing.T) {
		w, err := NewHostWhitelist([]string{"example.com", "192.168.1.100", "a.b.org"})
		if err != nil {
			t.Fatalf("NewHostWhitelist() error = %v", err)
		}
		if !w.IsHostAllowed("sub.example.com") {
			t.Error("expected sub.example.com to be allowed")
		}
	})

	t.Run("duplicates_deduplicated", func(t *testing.T) {
		w, err := NewHostWhitelist([]string{"example.com", "example.com", "EXAMPLE.COM"})
		if err != nil {
			t.Fatalf("NewHostWhitelist() error = %v", err)
		}
		if len(w.suffixes) != 1 {
			t.Errorf("expected 1 suffix, got %d (%v)", len(w.suffixes), w.suffixes)
		}
	})

	t.Run("uppercase_normalized", func(t *testing.T) {
		w, err := NewHostWhitelist([]string{"EXAMPLE.COM"})
		if err != nil {
			t.Fatalf("NewHostWhitelist() error = %v", err)
		}
		if !w.IsHostAllowed("sub.example.com") {
			t.Error("expected sub.example.com to be allowed after normalization")
		}
	})

	t.Run("nil_input_empty_whitelist", func(t *testing.T) {
		w, err := NewHostWhitelist(nil)
		if err != nil {
			t.Fatalf("NewHostWhitelist(nil) error = %v, want nil", err)
		}
		if w == nil {
			t.Fatal("expected non-nil whitelist for nil input")
		}
		if len(w.suffixes) != 0 {
			t.Errorf("expected 0 suffixes, got %d", len(w.suffixes))
		}
		if w.IsHostAllowed("example.com") {
			t.Error("expected empty whitelist to deny everything")
		}
	})

	t.Run("one_invalid_entry_errors", func(t *testing.T) {
		_, err := NewHostWhitelist([]string{"example.com", "bad entry!"})
		if err == nil {
			t.Fatal("expected error for invalid entry, got nil")
		}
		if !strings.Contains(err.Error(), "bad entry!") {
			t.Errorf("expected error to mention the entry, got: %v", err)
		}
	})
}

func TestHostWhitelistIsHostAllowed(t *testing.T) {
	w, err := NewHostWhitelist([]string{"example.com"})
	if err != nil {
		t.Fatalf("NewHostWhitelist() error = %v", err)
	}

	tests := []struct {
		name  string
		host  string
		want  bool
	}{
		{"apex", "example.com", true},
		{"subdomain", "sub.example.com", true},
		{"nested_subdomain", "a.b.example.com", true},
		{"suffix_spoofing", "example.com.evil.com", false},
		{"evil_prefix", "evilexample.com", false},
		{"not_matching", "notexample.com", false},
		{"different_tld", "example.org", false},
		{"case_insensitive", "SUB.EXAMPLE.COM", true},
		{"trailing_dot", "example.com.", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := w.IsHostAllowed(tt.host); got != tt.want {
				t.Errorf("IsHostAllowed(%q) = %v, want %v", tt.host, got, tt.want)
			}
		})
	}

	t.Run("nil_whitelist", func(t *testing.T) {
		var nilW *HostWhitelist
		if nilW.IsHostAllowed("example.com") {
			t.Error("expected nil whitelist to deny everything")
		}
	})
}

func parseURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("url.Parse(%q) error = %v", raw, err)
	}
	return u
}

func TestDecideURL(t *testing.T) {
	w := DefaultHostWhitelist()

	tests := []struct {
		name  string
		raw   string
		want  Decision
	}{
		// Allowed SIT
		{"sit_in4sit", "https://in4sit.singaporetech.edu.sg/psc/CSSISSTD/EMPLOYEE/SA/c/SA_LEARNER_SERVICES.SSR_SSENRL_LIST.GBL", DecisionAllow},
		{"sit_fs_port", "https://fs.singaporetech.edu.sg:8443/adfs/ls/", DecisionAllow},
		{"sit_xsite", "https://xsite.singaporetech.edu.sg/d2l/api/le/1.43/calendar/events/", DecisionAllow},
		{"sit_apex", "https://singaporetech.edu.sg", DecisionAllow},
		{"sit_www", "https://www.singaporetech.edu.sg", DecisionAllow},
		{"sit_uppercase", "https://IN4SIT.SINGAPORETECH.EDU.SG/x", DecisionAllow},

		// Blocked SIT spoofing
		{"spoof_suffix", "https://singaporetech.edu.sg.evil.com/", DecisionBlock},
		{"spoof_prefix_hyphen", "https://evil-singaporetech.edu.sg/", DecisionBlock},
		{"spoof_prefix", "https://mysingaporetech.edu.sg/", DecisionBlock},
		{"spoof_subdomain_suffix", "https://in4sit.singaporetech.edu.sg.attacker.com/", DecisionBlock},
		{"spoof_fs_suffix", "https://fs.singaporetech.edu.sg.fake.org/", DecisionBlock},

		// Blocked unrelated
		{"unrelated_evil", "https://evil.com/", DecisionBlock},
		{"unrelated_example", "https://example.com/", DecisionBlock},

		// Scheme behavior
		{"http_allowed", "http://in4sit.singaporetech.edu.sg/", DecisionAllow},
		{"ws_allowed", "ws://in4sit.singaporetech.edu.sg/ws", DecisionAllow},
		{"wss_blocked", "wss://evil.com/", DecisionBlock},
		{"data_exempt", "data:text/html,hello", DecisionExempt},
		{"data_no_host_exempt", "data:", DecisionExempt},
		{"blob_exempt", "blob:https://xsite.singaporetech.edu.sg/abc-123", DecisionExempt},
		{"about_exempt", "about:blank", DecisionExempt},
		{"chrome_extension_exempt", "chrome-extension://abcd/component.html", DecisionExempt},
		{"file_exempt", "file:///etc/hosts", DecisionExempt},

		// Fail-closed edge cases
		{"empty_host", "https://", DecisionBlock},
		{"unknown_scheme_host", "foo://example.com/", DecisionBlock},
		{"unknown_scheme_no_host", "foo://", DecisionExempt},

		// IP literals
		{"ip_not_whitelisted", "https://10.0.0.1/", DecisionBlock},
		{"ipv6_not_whitelisted", "https://[::1]/", DecisionBlock},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DecideURL(w, parseURL(t, tt.raw))
			if got != tt.want {
				t.Errorf("DecideURL(%q) = %v, want %v", tt.raw, got, tt.want)
			}
		})
	}

	t.Run("nil_url", func(t *testing.T) {
		if got := DecideURL(w, nil); got != DecisionBlock {
			t.Errorf("DecideURL(nil) = %v, want DecisionBlock", got)
		}
	})

	t.Run("ip_whitelisted", func(t *testing.T) {
		ipW, err := NewHostWhitelist([]string{"10.0.0.1"})
		if err != nil {
			t.Fatalf("NewHostWhitelist() error = %v", err)
		}
		if got := DecideURL(ipW, parseURL(t, "https://10.0.0.1/")); got != DecisionAllow {
			t.Errorf("DecideURL(whitelisted IP) = %v, want DecisionAllow", got)
		}
	})
}

func TestDefaultHostWhitelist(t *testing.T) {
	w := DefaultHostWhitelist()
	if w == nil {
		t.Fatal("DefaultHostWhitelist() = nil, want non-nil")
	}

	allowed := []string{
		"in4sit.singaporetech.edu.sg",
		"fs.singaporetech.edu.sg",
		"xsite.singaporetech.edu.sg",
		"login.microsoftonline.com",
		"aadcdn.msftauth.net",
	}
	for _, host := range allowed {
		if !w.IsHostAllowed(host) {
			t.Errorf("expected %q to be allowed", host)
		}
	}

	blocked := []string{
		"example.com",
		"singaporetech.edu.sg.evil.com",
	}
	for _, host := range blocked {
		if w.IsHostAllowed(host) {
			t.Errorf("expected %q to be blocked", host)
		}
	}
}

func TestBuildHostWhitelist(t *testing.T) {
	t.Run("empty_equals_default", func(t *testing.T) {
		w, err := BuildHostWhitelist(BrowserConfig{})
		if err != nil {
			t.Fatalf("BuildHostWhitelist() error = %v", err)
		}
		for _, host := range []string{"in4sit.singaporetech.edu.sg", "login.microsoftonline.com", "aadcdn.msftauth.net"} {
			if !w.IsHostAllowed(host) {
				t.Errorf("expected default entry %q to be allowed", host)
			}
		}
		for _, host := range []string{"example.com", "evil.com"} {
			if w.IsHostAllowed(host) {
				t.Errorf("expected %q to be blocked", host)
			}
		}
	})

	t.Run("additive", func(t *testing.T) {
		w, err := BuildHostWhitelist(BrowserConfig{AllowedHosts: []string{"example.com"}})
		if err != nil {
			t.Fatalf("BuildHostWhitelist() error = %v", err)
		}
		if !w.IsHostAllowed("sub.example.com") {
			t.Error("expected sub.example.com to be allowed (additive entry)")
		}
		if !w.IsHostAllowed("in4sit.singaporetech.edu.sg") {
			t.Error("expected default entries to still be allowed (additive)")
		}
	})

	t.Run("invalid_entry_errors", func(t *testing.T) {
		_, err := BuildHostWhitelist(BrowserConfig{AllowedHosts: []string{"bad entry!"}})
		if err == nil {
			t.Fatal("expected error for invalid entry, got nil")
		}
	})
}

func TestDecideURLZeroValueIsBlock(t *testing.T) {
	if Decision(0) != DecisionBlock {
		t.Errorf("Decision(0) = %v, want DecisionBlock (fail-closed invariant)", Decision(0))
	}
}
