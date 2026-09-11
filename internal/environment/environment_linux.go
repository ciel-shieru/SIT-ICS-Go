//go:build linux

package environment

import (
	"strings"
	"sync"
)

var containerMarkers = []string{
	"kubepods",
	"docker",
	"containerd",
	"libpod",
	"crio",
}

func detectWithFS(fs filesystem) bool {
	if checkCgroup(fs) {
		return true
	}

	if val, ok := fs.LookupEnv("KUBERNETES_SERVICE_HOST"); ok && val != "" {
		return true
	}

	if _, err := fs.Stat("/.dockerenv"); err == nil {
		return true
	}

	if _, err := fs.Stat("/run/.containerenv"); err == nil {
		return true
	}

	if _, err := fs.Stat("/var/run/secrets/kubernetes.io/serviceaccount/namespace"); err == nil {
		return true
	}

	return false
}

func checkCgroup(fs filesystem) bool {
	data, err := fs.ReadFile("/proc/self/cgroup")
	if err != nil {
		return false
	}

	for _, line := range strings.Split(string(data), "\n") {
		for _, marker := range containerMarkers {
			if strings.Contains(line, marker) {
				return true
			}
		}
	}

	return false
}

// fs is the filesystem implementation used for detection.
// In production, it is realFS{}. In tests, it can be replaced.
var fs filesystem = realFS{}

func detectContainerizedWithFS(fs filesystem) bool {
	return detectWithFS(fs)
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
