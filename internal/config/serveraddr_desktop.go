//go:build !container

package config

const defaultServerAddr = "127.0.0.1"

// DefaultServerAddr returns the compile-time default bind address for desktop builds.
func DefaultServerAddr() string {
	return defaultServerAddr
}
