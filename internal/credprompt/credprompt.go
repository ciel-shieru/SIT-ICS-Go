//go:build !container

package credprompt

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/ciel-shieru/sit-ics-go/internal/config"
	"github.com/ciel-shieru/sit-ics-go/internal/credentialstore"
	"golang.org/x/term"
)

// ErrNonInteractive is returned when stdin is not a terminal and interactive
// prompting cannot be performed.
var ErrNonInteractive = fmt.Errorf("interactive credential prompt requires a terminal")

// PromptIfNeeded prompts the user for credentials interactively if stdin is a
// terminal. Each secret is immediately stored to the OS keyring and the raw
// input buffers are zeroed from memory.
func PromptIfNeeded(cfg *config.Config) error {
	store := credentialstore.NewStore()

	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		return ErrNonInteractive
	}

	if !store.IsOpen() {
		return fmt.Errorf("credential store unavailable")
	}

	reader := bufio.NewReader(os.Stdin)

	// Username (unmasked)
	username, raw, err := readUsername(reader, os.Stdout)
	zeroBytes(raw)
	if err != nil {
		return fmt.Errorf("read username: %w", err)
	}
	if username == "" {
		return fmt.Errorf("username cannot be empty")
	}

	// Password (masked)
	passRaw, err := readSecret("Password: ", os.Stdin, os.Stdout)
	if err != nil {
		zeroBytes(passRaw)
		return fmt.Errorf("read password: %w", err)
	}

	// TOTP Secret (masked)
	totpRaw, err := readSecret("TOTP Secret: ", os.Stdin, os.Stdout)
	if err != nil {
		zeroBytes(totpRaw)
		return fmt.Errorf("read totp secret: %w", err)
	}

	// Capture string copies before zeroing raw buffers
	usernameStr := username
	passStr := string(passRaw)
	totpStr := string(totpRaw)

	// Store all three to keyring (keyring copies into OS memory)
	if err := store.SetUsername(usernameStr); err != nil {
		zeroBytes(passRaw)
		zeroBytes(totpRaw)
		return fmt.Errorf("store username: %w", err)
	}
	if err := store.SetPassword(passStr); err != nil {
		zeroBytes(passRaw)
		zeroBytes(totpRaw)
		return fmt.Errorf("store password: %w", err)
	}
	zeroBytes(passRaw)
	if err := store.SetTOTPSecret(totpStr); err != nil {
		zeroBytes(totpRaw)
		return fmt.Errorf("store totp secret: %w", err)
	}
	zeroBytes(totpRaw)

	// Set config fields (needed for auth flow)
	cfg.Username = usernameStr
	cfg.Password = passStr
	cfg.TOTPSecret = totpStr

	// Zero local string copies (config now holds them)
	usernameStr = ""
	passStr = ""
	totpStr = ""

	return nil
}

// readUsername reads an unmasked username from the reader.
func readUsername(reader *bufio.Reader, writer io.Writer) (string, []byte, error) {
	if _, err := writer.Write([]byte("Username: ")); err != nil {
		return "", nil, fmt.Errorf("write username prompt: %w", err)
	}
	line, err := reader.ReadString('\n')
	raw := []byte(line)
	if err != nil {
		return "", raw, fmt.Errorf("read username line: %w", err)
	}
	trimmed := strings.TrimSpace(line)
	return trimmed, raw, nil
}

// readSecret reads a masked secret (password or TOTP) from the terminal.
func readSecret(prompt string, reader io.Reader, writer io.Writer) ([]byte, error) {
	if _, err := writer.Write([]byte(prompt)); err != nil {
		return nil, fmt.Errorf("write secret prompt: %w", err)
	}

	type fdReader interface {
		Fd() uintptr
	}
	fr, ok := reader.(fdReader)
	if !ok {
		return nil, fmt.Errorf("reader does not support Fd()")
	}

	secret, err := term.ReadPassword(int(fr.Fd()))
	if err != nil {
		return secret, fmt.Errorf("read secret: %w", err)
	}

	// Print newline after masked input
	if _, err := writer.Write([]byte("\n")); err != nil {
		zeroBytes(secret)
		return secret, fmt.Errorf("write newline: %w", err)
	}

	return secret, nil
}

// zeroBytes fills a byte slice with zeros to prevent secrets from remaining in memory.
func zeroBytes(b []byte) {
	for i := range b {
		b[i] = 0
	}
}


