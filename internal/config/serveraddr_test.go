//go:build !container

package config

import "testing"

func TestDefaultServerAddr(t *testing.T) {
	if got := DefaultServerAddr(); got != "127.0.0.1" {
		t.Errorf("DefaultServerAddr() = %q, want %q", got, "127.0.0.1")
	}
}
