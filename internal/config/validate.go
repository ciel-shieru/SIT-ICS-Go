package config

import (
	"fmt"
	"net"
	"strings"
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
	if err := validateServerAddr(cfg.ServerAddr); err != nil {
		return err
	}
	if err := validateTimezone(cfg.TZ); err != nil {
		return err
	}
	if err := validateServerTrustedProxies(cfg.ServerTrustedProxies); err != nil {
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

func validateServerAddr(addr string) error {
	if addr == "" {
		return nil // empty = use build-tagged default
	}
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		// addr has no port component; it's a bare IP or empty
		if net.ParseIP(addr) == nil {
			return fmt.Errorf("invalid server address %q: not a valid IP address", addr)
		}
	} else {
		if net.ParseIP(host) == nil {
			return fmt.Errorf("invalid server address %q: host part is not a valid IP address", addr)
		}
	}
	return nil
}

func validateServerTrustedProxies(s string) error {
	if s == "" {
		return nil
	}
	for _, entry := range strings.Split(s, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		if strings.Contains(entry, "/") {
			if _, _, err := net.ParseCIDR(entry); err != nil {
				return fmt.Errorf("invalid trusted proxy %q: %w", entry, err)
			}
		} else {
			if net.ParseIP(entry) == nil {
				return fmt.Errorf("invalid trusted proxy %q: not a valid IP address", entry)
			}
		}
	}
	return nil
}
