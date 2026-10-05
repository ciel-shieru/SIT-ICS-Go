package config

import (
	"os"
	"testing"

	"github.com/spf13/pflag"
)

func TestCredentialFlagsRemoved(t *testing.T) {
	fs := pflag.NewFlagSet("app", pflag.ContinueOnError)
	_ = fs.String("start-date", "", "Start date (YYYY-MM-DD)")

	if fs.Lookup("username") != nil {
		t.Error("flag 'username' should not be registered (removed for keyring-based auth)")
	}
	if fs.Lookup("password") != nil {
		t.Error("flag 'password' should not be registered (removed for keyring-based auth)")
	}
	if fs.Lookup("totp-secret") != nil {
		t.Error("flag 'totp-secret' should not be registered (removed for keyring-based auth)")
	}
}

func TestCredentialFlagsRemoved_FromApplyFlags(t *testing.T) {
	cfg := &Config{BrowserMode: BrowserAuto, TZ: "Asia/Singapore", ServerPort: 42748}
	_ = cfg

	fs := pflag.NewFlagSet("app", pflag.ContinueOnError)

	if fs.Lookup("username") != nil {
		t.Error("flag 'username' should not exist in a fresh FlagSet")
	}
	if fs.Lookup("password") != nil {
		t.Error("flag 'password' should not exist in a fresh FlagSet")
	}
	if fs.Lookup("totp-secret") != nil {
		t.Error("flag 'totp-secret' should not exist in a fresh FlagSet")
	}
}

func TestOverrideCredentialsFlagExists(t *testing.T) {
	fs := pflag.NewFlagSet("app", pflag.ContinueOnError)
	_ = fs.Bool("override-credentials", false, "Force credential prompt")

	if fs.Lookup("override-credentials") == nil {
		t.Error("flag 'override-credentials' should be registerable")
	}

	// Verify applyFlags actually sets the field
	os.Args = []string{"test", "--override-credentials"}
	cfg2 := &Config{BrowserMode: BrowserAuto, TZ: "Asia/Singapore", ServerPort: 42748}
	if err := applyFlags(cfg2); err != nil {
		t.Fatalf("applyFlags: %v", err)
	}
	os.Args = []string{"test"}
	if !cfg2.OverrideCredentials {
		t.Error("OverrideCredentials should be true when --override-credentials is passed")
	}
}

func TestApplyFlags_BrowserAllowedHosts(t *testing.T) {
	t.Run("flag_sets_field", func(t *testing.T) {
		os.Args = []string{"test", "--browser-allowed-hosts", "example.com, cdn.example.org"}
		cfg := &Config{BrowserMode: BrowserAuto, TZ: "Asia/Singapore", ServerPort: 42748}
		if err := applyFlags(cfg); err != nil {
			t.Fatalf("applyFlags: %v", err)
		}
		os.Args = []string{"test"}
		if cfg.BrowserAllowedHosts != "example.com, cdn.example.org" {
			t.Errorf("BrowserAllowedHosts = %q, want %q", cfg.BrowserAllowedHosts, "example.com, cdn.example.org")
		}
	})

	t.Run("unchanged_flag_leaves_env_value", func(t *testing.T) {
		os.Args = []string{"test"}
		cfg := &Config{
			BrowserMode:         BrowserAuto,
			TZ:                  "Asia/Singapore",
			ServerPort:          42748,
			BrowserAllowedHosts: "envvalue.com",
		}
		if err := applyFlags(cfg); err != nil {
			t.Fatalf("applyFlags: %v", err)
		}
		os.Args = []string{"test"}
		if cfg.BrowserAllowedHosts != "envvalue.com" {
			t.Errorf("BrowserAllowedHosts = %q, want %q (unchanged flag should leave env value)", cfg.BrowserAllowedHosts, "envvalue.com")
		}
	})
}
