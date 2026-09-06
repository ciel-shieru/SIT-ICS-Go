package brightspace

import (
	"testing"
	"time"

	"github.com/ciel-shieru/sit-ics-go/internal/ics"
)

func TestBlocklist_IsCourseBlocked(t *testing.T) {
	tests := []struct {
		name          string
		blocklist     Blocklist
		orgUnitID     string
		courseName    string
		expectedBlocked bool
	}{
		{
			name: "exact course ID match",
			blocklist: Blocklist{
				CourseIDs: []string{"210875"},
			},
			orgUnitID:     "210875",
			courseName:    "Safety 101 (AY2026/27)",
			expectedBlocked: true,
		},
		{
			name: "case-insensitive course ID match",
			blocklist: Blocklist{
				CourseIDs: []string{"210875"},
			},
			orgUnitID:     "210875",
			courseName:    "Safety 101",
			expectedBlocked: true,
		},
		{
			name: "no course ID blocklist",
			blocklist: Blocklist{
				CourseIDs: []string{},
			},
			orgUnitID:     "210875",
			courseName:    "Safety 101",
			expectedBlocked: false,
		},
		{
			name: "course name substring match",
			blocklist: Blocklist{
				CourseNamePatterns: []string{"safety"},
			},
			orgUnitID:     "210875",
			courseName:    "Safety 101 (AY2026/27)",
			expectedBlocked: true,
		},
		{
			name: "course name substring match case-insensitive",
			blocklist: Blocklist{
				CourseNamePatterns: []string{"SAFETY"},
			},
			orgUnitID:     "210875",
			courseName:    "Safety 101 (AY2026/27)",
			expectedBlocked: true,
		},
		{
			name: "course name no match",
			blocklist: Blocklist{
				CourseNamePatterns: []string{"chemistry"},
			},
			orgUnitID:     "210875",
			courseName:    "Safety 101 (AY2026/27)",
			expectedBlocked: false,
		},
		{
			name: "multiple patterns - one matches",
			blocklist: Blocklist{
				CourseNamePatterns: []string{"chemistry", "safety"},
			},
			orgUnitID:     "210875",
			courseName:    "Safety 101 (AY2026/27)",
			expectedBlocked: true,
		},
		{
			name: "whitespace in patterns trimmed",
			blocklist: Blocklist{
				CourseNamePatterns: []string{" safety "},
			},
			orgUnitID:     "210875",
			courseName:    "Safety 101 (AY2026/27)",
			expectedBlocked: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.blocklist.IsCourseBlocked(tt.orgUnitID, tt.courseName)
			if got != tt.expectedBlocked {
				t.Errorf("IsCourseBlocked() = %v, want %v", got, tt.expectedBlocked)
			}
		})
	}
}

func TestBlocklist_IsEventBlocked(t *testing.T) {
	tests := []struct {
		name          string
		blocklist     Blocklist
		eventTitle    string
		expectedBlocked bool
	}{
		{
			name: "event title substring match",
			blocklist: Blocklist{
				EventTitlePatterns: []string{"Lecture"},
			},
			eventTitle:    "INF1104 Lecture Discussion (Online)",
			expectedBlocked: true,
		},
		{
			name: "event title substring match case-insensitive",
			blocklist: Blocklist{
				EventTitlePatterns: []string{"lecture"},
			},
			eventTitle:    "INF1104 Lecture Discussion (Online)",
			expectedBlocked: true,
		},
		{
			name: "event title no match",
			blocklist: Blocklist{
				EventTitlePatterns: []string{"workshop"},
			},
			eventTitle:    "INF1104 Lecture Discussion (Online)",
			expectedBlocked: false,
		},
		{
			name: "multiple patterns - one matches",
			blocklist: Blocklist{
				EventTitlePatterns: []string{"workshop", "lecture"},
			},
			eventTitle:    "INF1104 Lecture Discussion (Online)",
			expectedBlocked: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.blocklist.IsEventBlocked(tt.eventTitle)
			if got != tt.expectedBlocked {
				t.Errorf("IsEventBlocked() = %v, want %v", got, tt.expectedBlocked)
			}
		})
	}
}

func TestParseCommaSeparated(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "single value",
			input:    "safety",
			expected: []string{"safety"},
		},
		{
			name:     "multiple values",
			input:    "safety,chemistry,physics",
			expected: []string{"safety", "chemistry", "physics"},
		},
		{
			name:     "values with whitespace",
			input:    " safety , chemistry , physics ",
			expected: []string{"safety", "chemistry", "physics"},
		},
		{
			name:     "empty input",
			input:    "",
			expected: []string{},
		},
		{
			name:     "only whitespace",
			input:    "   ",
			expected: []string{},
		},
		{
			name:     "empty values between commas",
			input:    "safety,,chemistry",
			expected: []string{"safety", "chemistry"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseCommaSeparated(tt.input)
			if len(got) != len(tt.expected) {
				t.Errorf("ParseCommaSeparated() = %v (len=%d), want %v (len=%d)", got, len(got), tt.expected, len(tt.expected))
				return
			}
			for i, v := range got {
				if v != tt.expected[i] {
					t.Errorf("ParseCommaSeparated()[%d] = %q, want %q", i, v, tt.expected[i])
				}
			}
		})
	}
}

func TestHTMLToPlainText(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "plain text",
			input:    "Hello world",
			expected: "Hello world",
		},
		{
			name:     "HTML tags stripped",
			input:    "<p>Hello <b>world</b></p>",
			expected: "Hello world",
		},
		{
			name:     "br tags converted to newlines",
			input:    "Line 1<br>Line 2<br/>Line 3",
			expected: "Line 1\nLine 2\nLine 3",
		},
		{
			name:     "HTML entities decoded",
			input:    "5 &gt; 3 &amp; 3 &lt; 5",
			expected: "5 > 3 & 3 < 5",
		},
		{
			name:     "nbsp converted to space",
			input:    "Hello&nbsp;world",
			expected: "Hello world",
		},
		{
			name:     "zoom link description",
			input:    "<p><a rel=\"noopener\" href=\"https://singaporetech.zoom.us/j/92382132350?pwd=PlwtYX2zPjWJaDLoKSgGTtwKfa324b.1\" target=\"_blank\">Click here to join Zoom Meeting: 923 8213 2350</a></p>",
			expected: "Click here to join Zoom Meeting: 923 8213 2350",
		},
		{
			name:     "chinese text in zoom link",
			input:    "<p><a rel=\"noopener\" href=\"https://singaporetech.zoom.us/j/92382132350?pwd=PlwtYX2zPjWJaDLoKSgGTtwKfa324b.1\" target=\"_blank\">点击此处加入Zoom会议: 923 8213 2350</a></p>",
			expected: "点击此处加入Zoom会议: 923 8213 2350",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := htmlToPlainText(tt.input)
			if got != tt.expected {
				t.Errorf("htmlToPlainText() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestEntryToICSEvent(t *testing.T) {
	loc, err := time.LoadLocation("Asia/Singapore")
	if err != nil {
		t.Fatalf("failed to load timezone: %v", err)
	}

	tests := []struct {
		name     string
		entry    BrightSpaceEntry
		wantNil  bool
		wantSummaryPrefix string
	}{
		{
			name: "calendar event with org unit code",
			entry: BrightSpaceEntry{
				Source:      SourceCalendar,
				Title:       "INF1104-Discrete Mathematics for Computing [2026/27 T1]",
				OrgUnitId:   "213437",
				OrgUnitName: "INF1104-Discrete Mathematics for Computing [2026/27 T1]",
				OrgUnitCode: "SIT-2610-INF1104",
				Location:    "Zoom Online Meeting",
				DTStart:     time.Date(2026, 8, 31, 8, 0, 0, 0, time.UTC),
				DTEnd:       time.Date(2026, 8, 31, 10, 0, 0, 0, time.UTC),
			},
			wantNil:             false,
			wantSummaryPrefix:   "[SIT-2610-INF1104]",
		},
		{
			name: "calendar event without org unit code",
			entry: BrightSpaceEntry{
				Source:      SourceCalendar,
				Title:       "Tutorial Sheet",
				OrgUnitId:   "213437",
				OrgUnitName: "INF1104-Discrete Mathematics for Computing [2026/27 T1]",
				OrgUnitCode: "",
				DTStart:     time.Date(2026, 9, 6, 14, 0, 0, 0, time.UTC),
				DTEnd:       time.Date(2026, 9, 6, 14, 0, 0, 0, time.UTC),
			},
			wantNil:             false,
			wantSummaryPrefix:   "[INF1104-Discrete Mathematics for Computing [2026/27 T1]]",
		},
		{
			name: "dropbox due date",
			entry: BrightSpaceEntry{
				Source:      SourceDropbox,
				Title:       "[Submission Due] Weekly Lab 1",
				OrgUnitId:   "213437",
				OrgUnitName: "INF1104-Discrete Mathematics for Computing [2026/27 T1]",
				OrgUnitCode: "SIT-2610-INF1104",
				DTStart:     time.Date(2026, 9, 10, 15, 59, 0, 0, time.UTC),
				DTEnd:       time.Date(2026, 9, 10, 15, 59, 0, 0, time.UTC),
			},
			wantNil:             false,
			wantSummaryPrefix:   "[SIT-2610-INF1104]",
		},
		{
			name: "all-day event",
			entry: BrightSpaceEntry{
				Source:      SourceCalendar,
				Title:       "Holiday",
				OrgUnitId:   "213437",
				OrgUnitName: "Test Course",
				OrgUnitCode: "TEST",
				DTStart:     time.Date(2026, 12, 25, 0, 0, 0, 0, time.UTC),
				DTEnd:       time.Date(2026, 12, 25, 0, 0, 0, 0, time.UTC),
				IsAllDay:    true,
			},
			wantNil:             false,
			wantSummaryPrefix:   "[TEST]",
		},
		{
			name: "zero start time returns nil",
			entry: BrightSpaceEntry{
				Source:      SourceCalendar,
				Title:       "Test Event",
				OrgUnitId:   "213437",
				OrgUnitName: "Test Course",
				DTStart:     time.Time{},
				DTEnd:       time.Time{},
			},
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := entryToICSEvent(&tt.entry, loc)
			if tt.wantNil {
				if got != nil {
					t.Errorf("entryToICSEvent() = %v, want nil", got)
				}
				return
			}
			if got == nil {
				t.Fatalf("entryToICSEvent() = nil, want non-nil event")
			}
			if got.DTStart.IsZero() {
				t.Error("entryToICSEvent() DTStart is zero")
			}
			if got.DTEnd.IsZero() {
				t.Error("entryToICSEvent() DTEnd is zero")
			}
			if got.Location != tt.entry.Location {
				t.Errorf("entryToICSEvent() Location = %q, want %q", got.Location, tt.entry.Location)
			}
			if len(tt.wantSummaryPrefix) > 0 && len(got.Summary) < len(tt.wantSummaryPrefix) {
				t.Errorf("entryToICSEvent() Summary = %q, too short to contain prefix %q", got.Summary, tt.wantSummaryPrefix)
			}
		})
	}
}

func TestGenerateUIDWithSource(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Singapore")
	event1 := &ics.Event{
		Summary:     "Test Event",
		Location:    "Room 1",
		DTStart:     time.Date(2026, 8, 31, 8, 0, 0, 0, loc),
		DTEnd:       time.Date(2026, 8, 31, 10, 0, 0, 0, loc),
		Source:      "brightspace-calendar",
	}
	event2 := &ics.Event{
		Summary:     "Test Event",
		Location:    "Room 1",
		DTStart:     time.Date(2026, 8, 31, 8, 0, 0, 0, loc),
		DTEnd:       time.Date(2026, 8, 31, 10, 0, 0, 0, loc),
		Source:      "brightspace-dropbox",
	}
	event3 := &ics.Event{
		Summary:     "Test Event",
		Location:    "Room 1",
		DTStart:     time.Date(2026, 8, 31, 8, 0, 0, 0, loc),
		DTEnd:       time.Date(2026, 8, 31, 10, 0, 0, 0, loc),
		Source:      "",
	}

	uid1 := ics.GenerateUID(*event1)
	uid2 := ics.GenerateUID(*event2)
	uid3 := ics.GenerateUID(*event3)

	if uid1 == uid2 {
		t.Errorf("UIDs should differ for different sources: %s == %s", uid1, uid2)
	}
	if uid1 == uid3 {
		t.Errorf("UIDs should differ when source is set vs unset: %s == %s", uid1, uid3)
	}
	if uid2 == uid3 {
		t.Errorf("UIDs should differ when source is set vs unset: %s == %s", uid2, uid3)
	}

	// Same event should produce same UID (idempotency)
	uid1b := ics.GenerateUID(*event1)
	if uid1 != uid1b {
		t.Errorf("Same event should produce same UID: %s != %s", uid1, uid1b)
	}
}

func TestCalendarEventToEntry(t *testing.T) {
	loc, _ := time.LoadLocation("UTC")
	ev := CalendarEvent{
		Title:         "INF1104 Lecture",
		Description:   "<p>Click <a href=\"https://zoom.us/j/123\">here</a></p>",
		LocationName:  "Zoom Online Meeting",
		OrgUnitName:   "INF1104-Discrete Mathematics",
		OrgUnitCode:   "SIT-2610-INF1104",
		StartDateTime: time.Date(2026, 8, 31, 8, 0, 0, 0, loc),
		EndDateTime:   time.Date(2026, 8, 31, 10, 0, 0, 0, loc),
	}

	entry := calendarEventToEntry(ev, SourceCalendar, "213437", "INF1104-Discrete Mathematics", "SIT-2610-INF1104")

	if entry.Title != "INF1104 Lecture" {
		t.Errorf("Title = %q, want %q", entry.Title, "INF1104 Lecture")
	}
	if entry.OrgUnitId != "213437" {
		t.Errorf("OrgUnitId = %q, want %q", entry.OrgUnitId, "213437")
	}
	if entry.Source != SourceCalendar {
		t.Errorf("Source = %v, want %v", entry.Source, SourceCalendar)
	}
	if entry.Location != "Zoom Online Meeting" {
		t.Errorf("Location = %q, want %q", entry.Location, "Zoom Online Meeting")
	}
}

func TestDropboxFolderToEntry(t *testing.T) {
	loc, _ := time.LoadLocation("UTC")
	folder := DropboxFolder{
		Name:    "Weekly Lab 1",
		DueDate: time.Date(2026, 9, 10, 15, 59, 0, 0, loc),
	}

	entry := dropboxFolderToEntry(folder, SourceDropbox, "213437", "INF1104-Discrete Mathematics", "SIT-2610-INF1104")

	if entry.Title != "[Submission Due] Weekly Lab 1" {
		t.Errorf("Title = %q, want %q", entry.Title, "[Submission Due] Weekly Lab 1")
	}
	if entry.Source != SourceDropbox {
		t.Errorf("Source = %v, want %v", entry.Source, SourceDropbox)
	}
	if entry.OrgUnitId != "213437" {
		t.Errorf("OrgUnitId = %q, want %q", entry.OrgUnitId, "213437")
	}
}
