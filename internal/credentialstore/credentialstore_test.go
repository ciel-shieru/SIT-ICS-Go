package credentialstore

import (
	"fmt"
	"os"
	"sync"
	"testing"
)

func TestNewStore(t *testing.T) {
	defer resetForTesting()
	store := NewStore()
	if store == nil {
		t.Fatal("NewStore() returned nil")
	}
	store.Close()
}

func TestGetNotFound(t *testing.T) {
	defer resetForTesting()
	store := NewStore()
	defer store.Close()

	os.Unsetenv("USERNAME")
	os.Unsetenv("PASSWORD")
	os.Unsetenv("TOTP_SECRET")

	username, err := store.GetUsername()
	if err != ErrNotFound && err != ErrStoreUnavailable {
		t.Errorf("GetUsername() error = %v, want %v or %v", err, ErrNotFound, ErrStoreUnavailable)
	}
	if err == nil && username != "" {
		t.Error("GetUsername() returned non-empty string without error")
	}

	password, err := store.GetPassword()
	if err != ErrNotFound && err != ErrStoreUnavailable {
		t.Errorf("GetPassword() error = %v, want %v or %v", err, ErrNotFound, ErrStoreUnavailable)
	}
	if err == nil && password != "" {
		t.Error("GetPassword() returned non-empty string without error")
	}

	totp, err := store.GetTOTPSecret()
	if err != ErrNotFound && err != ErrStoreUnavailable {
		t.Errorf("GetTOTPSecret() error = %v, want %v or %v", err, ErrNotFound, ErrStoreUnavailable)
	}
	if err == nil && totp != "" {
		t.Error("GetTOTPSecret() returned non-empty string without error")
	}
}

func TestSetAndGet(t *testing.T) {
	defer resetForTesting()
	store := NewStore()
	defer store.Close()

	testUsername := "test-user"
	testPassword := "test-password"
	testTOTP := "JBSWY3DPEHPK3PXP"

	if err := store.SetUsername(testUsername); err != nil {
		t.Skipf("SetUsername skipped: %v", err)
	}
	if err := store.SetPassword(testPassword); err != nil {
		t.Skipf("SetPassword skipped: %v", err)
	}
	if err := store.SetTOTPSecret(testTOTP); err != nil {
		t.Skipf("SetTOTPSecret skipped: %v", err)
	}

	gotUsername, err := store.GetUsername()
	if err != nil {
		t.Fatalf("GetUsername after SetUsername: %v", err)
	}
	if gotUsername != testUsername {
		t.Errorf("GetUsername() = %q, want %q", gotUsername, testUsername)
	}

	gotPassword, err := store.GetPassword()
	if err != nil {
		t.Fatalf("GetPassword after SetPassword: %v", err)
	}
	if gotPassword != testPassword {
		t.Errorf("GetPassword() = %q, want %q", gotPassword, testPassword)
	}

	gotTOTP, err := store.GetTOTPSecret()
	if err != nil {
		t.Fatalf("GetTOTPSecret after SetTOTPSecret: %v", err)
	}
	if gotTOTP != testTOTP {
		t.Errorf("GetTOTPSecret() = %q, want %q", gotTOTP, testTOTP)
	}

	if err := store.Delete(); err != nil {
		t.Logf("Delete returned non-nil (acceptable in CI): %v", err)
	}
}

func TestDelete(t *testing.T) {
	defer resetForTesting()
	store := NewStore()
	defer store.Close()

	testUsername := "test-delete-user"

	if err := store.SetUsername(testUsername); err != nil {
		t.Skipf("SetUsername skipped: %v", err)
	}

	got, err := store.GetUsername()
	if err != nil {
		t.Fatalf("GetUsername before Delete: %v", err)
	}
	if got != testUsername {
		t.Errorf("Before delete: GetUsername() = %q, want %q", got, testUsername)
	}

	if err := store.Delete(); err != nil {
		t.Logf("Delete returned non-nil (acceptable in CI): %v", err)
	}

	_, err = store.GetUsername()
	if err == nil {
		t.Error("GetUsername after Delete() should return error")
	}
}

func TestConcurrentAccess(t *testing.T) {
	defer resetForTesting()
	store := NewStore()
	defer store.Close()

	const goroutines = 10
	const iterations = 100

	var wg sync.WaitGroup
	errs := make(chan error, goroutines*iterations*3)

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				val := fmt.Sprintf("user-%d", n)
				if err := store.SetUsername(val); err != nil && err != ErrStoreUnavailable {
					errs <- fmt.Errorf("concurrent SetUsername: %w", err)
				}
			}
		}(i)
	}

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				if _, err := store.GetUsername(); err != nil && err != ErrNotFound && err != ErrStoreUnavailable {
					errs <- fmt.Errorf("concurrent GetUsername: %w", err)
				}
			}
		}()
	}

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				if err := store.Delete(); err != nil && err != ErrNotFound && err != ErrStoreUnavailable {
					errs <- fmt.Errorf("concurrent Delete: %w", err)
				}
			}
		}()
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		t.Errorf("concurrent access error: %v", err)
	}
}

func TestDesktopUnavailable(t *testing.T) {
	// Create a desktopStore directly with open=false to simulate unavailable keyring.
	store := &desktopStore{open: false}

	username, err := store.GetUsername()
	if err != ErrStoreUnavailable {
		t.Errorf("GetUsername() on closed store: error = %v, want %v", err, ErrStoreUnavailable)
	}
	if username != "" {
		t.Error("GetUsername() on closed store should return empty string")
	}

	_, err = store.GetPassword()
	if err != ErrStoreUnavailable {
		t.Errorf("GetPassword() on closed store: error = %v, want %v", err, ErrStoreUnavailable)
	}

	_, err = store.GetTOTPSecret()
	if err != ErrStoreUnavailable {
		t.Errorf("GetTOTPSecret() on closed store: error = %v, want %v", err, ErrStoreUnavailable)
	}

	err = store.SetUsername("test")
	if err != ErrStoreUnavailable {
		t.Errorf("SetUsername() on closed store: error = %v, want %v", err, ErrStoreUnavailable)
	}

	err = store.SetPassword("test")
	if err != ErrStoreUnavailable {
		t.Errorf("SetPassword() on closed store: error = %v, want %v", err, ErrStoreUnavailable)
	}

	err = store.SetTOTPSecret("test")
	if err != ErrStoreUnavailable {
		t.Errorf("SetTOTPSecret() on closed store: error = %v, want %v", err, ErrStoreUnavailable)
	}

	err = store.Delete()
	if err != ErrStoreUnavailable {
		t.Errorf("Delete() on closed store: error = %v, want %v", err, ErrStoreUnavailable)
	}
}

func TestContainerHasCredsGuard(t *testing.T) {
	defer resetForTesting()

	store := NewStore()
	defer store.Close()

	os.Unsetenv("USERNAME")
	os.Unsetenv("PASSWORD")
	os.Unsetenv("TOTP_SECRET")

	username, err := store.GetUsername()
	if err != ErrNotFound && err != ErrStoreUnavailable {
		t.Errorf("GetUsername() before SetUsername: error = %v, want %v or %v", err, ErrNotFound, ErrStoreUnavailable)
	}
	if err == nil && username != "" {
		t.Error("GetUsername() before SetUsername returned non-empty value")
	}

	password, err := store.GetPassword()
	if err != ErrNotFound && err != ErrStoreUnavailable {
		t.Errorf("GetPassword() before SetPassword: error = %v, want %v or %v", err, ErrNotFound, ErrStoreUnavailable)
	}
	if err == nil && password != "" {
		t.Error("GetPassword() before SetPassword returned non-empty value")
	}

	totp, err := store.GetTOTPSecret()
	if err != ErrNotFound && err != ErrStoreUnavailable {
		t.Errorf("GetTOTPSecret() before SetTOTPSecret: error = %v, want %v or %v", err, ErrNotFound, ErrStoreUnavailable)
	}
	if err == nil && totp != "" {
		t.Error("GetTOTPSecret() before SetTOTPSecret returned non-empty value")
	}
}
