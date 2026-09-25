package calendar

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestParseICS_ParsesCalendarEventID(t *testing.T) {
	ics := strings.Join([]string{
		"BEGIN:VCALENDAR",
		"VERSION:2.0",
		"PRODID:-//SIT Timetable//EN",
		"CALSCALE:GREGORIAN",
		"METHOD:PUBLISH",
		"REFRESH-INTERVAL;VALUE=DURATION:PT1H",
		"BEGIN:VEVENT",
		"UID:test-uid-001",
		"DTSTART;TZID=Asia/Singapore:20260915T100000",
		"DTEND;TZID=Asia/Singapore:20260915T110000",
		"SUMMARY:Test Event",
		"LOCATION:Room 101",
		"X-CalendarEventId:99001",
		"END:VEVENT",
		"END:VCALENDAR",
	}, "\r\n") + "\r\n"

	loc, _ := time.LoadLocation("Asia/Singapore")
	events := parseICS([]byte(ics), loc)

	if len(events) != 1 {
		t.Fatalf("Expected 1 event, got %d", len(events))
	}
	if events[0].CalendarEventID != 99001 {
		t.Errorf("CalendarEventID = %d, want 99001", events[0].CalendarEventID)
	}
}

func TestParseICS_ParsesQuizID(t *testing.T) {
	ics := strings.Join([]string{
		"BEGIN:VCALENDAR",
		"VERSION:2.0",
		"PRODID:-//SIT Timetable//EN",
		"CALSCALE:GREGORIAN",
		"METHOD:PUBLISH",
		"REFRESH-INTERVAL;VALUE=DURATION:PT1H",
		"BEGIN:VEVENT",
		"UID:test-uid-002",
		"DTSTART;TZID=Asia/Singapore:20260915T100000",
		"DTEND;TZID=Asia/Singapore:20260915T110000",
		"SUMMARY:Quiz Event",
		"LOCATION:Online",
		"X-QuizId:99010",
		"END:VEVENT",
		"END:VCALENDAR",
	}, "\r\n") + "\r\n"

	loc, _ := time.LoadLocation("Asia/Singapore")
	events := parseICS([]byte(ics), loc)

	if len(events) != 1 {
		t.Fatalf("Expected 1 event, got %d", len(events))
	}
	if events[0].QuizID != 99010 {
		t.Errorf("QuizID = %d, want 99010", events[0].QuizID)
	}
}

func TestParseICS_ParsesBothIDs(t *testing.T) {
	ics := strings.Join([]string{
		"BEGIN:VCALENDAR",
		"VERSION:2.0",
		"PRODID:-//SIT Timetable//EN",
		"CALSCALE:GREGORIAN",
		"METHOD:PUBLISH",
		"REFRESH-INTERVAL;VALUE=DURATION:PT1H",
		"BEGIN:VEVENT",
		"UID:test-uid-003",
		"DTSTART;TZID=Asia/Singapore:20260915T100000",
		"DTEND;TZID=Asia/Singapore:20260915T110000",
		"SUMMARY:Quiz Calendar Event",
		"LOCATION:Zoom Online Meeting",
		"X-CalendarEventId:99020",
		"X-QuizId:99021",
		"END:VEVENT",
		"END:VCALENDAR",
	}, "\r\n") + "\r\n"

	loc, _ := time.LoadLocation("Asia/Singapore")
	events := parseICS([]byte(ics), loc)

	if len(events) != 1 {
		t.Fatalf("Expected 1 event, got %d", len(events))
	}
	if events[0].CalendarEventID != 99020 {
		t.Errorf("CalendarEventID = %d, want 99020", events[0].CalendarEventID)
	}
	if events[0].QuizID != 99021 {
		t.Errorf("QuizID = %d, want 99021", events[0].QuizID)
	}
}

func TestParseICS_ZeroIDsWhenAbsent(t *testing.T) {
	ics := strings.Join([]string{
		"BEGIN:VCALENDAR",
		"VERSION:2.0",
		"PRODID:-//SIT Timetable//EN",
		"CALSCALE:GREGORIAN",
		"METHOD:PUBLISH",
		"REFRESH-INTERVAL;VALUE=DURATION:PT1H",
		"BEGIN:VEVENT",
		"UID:test-uid-004",
		"DTSTART;TZID=Asia/Singapore:20260915T100000",
		"DTEND;TZID=Asia/Singapore:20260915T110000",
		"SUMMARY:Regular Event",
		"LOCATION:Room 201",
		"END:VEVENT",
		"END:VCALENDAR",
	}, "\r\n") + "\r\n"

	loc, _ := time.LoadLocation("Asia/Singapore")
	events := parseICS([]byte(ics), loc)

	if len(events) != 1 {
		t.Fatalf("Expected 1 event, got %d", len(events))
	}
	if events[0].CalendarEventID != 0 {
		t.Errorf("CalendarEventID = %d, want 0", events[0].CalendarEventID)
	}
	if events[0].QuizID != 0 {
		t.Errorf("QuizID = %d, want 0", events[0].QuizID)
	}
}

func TestParseICS_MultipleEventsWithMixedIDs(t *testing.T) {
	ics := strings.Join([]string{
		"BEGIN:VCALENDAR",
		"VERSION:2.0",
		"PRODID:-//SIT Timetable//EN",
		"CALSCALE:GREGORIAN",
		"METHOD:PUBLISH",
		"REFRESH-INTERVAL;VALUE=DURATION:PT1H",
		"BEGIN:VEVENT",
		"UID:test-uid-005",
		"DTSTART;TZID=Asia/Singapore:20260915T100000",
		"DTEND;TZID=Asia/Singapore:20260915T110000",
		"SUMMARY:Event with both IDs",
		"LOCATION:Room 101",
		"X-CalendarEventId:99030",
		"X-QuizId:99031",
		"END:VEVENT",
		"BEGIN:VEVENT",
		"UID:test-uid-006",
		"DTSTART;TZID=Asia/Singapore:20260915T140000",
		"DTEND;TZID=Asia/Singapore:20260915T150000",
		"SUMMARY:Event with only CalendarEventID",
		"LOCATION:Room 102",
		"X-CalendarEventId:99032",
		"END:VEVENT",
		"BEGIN:VEVENT",
		"UID:test-uid-007",
		"DTSTART;TZID=Asia/Singapore:20260916T100000",
		"DTEND;TZID=Asia/Singapore:20260916T110000",
		"SUMMARY:Event with only QuizID",
		"LOCATION:Room 103",
		"X-QuizId:99033",
		"END:VEVENT",
		"BEGIN:VEVENT",
		"UID:test-uid-008",
		"DTSTART;TZID=Asia/Singapore:20260916T140000",
		"DTEND;TZID=Asia/Singapore:20260916T150000",
		"SUMMARY:Event with no IDs",
		"LOCATION:Room 104",
		"END:VEVENT",
		"END:VCALENDAR",
	}, "\r\n") + "\r\n"

	loc, _ := time.LoadLocation("Asia/Singapore")
	events := parseICS([]byte(ics), loc)

	if len(events) != 4 {
		t.Fatalf("Expected 4 events, got %d", len(events))
	}

	if events[0].CalendarEventID != 99030 || events[0].QuizID != 99031 {
		t.Errorf("Event 0: CalendarEventID=%d, QuizID=%d, want 99030, 99031", events[0].CalendarEventID, events[0].QuizID)
	}
	if events[1].CalendarEventID != 99032 || events[1].QuizID != 0 {
		t.Errorf("Event 1: CalendarEventID=%d, QuizID=%d, want 99032, 0", events[1].CalendarEventID, events[1].QuizID)
	}
	if events[2].CalendarEventID != 0 || events[2].QuizID != 99033 {
		t.Errorf("Event 2: CalendarEventID=%d, QuizID=%d, want 0, 99033", events[2].CalendarEventID, events[2].QuizID)
	}
	if events[3].CalendarEventID != 0 || events[3].QuizID != 0 {
		t.Errorf("Event 3: CalendarEventID=%d, QuizID=%d, want 0, 0", events[3].CalendarEventID, events[3].QuizID)
	}
}

func TestParseICS_InvalidIDValues(t *testing.T) {
	ics := strings.Join([]string{
		"BEGIN:VCALENDAR",
		"VERSION:2.0",
		"PRODID:-//SIT Timetable//EN",
		"CALSCALE:GREGORIAN",
		"METHOD:PUBLISH",
		"REFRESH-INTERVAL;VALUE=DURATION:PT1H",
		"BEGIN:VEVENT",
		"UID:test-uid-009",
		"DTSTART;TZID=Asia/Singapore:20260915T100000",
		"DTEND;TZID=Asia/Singapore:20260915T110000",
		"SUMMARY:Event with invalid IDs",
		"X-CalendarEventId:not-a-number",
		"X-QuizId:also-invalid",
		"END:VEVENT",
		"END:VCALENDAR",
	}, "\r\n") + "\r\n"

	loc, _ := time.LoadLocation("Asia/Singapore")
	events := parseICS([]byte(ics), loc)

	if len(events) != 1 {
		t.Fatalf("Expected 1 event, got %d", len(events))
	}
	if events[0].CalendarEventID != 0 {
		t.Errorf("CalendarEventID = %d, want 0 (invalid value)", events[0].CalendarEventID)
	}
	if events[0].QuizID != 0 {
		t.Errorf("QuizID = %d, want 0 (invalid value)", events[0].QuizID)
	}
}

func TestRoundTrip_CalendarEventIDAndQuizID(t *testing.T) {
	tmpDir := t.TempDir()
	icsPath := filepath.Join(tmpDir, "test.ics")

	loc, _ := time.LoadLocation("Asia/Singapore")

	cache := NewICSCache()

	events := []Event{
		{
			Summary:         "Quiz-associated calendar event",
			Location:        "Zoom Online Meeting",
			DTStart:         time.Date(2026, 9, 15, 10, 0, 0, 0, loc),
			DTEnd:           time.Date(2026, 9, 15, 11, 0, 0, 0, loc),
			CalendarEventID: 99040,
			QuizID:          99041,
			Source:          "calendar",
			OrgUnitCode:     "MOD1001",
		},
		{
			Summary:         "Quiz-only event",
			Location:        "Online",
			DTStart:         time.Date(2026, 9, 15, 14, 0, 0, 0, loc),
			DTEnd:           time.Date(2026, 9, 15, 15, 0, 0, 0, loc),
			CalendarEventID: 0,
			QuizID:          99042,
			Source:          "brightspace-quizzes",
			OrgUnitCode:     "MOD1002",
		},
		{
			Summary:         "Regular event without IDs",
			Location:        "Room 101",
			DTStart:         time.Date(2026, 9, 16, 9, 0, 0, 0, loc),
			DTEnd:           time.Date(2026, 9, 16, 10, 0, 0, 0, loc),
			CalendarEventID: 0,
			QuizID:          0,
			Source:          "",
			OrgUnitCode:     "MOD1003",
		},
	}

	err := cache.Update(events, "Asia/Singapore")
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	err = cache.SaveToFile(icsPath, time.Hour)
	if err != nil {
		t.Fatalf("SaveToFile() error = %v", err)
	}

	newCache := NewICSCache()
	err = newCache.LoadFromFile(icsPath, loc)
	if err != nil {
		t.Fatalf("LoadFromFile() error = %v", err)
	}

	loadedEvents := newCache.events
	if len(loadedEvents) != 3 {
		t.Fatalf("Expected 3 loaded events, got %d", len(loadedEvents))
	}

	if loadedEvents[0].CalendarEventID != 99040 {
		t.Errorf("Event 0 CalendarEventID = %d, want 99040", loadedEvents[0].CalendarEventID)
	}
	if loadedEvents[0].QuizID != 99041 {
		t.Errorf("Event 0 QuizID = %d, want 99041", loadedEvents[0].QuizID)
	}

	if loadedEvents[1].CalendarEventID != 0 {
		t.Errorf("Event 1 CalendarEventID = %d, want 0", loadedEvents[1].CalendarEventID)
	}
	if loadedEvents[1].QuizID != 99042 {
		t.Errorf("Event 1 QuizID = %d, want 99042", loadedEvents[1].QuizID)
	}

	if loadedEvents[2].CalendarEventID != 0 {
		t.Errorf("Event 2 CalendarEventID = %d, want 0", loadedEvents[2].CalendarEventID)
	}
	if loadedEvents[2].QuizID != 0 {
		t.Errorf("Event 2 QuizID = %d, want 0", loadedEvents[2].QuizID)
	}
}

func TestRoundTrip_MultipleSaveLoadCycles(t *testing.T) {
	tmpDir := t.TempDir()
	icsPath := filepath.Join(tmpDir, "test.ics")

	loc, _ := time.LoadLocation("Asia/Singapore")

	cache := NewICSCache()

	events := []Event{
		{
			Summary:         "Persistent event",
			Location:        "Room 101",
			DTStart:         time.Date(2026, 9, 15, 10, 0, 0, 0, loc),
			DTEnd:           time.Date(2026, 9, 15, 11, 0, 0, 0, loc),
			CalendarEventID: 99050,
			QuizID:          99051,
			Source:          "calendar",
			OrgUnitCode:     "MOD1001",
		},
	}

	err := cache.Update(events, "Asia/Singapore")
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	for i := 0; i < 3; i++ {
		err = cache.SaveToFile(icsPath, time.Hour)
		if err != nil {
			t.Fatalf("Cycle %d: SaveToFile() error = %v", i, err)
		}

		newCache := NewICSCache()
		err = newCache.LoadFromFile(icsPath, loc)
		if err != nil {
			t.Fatalf("Cycle %d: LoadFromFile() error = %v", i, err)
		}

		if len(newCache.events) != 1 {
			t.Fatalf("Cycle %d: Expected 1 event, got %d", i, len(newCache.events))
		}

		if newCache.events[0].CalendarEventID != 99050 {
			t.Errorf("Cycle %d: CalendarEventID = %d, want 99050", i, newCache.events[0].CalendarEventID)
		}
		if newCache.events[0].QuizID != 99051 {
			t.Errorf("Cycle %d: QuizID = %d, want 99051", i, newCache.events[0].QuizID)
		}

		cache = newCache
	}
}

func TestParseICS_FileDoesNotExist(t *testing.T) {
	tmpDir := t.TempDir()
	nonExistentPath := filepath.Join(tmpDir, "nonexistent.ics")

	cache := NewICSCache()
	err := cache.LoadFromFile(nonExistentPath, time.UTC)
	if err != nil {
		t.Fatalf("LoadFromFile() should return nil for non-existent file, got error = %v", err)
	}
}

func TestParseIntHelper(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{"valid number", "99001", 99001},
		{"zero", "0", 0},
		{"with whitespace", "  99002  ", 99002},
		{"non-numeric", "not-a-number", 0},
		{"empty string", "", 0},
		{"negative", "-1", -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseInt(tt.input)
			if got != tt.expected {
				t.Errorf("parseInt(%q) = %d, want %d", tt.input, got, tt.expected)
			}
		})
	}
}

func TestWriteAtomic(t *testing.T) {
	tmpDir := t.TempDir()
	testPath := filepath.Join(tmpDir, "atomic.ics")

	data := []byte("BEGIN:VCALENDAR\r\nVERSION:2.0\r\nEND:VCALENDAR\r\n")

	err := writeAtomic(testPath, data)
	if err != nil {
		t.Fatalf("writeAtomic() error = %v", err)
	}

	content, err := os.ReadFile(testPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	if string(content) != string(data) {
		t.Errorf("File content = %q, want %q", string(content), string(data))
	}

	// Verify no .tmp file remains
	_, err = os.Stat(testPath + ".tmp")
	if err == nil {
		t.Error("tmp file should not exist after successful writeAtomic")
	}
}

func TestWriteAtomic_ProducesValidICS(t *testing.T) {
	tmpDir := t.TempDir()
	testPath := filepath.Join(tmpDir, "valid.ics")

	events := []Event{
		{
			Summary:         "Test",
			Location:        "Room 101",
			DTStart:         time.Date(2026, 9, 15, 10, 0, 0, 0, time.UTC),
			DTEnd:           time.Date(2026, 9, 15, 11, 0, 0, 0, time.UTC),
			CalendarEventID: 99060,
			QuizID:          99061,
		},
	}

	cache := NewICSCache()
	err := cache.Update(events, "UTC")
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	err = cache.SaveToFile(testPath, time.Hour)
	if err != nil {
		t.Fatalf("SaveToFile() error = %v", err)
	}

	loc := time.UTC
	loadedCache := NewICSCache()
	err = loadedCache.LoadFromFile(testPath, loc)
	if err != nil {
		t.Fatalf("LoadFromFile() error = %v", err)
	}

	if len(loadedCache.events) != 1 {
		t.Fatalf("Expected 1 loaded event, got %d", len(loadedCache.events))
	}

	if loadedCache.events[0].CalendarEventID != 99060 {
		t.Errorf("CalendarEventID = %d, want 99060", loadedCache.events[0].CalendarEventID)
	}
	if loadedCache.events[0].QuizID != 99061 {
		t.Errorf("QuizID = %d, want 99061", loadedCache.events[0].QuizID)
	}
}

func TestParseICS_FoldedLines(t *testing.T) {
	foldedDesc := "DESCRIPTION:This is a very long description that exceeds seventy five octets per line limit per RFC 5545 Section 3.1 and must be folded correctly when parsed back from the ICS file content to ensure round-trip fidelity for the calendar application"
	ics := strings.Join([]string{
		"BEGIN:VCALENDAR",
		"VERSION:2.0",
		"PRODID:-//SIT Timetable//EN",
		"CALSCALE:GREGORIAN",
		"METHOD:PUBLISH",
		"REFRESH-INTERVAL;VALUE=DURATION:PT1H",
		"BEGIN:VEVENT",
		"UID:test-uid-folded",
		"DTSTART;TZID=Asia/Singapore:20260915T100000",
		"DTEND;TZID=Asia/Singapore:20260915T110000",
		"SUMMARY:Folded Test Event",
		foldedDesc,
		"LOCATION:Room 101",
		"END:VEVENT",
		"END:VCALENDAR",
	}, "\r\n") + "\r\n"

	loc, _ := time.LoadLocation("Asia/Singapore")
	events := parseICS([]byte(ics), loc)

	if len(events) != 1 {
		t.Fatalf("Expected 1 event, got %d", len(events))
	}

	expectedDesc := "This is a very long description that exceeds seventy five octets per line limit per RFC 5545 Section 3.1 and must be folded correctly when parsed back from the ICS file content to ensure round-trip fidelity for the calendar application"
	if events[0].Description != expectedDesc {
		t.Errorf("Description = %q, want %q", events[0].Description, expectedDesc)
	}
}

func TestParseICS_MultipleFoldedProperties(t *testing.T) {
	foldedSummary := "SUMMARY:This is a very long event summary that exceeds seventy five octets per line limit per RFC 5545 Section 3.1 and must be folded correctly when parsed back from the ICS file content"
	foldedLocation := "LOCATION:This is a very long location name that exceeds seventy five octets per line limit per RFC 5545 Section 3.1 and must be folded correctly when parsed back"

	ics := strings.Join([]string{
		"BEGIN:VCALENDAR",
		"VERSION:2.0",
		"PRODID:-//SIT Timetable//EN",
		"CALSCALE:GREGORIAN",
		"METHOD:PUBLISH",
		"REFRESH-INTERVAL;VALUE=DURATION:PT1H",
		"BEGIN:VEVENT",
		"UID:test-uid-folded-multi",
		"DTSTART;TZID=Asia/Singapore:20260915T100000",
		"DTEND;TZID=Asia/Singapore:20260915T110000",
		foldedSummary,
		foldedLocation,
		"END:VEVENT",
		"END:VCALENDAR",
	}, "\r\n") + "\r\n"

	loc, _ := time.LoadLocation("Asia/Singapore")
	events := parseICS([]byte(ics), loc)

	if len(events) != 1 {
		t.Fatalf("Expected 1 event, got %d", len(events))
	}

	expectedSummary := "This is a very long event summary that exceeds seventy five octets per line limit per RFC 5545 Section 3.1 and must be folded correctly when parsed back from the ICS file content"
	if events[0].Summary != expectedSummary {
		t.Errorf("Summary = %q, want %q", events[0].Summary, expectedSummary)
	}

	expectedLocation := "This is a very long location name that exceeds seventy five octets per line limit per RFC 5545 Section 3.1 and must be folded correctly when parsed back"
	if events[0].Location != expectedLocation {
		t.Errorf("Location = %q, want %q", events[0].Location, expectedLocation)
	}
}

func TestRoundTrip_FoldedLines(t *testing.T) {
	tmpDir := t.TempDir()
	icsPath := filepath.Join(tmpDir, "folded.ics")

	longDesc := strings.Repeat("Lorem ipsum dolor sit amet. ", 10)
	longSummary := strings.Repeat("A conference session title that is very long and exceeds the line limit. ", 5)

	loc, _ := time.LoadLocation("Asia/Singapore")

	cache := NewICSCache()
	err := cache.Update([]Event{
		{
			Summary:     longSummary,
			Location:    "Conference Hall A",
			Description: longDesc,
			DTStart:     time.Date(2026, 9, 15, 10, 0, 0, 0, loc),
			DTEnd:       time.Date(2026, 9, 15, 11, 0, 0, 0, loc),
		},
	}, "Asia/Singapore")
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	err = cache.SaveToFile(icsPath, time.Hour)
	if err != nil {
		t.Fatalf("SaveToFile() error = %v", err)
	}

	newCache := NewICSCache()
	err = newCache.LoadFromFile(icsPath, loc)
	if err != nil {
		t.Fatalf("LoadFromFile() error = %v", err)
	}

	if len(newCache.events) != 1 {
		t.Fatalf("Expected 1 loaded event, got %d", len(newCache.events))
	}

	if newCache.events[0].Summary != longSummary {
		t.Errorf("Summary mismatch: got %d chars, want %d chars", len(newCache.events[0].Summary), len(longSummary))
	}
	if newCache.events[0].Description != longDesc {
		t.Errorf("Description mismatch: got %d chars, want %d chars", len(newCache.events[0].Description), len(longDesc))
	}
}

func TestParseICS_DTSTAMP(t *testing.T) {
	ics := strings.Join([]string{
		"BEGIN:VCALENDAR",
		"VERSION:2.0",
		"PRODID:-//SIT Timetable//EN",
		"CALSCALE:GREGORIAN",
		"METHOD:PUBLISH",
		"REFRESH-INTERVAL;VALUE=DURATION:PT1H",
		"BEGIN:VEVENT",
		"UID:test-uid-dtstamp",
		"DTSTAMP:20260907T123045Z",
		"DTSTART;TZID=Asia/Singapore:20260915T100000",
		"DTEND;TZID=Asia/Singapore:20260915T110000",
		"SUMMARY:Test Event with DTSTAMP",
		"LOCATION:Room 101",
		"END:VEVENT",
		"END:VCALENDAR",
	}, "\r\n") + "\r\n"

	loc, _ := time.LoadLocation("Asia/Singapore")
	events := parseICS([]byte(ics), loc)

	if len(events) != 1 {
		t.Fatalf("Expected 1 event, got %d", len(events))
	}

	expectedDTStamp := time.Date(2026, 9, 7, 12, 30, 45, 0, time.UTC)
	if !events[0].DTStamp.Equal(expectedDTStamp) {
		t.Errorf("DTStamp = %v, want %v", events[0].DTStamp, expectedDTStamp)
	}

	if events[0].DTStamp.Location() != time.UTC {
		t.Errorf("DTStamp location = %v, want UTC", events[0].DTStamp.Location())
	}
}

func TestParseICS_DTSTAMP_RoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	icsPath := filepath.Join(tmpDir, "dtstamp.ics")

	loc, _ := time.LoadLocation("Asia/Singapore")

	cache := NewICSCache()

	dtstamp := time.Date(2026, 9, 7, 12, 30, 45, 0, time.UTC)
	events := []Event{
		{
			Summary:     "Event with DTSTAMP",
			Location:    "Room 101",
			DTStamp:     dtstamp,
			DTStart:     time.Date(2026, 9, 15, 10, 0, 0, 0, loc),
			DTEnd:       time.Date(2026, 9, 15, 11, 0, 0, 0, loc),
			Source:      "calendar",
			OrgUnitCode: "MOD1001",
		},
	}

	err := cache.Update(events, "Asia/Singapore")
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	err = cache.SaveToFile(icsPath, time.Hour)
	if err != nil {
		t.Fatalf("SaveToFile() error = %v", err)
	}

	newCache := NewICSCache()
	err = newCache.LoadFromFile(icsPath, loc)
	if err != nil {
		t.Fatalf("LoadFromFile() error = %v", err)
	}

	if len(newCache.events) != 1 {
		t.Fatalf("Expected 1 loaded event, got %d", len(newCache.events))
	}

	expectedDTStamp := time.Date(2026, 9, 7, 12, 30, 45, 0, time.UTC)
	if !newCache.events[0].DTStamp.Equal(expectedDTStamp) {
		t.Errorf("DTStamp after round-trip = %v, want %v", newCache.events[0].DTStamp, expectedDTStamp)
	}

	if newCache.events[0].DTStamp.Location() != time.UTC {
		t.Errorf("DTStamp location after round-trip = %v, want UTC", newCache.events[0].DTStamp.Location())
	}
}

func TestParseICS_DTSTAMP_Missing(t *testing.T) {
	ics := strings.Join([]string{
		"BEGIN:VCALENDAR",
		"VERSION:2.0",
		"PRODID:-//SIT Timetable//EN",
		"CALSCALE:GREGORIAN",
		"METHOD:PUBLISH",
		"REFRESH-INTERVAL;VALUE=DURATION:PT1H",
		"BEGIN:VEVENT",
		"UID:test-uid-no-dtstamp",
		"DTSTART;TZID=Asia/Singapore:20260915T100000",
		"DTEND;TZID=Asia/Singapore:20260915T110000",
		"SUMMARY:Test Event without DTSTAMP",
		"LOCATION:Room 101",
		"END:VEVENT",
		"END:VCALENDAR",
	}, "\r\n") + "\r\n"

	loc, _ := time.LoadLocation("Asia/Singapore")
	events := parseICS([]byte(ics), loc)

	if len(events) != 1 {
		t.Fatalf("Expected 1 event, got %d", len(events))
	}

	if !events[0].DTStamp.IsZero() {
		t.Errorf("DTStamp should be zero when DTSTAMP is absent, got %v", events[0].DTStamp)
	}
}

func TestParseICS_DTSTAMP_InvalidFormat(t *testing.T) {
	ics := strings.Join([]string{
		"BEGIN:VCALENDAR",
		"VERSION:2.0",
		"PRODID:-//SIT Timetable//EN",
		"CALSCALE:GREGORIAN",
		"METHOD:PUBLISH",
		"REFRESH-INTERVAL;VALUE=DURATION:PT1H",
		"BEGIN:VEVENT",
		"UID:test-uid-invalid-dtstamp",
		"DTSTAMP:invalid",
		"DTSTART;TZID=Asia/Singapore:20260915T100000",
		"DTEND;TZID=Asia/Singapore:20260915T110000",
		"SUMMARY:Test Event with invalid DTSTAMP",
		"LOCATION:Room 101",
		"END:VEVENT",
		"END:VCALENDAR",
	}, "\r\n") + "\r\n"

	loc, _ := time.LoadLocation("Asia/Singapore")
	events := parseICS([]byte(ics), loc)

	if len(events) != 1 {
		t.Fatalf("Expected 1 event, got %d", len(events))
	}

	if !events[0].DTStamp.IsZero() {
		t.Errorf("DTStamp should be zero when DTSTAMP has invalid format, got %v", events[0].DTStamp)
	}
}

func TestParseICS_DTSTAMP_MultipleEvents(t *testing.T) {
	ics := strings.Join([]string{
		"BEGIN:VCALENDAR",
		"VERSION:2.0",
		"PRODID:-//SIT Timetable//EN",
		"CALSCALE:GREGORIAN",
		"METHOD:PUBLISH",
		"REFRESH-INTERVAL;VALUE=DURATION:PT1H",
		"BEGIN:VEVENT",
		"UID:test-uid-dtstamp-001",
		"DTSTAMP:20260907T100000Z",
		"DTSTART;TZID=Asia/Singapore:20260915T100000",
		"DTEND;TZID=Asia/Singapore:20260915T110000",
		"SUMMARY:Event 1",
		"END:VEVENT",
		"BEGIN:VEVENT",
		"UID:test-uid-dtstamp-002",
		"DTSTAMP:20260907T140000Z",
		"DTSTART;TZID=Asia/Singapore:20260915T140000",
		"DTEND;TZID=Asia/Singapore:20260915T150000",
		"SUMMARY:Event 2",
		"END:VEVENT",
		"BEGIN:VEVENT",
		"UID:test-uid-dtstamp-003",
		"DTSTART;TZID=Asia/Singapore:20260916T100000",
		"DTEND;TZID=Asia/Singapore:20260916T110000",
		"SUMMARY:Event 3 no DTSTAMP",
		"END:VEVENT",
		"END:VCALENDAR",
	}, "\r\n") + "\r\n"

	loc, _ := time.LoadLocation("Asia/Singapore")
	events := parseICS([]byte(ics), loc)

	if len(events) != 3 {
		t.Fatalf("Expected 3 events, got %d", len(events))
	}

	expectedDTStamp1 := time.Date(2026, 9, 7, 10, 0, 0, 0, time.UTC)
	expectedDTStamp2 := time.Date(2026, 9, 7, 14, 0, 0, 0, time.UTC)

	if !events[0].DTStamp.Equal(expectedDTStamp1) {
		t.Errorf("Event 0 DTStamp = %v, want %v", events[0].DTStamp, expectedDTStamp1)
	}
	if !events[1].DTStamp.Equal(expectedDTStamp2) {
		t.Errorf("Event 1 DTStamp = %v, want %v", events[1].DTStamp, expectedDTStamp2)
	}
	if !events[2].DTStamp.IsZero() {
		t.Errorf("Event 2 DTStamp should be zero, got %v", events[2].DTStamp)
	}
}
