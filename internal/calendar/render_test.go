package calendar

import (
	"strings"
	"testing"
	"time"
)

func TestRender_IncludesXCalendarEventId(t *testing.T) {
	events := []Event{
		{
			Summary:         "Test Event",
			Location:        "Room 101",
			DTStart:         time.Date(2026, 9, 7, 9, 0, 0, 0, time.UTC),
			DTEnd:           time.Date(2026, 9, 7, 11, 0, 0, 0, time.UTC),
			CalendarEventID: 99001,
		},
	}

	data, err := Render(events, RenderOptions{
		Timezone:        time.UTC,
		RefreshInterval: time.Hour,
	})
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "X-CalendarEventId:99001") {
		t.Error("Rendered ICS missing X-CalendarEventId:99001")
	}
}

func TestRender_IncludesXQuizId(t *testing.T) {
	events := []Event{
		{
			Summary: "Test Event",
			Location: "Room 101",
			DTStart: time.Date(2026, 9, 7, 9, 0, 0, 0, time.UTC),
			DTEnd:   time.Date(2026, 9, 7, 11, 0, 0, 0, time.UTC),
			QuizID:  99010,
		},
	}

	data, err := Render(events, RenderOptions{
		Timezone:        time.UTC,
		RefreshInterval: time.Hour,
	})
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "X-QuizId:99010") {
		t.Error("Rendered ICS missing X-QuizId:99010")
	}
}

func TestRender_ExcludesBothWhenZero(t *testing.T) {
	events := []Event{
		{
			Summary:         "Test Event",
			Location:        "Room 101",
			DTStart:         time.Date(2026, 9, 7, 9, 0, 0, 0, time.UTC),
			DTEnd:           time.Date(2026, 9, 7, 11, 0, 0, 0, time.UTC),
			CalendarEventID: 0,
			QuizID:          0,
		},
	}

	data, err := Render(events, RenderOptions{
		Timezone:        time.UTC,
		RefreshInterval: time.Hour,
	})
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	content := string(data)
	if strings.Contains(content, "X-CalendarEventId:") {
		t.Error("Rendered ICS should not contain X-CalendarEventId when CalendarEventID is 0")
	}
	if strings.Contains(content, "X-QuizId:") {
		t.Error("Rendered ICS should not contain X-QuizId when QuizID is 0")
	}
}

func TestRender_BothIDsNonZero(t *testing.T) {
	events := []Event{
		{
			Summary:         "Test Event",
			Location:        "Room 101",
			DTStart:         time.Date(2026, 9, 7, 9, 0, 0, 0, time.UTC),
			DTEnd:           time.Date(2026, 9, 7, 11, 0, 0, 0, time.UTC),
			CalendarEventID: 99001,
			QuizID:          99010,
		},
	}

	data, err := Render(events, RenderOptions{
		Timezone:        time.UTC,
		RefreshInterval: time.Hour,
	})
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "X-CalendarEventId:99001") {
		t.Error("Rendered ICS missing X-CalendarEventId:99001")
	}
	if !strings.Contains(content, "X-QuizId:99010") {
		t.Error("Rendered ICS missing X-QuizId:99010")
	}
}

func TestRender_MultipleEventsMixedIDs(t *testing.T) {
	events := []Event{
		{
			Summary:         "Event with both IDs",
			Location:        "Room 101",
			DTStart:         time.Date(2026, 9, 7, 9, 0, 0, 0, time.UTC),
			DTEnd:           time.Date(2026, 9, 7, 11, 0, 0, 0, time.UTC),
			CalendarEventID: 99001,
			QuizID:          99010,
		},
		{
			Summary:         "Event with only CalendarEventID",
			Location:        "Room 102",
			DTStart:         time.Date(2026, 9, 7, 14, 0, 0, 0, time.UTC),
			DTEnd:           time.Date(2026, 9, 7, 16, 0, 0, 0, time.UTC),
			CalendarEventID: 99002,
			QuizID:          0,
		},
		{
			Summary:         "Event with only QuizID",
			Location:        "Room 103",
			DTStart:         time.Date(2026, 9, 8, 9, 0, 0, 0, time.UTC),
			DTEnd:           time.Date(2026, 9, 8, 11, 0, 0, 0, time.UTC),
			CalendarEventID: 0,
			QuizID:          99011,
		},
		{
			Summary:         "Event with no IDs",
			Location:        "Room 104",
			DTStart:         time.Date(2026, 9, 8, 14, 0, 0, 0, time.UTC),
			DTEnd:           time.Date(2026, 9, 8, 16, 0, 0, 0, time.UTC),
			CalendarEventID: 0,
			QuizID:          0,
		},
	}

	data, err := Render(events, RenderOptions{
		Timezone:        time.UTC,
		RefreshInterval: time.Hour,
	})
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	content := string(data)

	if !strings.Contains(content, "X-CalendarEventId:99001") {
		t.Error("Missing X-CalendarEventId:99001")
	}
	if !strings.Contains(content, "X-CalendarEventId:99002") {
		t.Error("Missing X-CalendarEventId:99002")
	}
	if !strings.Contains(content, "X-QuizId:99010") {
		t.Error("Missing X-QuizId:99010")
	}
	if !strings.Contains(content, "X-QuizId:99011") {
		t.Error("Missing X-QuizId:99011")
	}

	count := strings.Count(content, "X-CalendarEventId:")
	if count != 2 {
		t.Errorf("Expected 2 X-CalendarEventId properties, got %d", count)
	}
	count = strings.Count(content, "X-QuizId:")
	if count != 2 {
		t.Errorf("Expected 2 X-QuizId properties, got %d", count)
	}
}

func TestVTimezoneBlock_AsiaSingapore(t *testing.T) {
	loc, err := time.LoadLocation("Asia/Singapore")
	if err != nil {
		t.Fatalf("failed to load Asia/Singapore timezone: %v", err)
	}

	block := vtimezoneBlock(loc)
	if block == "" {
		t.Fatal("vtimezoneBlock(Asia/Singapore) returned empty string")
	}

	expected := "BEGIN:VTIMEZONE\r\nTZID:Asia/Singapore\r\nBEGIN:STANDARD\r\nDTSTART:19820101T000000\r\nTZOFFSETFROM:+0800\r\nTZOFFSETTO:+0800\r\nTZNAME:SGT\r\nEND:STANDARD\r\nEND:VTIMEZONE\r\n"
	if block != expected {
		t.Errorf("vtimezoneBlock(Asia/Singapore) =\n%q\nwant\n%q", block, expected)
	}

	if !strings.Contains(block, "BEGIN:VTIMEZONE") {
		t.Error("missing BEGIN:VTIMEZONE")
	}
	if !strings.Contains(block, "TZID:Asia/Singapore") {
		t.Error("missing TZID:Asia/Singapore")
	}
	if !strings.Contains(block, "BEGIN:STANDARD") {
		t.Error("missing BEGIN:STANDARD")
	}
	if !strings.Contains(block, "END:STANDARD") {
		t.Error("missing END:STANDARD")
	}
	if !strings.Contains(block, "END:VTIMEZONE") {
		t.Error("missing END:VTIMEZONE")
	}
	if !strings.Contains(block, "TZNAME:SGT") {
		t.Error("missing TZNAME:SGT")
	}
}

func TestVTimezoneBlock_UTC(t *testing.T) {
	block := vtimezoneBlock(time.UTC)
	if block != "" {
		t.Errorf("vtimezoneBlock(UTC) = %q, want empty string", block)
	}

	block = vtimezoneBlock(nil)
	if block != "" {
		t.Errorf("vtimezoneBlock(nil) = %q, want empty string", block)
	}
}

func TestRender_IncludesVTIMEZONE(t *testing.T) {
	loc, err := time.LoadLocation("Asia/Singapore")
	if err != nil {
		t.Fatalf("failed to load Asia/Singapore timezone: %v", err)
	}

	events := []Event{
		{
			Summary: "Test Event",
			Location: "Room 101",
			DTStart: time.Date(2026, 9, 7, 9, 0, 0, 0, loc),
			DTEnd:   time.Date(2026, 9, 7, 11, 0, 0, 0, loc),
		},
	}

	data, err := Render(events, RenderOptions{
		Timezone:        loc,
		RefreshInterval: time.Hour,
	})
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "BEGIN:VTIMEZONE") {
		t.Error("Rendered ICS missing BEGIN:VTIMEZONE")
	}
	if !strings.Contains(content, "TZID:Asia/Singapore") {
		t.Error("Rendered ICS missing TZID:Asia/Singapore")
	}
	if !strings.Contains(content, "TZNAME:SGT") {
		t.Error("Rendered ICS missing TZNAME:SGT")
	}

	vtzIdx := strings.Index(content, "BEGIN:VTIMEZONE")
	methodIdx := strings.Index(content, "METHOD:PUBLISH")
	if vtzIdx < 0 || methodIdx < 0 || vtzIdx < methodIdx {
		t.Error("VTIMEZONE should appear after METHOD:PUBLISH")
	}
}

func TestRender_ExcludesVTIMEZONE_UTC(t *testing.T) {
	events := []Event{
		{
			Summary: "Test Event",
			Location: "Room 101",
			DTStart: time.Date(2026, 9, 7, 9, 0, 0, 0, time.UTC),
			DTEnd:   time.Date(2026, 9, 7, 11, 0, 0, 0, time.UTC),
		},
	}

	data, err := Render(events, RenderOptions{
		Timezone:        time.UTC,
		RefreshInterval: time.Hour,
	})
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	content := string(data)
	if strings.Contains(content, "BEGIN:VTIMEZONE") {
		t.Error("Rendered ICS should not contain VTIMEZONE when timezone is UTC")
	}
}

func TestFoldText_NoFolding(t *testing.T) {
	tests := []string{
		"",
		"short",
		strings.Repeat("a", 75),
	}
	for _, tc := range tests {
		got := FoldText(tc)
		if got != tc {
			t.Errorf("FoldText(%q) = %q, want %q", tc, got, tc)
		}
	}
}

func TestFoldText_Exact75(t *testing.T) {
	input := strings.Repeat("a", 75)
	got := FoldText(input)
	if got != input {
		t.Errorf("FoldText(75 chars) = %q, want unchanged", got)
	}
	if strings.Contains(got, "\r\n ") {
		t.Error("FoldText(75 chars) should not contain fold sequence")
	}
}

func TestFoldText_Exact76(t *testing.T) {
	input := strings.Repeat("a", 76)
	got := FoldText(input)
	expected := strings.Repeat("a", 75) + "\r\n a"
	if got != expected {
		t.Errorf("FoldText(76 chars) = %q, want %q", got, expected)
	}
}

func TestFoldText_SingleFold(t *testing.T) {
	input := strings.Repeat("a", 76)
	got := FoldText(input)
	if !strings.Contains(got, "\r\n ") {
		t.Error("FoldText(76 chars) should contain fold sequence")
	}
	parts := strings.Split(got, "\r\n ")
	if len(parts) != 2 {
		t.Fatalf("FoldText(76 chars) should produce 2 parts, got %d", len(parts))
	}
	if len(parts[0]) != 75 {
		t.Errorf("First line length = %d, want 75", len(parts[0]))
	}
	if len(parts[1]) != 1 {
		t.Errorf("Continuation content length = %d, want 1", len(parts[1]))
	}
}

func TestFoldText_MultipleFolds(t *testing.T) {
	input := strings.Repeat("a", 150)
	got := FoldText(input)
	parts := strings.Split(got, "\r\n ")
	if len(parts) != 3 {
		t.Fatalf("FoldText(150 chars) should produce 3 parts, got %d", len(parts))
	}
	if len(parts[0]) != 75 {
		t.Errorf("First line length = %d, want 75", len(parts[0]))
	}
	if len(parts[1]) != 74 {
		t.Errorf("Second line length = %d, want 74", len(parts[1]))
	}
	if len(parts[2]) != 1 {
		t.Errorf("Third line length = %d, want 1", len(parts[2]))
	}
}

func TestFoldText_VeryLong(t *testing.T) {
	input := strings.Repeat("x", 300)
	got := FoldText(input)
	parts := strings.Split(got, "\r\n ")
	if len(parts) != 5 {
		t.Fatalf("FoldText(300 chars) should produce 5 parts, got %d", len(parts))
	}
	if len(parts[0]) != 75 {
		t.Errorf("First line length = %d, want 75", len(parts[0]))
	}
	for i := 1; i < len(parts); i++ {
		if len(parts[i]) != 74 && i != len(parts)-1 {
			t.Errorf("Part %d length = %d, want 74 (except last)", i, len(parts[i]))
		}
	}
}

func TestRender_FoldedLines(t *testing.T) {
	longSummary := strings.Repeat("A", 100)
	events := []Event{
		{
			Summary: longSummary,
			Location: "Room 101",
			DTStart: time.Date(2026, 9, 7, 9, 0, 0, 0, time.UTC),
			DTEnd:   time.Date(2026, 9, 7, 11, 0, 0, 0, time.UTC),
		},
	}

	data, err := Render(events, RenderOptions{
		Timezone:        time.UTC,
		RefreshInterval: time.Hour,
	})
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "\r\n ") {
		t.Error("Rendered ICS should contain folded lines for 100-char SUMMARY")
	}

	// Verify the SUMMARY line starts correctly
	if !strings.Contains(content, "SUMMARY:"+strings.Repeat("A", 75)) {
		t.Error("SUMMARY first line should contain first 75 chars")
	}
}

func TestRender_FoldedVALARMDescription(t *testing.T) {
	longDesc := strings.Repeat("B", 100)
	events := []Event{
		{
			Summary: "Test Event",
			Location: "Room 101",
			DTStart: time.Date(2026, 9, 7, 9, 0, 0, 0, time.UTC),
			DTEnd:   time.Date(2026, 9, 7, 11, 0, 0, 0, time.UTC),
		},
	}

	data, err := Render(events, RenderOptions{
		Timezone:        time.UTC,
		RefreshInterval: time.Hour,
		Alerts: []Alert{
			{
				Duration:    15 * time.Minute,
				Action:      AlertDisplay,
				Description: longDesc,
			},
		},
	})
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "BEGIN:VALARM") {
		t.Fatal("Rendered ICS missing VALARM")
	}
	if !strings.Contains(content, "\r\n ") {
		t.Error("VALARM DESCRIPTION should be folded for 100-char description")
	}
}

func TestRender_NoFoldingForShortText(t *testing.T) {
	events := []Event{
		{
			Summary: "Short",
			Location: "Room 101",
			DTStart: time.Date(2026, 9, 7, 9, 0, 0, 0, time.UTC),
			DTEnd:   time.Date(2026, 9, 7, 11, 0, 0, 0, time.UTC),
		},
	}

	data, err := Render(events, RenderOptions{
		Timezone:        time.UTC,
		RefreshInterval: time.Hour,
	})
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	content := string(data)
	if strings.Contains(content, "\r\n ") {
		t.Error("Rendered ICS should not contain fold sequences for short text")
	}
}

func TestRender_NoFoldingForNumericProperties(t *testing.T) {
	events := []Event{
		{
			Summary:         "Test Event",
			Location:        "Room 101",
			DTStart:         time.Date(2026, 9, 7, 9, 0, 0, 0, time.UTC),
			DTEnd:           time.Date(2026, 9, 7, 11, 0, 0, 0, time.UTC),
			CalendarEventID: 99001,
			QuizID:          99010,
		},
	}

	data, err := Render(events, RenderOptions{
		Timezone:        time.UTC,
		RefreshInterval: time.Hour,
	})
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	content := string(data)
	// Numeric properties should not be folded
	if strings.Contains(content, "X-CalendarEventId:\r\n ") {
		t.Error("X-CalendarEventId should not be folded")
	}
	if strings.Contains(content, "X-QuizId:\r\n ") {
		t.Error("X-QuizId should not be folded")
	}
}

func TestRender_XPropertyOrder(t *testing.T) {
	events := []Event{
		{
			Summary:         "Test Event",
			Location:        "Room 101",
			DTStart:         time.Date(2026, 9, 7, 9, 0, 0, 0, time.UTC),
			DTEnd:           time.Date(2026, 9, 7, 11, 0, 0, 0, time.UTC),
			Title:           "Test Title",
			CalendarEventID: 99001,
			QuizID:          99010,
		},
	}

	data, err := Render(events, RenderOptions{
		Timezone:        time.UTC,
		RefreshInterval: time.Hour,
	})
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	content := string(data)

	titleIdx := strings.Index(content, "X-Title:")
	calIdx := strings.Index(content, "X-CalendarEventId:")
	quizIdx := strings.Index(content, "X-QuizId:")

	if titleIdx < 0 || calIdx < 0 || quizIdx < 0 {
		t.Error("Missing expected X- properties")
	}
	if titleIdx >= calIdx {
		t.Error("X-Title should come before X-CalendarEventId")
	}
	if calIdx >= quizIdx {
		t.Error("X-CalendarEventId should come before X-QuizId")
	}
}

func TestRender_IncludesDTSTAMP(t *testing.T) {
	events := []Event{
		{
			Summary: "Test Event",
			Location: "Room 101",
			DTStamp: time.Date(2026, 9, 7, 12, 30, 45, 0, time.UTC),
			DTStart: time.Date(2026, 9, 7, 9, 0, 0, 0, time.UTC),
			DTEnd:   time.Date(2026, 9, 7, 11, 0, 0, 0, time.UTC),
		},
	}

	data, err := Render(events, RenderOptions{
		Timezone:        time.UTC,
		RefreshInterval: time.Hour,
	})
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "DTSTAMP:20260907T123045Z") {
		t.Error("Rendered ICS missing DTSTAMP:20260907T123045Z")
	}

	if !strings.Contains(content, "DTSTAMP:") {
		t.Error("Rendered ICS missing DTSTAMP property")
	}

	// Verify DTSTAMP ends with Z (UTC format)
	dtsIdx := strings.Index(content, "DTSTAMP:")
	if dtsIdx < 0 {
		t.Fatal("DTSTAMP not found")
	}
	lineEnd := strings.Index(content[dtsIdx:], "\r\n")
	if lineEnd < 0 {
		t.Fatal("DTSTAMP line not terminated")
	}
	line := content[dtsIdx : dtsIdx+lineEnd]
	if !strings.HasSuffix(line, "Z") {
		t.Errorf("DTSTAMP should end with Z for UTC format, got %q", line)
	}
}

func TestRender_DTSTAMP_UTCConversion(t *testing.T) {
	loc, err := time.LoadLocation("Asia/Singapore")
	if err != nil {
		t.Fatalf("failed to load Asia/Singapore timezone: %v", err)
	}

	singaporeTime := time.Date(2026, 9, 7, 20, 30, 45, 0, loc)
	data, err := Render([]Event{
		{
			Summary: "Test Event",
			Location: "Room 101",
			DTStamp: singaporeTime,
			DTStart: time.Date(2026, 9, 7, 9, 0, 0, 0, loc),
			DTEnd:   time.Date(2026, 9, 7, 11, 0, 0, 0, loc),
		},
	}, RenderOptions{
		Timezone:        loc,
		RefreshInterval: time.Hour,
	})
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	content := string(data)
	// 20:30:45 SGT (UTC+8) = 12:30:45 UTC
	if !strings.Contains(content, "DTSTAMP:20260907T123045Z") {
		t.Error("Rendered ICS DTSTAMP should be converted to UTC, got content with DTSTAMP line")
	}
}

func TestRender_DTSTAMPPropertyOrder(t *testing.T) {
	events := []Event{
		{
			Summary: "Test Event",
			Location: "Room 101",
			DTStamp: time.Date(2026, 9, 7, 12, 30, 45, 0, time.UTC),
			DTStart: time.Date(2026, 9, 7, 9, 0, 0, 0, time.UTC),
			DTEnd:   time.Date(2026, 9, 7, 11, 0, 0, 0, time.UTC),
		},
	}

	data, err := Render(events, RenderOptions{
		Timezone:        time.UTC,
		RefreshInterval: time.Hour,
	})
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	content := string(data)

	uidIdx := strings.Index(content, "UID:")
	dtsIdx := strings.Index(content, "DTSTAMP:")
	dtsStartIdx := strings.Index(content, "DTSTART;TZID=")

	if uidIdx < 0 || dtsIdx < 0 || dtsStartIdx < 0 {
		t.Fatal("Missing expected properties")
	}
	if uidIdx >= dtsIdx {
		t.Error("UID should come before DTSTAMP")
	}
	if dtsIdx >= dtsStartIdx {
		t.Error("DTSTAMP should come before DTSTART")
	}
}

func TestRender_DTSTAMP_ZeroTime(t *testing.T) {
	events := []Event{
		{
			Summary: "Test Event",
			Location: "Room 101",
			DTStamp: time.Time{},
			DTStart: time.Date(2026, 9, 7, 9, 0, 0, 0, time.UTC),
			DTEnd:   time.Date(2026, 9, 7, 11, 0, 0, 0, time.UTC),
		},
	}

	data, err := Render(events, RenderOptions{
		Timezone:        time.UTC,
		RefreshInterval: time.Hour,
	})
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "DTSTAMP:00010101T000000Z") {
		t.Error("Rendered ICS should include DTSTAMP even for zero time")
	}
}

func TestEscapeText_StripsNonASCII(t *testing.T) {
	input := "Hello ∞ World 入力"
	got := EscapeText(input)
	want := "Hello  World " // non-ASCII stripped
	if got != want {
		t.Errorf("EscapeText(%q) = %q, want %q", input, got, want)
	}
}

func TestEscapeText_EscapesSpecialChars(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"a\\b", "a\\\\b"},
		{"a;b", "a\\;b"},
		{"a,b", "a\\,b"},
		{"a\nb", "a\\nb"},
		{"&#160;", "&#160\\;"}, // EscapeText escapes semicolons
	}
	for _, tc := range tests {
		got := EscapeText(tc.input)
		if got != tc.want {
			t.Errorf("EscapeText(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestFoldText_ByteCount(t *testing.T) {
	// 75 ASCII chars = 75 bytes, should not fold
	input := strings.Repeat("a", 75)
	got := FoldText(input)
	if got != input {
		t.Errorf("FoldText(75 ASCII chars) = folded, should not fold")
	}

	// 76 ASCII chars = 76 bytes, should fold
	input = strings.Repeat("a", 76)
	got = FoldText(input)
	parts := strings.Split(got, "\r\n ")
	if len(parts) != 2 {
		t.Fatalf("FoldText(76 ASCII chars) should produce 2 parts, got %d", len(parts))
	}
	if len([]byte(parts[0])) != 75 {
		t.Errorf("First part = %d bytes, want 75", len([]byte(parts[0])))
	}
}

func TestFoldText_NoSplitEscapeSequence(t *testing.T) {
	// Text ending with escape sequence at fold boundary
	text := strings.Repeat("a", 73) + "\\n" + strings.Repeat("b", 10)
	got := FoldText(text)

	// The \n should not be split across lines
	if strings.Contains(got, "\\") && strings.Contains(got, "\r\n n") {
		t.Error("FoldText split a \\n escape sequence across line boundary")
	}

	// Verify the escaped newline is preserved intact
	if !strings.Contains(got, "\\n") {
		t.Error("FoldText should preserve escaped newlines")
	}
}

func TestFoldText_MultiByteUTF8(t *testing.T) {
	// After ASCII stripping in EscapeText, only ASCII remains.
	// But FoldText should still use byte counting for correctness.
	input := strings.Repeat("x", 80) // all ASCII
	got := FoldText(input)
	parts := strings.Split(got, "\r\n ")
	// First part should be exactly 75 bytes
	if len([]byte(parts[0])) != 75 {
		t.Errorf("First part = %d bytes, want 75", len([]byte(parts[0])))
	}
}

func TestRender_NoMalformedEntities(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Singapore")
	events := []Event{
		{
			Summary:     "Test Event",
			Description: "Space here's and symbol",
			DTStart:     time.Date(2026, 9, 7, 9, 0, 0, 0, loc),
			DTEnd:       time.Date(2026, 9, 7, 11, 0, 0, 0, loc),
		},
	}
	data, err := Render(events, RenderOptions{
		Timezone:        loc,
		RefreshInterval: time.Hour,
	})
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	content := string(data)
	// Should not contain malformed &#...; patterns
	if strings.Contains(content, "&#") {
		t.Errorf("Rendered ICS should not contain HTML entities, found: &# in content")
	}
	// Should not contain trailing backslash before CRLF
	lines := strings.Split(string(data), "\r\n")
	for i, line := range lines {
		if strings.HasSuffix(line, "\\") {
			t.Errorf("Line %d ends with trailing backslash: %q", i, line)
		}
	}
	// All lines should be <= 75 octets
	for i, line := range lines {
		if len([]byte(line)) > 75 {
			t.Errorf("Line %d exceeds 75 octets: %d bytes", i, len([]byte(line)))
		}
	}
}
