package brightspace

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestAPIToStringEntry_SetsCalendarEventIDAndQuizID(t *testing.T) {
	ev := CalendarEventAPI{
		CalendarEventId: 99001,
		OrgUnitId:       12345,
		Title:           "Quiz Event",
		StartDateTime:   "2026-09-15T10:00:00Z",
		EndDateTime:     "2026-09-15T11:00:00Z",
		OrgUnitName:     "MOD1001-Sample Module",
		OrgUnitCode:     "MOD1001",
		QuizId:          99010,
		AssociatedEntity: &AssociatedEntity{
			AssociatedEntityType: "D2L.LE.Quizzing.Quiz",
			AssociatedEntityId:   99002,
		},
	}

	entry := APIToStringEntry(ev, "calendar")

	if entry.CalendarEventID != 99001 {
		t.Errorf("CalendarEventID = %d, want 99001", entry.CalendarEventID)
	}
	if entry.QuizID != 99002 {
		t.Errorf("QuizID = %d, want 99002", entry.QuizID)
	}
	if entry.Title != "Quiz Event" {
		t.Errorf("Title = %q, want %q", entry.Title, "Quiz Event")
	}
}

func TestAPIToStringEntry_ZeroIDs(t *testing.T) {
	ev := CalendarEventAPI{
		CalendarEventId: 99006,
		OrgUnitId:       12346,
		Title:           "Regular Event",
		StartDateTime:   "2026-09-15T09:00:00Z",
		EndDateTime:     "2026-09-15T10:00:00Z",
		OrgUnitName:     "MOD1002-Regular Module",
		OrgUnitCode:     "MOD1002",
		QuizId:          0,
	}

	entry := APIToStringEntry(ev, "calendar")

	if entry.CalendarEventID != 99006 {
		t.Errorf("CalendarEventID = %d, want 99006", entry.CalendarEventID)
	}
	if entry.QuizID != 0 {
		t.Errorf("QuizID = %d, want 0", entry.QuizID)
	}
}

func TestQuizToStringEntry_SetsQuizID(t *testing.T) {
	quiz := QuizAPI{
		QuizId:      99020,
		Name:        "Practice Quiz",
		StartDate:   "2026-09-15T10:00:00Z",
		DueDate:     "2026-09-15T10:30:00Z",
		OrgUnitId:   "12345",
		OrgUnitName: "MOD1001-Sample Module",
		OrgUnitCode: "MOD1001",
	}

	entry := QuizToStringEntry(quiz)

	if entry.QuizID != 99020 {
		t.Errorf("QuizID = %d, want 99020", entry.QuizID)
	}
	if entry.CalendarEventID != 0 {
		t.Errorf("CalendarEventID = %d, want 0", entry.CalendarEventID)
	}
	if entry.Title != "Practice Quiz" {
		t.Errorf("Title = %q, want %q", entry.Title, "Practice Quiz")
	}
}

func TestEntriesToEvents_PassesCalendarEventIDAndQuizID(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Singapore")
	blocklist := &Blocklist{}

	entries := []BrightSpaceStringEntry{
		{
			Title:           "Quiz Event",
			OrgUnitId:       "12345",
			OrgUnitName:     "MOD1001",
			OrgUnitCode:     "MOD1001",
			DTStart:         "2026-09-15T10:00:00Z",
			DTEnd:           "2026-09-15T11:00:00Z",
			Source:          "calendar",
			CalendarEventID: 99001,
			QuizID:          99010,
		},
	}

	events := EntriesToEvents(entries, blocklist, loc)

	if len(events) != 1 {
		t.Fatalf("Expected 1 event, got %d", len(events))
	}

	event := events[0]
	if event.CalendarEventID != 99001 {
		t.Errorf("CalendarEventID = %d, want 99001", event.CalendarEventID)
	}
	if event.QuizID != 99010 {
		t.Errorf("QuizID = %d, want 99010", event.QuizID)
	}
}

func TestQuizEntriesToEvents_PassesQuizID(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Singapore")
	blocklist := &Blocklist{}

	entries := []BrightSpaceStringEntry{
		{
			Title:           "Quiz Due",
			OrgUnitId:       "12345",
			OrgUnitName:     "MOD1001",
			OrgUnitCode:     "MOD1001",
			DTStart:         "2026-09-15T10:00:00Z",
			DTEnd:           "2026-09-15T11:00:00Z",
			Source:          "brightspace-quizzes",
			CalendarEventID: 0,
			QuizID:          99020,
		},
	}

	events := QuizEntriesToEvents(entries, blocklist, loc)

	if len(events) != 1 {
		t.Fatalf("Expected 1 event, got %d", len(events))
	}

	event := events[0]
	if event.QuizID != 99020 {
		t.Errorf("QuizID = %d, want 99020", event.QuizID)
	}
	if event.CalendarEventID != 0 {
		t.Errorf("CalendarEventID = %d, want 0", event.CalendarEventID)
	}
}

func TestEntriesToEvents_ZeroIDsPassthrough(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Singapore")
	blocklist := &Blocklist{}

	entries := []BrightSpaceStringEntry{
		{
			Title:           "Regular Event",
			OrgUnitId:       "12346",
			OrgUnitName:     "MOD1002",
			OrgUnitCode:     "MOD1002",
			DTStart:         "2026-09-15T09:00:00Z",
			DTEnd:           "2026-09-15T10:00:00Z",
			Source:          "calendar",
			CalendarEventID: 0,
			QuizID:          0,
		},
	}

	events := EntriesToEvents(entries, blocklist, loc)

	if len(events) != 1 {
		t.Fatalf("Expected 1 event, got %d", len(events))
	}

	event := events[0]
	if event.CalendarEventID != 0 {
		t.Errorf("CalendarEventID = %d, want 0", event.CalendarEventID)
	}
	if event.QuizID != 0 {
		t.Errorf("QuizID = %d, want 0", event.QuizID)
	}
}

func TestEntriesToEvents_EventStructFields(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Singapore")
	blocklist := &Blocklist{}

	entries := []BrightSpaceStringEntry{
		{
			Title:           "Full Event",
			OrgUnitId:       "12362",
			OrgUnitName:     "MOD1017",
			OrgUnitCode:     "MOD1017",
			Location:        "Room 201",
			Description:     "Full description",
			DTStart:         "2026-09-30T10:00:00Z",
			DTEnd:           "2026-09-30T11:00:00Z",
			Source:          "calendar",
			CalendarEventID: 99034,
			QuizID:          99035,
		},
	}

	events := EntriesToEvents(entries, blocklist, loc)

	if len(events) != 1 {
		t.Fatalf("Expected 1 event, got %d", len(events))
	}

	event := events[0]
	if event.CourseCode != "MOD1017" {
		t.Errorf("CourseCode = %q, want %q", event.CourseCode, "MOD1017")
	}
	if event.Summary != "[MOD1017] Full Event" {
		t.Errorf("Summary = %q, want %q", event.Summary, "[MOD1017] Full Event")
	}
	if event.Title != "Full Event" {
		t.Errorf("Title = %q, want %q", event.Title, "Full Event")
	}
	if event.OrgUnitID != "12362" {
		t.Errorf("OrgUnitID = %q, want %q", event.OrgUnitID, "12362")
	}
	if event.OrgUnitName != "MOD1017" {
		t.Errorf("OrgUnitName = %q, want %q", event.OrgUnitName, "MOD1017")
	}
	if event.OrgUnitCode != "MOD1017" {
		t.Errorf("OrgUnitCode = %q, want %q", event.OrgUnitCode, "MOD1017")
	}
	if event.Location != "Room 201" {
		t.Errorf("Location = %q, want %q", event.Location, "Room 201")
	}
	if event.Description != "Full description" {
		t.Errorf("Description = %q, want %q", event.Description, "Full description")
	}
	if event.Source != "calendar" {
		t.Errorf("Source = %q, want %q", event.Source, "calendar")
	}
	if event.CalendarEventID != 99034 {
		t.Errorf("CalendarEventID = %d, want 99034", event.CalendarEventID)
	}
	if event.QuizID != 99035 {
		t.Errorf("QuizID = %d, want 99035", event.QuizID)
	}
}

func TestCalendarEventAPI_JSONRoundTrip(t *testing.T) {
	original := CalendarEventAPI{
		CalendarEventId: 99030,
		OrgUnitId:       12360,
		Title:           "Round Trip Test",
		StartDateTime:   "2026-09-28T10:00:00Z",
		EndDateTime:     "2026-09-28T11:00:00Z",
		OrgUnitName:     "MOD1015-Round Trip",
		OrgUnitCode:     "MOD1015",
		QuizId:          99031,
		AssociatedEntity: &AssociatedEntity{
			AssociatedEntityType: "D2L.LE.Quizzing.Quiz",
			AssociatedEntityId:   99032,
			Link:                 "/d2l/le/quizzing/99032",
		},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var decoded CalendarEventAPI
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if decoded.CalendarEventId != original.CalendarEventId {
		t.Errorf("CalendarEventId = %d, want %d", decoded.CalendarEventId, original.CalendarEventId)
	}
	if decoded.QuizId != original.QuizId {
		t.Errorf("QuizId = %d, want %d", decoded.QuizId, original.QuizId)
	}
	if decoded.AssociatedEntity == nil {
		t.Fatal("AssociatedEntity should not be nil after round trip")
	}
	if decoded.AssociatedEntity.AssociatedEntityType != original.AssociatedEntity.AssociatedEntityType {
		t.Errorf("AssociatedEntityType = %q, want %q", decoded.AssociatedEntity.AssociatedEntityType, original.AssociatedEntity.AssociatedEntityType)
	}
}

func TestCalendarEventAPI_JSONOmitEmpty(t *testing.T) {
	original := CalendarEventAPI{
		CalendarEventId: 99033,
		OrgUnitId:       12361,
		Title:           "No Associated Entity",
		StartDateTime:   "2026-09-29T10:00:00Z",
		EndDateTime:     "2026-09-29T11:00:00Z",
		OrgUnitName:     "MOD1016",
		OrgUnitCode:     "MOD1016",
		QuizId:          0,
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	content := string(data)
	if containsStr(content, "AssociatedEntity") {
		t.Error("AssociatedEntity should be omitted when nil")
	}
}

func TestEntriesToEvents_MultipleEventsWithIDs(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Singapore")
	blocklist := &Blocklist{}

	entries := []BrightSpaceStringEntry{
		{
			Title:           "Event 1",
			OrgUnitId:       "12354",
			OrgUnitName:     "MOD1010",
			OrgUnitCode:     "MOD1010",
			DTStart:         "2026-09-22T10:00:00Z",
			DTEnd:           "2026-09-22T11:00:00Z",
			Source:          "calendar",
			CalendarEventID: 99013,
			QuizID:          99026,
		},
		{
			Title:           "Event 2",
			OrgUnitId:       "12355",
			OrgUnitName:     "MOD1011",
			OrgUnitCode:     "MOD1011",
			DTStart:         "2026-09-23T10:00:00Z",
			DTEnd:           "2026-09-23T11:00:00Z",
			Source:          "calendar",
			CalendarEventID: 99014,
			QuizID:          0,
		},
		{
			Title:           "Quiz Event",
			OrgUnitId:       "12356",
			OrgUnitName:     "MOD1012",
			OrgUnitCode:     "MOD1012",
			DTStart:         "2026-09-24T10:00:00Z",
			DTEnd:           "2026-09-24T11:00:00Z",
			Source:          "brightspace-quizzes",
			CalendarEventID: 0,
			QuizID:          99027,
		},
	}

	events := EntriesToEvents(entries, blocklist, loc)

	if len(events) != 3 {
		t.Fatalf("Expected 3 events, got %d", len(events))
	}

	if events[0].CalendarEventID != 99013 || events[0].QuizID != 99026 {
		t.Errorf("Event 0: CalendarEventID=%d, QuizID=%d, want 99013, 99026", events[0].CalendarEventID, events[0].QuizID)
	}
	if events[1].CalendarEventID != 99014 || events[1].QuizID != 0 {
		t.Errorf("Event 1: CalendarEventID=%d, QuizID=%d, want 99014, 0", events[1].CalendarEventID, events[1].QuizID)
	}
	if events[2].CalendarEventID != 0 || events[2].QuizID != 99027 {
		t.Errorf("Event 2: CalendarEventID=%d, QuizID=%d, want 0, 99027", events[2].CalendarEventID, events[2].QuizID)
	}
}

func TestAPIToStringEntry_SourceCalendar(t *testing.T) {
	ev := CalendarEventAPI{
		CalendarEventId: 99129,
		OrgUnitId:       12423,
		Title:           "Source Calendar",
		StartDateTime:   "2026-12-01T10:00:00Z",
		EndDateTime:     "2026-12-01T11:00:00Z",
		OrgUnitName:     "MOD1077",
		OrgUnitCode:     "MOD1077",
		AssociatedEntity: &AssociatedEntity{
			AssociatedEntityType: "D2L.LE.Quizzing.Quiz",
			AssociatedEntityId:   99130,
		},
	}

	entry := APIToStringEntry(ev, "calendar")

	if entry.Source != "calendar" {
		t.Errorf("Source = %q, want %q", entry.Source, "calendar")
	}
	if entry.CalendarEventID != 99129 {
		t.Errorf("CalendarEventID = %d, want 99129", entry.CalendarEventID)
	}
	if entry.QuizID != 99130 {
		t.Errorf("QuizID = %d, want 99130", entry.QuizID)
	}
}

func TestEntriesToEvents_SourcePreserved(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Singapore")
	blocklist := &Blocklist{}

	entries := []BrightSpaceStringEntry{
		{
			Title:           "Source Calendar",
			OrgUnitId:       "12425",
			OrgUnitName:     "MOD1079",
			OrgUnitCode:     "MOD1079",
			DTStart:         "2026-12-03T10:00:00Z",
			DTEnd:           "2026-12-03T11:00:00Z",
			Source:          "calendar",
			CalendarEventID: 99133,
			QuizID:          99134,
		},
		{
			Title:           "Source Dropbox",
			OrgUnitId:       "12426",
			OrgUnitName:     "MOD1080",
			OrgUnitCode:     "MOD1080",
			DTStart:         "2026-12-04T10:00:00Z",
			DTEnd:           "2026-12-04T11:00:00Z",
			Source:          "brightspace-dropbox",
			CalendarEventID: 99135,
			QuizID:          0,
		},
		{
			Title:           "Source Quizzes",
			OrgUnitId:       "12427",
			OrgUnitName:     "MOD1081",
			OrgUnitCode:     "MOD1081",
			DTStart:         "2026-12-05T10:00:00Z",
			DTEnd:           "2026-12-05T11:00:00Z",
			Source:          "brightspace-quizzes",
			CalendarEventID: 0,
			QuizID:          99136,
		},
	}

	events := EntriesToEvents(entries, blocklist, loc)

	if len(events) != 3 {
		t.Fatalf("Expected 3 events, got %d", len(events))
	}

	if events[0].Source != "calendar" {
		t.Errorf("Event 0 Source = %q, want %q", events[0].Source, "calendar")
	}
	if events[1].Source != "brightspace-dropbox" {
		t.Errorf("Event 1 Source = %q, want %q", events[1].Source, "brightspace-dropbox")
	}
	if events[2].Source != "brightspace-quizzes" {
		t.Errorf("Event 2 Source = %q, want %q", events[2].Source, "brightspace-quizzes")
	}

	if events[0].CalendarEventID != 99133 || events[0].QuizID != 99134 {
		t.Errorf("Event 0: CalendarEventID=%d, QuizID=%d", events[0].CalendarEventID, events[0].QuizID)
	}
	if events[1].CalendarEventID != 99135 || events[1].QuizID != 0 {
		t.Errorf("Event 1: CalendarEventID=%d, QuizID=%d", events[1].CalendarEventID, events[1].QuizID)
	}
	if events[2].CalendarEventID != 0 || events[2].QuizID != 99136 {
		t.Errorf("Event 2: CalendarEventID=%d, QuizID=%d", events[2].CalendarEventID, events[2].QuizID)
	}
}

func TestQuizEntriesToEvents_SourcePreserved(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Singapore")
	blocklist := &Blocklist{}

	entries := []BrightSpaceStringEntry{
		{
			Title:           "Quiz Source Preserved",
			OrgUnitId:       "12428",
			OrgUnitName:     "MOD1082",
			OrgUnitCode:     "MOD1082",
			DTStart:         "2026-12-06T10:00:00Z",
			DTEnd:           "2026-12-06T11:00:00Z",
			Source:          "brightspace-quizzes",
			CalendarEventID: 0,
			QuizID:          99137,
		},
	}

	events := QuizEntriesToEvents(entries, blocklist, loc)

	if len(events) != 1 {
		t.Fatalf("Expected 1 event, got %d", len(events))
	}

	event := events[0]
	if event.Source != "brightspace-quizzes" {
		t.Errorf("Source = %q, want %q", event.Source, "brightspace-quizzes")
	}
	if event.QuizID != 99137 {
		t.Errorf("QuizID = %d, want 99137", event.QuizID)
	}
	if event.CalendarEventID != 0 {
		t.Errorf("CalendarEventID = %d, want 0", event.CalendarEventID)
	}
}

func TestEntriesToEvents_BlocklistWithIDs(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Singapore")
	blocklist := &Blocklist{
		CourseCodePatterns: []string{"MOD9999"},
	}
	blocklist.CompilePatterns()

	entries := []BrightSpaceStringEntry{
		{
			Title:           "Blocked Event",
			OrgUnitId:       "12357",
			OrgUnitName:     "MOD9999",
			OrgUnitCode:     "MOD9999",
			DTStart:         "2026-09-25T10:00:00Z",
			DTEnd:           "2026-09-25T11:00:00Z",
			Source:          "calendar",
			CalendarEventID: 99015,
			QuizID:          99028,
		},
		{
			Title:           "Allowed Event",
			OrgUnitId:       "12358",
			OrgUnitName:     "MOD1013",
			OrgUnitCode:     "MOD1013",
			DTStart:         "2026-09-26T10:00:00Z",
			DTEnd:           "2026-09-26T11:00:00Z",
			Source:          "calendar",
			CalendarEventID: 99016,
			QuizID:          0,
		},
	}

	events := EntriesToEvents(entries, blocklist, loc)

	if len(events) != 1 {
		t.Fatalf("Expected 1 event (blocked one filtered), got %d", len(events))
	}
	if events[0].CalendarEventID != 99016 {
		t.Errorf("CalendarEventID = %d, want 99016", events[0].CalendarEventID)
	}
}

func TestQuizEntriesToEvents_PreservesAllFields(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Singapore")
	blocklist := &Blocklist{}

	entries := []BrightSpaceStringEntry{
		{
			Title:           "Comprehensive Quiz",
			OrgUnitId:       "12359",
			OrgUnitName:     "MOD1014",
			OrgUnitCode:     "MOD1014",
			Location:        "Online",
			Description:     "Quiz description",
			DTStart:         "2026-09-27T10:00:00Z",
			DTEnd:           "2026-09-27T11:00:00Z",
			Source:          "brightspace-quizzes",
			CalendarEventID: 0,
			QuizID:          99029,
		},
	}

	events := QuizEntriesToEvents(entries, blocklist, loc)

	if len(events) != 1 {
		t.Fatalf("Expected 1 event, got %d", len(events))
	}

	event := events[0]
	if event.QuizID != 99029 {
		t.Errorf("QuizID = %d, want 99029", event.QuizID)
	}
	if event.CalendarEventID != 0 {
		t.Errorf("CalendarEventID = %d, want 0", event.CalendarEventID)
	}
	if event.Title != "Comprehensive Quiz" {
		t.Errorf("Title = %q, want %q", event.Title, "Comprehensive Quiz")
	}
	if event.Location != "Online" {
		t.Errorf("Location = %q, want %q", event.Location, "Online")
	}
}

func TestAPIToStringEntry_WithRecurrenceInfo(t *testing.T) {
	ev := CalendarEventAPI{
		CalendarEventId: 212823,
		OrgUnitId:       213426,
		Title:           "Test Recurring Event",
		StartDateTime:   "2026-09-04T01:00:00.000Z",
		EndDateTime:     "2026-09-04T03:00:00.000Z",
		OrgUnitName:     "MOD1001-Sample Module",
		OrgUnitCode:     "MOD1001",
		IsRecurring:     true,
		RecurrenceInfo: &RecurrenceInfo{
			RepeatType:      3,
			RepeatEvery:     1,
			RepeatOnInfo:    &RepeatOnInfo{Friday: true},
			RepeatUntilDate: "2026-10-02T01:00:00.000Z",
		},
	}

	entry := APIToStringEntry(ev, "brightspace-calendar")

	if !entry.IsRecurring {
		t.Error("IsRecurring should be true")
	}
	if entry.RepeatType != 3 {
		t.Errorf("RepeatType = %d, want 3", entry.RepeatType)
	}
	if entry.RepeatEvery != 1 {
		t.Errorf("RepeatEvery = %d, want 1", entry.RepeatEvery)
	}
	if entry.RepeatOnInfo == nil || entry.RepeatOnInfo.RepeatOnInfo == nil || !entry.RepeatOnInfo.RepeatOnInfo.Friday {
		t.Error("RepeatOnInfo.Friday should be true")
	}
	if entry.RepeatUntilDateString != "2026-10-02T01:00:00.000Z" {
		t.Errorf("RepeatUntilDateString = %q, want %q", entry.RepeatUntilDateString, "2026-10-02T01:00:00.000Z")
	}
}

func TestAPIToStringEntry_WithoutRecurrenceInfo(t *testing.T) {
	ev := CalendarEventAPI{
		CalendarEventId: 12345,
		OrgUnitId:       12345,
		Title:           "Non-Recurring Event",
		StartDateTime:   "2026-09-04T01:00:00.000Z",
		EndDateTime:     "2026-09-04T03:00:00.000Z",
		OrgUnitName:     "MOD1002",
		OrgUnitCode:     "MOD1002",
		IsRecurring:     false,
	}

	entry := APIToStringEntry(ev, "brightspace-calendar")

	if entry.IsRecurring {
		t.Error("IsRecurring should be false")
	}
	if entry.RepeatType != 1 {
		t.Errorf("RepeatType = %d, want 1 (default None)", entry.RepeatType)
	}
	if entry.RepeatEvery != 0 {
		t.Errorf("RepeatEvery = %d, want 0", entry.RepeatEvery)
	}
	if entry.RepeatUntilDateString != "" {
		t.Errorf("RepeatUntilDateString = %q, want empty", entry.RepeatUntilDateString)
	}
}

func TestParseEntryTimes_ZeroDuration(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Singapore")

	entry := &BrightSpaceStringEntry{
		DTStart: "2026-09-15T14:00:00Z",
		DTEnd:   "2026-09-15T14:00:00Z",
		IsAllDay: false,
	}

	dtStart, dtEnd, err := parseEntryTimes(entry, loc)
	if err != nil {
		t.Fatalf("parseEntryTimes() error = %v", err)
	}

	if !dtStart.Equal(dtEnd.Add(-1 * time.Hour)) {
		t.Errorf("DTSTART = %v, want DTEND - 1h (%v)", dtStart, dtEnd.Add(-1*time.Hour))
	}
	if !dtEnd.Equal(dtStart.Add(1 * time.Hour)) {
		t.Errorf("DTEnd = %v, want DTSTART + 1h (%v)", dtEnd, dtStart.Add(1*time.Hour))
	}
}

func TestParseEntryTimes_AllDayNoAdjustment(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Singapore")

	entry := &BrightSpaceStringEntry{
		DTStart:  "2026-09-15T00:00:00Z",
		DTEnd:    "2026-09-15T00:00:00Z",
		IsAllDay: true,
	}

	dtStart, dtEnd, err := parseEntryTimes(entry, loc)
	if err != nil {
		t.Fatalf("parseEntryTimes() error = %v", err)
	}

	if !dtEnd.Equal(dtStart.Add(24 * time.Hour)) {
		t.Errorf("DTEnd = %v, want DTSTART + 24h (%v)", dtEnd, dtStart.Add(24*time.Hour))
	}
}

func TestParseEntryTimes_NonZeroDurationNoAdjustment(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Singapore")

	entry := &BrightSpaceStringEntry{
		DTStart:  "2026-09-15T10:00:00Z",
		DTEnd:    "2026-09-15T11:30:00Z",
		IsAllDay: false,
	}

	dtStart, dtEnd, err := parseEntryTimes(entry, loc)
	if err != nil {
		t.Fatalf("parseEntryTimes() error = %v", err)
	}

	expectedStart, _ := time.Parse(time.RFC3339, "2026-09-15T10:00:00Z")
	expectedEnd, _ := time.Parse(time.RFC3339, "2026-09-15T11:30:00Z")

	if !dtStart.Equal(expectedStart) {
		t.Errorf("DTSTART = %v, want %v", dtStart, expectedStart)
	}
	if !dtEnd.Equal(expectedEnd) {
		t.Errorf("DTEnd = %v, want %v", dtEnd, expectedEnd)
	}
}

func TestParseEntryTimes_EmptyDTStartUsesDTEnd(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Singapore")

	entry := &BrightSpaceStringEntry{
		DTStart: "",
		DTEnd:   "2026-09-15T14:00:00Z",
		IsAllDay: false,
	}

	dtStart, dtEnd, err := parseEntryTimes(entry, loc)
	if err != nil {
		t.Fatalf("parseEntryTimes() error = %v", err)
	}

	expectedEnd, _ := time.Parse(time.RFC3339, "2026-09-15T14:00:00Z")
	if !dtEnd.Equal(expectedEnd) {
		t.Errorf("DTEnd = %v, want %v", dtEnd, expectedEnd)
	}
	if !dtStart.Equal(dtEnd.Add(-1 * time.Hour)) {
		t.Errorf("DTSTART = %v, want DTEND - 1h (%v)", dtStart, dtEnd.Add(-1*time.Hour))
	}
}

func TestHtmlToPlainText_NumericEntities(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"hello&#160;world", "hello world"},
		{"test&#39;s", "test's"},
		{"&#169; 2026", "\u00a9 2026"},
		{"a&amp;b", "a&b"},
		{"hello&lt;world&gt;", "hello<world>"},
		{"&#8211; dash", "\u2013 dash"},
		{"&#8230; ellipsis", "\u2026 ellipsis"},
	}
	for _, tc := range tests {
		got := htmlToPlainText(tc.input)
		if got != tc.want {
			t.Errorf("htmlToPlainText(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestHtmlToPlainText_RandomEntitiesStripped(t *testing.T) {
	input := "test&#99999;remaining&#xyz;end"
	got := htmlToPlainText(input)
	// Entities should be stripped, not left as malformed &#...;
	if strings.Contains(got, "&#") {
		t.Errorf("htmlToPlainText should strip remaining entities, got %q", got)
	}
}
