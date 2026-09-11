//go:build !container

package credentialstore

import (
	"fmt"
	"sync"

	"github.com/zalando/go-keyring"
)

const serviceName = "sit-ics-go"

// desktopStore wraps the OS keyring for credential persistence.
type desktopStore struct {
	mu   sync.RWMutex
	open bool
}

// NewStore returns the singleton desktop-backed credential store.
func NewStore() Store {
	once.Do(func() {
		s := &desktopStore{}
		if err := testKeyring(); err != nil {
			s.open = false
		} else {
			s.open = true
		}
		defaultStore = s
	})
	return defaultStore
}

// testKeyring checks whether the keyring backend is accessible by performing a
// round-trip Set+Get+Delete on a dummy credential. A failure at any step
// indicates no keyring service is available (e.g. headless CI, missing libsecret).
func testKeyring() error {
	if err := keyring.Set(serviceName, "_test", "probe"); err != nil {
		return fmt.Errorf("keyring unavailable (set): %w", err)
	}
	if _, err := keyring.Get(serviceName, "_test"); err != nil {
		keyring.Delete(serviceName, "_test") // best-effort cleanup
		return fmt.Errorf("keyring unavailable (get): %w", err)
	}
	if err := keyring.Delete(serviceName, "_test"); err != nil {
		return fmt.Errorf("keyring unavailable (delete): %w", err)
	}
	return nil
}

func (s *desktopStore) GetUsername() (string, error) {
	if !s.open {
		return "", ErrStoreUnavailable
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	secret, err := keyring.Get(serviceName, "username")
	if err != nil {
		return "", fmt.Errorf("get username: %w", err)
	}
	return secret, nil
}

func (s *desktopStore) SetUsername(username string) error {
	if !s.open {
		return ErrStoreUnavailable
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := keyring.Set(serviceName, "username", username); err != nil {
		return fmt.Errorf("set username: %w", err)
	}
	return nil
}

func (s *desktopStore) GetPassword() (string, error) {
	if !s.open {
		return "", ErrStoreUnavailable
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	secret, err := keyring.Get(serviceName, "password")
	if err != nil {
		return "", fmt.Errorf("get password: %w", err)
	}
	return secret, nil
}

func (s *desktopStore) SetPassword(password string) error {
	if !s.open {
		return ErrStoreUnavailable
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := keyring.Set(serviceName, "password", password); err != nil {
		return fmt.Errorf("set password: %w", err)
	}
	return nil
}

func (s *desktopStore) GetTOTPSecret() (string, error) {
	if !s.open {
		return "", ErrStoreUnavailable
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	secret, err := keyring.Get(serviceName, "totp_secret")
	if err != nil {
		return "", fmt.Errorf("get totp_secret: %w", err)
	}
	return secret, nil
}

func (s *desktopStore) SetTOTPSecret(secret string) error {
	if !s.open {
		return ErrStoreUnavailable
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := keyring.Set(serviceName, "totp_secret", secret); err != nil {
		return fmt.Errorf("set totp_secret: %w", err)
	}
	return nil
}

func (s *desktopStore) Delete() error {
	if !s.open {
		return ErrStoreUnavailable
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	var firstErr error
	for _, id := range []string{"username", "password", "totp_secret"} {
		if err := keyring.Delete(serviceName, id); err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("delete %s: %w", id, err)
			}
		}
	}
	return firstErr
}

func (s *desktopStore) Close() {}
