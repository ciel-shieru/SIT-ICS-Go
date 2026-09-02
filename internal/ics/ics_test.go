package ics

import (
	"strings"
	"testing"
	"time"
)

func TestWrite(t *testing.T) {
	events := []Event{
		{
			UID:         "test-uid-1",
			DTStart:     time.Date(2026, 9, 7, 9, 0, 0, 0, time.UTC),
			DTEnd:       time.Date(2026, 9, 7, 11, 0, 0, 0, time.UTC),
			CourseCode:  "ABC 0002",
			Summary:     "ABC 0002 - ALL (Lecture)",
			Location:    "W1-05-07",
			Description: "Course: ABC 0002\nSection: ALL\nType: Lecture",
		},
	}

	data, err := Write(events, "Asia/Singapore")
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	content := string(data)

	if !strings.Contains(content, "BEGIN:VCALENDAR") {
		t.Error("Write() missing BEGIN:VCALENDAR")
	}
	if !strings.Contains(content, "END:VCALENDAR") {
		t.Error("Write() missing END:VCALENDAR")
	}
	if !strings.Contains(content, "BEGIN:VEVENT") {
		t.Error("Write() missing BEGIN:VEVENT")
	}
	if !strings.Contains(content, "END:VEVENT") {
		t.Error("Write() missing END:VEVENT")
	}
	if !strings.Contains(content, "UID:test-uid-1") {
		t.Error("Write() missing UID")
	}
	if !strings.Contains(content, "SUMMARY:ABC 0002 - ALL (Lecture)") {
		t.Error("Write() missing SUMMARY")
	}
	if !strings.Contains(content, "LOCATION:W1-05-07") {
		t.Error("Write() missing LOCATION")
	}
}

func TestWriteEmpty(t *testing.T) {
	data, err := Write([]Event{}, "Asia/Singapore")
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "BEGIN:VCALENDAR") {
		t.Error("Write() missing BEGIN:VCALENDAR")
	}
	if !strings.Contains(content, "END:VCALENDAR") {
		t.Error("Write() missing END:VCALENDAR")
	}
	if strings.Contains(content, "BEGIN:VEVENT") {
		t.Error("Write() should not contain VEVENT for empty events")
	}
}

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
			got := escapeText(tt.input)
			if got != tt.want {
				t.Errorf("escapeText() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGenerateUID(t *testing.T) {
	event1 := Event{
		DTStart: time.Date(2026, 9, 7, 9, 0, 0, 0, time.UTC),
		DTEnd:   time.Date(2026, 9, 7, 11, 0, 0, 0, time.UTC),
		Summary: "ABC 0002 - ALL (Lecture)",
		Location: "W1-05-07",
	}

	event2 := Event{
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
	event1 := Event{
		DTStart: time.Date(2026, 9, 7, 9, 0, 0, 0, time.UTC),
		DTEnd:   time.Date(2026, 9, 7, 11, 0, 0, 0, time.UTC),
		Summary: "ABC 0002 - ALL (Lecture)",
		Location: "W1-05-07",
	}

	event2 := Event{
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
