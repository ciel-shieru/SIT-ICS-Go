// Package environment detects whether the current process appears to be running
// in a container deployment.
//
// Detection is based on locally observable runtime and deployment signals:
// cgroup hierarchy markers, Kubernetes environment variables, container marker
// files, and Kubernetes ServiceAccount namespace files.
//
// The result is intentionally a boolean: true means the process is running in a
// container deployment, false means it is running directly on the host or in a
// non-container environment. Absence of one particular signal does not imply a
// non-container environment — multiple independent signals are checked and any
// single positive match is sufficient.
//
// The result is not a security boundary. It provides best-effort detection suitable
// for operational decisions only.
//
// On non-Linux platforms, IsContainerized always returns false.
//
// The result is cached after the first call via sync.Once.
package environment

import (
	"os"
)

// filesystem abstracts OS operations for testability.
type filesystem interface {
	Stat(name string) (os.FileInfo, error)
	ReadFile(name string) ([]byte, error)
	LookupEnv(key string) (string, bool)
}

// realFS implements filesystem using the real OS.
type realFS struct{}

func (realFS) Stat(name string) (os.FileInfo, error) { return os.Stat(name) }
func (realFS) ReadFile(name string) ([]byte, error)  { return os.ReadFile(name) }
func (realFS) LookupEnv(key string) (string, bool)   { return os.LookupEnv(key) }
