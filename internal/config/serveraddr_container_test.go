//go:build container

package config

import "testing"

func TestDefaultServerAddr(t *testing.T) {
	if got := DefaultServerAddr(); got != "0.0.0.0" {
		t.Errorf("DefaultServerAddr() = %q, want %q", got, "0.0.0.0")
	}
}
