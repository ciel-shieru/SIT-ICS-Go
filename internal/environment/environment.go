// Package environment provides compile-time container detection via Go build
// tags. The presence of the "container" build tag causes IsContainerized to
// return true; its absence causes it to return false.
//
// This replaces the previous runtime cgroup/.dockerenv-based detection with
// a compile-time gate, ensuring that sandbox disablement is decided at build
// time rather than at runtime.
//
// The result is not a security boundary. It provides best-effort detection
// suitable for operational decisions only.
//
// IsContainerized() returns true when built with -tags container (Docker
// images). It returns false for all other builds (desktop binaries, CI
// binaries, release binaries).
//
// The result is cached after the first call via sync.Once.
package environment

import "sync"

var (
	containerized bool
	once          sync.Once
)

// IsContainerized reports whether the current binary was built with the
// "container" build tag. This determines Chromium sandbox behavior:
// true = sandbox disabled (for container deployments),
// false = sandbox enabled (for desktop and server builds).
func IsContainerized() bool {
	once.Do(func() {
		containerized = isContainerizedImpl()
	})
	return containerized
}

// resetForTesting resets the cached containerized result.
// Only for use in tests.
func resetForTesting() {
	once = sync.Once{}
	containerized = false
}
