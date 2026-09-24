package totp

import (
	"fmt"
	"time"

	"github.com/pquerna/otp/totp"
)

const defaultPeriod = 30

func Generate(secret string, now time.Time) (string, error) {
	if secret == "" {
		return "", fmt.Errorf("TOTP secret is empty")
	}

	return totp.GenerateCode(secret, now)
}

// GenerateAtOffset generates a TOTP code by trying multiple time steps
// within ±tolerancePeriods of the current time. This handles clock skew and
// processing delays between generating the code and submitting it.
func GenerateAtOffset(secret string, now time.Time, tolerancePeriods int) (string, error) {
	if secret == "" {
		return "", fmt.Errorf("TOTP secret is empty")
	}

	currentStep := uint64(now.Unix()) / defaultPeriod

	for offset := -tolerancePeriods; offset <= tolerancePeriods; offset++ {
		step := currentStep + uint64(offset)
		if step == 0 && offset < 0 {
			continue
		}
		timestamp := time.Unix(int64(step*defaultPeriod), 0)
		code, err := totp.GenerateCode(secret, timestamp)
		if err != nil {
			continue
		}
		return code, nil
	}

	return "", fmt.Errorf("failed to generate TOTP code for any time step in range")
}
