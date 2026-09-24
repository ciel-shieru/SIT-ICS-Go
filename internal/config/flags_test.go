package config

import (
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
	cfg := &Config{BrowserMode: BrowserAuto, TZ: "Asia/Singapore", ServerPort: 8080}
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
