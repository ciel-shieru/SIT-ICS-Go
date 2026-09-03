package totp

import (
	"testing"
	"time"
)

func TestGenerate(t *testing.T) {
	tests := []struct {
		name     string
		secret   string
		now      time.Time
		wantLen  int
		wantErr  bool
	}{
		{
			name:    "valid secret",
			secret:  "JBSWY3DPEHPK3PXP",
			now:     time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC),
			wantLen: 6,
			wantErr: false,
		},
		{
			name:    "empty secret",
			secret:  "",
			now:     time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC),
			wantLen: 0,
			wantErr: true,
		},
		{
			name:    "different time",
			secret:  "JBSWY3DPEHPK3PXP",
			now:     time.Date(2026, 9, 1, 12, 0, 30, 0, time.UTC),
			wantLen: 6,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Generate(tt.secret, tt.now)
			if (err != nil) != tt.wantErr {
				t.Errorf("Generate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && len(got) != tt.wantLen {
				t.Errorf("Generate() length = %d, want %d", len(got), tt.wantLen)
			}
			if !tt.wantErr {
				for _, c := range got {
					if c < '0' || c > '9' {
						t.Errorf("Generate() contains non-digit character: %c", c)
					}
				}
			}
		})
	}
}

func TestGenerateDeterministic(t *testing.T) {
	secret := "JBSWY3DPEHPK3PXP"
	now := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)

	got1, err := Generate(secret, now)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	got2, err := Generate(secret, now)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if got1 != got2 {
		t.Errorf("Generate() not deterministic: got1=%s, got2=%s", got1, got2)
	}
}

func TestGenerateDifferentTimes(t *testing.T) {
	secret := "JBSWY3DPEHPK3PXP"
	now1 := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	now2 := time.Date(2026, 9, 1, 12, 1, 0, 0, time.UTC)

	got1, err := Generate(secret, now1)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	got2, err := Generate(secret, now2)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if got1 == got2 {
		t.Errorf("Generate() returned same code for different times: %s", got1)
	}
}

func TestGenerateWithTolerance(t *testing.T) {
	secret := "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ"

	tests := []struct {
		name string
		// time offset in seconds from the expected time step
		timeOffset int64
		want       string
	}{
		{
			name:       "exact time",
			timeOffset: 0,
			want:       "287082",
		},
		{
			name:       "-1 period",
			timeOffset: -30,
			want:       "094451",
		},
		{
			name:       "+1 period",
			timeOffset: 30,
			want:       "287082",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			now := time.Unix(59, 0).UTC().Add(time.Duration(tt.timeOffset) * time.Second)
			got, err := GenerateWithTolerance(secret, now, 1)
			if err != nil {
				t.Fatalf("GenerateWithTolerance() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("GenerateWithTolerance() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestGenerateWithToleranceOutsideRange(t *testing.T) {
	secret := "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ"
	// time=59, step=1. Offset by 2 periods (60s) which is outside tolerance of 1.
	now := time.Unix(59-60, 0).UTC()
	got, err := GenerateWithTolerance(secret, now, 1)
	if err != nil {
		t.Fatalf("GenerateWithTolerance() error = %v", err)
	}
	// Should return the code for step=0 (the closest valid step), not fail
	if got == "" {
		t.Error("GenerateWithTolerance() returned empty string")
	}
}

// TestGenerateRFC6238 verifies against RFC 6238 Section B test vectors.
// Secret is "12345678901234567890" (ASCII), which is "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ" in base32.
// RFC 6238 Appendix B uses 8-digit codes; we verify the last 6 digits match our 6-digit output.
func TestGenerateRFC6238(t *testing.T) {
	secret := "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ"

	tests := []struct {
		name   string
		unixTs int64
		want   string
	}{
		{
			name:   "time=59",
			unixTs: 59,
			want:   "287082",
		},
		{
			name:   "time=1111111109",
			unixTs: 1111111109,
			want:   "081804",
		},
		{
			name:   "time=1111111111",
			unixTs: 1111111111,
			want:   "050471",
		},
		{
			name:   "time=1234567890",
			unixTs: 1234567890,
			want:   "005924",
		},
		{
			name:   "time=2000000000",
			unixTs: 2000000000,
			want:   "279037",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Generate(secret, time.Unix(tt.unixTs, 0).UTC())
			if err != nil {
				t.Fatalf("Generate() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("Generate() = %s, want %s", got, tt.want)
			}
		})
	}
}
