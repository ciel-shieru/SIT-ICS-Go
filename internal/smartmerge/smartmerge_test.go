package smartmerge

import (
	"testing"
	"time"

	"github.com/ciel-shieru/sit-ics-go/internal/calendar"
)

func TestNormalizeModuleCode(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"already normalized", "MOD1001", "MOD1001"},
		{"with space", "MOD 1001", "MOD1001"},
		{"lowercase", "mod1001", "MOD1001"},
		{"mixed case with space", "MOD 1001", "MOD1001"},
		{"extra whitespace", "  MOD 1001  ", "MOD1001"},
		{"multiple spaces", "MOD   1001", "MOD1001"},
		{"empty string", "", ""},
		{"single letter", "A", "A"},
		{"alphanumeric", "COR2003", "COR2003"},
		{"with dashes", "MOD-1001", "MOD-1001"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeModuleCode(tt.input)
			if got != tt.expected {
				t.Errorf("NormalizeModuleCode(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestIsValidModuleCode(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"valid: 3 letters + 4 digits", "MOD1002", true},
		{"valid: with suffix letter", "MOD1001T1", true},
		{"valid: with suffix dash+alphanumeric", "COR2001-SEC01", true},
		{"valid: minimal valid", "ABC1234", true},
		{"invalid: only 3 digits", "MOD100", false},
		{"invalid: 5 digits", "MOD10023", false},
		{"invalid: 2 letters", "MO1002", false},
		{"invalid: 6 letters", "MODULE1234", false},
		{"invalid: only letters", "ABCDEF", false},
		{"invalid: only digits", "12345", false},
		{"invalid: empty string", "", false},
		{"invalid: special chars only", "!@#$%", false},
		{"valid: suffix with numbers after non-digit", "ABC1234T1", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsValidModuleCode(tt.input)
			if got != tt.expected {
				t.Errorf("IsValidModuleCode(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestExtractModuleCodeFromOrgUnitName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"standard format", "MOD1001-Sample Module Title [2026/27 T1]", "MOD1001"},
		{"lowercase", "mod1001-intro [2026]", "MOD1001"},
		{"with spaces after code", "MOD1001 - Intro [2026]", "MOD1001"},
		{"alphanumeric code", "COR2003ABC-something", "COR2003ABC"},
		{"no alphanumeric prefix", "-intro something", ""},
		{"empty string", "", ""},
		{"only brackets", "[2026/27 T1]", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractModuleCodeFromOrgUnitName(tt.input)
			if got != tt.expected {
				t.Errorf("ExtractModuleCodeFromOrgUnitName(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestModulesMatch(t *testing.T) {
	tests := []struct {
		name          string
		psCode        string
		bsOrgUnitCode string
		want          bool
	}{
		{"exact match", "MOD1001", "MOD1001", true},
		{"case insensitive", "mod1001", "MOD1001", true},
		{"ps with space", "MOD 1001", "MOD1001", true},
		{"bs code has prefix", "MOD1001", "SIT-2610-MOD1001", true},
		{"ps code is substring of bs code", "MOD1002", "SIT-2610-MOD1002", true},
		{"ps code not in bs code", "MOD1001", "SIT-2610-MOD1002", false},
		{"no match different module", "COR2003", "MOD1001", false},
		{"empty ps code", "", "MOD1001", false},
		{"empty bs code", "MOD1001", "", false},
		{"valid: base format", "MOD1002", "MOD1002", true},
		{"valid: with suffix letter", "MOD1001T1", "SIT-2610-MOD1001T1", true},
		{"invalid: only 3 digits", "MOD100", "MOD100", false},
		{"invalid: 5 digits", "MOD10023", "MOD10023", false},
		{"invalid: only 2 letters", "MO1002", "MO1002", false},
		{"invalid: 6 letters", "MODULE1234", "MODULE1234", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ModulesMatch(tt.psCode, calendar.Event{OrgUnitCode: tt.bsOrgUnitCode})
			if got != tt.want {
				t.Errorf("ModulesMatch(%q, calendar.Event{OrgUnitCode: %q}) = %v, want %v", tt.psCode, tt.bsOrgUnitCode, got, tt.want)
			}
		})
	}
}

func TestTimesOverlap(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Singapore")
	base := time.Date(2026, 8, 31, 9, 0, 0, 0, loc)

	tests := []struct {
		name    string
		psStart time.Time
		psEnd   time.Time
		bsStart time.Time
		bsEnd   time.Time
		want    bool
	}{
		{"identical ranges", base, base.Add(2 * time.Hour), base, base.Add(2 * time.Hour), true},
		{"partial overlap start", base, base.Add(2 * time.Hour), base.Add(-1 * time.Hour), base.Add(1 * time.Hour), true},
		{"partial overlap end", base, base.Add(2 * time.Hour), base.Add(1 * time.Hour), base.Add(3 * time.Hour), true},
		{"contained", base, base.Add(4 * time.Hour), base.Add(1 * time.Hour), base.Add(2 * time.Hour), true},
		{"adjacent no overlap", base, base.Add(2 * time.Hour), base.Add(2 * time.Hour), base.Add(4 * time.Hour), false},
		{"gap", base, base.Add(2 * time.Hour), base.Add(3 * time.Hour), base.Add(5 * time.Hour), false},
		{"bs before ps", base.Add(3 * time.Hour), base.Add(5 * time.Hour), base, base.Add(1 * time.Hour), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TimesOverlap(tt.psStart, tt.psEnd, tt.bsStart, tt.bsEnd)
			if got != tt.want {
				t.Errorf("TimesOverlap(%v, %v, %v, %v) = %v, want %v",
					tt.psStart, tt.psEnd, tt.bsStart, tt.bsEnd, got, tt.want)
			}
		})
	}
}

func TestMatchesLocationConditions(t *testing.T) {
	tests := []struct {
		name       string
		psLocation string
		bsLocation string
		want       bool
	}{
		{"online + zoom", "Online", "Zoom Online Meeting", true},
		{"online lowercase + zoom", "online", "ZOOM ONLINE MEETING", true},
		{"tbd + zoom", "TBD", "Zoom Online Meeting", true},
		{"tbd lowercase + zoom", "tbd", "ZOOM ONLINE MEETING", true},
		{"to be advised + zoom", "To Be Advised", "Zoom Online Meeting", true},
		{"to be advised lowercase + zoom", "to be advised", "ZOOM ONLINE MEETING", true},
		{"tba + zoom", "TBA", "Zoom Online Meeting", true},
		{"campus location", "SIS Building Room 101", "Zoom Online Meeting", false},
		{"tba prefix not exact match", "TBA - To Be Advised", "Zoom Online Meeting", false},
		{"online + non-zoom", "Online", "SIS Building Room 101", false},
		{"empty locations", "", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			psEvent := calendar.Event{Location: tt.psLocation}
			bsEvent := calendar.Event{Location: tt.bsLocation}
			got := MatchesLocationConditions(psEvent, bsEvent)
			if got != tt.want {
				t.Errorf("MatchesLocationConditions(%q, %q) = %v, want %v",
					tt.psLocation, tt.bsLocation, got, tt.want)
			}
		})
	}
}

func TestExtractZoomDetails(t *testing.T) {
	tests := []struct {
		name        string
		description string
		wantLink    string
		wantID      string
		wantPass    string
	}{
		{
			"full zoom link with meeting ID and passcode",
			`<p>Join Zoom Meeting<br>http://zoom.us/j/0000000000?pwd=000000</p>`,
			"http://zoom.us/j/0000000000?pwd=000000",
			"0000000000",
			"000000",
		},
		{
			"meeting ID in link text",
			`<p><a href="http://zoom.us/j/0000000000?pwd=000000">Meeting 0000000000</a></p>`,
			"http://zoom.us/j/0000000000?pwd=000000",
			"0000000000",
			"000000",
		},
		{
			"no zoom link",
			`<p>No meeting scheduled</p>`,
			"",
			"",
			"",
		},
		{
			"empty description",
			"",
			"",
			"",
			"",
		},
		{
			"zoom link without passcode",
			`<p><a href="http://zoom.us/j/1111111111">Join Meeting</a></p>`,
			"http://zoom.us/j/1111111111",
			"1111111111",
			"",
		},
		{
			"multiple zoom links - takes first",
			`<p><a href="http://zoom.us/j/2222222222?pwd=000000">First</a> <a href="http://zoom.us/j/3333333333?pwd=000000">Second</a></p>`,
			"http://zoom.us/j/2222222222?pwd=000000",
			"2222222222",
			"000000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractZoomDetails(tt.description)
			if got.Link != tt.wantLink {
				t.Errorf("ExtractZoomDetails() Link = %q, want %q", got.Link, tt.wantLink)
			}
			if got.MeetingID != tt.wantID {
				t.Errorf("ExtractZoomDetails() MeetingID = %q, want %q", got.MeetingID, tt.wantID)
			}
			if got.Passcode != tt.wantPass {
				t.Errorf("ExtractZoomDetails() Passcode = %q, want %q", got.Passcode, tt.wantPass)
			}
		})
	}
}

func TestMergeEvents(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Singapore")
	base := time.Date(2026, 8, 31, 9, 0, 0, 0, loc)

	psEvent := calendar.Event{
		CourseCode:  "MOD1001",
		Summary:     "MOD1001 - L01 (Tutorial)",
		Location:    "Online",
		Description: "Course: MOD1001\nClass: Sample Module Title\nSection: L01\nType: Tutorial",
		DTStart:     base,
		DTEnd:       base.Add(2 * time.Hour),
	}

	bsEvent := calendar.Event{
		Source:      "calendar",
		Title:       "Tutorial 1",
		OrgUnitName: "MOD1001-Sample Module Title [2026/27 T1]",
		OrgUnitCode: "MOD1001",
		Location:    "Zoom Online Meeting",
		Description: `<p>Join Zoom Meeting<br>http://zoom.us/j/0000000000?pwd=000000</p>`,
		DTStart:     base,
		DTEnd:       base.Add(2 * time.Hour),
	}

	tests := []struct {
		name            string
		psEvents        []calendar.Event
		bsEvents        []calendar.Event
		wantPSCount     int
		wantTotalCount  int
		wantMergedCount int
		wantDescription string
	}{
		{
			"single match",
			[]calendar.Event{psEvent},
			[]calendar.Event{bsEvent},
			1,
			1,
			1,
			"Course: MOD1001\nClass: Sample Module Title\nSection: L01\nType: Tutorial\n\nZoom Meeting Details:\nLink: http://zoom.us/j/0000000000?pwd=000000\nMeeting ID: 0000000000\nPasscode: 000000",
		},
		{
			"no match - different module",
			[]calendar.Event{psEvent},
			[]calendar.Event{{
				Title:       "Tutorial 1",
				OrgUnitName: "COR2003-Software Engineering [2026/27 T1]",
				OrgUnitCode: "COR2003",
				Location:    "Zoom Online Meeting",
				DTStart:     base,
				DTEnd:       base.Add(2 * time.Hour),
			}},
			1,
			2,
			0,
			"",
		},
		{
			"no match - different location",
			[]calendar.Event{{
				CourseCode: "MOD1001",
				Location:   "SIS Building Room 101",
				DTStart:    base,
				DTEnd:      base.Add(2 * time.Hour),
			}},
			[]calendar.Event{bsEvent},
			1,
			2,
			0,
			"",
		},
		{
			"no match - no time overlap",
			[]calendar.Event{psEvent},
			[]calendar.Event{{
				Title:       "Tutorial 1",
				OrgUnitName: "MOD1001-Sample Module Title [2026/27 T1]",
				OrgUnitCode: "MOD1001",
				Location:    "Zoom Online Meeting",
				DTStart:     base.Add(5 * time.Hour),
				DTEnd:       base.Add(7 * time.Hour),
			}},
			1,
			2,
			0,
			"",
		},
		{
			"multiple PS events, one match",
			[]calendar.Event{
				{CourseCode: "MOD1001", Location: "Online", Description: "Course: MOD1001\nClass: \nSection: \nType: ", DTStart: base, DTEnd: base.Add(2 * time.Hour)},
				{CourseCode: "COR2003", Location: "Online", DTStart: base, DTEnd: base.Add(2 * time.Hour)},
			},
			[]calendar.Event{bsEvent},
			2,
			2,
			1,
			"Course: MOD1001\nClass: \nSection: \nType: \n\nZoom Meeting Details:\nLink: http://zoom.us/j/0000000000?pwd=000000\nMeeting ID: 0000000000\nPasscode: 000000",
		},
		{
			"multiple BS events, no PS match",
			[]calendar.Event{psEvent},
			[]calendar.Event{
				{
					Title: "Tutorial 1", OrgUnitName: "COR2003-Software Engineering [2026/27 T1]",
					OrgUnitCode: "COR2003",
					Location:    "Zoom Online Meeting",
					DTStart:     base, DTEnd: base.Add(2 * time.Hour),
				},
				{
					Title: "Tutorial 2", OrgUnitName: "MAT1001-Applied Mathematics [2026/27 T1]",
					OrgUnitCode: "MAT1001",
					Location:    "Zoom Online Meeting",
					DTStart:     base, DTEnd: base.Add(2 * time.Hour),
				},
			},
			1,
			3,
			0,
			"",
		},
		{
			"empty inputs",
			[]calendar.Event{},
			[]calendar.Event{},
			0,
			0,
			0,
			"",
		},
		{
			"merge with Zoom link in href attribute",
			[]calendar.Event{{CourseCode: "MOD1001", Location: "Online", Description: "Course: MOD1001\nClass: \nSection: \nType: ", DTStart: base, DTEnd: base.Add(2 * time.Hour)}},
			[]calendar.Event{{
				Title:       "Tutorial 1",
				OrgUnitName: "MOD1001-Sample Module Title [2026/27 T1]",
				OrgUnitCode: "MOD1001",
				Location:    "Zoom Online Meeting",
				Description: `<p><a href="http://example.zoom.us/j/9102329792?pwd=abc123xyz">Click here to join Zoom Meeting: 910 2329 7921</a></p>`,
				DTStart:     base,
				DTEnd:       base.Add(2 * time.Hour),
			}},
			1,
			1,
			1,
			"Course: MOD1001\nClass: \nSection: \nType: \n\nZoom Meeting Details:\nLink: http://example.zoom.us/j/9102329792?pwd=abc123xyz\nMeeting ID: 9102329792\nPasscode: abc123xyz",
		},
		{
			"no merge when BS description has no Zoom link",
			[]calendar.Event{psEvent},
			[]calendar.Event{{
				Title:       "Tutorial 1",
				OrgUnitName: "MOD1001-Sample Module Title [2026/27 T1]",
				OrgUnitCode: "MOD1001",
				Location:    "Zoom Online Meeting",
				Description: "<p>No Zoom link here</p>",
				DTStart:     base,
				DTEnd:       base.Add(2 * time.Hour),
			}},
			1,
			2,
			0,
			"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, mergedCount := MergeEvents(tt.psEvents, tt.bsEvents)

			if len(result) != tt.wantTotalCount {
				t.Errorf("MergeEvents() total count = %d, want %d", len(result), tt.wantTotalCount)
			}
			if mergedCount != tt.wantMergedCount {
				t.Errorf("MergeEvents() merged count = %d, want %d", mergedCount, tt.wantMergedCount)
			}

			if tt.wantDescription != "" {
				found := false
				for _, e := range result {
					if e.CourseCode == "MOD1001" && e.Location == "Online" {
						if e.Description != tt.wantDescription {
							t.Errorf("merged event description:\ngot:\n%s\n\nwant:\n%s", e.Description, tt.wantDescription)
						}
						found = true
						break
					}
				}
				if !found {
					t.Error("did not find merged MOD1001 online event")
				}
			}
		})
	}
}

func TestMergeEvents_LocationFilter(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Singapore")
	base := time.Date(2026, 8, 31, 9, 0, 0, 0, loc)

	// PS event on campus - should NOT match even if other conditions are met
	campusPS := calendar.Event{
		CourseCode: "MOD1001",
		Location:   "SIS Building Room 101",
		DTStart:    base,
		DTEnd:      base.Add(2 * time.Hour),
	}

	// BS event at Zoom - should NOT merge with campus PS event
	zoomBS := calendar.Event{
		Title:       "Tutorial 1",
		OrgUnitName: "MOD1001-Sample Module Title [2026/27 T1]",
		OrgUnitCode: "MOD1001",
		Location:    "Zoom Online Meeting",
		Description: `<p><a href="http://zoom.us/j/0000000000?pwd=000000">Join</a></p>`,
		DTStart:     base,
		DTEnd:       base.Add(2 * time.Hour),
	}

	result, mergedCount := MergeEvents([]calendar.Event{campusPS}, []calendar.Event{zoomBS})

	if mergedCount != 0 {
		t.Errorf("MergeEvents() merged %d events, expected 0 (campus PS should not match zoom BS)", mergedCount)
	}
	if len(result) != 2 {
		t.Errorf("MergeEvents() returned %d events, expected 2 (both should be separate)", len(result))
	}
}

func TestMergeEvents_OnePSMatchedOnce(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Singapore")
	base := time.Date(2026, 8, 31, 9, 0, 0, 0, loc)

	psEvent := calendar.Event{
		CourseCode: "MOD1001",
		Location:   "Online",
		DTStart:    base,
		DTEnd:      base.Add(2 * time.Hour),
	}

	// Two BS events matching the same PS event
	bsEvent1 := calendar.Event{
		Title:       "Tutorial 1",
		OrgUnitName: "MOD1001-Sample Module Title [2026/27 T1]",
		OrgUnitCode: "MOD1001",
		Location:    "Zoom Online Meeting",
		Description: `<p><a href="http://zoom.us/j/2222222222?pwd=000000">Join</a></p>`,
		DTStart:     base,
		DTEnd:       base.Add(2 * time.Hour),
	}
	bsEvent2 := calendar.Event{
		Title:       "Tutorial 2",
		OrgUnitName: "MOD1001-Sample Module Title [2026/27 T1]",
		OrgUnitCode: "MOD1001",
		Location:    "Zoom Online Meeting",
		Description: `<p><a href="http://zoom.us/j/3333333333?pwd=000000">Join</a></p>`,
		DTStart:     base.Add(30 * time.Minute),
		DTEnd:       base.Add(4 * time.Hour),
	}

	result, mergedCount := MergeEvents([]calendar.Event{psEvent}, []calendar.Event{bsEvent1, bsEvent2})

	if mergedCount != 1 {
		t.Errorf("MergeEvents() merged %d events, expected 1 (PS event can only be matched once)", mergedCount)
	}
	if len(result) != 2 {
		t.Errorf("MergeEvents() returned %d events, expected 2 (1 merged PS + 1 unmatched BS)", len(result))
	}

	// The unmatched BS event should still be in the result
	foundUnmatched := false
	for _, e := range result {
		if e.Title == "Tutorial 2" {
			foundUnmatched = true
		}
	}
	if !foundUnmatched {
		t.Error("expected Tutorial 2 to be in results as unmatched BS event")
	}
}

func TestDedupQuizzes(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Singapore")
	base := time.Date(2026, 10, 18, 15, 59, 59, 0, loc)

	tests := []struct {
		name             string
		calendarEvents   []calendar.Event
		quizEvents       []calendar.Event
		wantTotalCount   int
		wantReplacements int
		wantVerify       func(t *testing.T, result []calendar.Event)
	}{
		{
			name: "basic replacement",
			calendarEvents: []calendar.Event{
				{Summary: "Quiz Event", CalendarEventID: 99001, QuizID: 99002, Source: "brightspace-calendar", DTStart: base, DTEnd: base},
			},
			quizEvents: []calendar.Event{
				{Summary: "Sample Quiz", QuizID: 99002, Source: "brightspace-quizzes", DTStart: base.Add(-1 * time.Hour), DTEnd: base},
			},
			wantTotalCount:   1,
			wantReplacements: 1,
			wantVerify: func(t *testing.T, result []calendar.Event) {
				if result[0].CalendarEventID != 99001 {
					t.Errorf("CalendarEventID = %d, want 99001", result[0].CalendarEventID)
				}
				if result[0].QuizID != 99002 {
					t.Errorf("QuizID = %d, want 99002", result[0].QuizID)
				}
			},
		},
		{
			name: "non-quiz calendar event passthrough",
			calendarEvents: []calendar.Event{
				{Summary: "Lecture", CalendarEventID: 99003, QuizID: 0, Source: "brightspace-calendar", DTStart: base, DTEnd: base},
			},
			quizEvents:       []calendar.Event{},
			wantTotalCount:   1,
			wantReplacements: 0,
			wantVerify: func(t *testing.T, result []calendar.Event) {
				if result[0].CalendarEventID != 99003 {
					t.Errorf("CalendarEventID = %d, want 99003", result[0].CalendarEventID)
				}
				if result[0].QuizID != 0 {
					t.Errorf("QuizID = %d, want 0", result[0].QuizID)
				}
			},
		},
		{
			name:             "unmatched quiz event passthrough",
			calendarEvents:   []calendar.Event{},
			quizEvents: []calendar.Event{
				{Summary: "Sample Quiz", QuizID: 99004, Source: "brightspace-quizzes", DTStart: base.Add(-1 * time.Hour), DTEnd: base},
			},
			wantTotalCount:   1,
			wantReplacements: 0,
			wantVerify: func(t *testing.T, result []calendar.Event) {
				if result[0].QuizID != 99004 {
					t.Errorf("QuizID = %d, want 99004", result[0].QuizID)
				}
				if result[0].CalendarEventID != 0 {
					t.Errorf("CalendarEventID = %d, want 0", result[0].CalendarEventID)
				}
			},
		},
		{
			name:             "empty inputs",
			calendarEvents:   []calendar.Event{},
			quizEvents:       []calendar.Event{},
			wantTotalCount:   0,
			wantReplacements: 0,
		},
		{
			name: "mixed scenario",
			calendarEvents: []calendar.Event{
				{Summary: "Quiz Event A", CalendarEventID: 99010, QuizID: 99011, Source: "brightspace-calendar", DTStart: base, DTEnd: base},
				{Summary: "Lecture", CalendarEventID: 99012, QuizID: 0, Source: "brightspace-calendar", DTStart: base.Add(1 * time.Hour), DTEnd: base.Add(2 * time.Hour)},
				{Summary: "Quiz Event B", CalendarEventID: 99013, QuizID: 99014, Source: "brightspace-calendar", DTStart: base.Add(3 * time.Hour), DTEnd: base.Add(4 * time.Hour)},
			},
			quizEvents: []calendar.Event{
				{Summary: "Quiz A", QuizID: 99011, Source: "brightspace-quizzes", DTStart: base.Add(-1 * time.Hour), DTEnd: base},
				{Summary: "Quiz B", QuizID: 99014, Source: "brightspace-quizzes", DTStart: base.Add(2 * time.Hour), DTEnd: base.Add(3 * time.Hour)},
				{Summary: "Unmatched Quiz", QuizID: 99015, Source: "brightspace-quizzes", DTStart: base.Add(5 * time.Hour), DTEnd: base.Add(6 * time.Hour)},
			},
			wantTotalCount:   4,
			wantReplacements: 2,
			wantVerify: func(t *testing.T, result []calendar.Event) {
				quizIDs := make(map[int]bool)
				nonQuizIDs := make(map[int]bool)
				for _, e := range result {
					if e.QuizID == 0 && e.CalendarEventID > 0 {
						nonQuizIDs[e.CalendarEventID] = true
					} else if e.QuizID > 0 {
						quizIDs[e.QuizID] = true
					}
				}
				if !nonQuizIDs[99012] {
					t.Error("expected non-quiz calendar event with CalendarEventID 99012")
				}
				if len(nonQuizIDs) != 1 {
					t.Errorf("expected 1 non-quiz calendar event, got %d", len(nonQuizIDs))
				}
				if !quizIDs[99011] {
					t.Error("expected matched quiz event with QuizID 99011")
				}
				if !quizIDs[99014] {
					t.Error("expected matched quiz event with QuizID 99014")
				}
				if !quizIDs[99015] {
					t.Error("expected unmatched quiz event with QuizID 99015")
				}
			},
		},
		{
			name: "quiz event has CalendarEventID set on match",
			calendarEvents: []calendar.Event{
				{Summary: "Quiz Event", CalendarEventID: 99005, QuizID: 99006, Source: "brightspace-calendar", DTStart: base, DTEnd: base},
			},
			quizEvents: []calendar.Event{
				{Summary: "Sample Quiz", QuizID: 99006, Source: "brightspace-quizzes", DTStart: base.Add(-1 * time.Hour), DTEnd: base},
			},
			wantTotalCount:   1,
			wantReplacements: 1,
			wantVerify: func(t *testing.T, result []calendar.Event) {
				if result[0].CalendarEventID != 99005 {
					t.Errorf("CalendarEventID = %d, want 99005", result[0].CalendarEventID)
				}
				if result[0].QuizID != 99006 {
					t.Errorf("QuizID = %d, want 99006", result[0].QuizID)
				}
			},
		},
		{
			name: "duplicate quiz IDs in calendar events",
			calendarEvents: []calendar.Event{
				{Summary: "Quiz Event A", CalendarEventID: 99020, QuizID: 99007, Source: "brightspace-calendar", DTStart: base, DTEnd: base},
				{Summary: "Quiz Event B", CalendarEventID: 99021, QuizID: 99007, Source: "brightspace-calendar", DTStart: base.Add(1 * time.Hour), DTEnd: base.Add(2 * time.Hour)},
			},
			quizEvents: []calendar.Event{
				{Summary: "Sample Quiz", QuizID: 99007, Source: "brightspace-quizzes", DTStart: base.Add(-1 * time.Hour), DTEnd: base},
			},
			wantTotalCount:   1,
			wantReplacements: 1,
			wantVerify: func(t *testing.T, result []calendar.Event) {
				// Both calendar events have QuizID > 0, so both are removed.
				// The quiz event should have CalendarEventID set to whichever calendar event was last in the map iteration.
				if len(result) != 1 {
					t.Fatalf("expected 1 result, got %d", len(result))
				}
				if result[0].QuizID != 99007 {
					t.Errorf("QuizID = %d, want 99007", result[0].QuizID)
				}
				if result[0].CalendarEventID != 99020 && result[0].CalendarEventID != 99021 {
					t.Errorf("CalendarEventID = %d, want 99020 or 99021", result[0].CalendarEventID)
				}
			},
		},
		{
			name: "multiple matches",
			calendarEvents: []calendar.Event{
				{Summary: "Quiz Event A", CalendarEventID: 99030, QuizID: 99031, Source: "brightspace-calendar", DTStart: base, DTEnd: base},
				{Summary: "Lecture", CalendarEventID: 99032, QuizID: 0, Source: "brightspace-calendar", DTStart: base.Add(1 * time.Hour), DTEnd: base.Add(2 * time.Hour)},
				{Summary: "Quiz Event B", CalendarEventID: 99033, QuizID: 99034, Source: "brightspace-calendar", DTStart: base.Add(3 * time.Hour), DTEnd: base.Add(4 * time.Hour)},
				{Summary: "Tutorial", CalendarEventID: 99035, QuizID: 0, Source: "brightspace-calendar", DTStart: base.Add(5 * time.Hour), DTEnd: base.Add(6 * time.Hour)},
			},
			quizEvents: []calendar.Event{
				{Summary: "Quiz A", QuizID: 99031, Source: "brightspace-quizzes", DTStart: base.Add(-1 * time.Hour), DTEnd: base},
				{Summary: "Quiz B", QuizID: 99034, Source: "brightspace-quizzes", DTStart: base.Add(2 * time.Hour), DTEnd: base.Add(3 * time.Hour)},
			},
			wantTotalCount:   4,
			wantReplacements: 2,
			wantVerify: func(t *testing.T, result []calendar.Event) {
				var nonQuizCount, matchedQuizCount int
				for _, e := range result {
					if e.QuizID == 0 && e.CalendarEventID > 0 {
						nonQuizCount++
					} else if e.QuizID > 0 && e.CalendarEventID > 0 {
						matchedQuizCount++
					}
				}
				if nonQuizCount != 2 {
					t.Errorf("expected 2 non-quiz calendar events, got %d", nonQuizCount)
				}
				if matchedQuizCount != 2 {
					t.Errorf("expected 2 matched quiz events, got %d", matchedQuizCount)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, replacements := DedupQuizzes(tt.calendarEvents, tt.quizEvents)
			if len(result) != tt.wantTotalCount {
				t.Errorf("DedupQuizzes() total count = %d, want %d", len(result), tt.wantTotalCount)
			}
			if replacements != tt.wantReplacements {
				t.Errorf("DedupQuizzes() replacements = %d, want %d", replacements, tt.wantReplacements)
			}
			if tt.wantVerify != nil {
				tt.wantVerify(t, result)
			}
		})
	}
}
