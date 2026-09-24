// Package credentialstore provides compile-time credential storage backend
// selection via Go build tags. Container builds use env-var passthrough;
// desktop builds use the OS keyring.
//
// The interface is shared across all builds. Implementations are selected
// at compile time by the "container" build tag.
package credentialstore

import (
	"errors"
	"sync"
)

var (
	// ErrNotFound is returned when a credential is not present in the store.
	ErrNotFound = errors.New("credential not found")

	// ErrStoreUnavailable is returned when the credential store cannot be
	// opened or accessed (desktop builds only, e.g. no keyring backend).
	ErrStoreUnavailable = errors.New("credential store unavailable")

	once         sync.Once
	defaultStore Store
)

// ResetForTesting clears the singleton so subsequent tests start fresh.
// Only used in tests.
func ResetForTesting() {
	once = sync.Once{}
	defaultStore = nil
}

// Store abstracts credential persistence. Implementations are build-tagged:
//   - container: reads/writes env vars (passthrough, no persistence)
//   - !container: uses OS keyring (desktop)
type Store interface {
	GetUsername() (string, error)
	SetUsername(username string) error
	GetPassword() (string, error)
	SetPassword(password string) error
	GetTOTPSecret() (string, error)
	SetTOTPSecret(secret string) error
	Delete() error
	Close()
	IsOpen() bool
}
