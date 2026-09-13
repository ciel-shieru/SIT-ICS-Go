//go:build container

package credprompt

import "github.com/ciel-shieru/sit-ics-go/internal/config"

// PromptIfNeeded is a no-op for container builds. Container builds source
// credentials from environment variables, not interactive prompts.
func PromptIfNeeded(_ *config.Config) error {
	return nil
}
