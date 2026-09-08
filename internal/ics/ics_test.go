package ics

import (
	"testing"
	"time"

	"github.com/ciel-shieru/sit-ics-go/internal/calendar"
)

func TestEscapeText(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "normal text",
			input: "Hello World",
			want:  "Hello World",
		},
		{
			name:  "text with semicolon",
			input: "Hello; World",
			want:  "Hello\\; World",
		},
		{
			name:  "text with comma",
			input: "Hello, World",
			want:  "Hello\\, World",
		},
		{
			name:  "text with newline",
			input: "Hello\nWorld",
			want:  "Hello\\nWorld",
		},
		{
			name:  "text with backslash",
			input: "Hello\\World",
			want:  "Hello\\\\World",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EscapeText(tt.input)
			if got != tt.want {
				t.Errorf("EscapeText() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGenerateUID(t *testing.T) {
	event1 := calendar.Event{
		DTStart: time.Date(2026, 9, 7, 9, 0, 0, 0, time.UTC),
		DTEnd:   time.Date(2026, 9, 7, 11, 0, 0, 0, time.UTC),
		Summary: "ABC 0002 - ALL (Lecture)",
		Location: "W1-05-07",
	}

	event2 := calendar.Event{
		DTStart: time.Date(2026, 9, 7, 9, 0, 0, 0, time.UTC),
		DTEnd:   time.Date(2026, 9, 7, 11, 0, 0, 0, time.UTC),
		Summary: "ABC 0002 - ALL (Lecture)",
		Location: "W1-05-07",
	}

	uid1 := GenerateUID(event1)
	uid2 := GenerateUID(event2)

	if uid1 != uid2 {
		t.Errorf("GenerateUID() not deterministic: %s != %s", uid1, uid2)
	}

	if len(uid1) == 0 {
		t.Error("GenerateUID() returned empty UID")
	}
}

func TestGenerateUIDDifferent(t *testing.T) {
	event1 := calendar.Event{
		DTStart: time.Date(2026, 9, 7, 9, 0, 0, 0, time.UTC),
		DTEnd:   time.Date(2026, 9, 7, 11, 0, 0, 0, time.UTC),
		Summary: "ABC 0002 - ALL (Lecture)",
		Location: "W1-05-07",
	}

	event2 := calendar.Event{
		DTStart: time.Date(2026, 9, 7, 14, 0, 0, 0, time.UTC),
		DTEnd:   time.Date(2026, 9, 7, 16, 0, 0, 0, time.UTC),
		Summary: "ABC 0002 - ALL (Tutorial)",
		Location: "W1-05-08",
	}

	uid1 := GenerateUID(event1)
	uid2 := GenerateUID(event2)

	if uid1 == uid2 {
		t.Errorf("GenerateUID() returned same UID for different events: %s", uid1)
	}
}
