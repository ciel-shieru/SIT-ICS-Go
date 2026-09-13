//go:build container

package credentialstore

import (
	"os"
	"sync"
)

// containerStore reads credentials directly from environment variables.
// No persistence is performed.
type containerStore struct {
	mu       sync.RWMutex
	hasCreds bool
}

// NewStore returns the singleton container-backed credential store.
func NewStore() Store {
	once.Do(func() {
		defaultStore = &containerStore{}
	})
	return defaultStore
}

func (s *containerStore) GetUsername() (string, error) {
	s.mu.RLock()
	ok := s.hasCreds
	s.mu.RUnlock()
	if !ok {
		return "", ErrNotFound
	}
	val := os.Getenv("USERNAME")
	if val == "" {
		return "", ErrNotFound
	}
	return val, nil
}

func (s *containerStore) SetUsername(username string) error {
	s.mu.Lock()
	s.hasCreds = true
	s.mu.Unlock()
	os.Setenv("USERNAME", username)
	return nil
}

func (s *containerStore) GetPassword() (string, error) {
	s.mu.RLock()
	ok := s.hasCreds
	s.mu.RUnlock()
	if !ok {
		return "", ErrNotFound
	}
	val := os.Getenv("PASSWORD")
	if val == "" {
		return "", ErrNotFound
	}
	return val, nil
}

func (s *containerStore) SetPassword(password string) error {
	s.mu.Lock()
	s.hasCreds = true
	s.mu.Unlock()
	os.Setenv("PASSWORD", password)
	return nil
}

func (s *containerStore) GetTOTPSecret() (string, error) {
	s.mu.RLock()
	ok := s.hasCreds
	s.mu.RUnlock()
	if !ok {
		return "", ErrNotFound
	}
	val := os.Getenv("TOTP_SECRET")
	if val == "" {
		return "", ErrNotFound
	}
	return val, nil
}

func (s *containerStore) SetTOTPSecret(secret string) error {
	s.mu.Lock()
	s.hasCreds = true
	s.mu.Unlock()
	os.Setenv("TOTP_SECRET", secret)
	return nil
}

func (s *containerStore) Delete() error {
	s.mu.Lock()
	s.hasCreds = false
	s.mu.Unlock()
	os.Unsetenv("USERNAME")
	os.Unsetenv("PASSWORD")
	os.Unsetenv("TOTP_SECRET")
	return nil
}

func (s *containerStore) Close() {}

func (s *containerStore) IsOpen() bool {
	return true
}
