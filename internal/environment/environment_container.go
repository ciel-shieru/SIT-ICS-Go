//go:build container

package environment

// isContainerizedImpl always returns true when built with -tags container.
func isContainerizedImpl() bool {
	return true
}
