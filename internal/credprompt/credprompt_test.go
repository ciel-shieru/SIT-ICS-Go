//go:build !container

package credprompt

import (
	"fmt"
	"os"
	"testing"

	"github.com/ciel-shieru/sit-ics-go/internal/config"
	"github.com/ciel-shieru/sit-ics-go/internal/credentialstore"
)

// testStore is a minimal mock of credentialstore.Store for testing.
type testStore struct {
	username    string
	password    string
	totp        string
	usernameErr error
	passwordErr error
	totpErr     error
	open        bool
}

func (t *testStore) GetUsername() (string, error)    { return t.username, t.usernameErr }
func (t *testStore) SetUsername(string) error        { return nil }
func (t *testStore) GetPassword() (string, error)    { return t.password, t.passwordErr }
func (t *testStore) SetPassword(string) error        { return nil }
func (t *testStore) GetTOTPSecret() (string, error)  { return t.totp, t.totpErr }
func (t *testStore) SetTOTPSecret(string) error      { return nil }
func (t *testStore) Delete() error                   { return nil }
func (t *testStore) Close()                          {}
func (t *testStore) IsOpen() bool                    { return t.open }

func TestZeroBytes(t *testing.T) {
	b := []byte{0x41, 0x42, 0x43, 0x44}
	zeroBytes(b)
	for i, v := range b {
		if v != 0 {
			t.Errorf("zeroBytes[%d] = %d, want 0", i, v)
		}
	}
}

func TestZeroBytesEmpty(t *testing.T) {
	b := []byte{}
	zeroBytes(b) // should not panic
}

func TestErrNonInteractive(t *testing.T) {
	if ErrNonInteractive == nil {
		t.Fatal("ErrNonInteractive should not be nil")
	}
	msg := ErrNonInteractive.Error()
	if msg != "interactive credential prompt requires a terminal" {
		t.Errorf("ErrNonInteractive message = %q, want %q", msg, "interactive credential prompt requires a terminal")
	}
}

func TestPromptIfNeeded_RequiresTerminal(t *testing.T) {
	credentialstore.ResetForTesting()
	// When stdin is a pipe, PromptIfNeeded must return ErrNonInteractive
	// and must NOT modify the config.
	store := credentialstore.NewStore()
	store.Delete() // best-effort cleanup so hasAllCredentials returns false

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("create pipe: %v", err)
	}
	defer r.Close()
	defer w.Close()

	// Pre-populate config to ensure it's not overwritten
	cfg := &config.Config{
		Username: "existing",
	}

	oldStdin := os.Stdin
	os.Stdin = r
	defer func() { os.Stdin = oldStdin }()

	err = PromptIfNeeded(cfg)
	if err == nil {
		t.Fatal("expected error from PromptIfNeeded with non-terminal stdin")
	}
	if err != ErrNonInteractive {
		t.Errorf("error = %v, want ErrNonInteractive", err)
	}
	// Config must be unchanged
	if cfg.Username != "existing" {
		t.Errorf("config.Username = %q, want %q (unchanged)", cfg.Username, "existing")
	}
}

func TestHasAllCredentials_AllPresent(t *testing.T) {
	store := &testStore{
		username: "user",
		password: "pass",
		totp:     "totp",
		open:     true,
	}
	if !hasAllCredentials(store) {
		t.Error("expected true when all credentials exist")
	}
}

func TestHasAllCredentials_MissingUsername(t *testing.T) {
	store := &testStore{
		password: "pass",
		totp:     "totp",
		open:     true,
	}
	if hasAllCredentials(store) {
		t.Error("expected false when username is missing")
	}
}

func TestHasAllCredentials_MissingPassword(t *testing.T) {
	store := &testStore{
		username: "user",
		totp:     "totp",
		open:     true,
	}
	if hasAllCredentials(store) {
		t.Error("expected false when password is missing")
	}
}

func TestHasAllCredentials_MissingTOTP(t *testing.T) {
	store := &testStore{
		username: "user",
		password: "pass",
		open:     true,
	}
	if hasAllCredentials(store) {
		t.Error("expected false when TOTP secret is missing")
	}
}

func TestHasAllCredentials_AllEmpty(t *testing.T) {
	store := &testStore{open: true}
	if hasAllCredentials(store) {
		t.Error("expected false when all credentials are empty")
	}
}

func TestHasAllCredentials_GetError(t *testing.T) {
	store := &testStore{
		username:    "user",
		password:    "pass",
		totpErr:     fmt.Errorf("store error"),
		open:        true,
	}
	if hasAllCredentials(store) {
		t.Error("expected false when GetTOTPSecret returns error")
	}
}

func TestPromptIfNeeded_NonInteractive(t *testing.T) {
	credentialstore.ResetForTesting()
	// Ensure store has no credentials so we reach the TTY check.
	store := credentialstore.NewStore()
	store.Delete() // best-effort cleanup

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("create pipe: %v", err)
	}
	defer r.Close()
	defer w.Close()

	oldStdin := os.Stdin
	os.Stdin = r
	defer func() { os.Stdin = oldStdin }()

	cfg := &config.Config{}
	err = PromptIfNeeded(cfg)

	if err != ErrNonInteractive {
		t.Errorf("PromptIfNeeded() error = %v, want ErrNonInteractive", err)
	}
	// Config should be unchanged
	if cfg.Username != "" || cfg.Password != "" || cfg.TOTPSecret != "" {
		t.Error("config should not be modified on error")
	}
}


