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
