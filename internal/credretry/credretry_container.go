//go:build container

package credretry

import "github.com/ciel-shieru/sit-ics-go/internal/config"

func PromptIfAuthFailed(cfg *config.Config, err error) error {
	return err // no-op on container builds
}
