package config

import (
	"testing"
)

func TestValidate_BrowserMode(t *testing.T) {
	tests := []struct {
		name    string
		mode    BrowserMode
		wantErr bool
	}{
		{"auto", BrowserAuto, false},
		{"system", BrowserSystem, false},
		{"rod", BrowserRod, false},
		{"remote", BrowserRemote, false},
		{"invalid", BrowserMode("foobar"), true},
		{"empty", BrowserMode(""), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{BrowserMode: tt.mode, TZ: "Asia/Singapore", ServerPort: 8080}
			err := Validate(cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidate_ServerPort(t *testing.T) {
	tests := []struct {
		name    string
		port    int
		wantErr bool
	}{
		{"valid", 8080, false},
		{"min", 1, false},
		{"max", 65535, false},
		{"too_low", 0, true},
		{"too_high", 65536, true},
		{"negative", -1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{BrowserMode: BrowserAuto, TZ: "Asia/Singapore", ServerPort: tt.port}
			err := Validate(cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidate_Timezone(t *testing.T) {
	tests := []struct {
		name    string
		tz      string
		wantErr bool
	}{
		{"singapore", "Asia/Singapore", false},
		{"utc", "UTC", false},
		{"london", "Europe/London", false},
		{"new_york", "America/New_York", false},
		{"invalid", "NotA/Timezone", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{BrowserMode: BrowserAuto, TZ: tt.tz, ServerPort: 8080}
			err := Validate(cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidate_ServerAddr(t *testing.T) {
	tests := []struct {
		name    string
		addr    string
		wantErr bool
	}{
		{"empty", "", false},
		{"loopback", "127.0.0.1", false},
		{"all_interfaces", "0.0.0.0", false},
		{"ipv6_loopback", "::1", false},
		{"ipv6_all", "::", false},
		{"with_port", "127.0.0.1:8080", false},
		{"ipv6_with_port", "[::1]:8080", false},
		{"invalid_ip", "not_an_ip", true},
		{"hostname", "localhost", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{BrowserMode: BrowserAuto, TZ: "Asia/Singapore", ServerPort: 8080, ServerAddr: tt.addr}
			err := Validate(cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidate_ValidConfig(t *testing.T) {
	cfg := &Config{
		BrowserMode: BrowserAuto,
		TZ:          "Asia/Singapore",
		ServerPort:  8080,
	}
	if err := Validate(cfg); err != nil {
		t.Errorf("Validate() = %v, want nil", err)
	}
}

func TestValidateServerTrustedProxies(t *testing.T) {
	tests := []struct {
		name    string
		proxies string
		wantErr bool
	}{
		{"empty", "", false},
		{"single_ipv4", "10.0.0.1", false},
		{"single_ipv6", "::1", false},
		{"ipv4_cidr", "192.168.0.0/16", false},
		{"ipv6_cidr", "2001:db8::/32", false},
		{"mixed", "10.0.0.1,192.168.0.0/16,::1,2001:db8::/32", false},
		{"not_an_ip", "not_an_ip", true},
		{"ipv6_too_long", "192.168.1.0/33", true},
		{"invalid_cidr", "192.168.1.0/abc", true},
		{"mixed_valid_invalid", "10.0.0.1,not_an_ip", true},
		{"whitespace_trimmed", "  10.0.0.1 , 192.168.0.0/16  ", false},
		{"bare_ipv6", "::ffff:192.0.2.1", false},
		{"single_entry_with_spaces", "  10.0.0.1  ", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				BrowserMode:        BrowserAuto,
				TZ:                 "Asia/Singapore",
				ServerPort:         8080,
				ServerTrustedProxies: tt.proxies,
			}
			err := Validate(cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
