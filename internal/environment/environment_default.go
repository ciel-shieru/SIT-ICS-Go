//go:build !container

package environment

// isContainerizedImpl always returns false when NOT built with -tags container.
func isContainerizedImpl() bool {
	return false
}
