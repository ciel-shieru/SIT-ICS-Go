package config

import (
	"os"
	"testing"

	"github.com/ciel-shieru/sit-ics-go/internal/credentialstore"
)

type mockStore struct {
	username string
	password string
	totp     string
	notFound bool
}

func (m *mockStore) GetUsername() (string, error) {
	if m.notFound {
		return "", credentialstore.ErrNotFound
	}
	return m.username, nil
}

func (m *mockStore) SetUsername(string) error { return nil }
func (m *mockStore) GetPassword() (string, error) {
	if m.notFound {
		return "", credentialstore.ErrNotFound
	}
	return m.password, nil
}
func (m *mockStore) SetPassword(string) error { return nil }
func (m *mockStore) GetTOTPSecret() (string, error) {
	if m.notFound {
		return "", credentialstore.ErrNotFound
	}
	return m.totp, nil
}
func (m *mockStore) SetTOTPSecret(string) error { return nil }
func (m *mockStore) Delete() error              { return nil }
func (m *mockStore) Close()                     {}
func (m *mockStore) IsOpen() bool               { return true }

func TestLoadCredentialsFromStore_EnvVarPriority(t *testing.T) {
	os.Setenv("USERNAME", "from-env")
	os.Setenv("PASSWORD", "from-env")
	os.Setenv("TOTP_SECRET", "from-env")
	defer func() {
		os.Unsetenv("USERNAME")
		os.Unsetenv("PASSWORD")
		os.Unsetenv("TOTP_SECRET")
	}()

	store := &mockStore{
		username: "from-store",
		password: "from-store",
		totp:     "from-store",
	}
	cfg := &Config{
		Username:     "from-env",
		Password:     "from-env",
		TOTPSecret:   "from-env",
		credStore:    store,
		BrowserMode:  BrowserAuto,
		TZ:           "Asia/Singapore",
		ServerPort:   8080,
	}

	if err := cfg.loadCredentialsFromStore(); err != nil {
		t.Fatalf("loadCredentialsFromStore() error = %v", err)
	}

	if cfg.Username != "from-env" {
		t.Errorf("Username = %q, want %q (env var should take priority)", cfg.Username, "from-env")
	}
	if cfg.Password != "from-env" {
		t.Errorf("Password = %q, want %q (env var should take priority)", cfg.Password, "from-env")
	}
	if cfg.TOTPSecret != "from-env" {
		t.Errorf("TOTPSecret = %q, want %q (env var should take priority)", cfg.TOTPSecret, "from-env")
	}
}

func TestLoadCredentialsFromStore_StoreFallback(t *testing.T) {
	os.Unsetenv("USERNAME")
	os.Unsetenv("PASSWORD")
	os.Unsetenv("TOTP_SECRET")

	store := &mockStore{
		username: "from-store",
		password: "from-store",
		totp:     "from-store",
	}
	cfg := &Config{
		Username:     "",
		Password:     "",
		TOTPSecret:   "",
		credStore:    store,
		BrowserMode:  BrowserAuto,
		TZ:           "Asia/Singapore",
		ServerPort:   8080,
	}

	if err := cfg.loadCredentialsFromStore(); err != nil {
		t.Fatalf("loadCredentialsFromStore() error = %v", err)
	}

	if cfg.Username != "from-store" {
		t.Errorf("Username = %q, want %q (should fall back to store)", cfg.Username, "from-store")
	}
	if cfg.Password != "from-store" {
		t.Errorf("Password = %q, want %q (should fall back to store)", cfg.Password, "from-store")
	}
	if cfg.TOTPSecret != "from-store" {
		t.Errorf("TOTPSecret = %q, want %q (should fall back to store)", cfg.TOTPSecret, "from-store")
	}
}

func TestLoadCredentialsFromStore_PartialFallback(t *testing.T) {
	os.Setenv("USERNAME", "from-env")
	os.Unsetenv("PASSWORD")
	os.Unsetenv("TOTP_SECRET")
	defer func() {
		os.Unsetenv("USERNAME")
		os.Unsetenv("PASSWORD")
		os.Unsetenv("TOTP_SECRET")
	}()

	store := &mockStore{
		username: "from-store",
		password: "from-store",
		totp:     "from-store",
	}
	cfg := &Config{
		Username:     "from-env",
		Password:     "",
		TOTPSecret:   "",
		credStore:    store,
		BrowserMode:  BrowserAuto,
		TZ:           "Asia/Singapore",
		ServerPort:   8080,
	}

	if err := cfg.loadCredentialsFromStore(); err != nil {
		t.Fatalf("loadCredentialsFromStore() error = %v", err)
	}

	if cfg.Username != "from-env" {
		t.Errorf("Username = %q, want %q (env var should take priority)", cfg.Username, "from-env")
	}
	if cfg.Password != "from-store" {
		t.Errorf("Password = %q, want %q (should fall back to store)", cfg.Password, "from-store")
	}
	if cfg.TOTPSecret != "from-store" {
		t.Errorf("TOTPSecret = %q, want %q (should fall back to store)", cfg.TOTPSecret, "from-store")
	}
}

func TestLoadCredentialsFromStore_StoreNotFound(t *testing.T) {
	os.Unsetenv("USERNAME")
	os.Unsetenv("PASSWORD")
	os.Unsetenv("TOTP_SECRET")

	store := &mockStore{notFound: true}
	cfg := &Config{
		Username:     "",
		Password:     "",
		TOTPSecret:   "",
		credStore:    store,
		BrowserMode:  BrowserAuto,
		TZ:           "Asia/Singapore",
		ServerPort:   8080,
	}

	if err := cfg.loadCredentialsFromStore(); err != nil {
		t.Fatalf("loadCredentialsFromStore() error = %v", err)
	}

	if cfg.Username != "" {
		t.Errorf("Username = %q, want empty (store returned not found)", cfg.Username)
	}
	if cfg.Password != "" {
		t.Errorf("Password = %q, want empty (store returned not found)", cfg.Password)
	}
	if cfg.TOTPSecret != "" {
		t.Errorf("TOTPSecret = %q, want empty (store returned not found)", cfg.TOTPSecret)
	}
}

func TestLoadCredentialsFromStore_NilStore(t *testing.T) {
	cfg := &Config{
		BrowserMode: BrowserAuto,
		TZ:          "Asia/Singapore",
		ServerPort:  8080,
	}

	if err := cfg.loadCredentialsFromStore(); err != nil {
		t.Fatalf("loadCredentialsFromStore() error = %v", err)
	}
}
