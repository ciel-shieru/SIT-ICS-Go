//go:build container

package config

const defaultServerAddr = "0.0.0.0"

// DefaultServerAddr returns the compile-time default bind address for container builds.
func DefaultServerAddr() string {
	return defaultServerAddr
}
