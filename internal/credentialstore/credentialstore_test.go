package credentialstore

import (
	"os"
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
