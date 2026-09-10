package config

import (
	"fmt"
	"time"
)

var validBrowserModes = map[BrowserMode]bool{
	BrowserAuto:   true,
	BrowserSystem: true,
	BrowserRod:    true,
	BrowserRemote: true,
}

func Validate(cfg *Config) error {
	if err := validateBrowserMode(cfg.BrowserMode); err != nil {
		return err
	}
	if err := validateServerPort(cfg.ServerPort); err != nil {
		return err
	}
	if err := validateTimezone(cfg.TZ); err != nil {
		return err
	}
	return nil
}

func validateBrowserMode(mode BrowserMode) error {
	if !validBrowserModes[mode] {
		return fmt.Errorf("invalid browser mode %q: must be one of auto, system, rod, remote", mode)
	}
	return nil
}

func validateServerPort(port int) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf("server port %d out of range (1-65535)", port)
	}
	return nil
}

func validateTimezone(tz string) error {
	if _, err := time.LoadLocation(tz); err != nil {
		return fmt.Errorf("invalid timezone %q: %w", tz, err)
	}
	return nil
}
