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
