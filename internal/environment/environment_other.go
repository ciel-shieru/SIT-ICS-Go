//go:build !linux

package environment

import "sync"

func detectWithFS(_ filesystem) bool { return false }

// fs is the filesystem implementation used for detection.
// In production, it is realFS{}. In tests, it can be replaced.
var fs filesystem = realFS{}

func detectContainerizedWithFS(_ filesystem) bool {
	return false
}

var (
	containerized bool
	once          *sync.Once
)

func init() {
	once = &sync.Once{}
}

// IsContainerized reports whether the current process appears to be running
// inside a container deployment. The result is cached after the first call.
// On non-Linux platforms, this always returns false.
func IsContainerized() bool {
	once.Do(func() {
		containerized = detectContainerizedWithFS(fs)
	})
	return containerized
}

// resetForTesting resets the cached containerized result.
// Only for use in tests.
func resetForTesting() {
	once = &sync.Once{}
	containerized = false
}
