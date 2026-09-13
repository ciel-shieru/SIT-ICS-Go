//go:build !container

package credprompt

import (
	"os"
	"testing"

	"github.com/ciel-shieru/sit-ics-go/internal/config"
)

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

func TestPromptIfNeeded_NonInteractive(t *testing.T) {
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

func TestPromptIfNeeded_RequiresTerminal(t *testing.T) {
	// When stdin is a pipe, PromptIfNeeded must return ErrNonInteractive
	// and must NOT modify the config.
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
