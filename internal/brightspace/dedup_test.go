package brightspace

import (
	"testing"
	"time"
)

func TestDeduplicateCalendarEntries_QuizThreeEventTypes(t *testing.T) {
	entries := []CalendarEventAPI{
		{
			CalendarEventId: 1,
			OrgUnitId:       12345,
			Title:           "Quiz 1",
			Description:     "Availability starts",
			StartDateTime:   "2026-09-15T09:00:00Z",
			EndDateTime:     "2026-09-15T09:05:00Z",
			OrgUnitCode:     "MOD1001",
			OrgUnitName:     "MOD1001 Sample Module",
			EventType:       EventTypeAvailabilityStarts,
			AssociatedEntity: &AssociatedEntity{
				AssociatedEntityType: "D2L.LE.Quizzing.Quiz",
				AssociatedEntityId:   99001,
			},
		},
		{
			CalendarEventId: 2,
			OrgUnitId:       12345,
			Title:           "Quiz 1",
			Description:     "Due date",
			StartDateTime:   "2026-09-15T10:00:00Z",
			EndDateTime:     "2026-09-15T10:30:00Z",
			OrgUnitCode:     "MOD1001",
			OrgUnitName:     "MOD1001 Sample Module",
			EventType:       EventTypeDueDate,
			AssociatedEntity: &AssociatedEntity{
				AssociatedEntityType: "D2L.LE.Quizzing.Quiz",
				AssociatedEntityId:   99001,
			},
		},
		{
			CalendarEventId: 3,
			OrgUnitId:       12345,
			Title:           "Quiz 1",
			Description:     "Availability ends",
			StartDateTime:   "2026-09-15T10:30:00Z",
			EndDateTime:     "2026-09-15T11:00:00Z",
			OrgUnitCode:     "MOD1001",
			OrgUnitName:     "MOD1001 Sample Module",
			EventType:       EventTypeAvailabilityEnds,
			AssociatedEntity: &AssociatedEntity{
				AssociatedEntityType: "D2L.LE.Quizzing.Quiz",
				AssociatedEntityId:   99001,
			},
		},
	}

	result := DeduplicateCalendarEntries(entries)

	if len(result) != 1 {
		t.Fatalf("Expected 1 entry, got %d", len(result))
	}

	ev := result[0]
	if ev.StartDateTime != "2026-09-15T09:00:00Z" {
		t.Errorf("StartDateTime = %q, want %q", ev.StartDateTime, "2026-09-15T09:00:00Z")
	}
	if ev.EndDateTime != "2026-09-15T11:00:00Z" {
		t.Errorf("EndDateTime = %q, want %q", ev.EndDateTime, "2026-09-15T11:00:00Z")
	}
	if ev.EventType != EventTypeDueDate {
		t.Errorf("EventType = %d, want %d (DueDate)", ev.EventType, EventTypeDueDate)
	}
	if ev.Description != "Due date" {
		t.Errorf("Description = %q, want %q", ev.Description, "Due date")
	}
}

func TestDeduplicateCalendarEntries_AssignmentTwoEventTypes(t *testing.T) {
	entries := []CalendarEventAPI{
		{
			CalendarEventId: 10,
			OrgUnitId:       12346,
			Title:           "Assignment 1",
			Description:     "Due date desc",
			StartDateTime:   "2026-09-20T10:00:00Z",
			EndDateTime:     "2026-09-20T10:30:00Z",
			OrgUnitCode:     "MOD1002",
			OrgUnitName:     "MOD1002 Another Module",
			EventType:       EventTypeDueDate,
			AssociatedEntity: &AssociatedEntity{
				AssociatedEntityType: "D2L.LE.Dropbox.Folder",
				AssociatedEntityId:   99010,
			},
		},
		{
			CalendarEventId: 11,
			OrgUnitId:       12346,
			Title:           "Assignment 1",
			Description:     "Availability ends desc",
			StartDateTime:   "2026-09-20T10:30:00Z",
			EndDateTime:     "2026-09-20T11:00:00Z",
			OrgUnitCode:     "MOD1002",
			OrgUnitName:     "MOD1002 Another Module",
			EventType:       EventTypeAvailabilityEnds,
			AssociatedEntity: &AssociatedEntity{
				AssociatedEntityType: "D2L.LE.Dropbox.Folder",
				AssociatedEntityId:   99010,
			},
		},
	}

	result := DeduplicateCalendarEntries(entries)

	if len(result) != 1 {
		t.Fatalf("Expected 1 entry, got %d", len(result))
	}

	ev := result[0]
	if ev.StartDateTime != "2026-09-20T10:00:00Z" {
		t.Errorf("StartDateTime = %q, want %q", ev.StartDateTime, "2026-09-20T10:00:00Z")
	}
	if ev.EndDateTime != "2026-09-20T11:00:00Z" {
		t.Errorf("EndDateTime = %q, want %q", ev.EndDateTime, "2026-09-20T11:00:00Z")
	}
	if ev.EventType != EventTypeDueDate {
		t.Errorf("EventType = %d, want %d", ev.EventType, EventTypeDueDate)
	}
}

func TestDeduplicateCalendarEntries_SingleEventPassThrough(t *testing.T) {
	entries := []CalendarEventAPI{
		{
			CalendarEventId: 20,
			OrgUnitId:       12347,
			Title:           "Single Event",
			Description:     "Only one",
			StartDateTime:   "2026-09-25T14:00:00Z",
			EndDateTime:     "2026-09-25T15:00:00Z",
			OrgUnitCode:     "MOD1003",
			OrgUnitName:     "MOD1003 Single Module",
			EventType:       EventTypeReminder,
		},
	}

	result := DeduplicateCalendarEntries(entries)

	if len(result) != 1 {
		t.Fatalf("Expected 1 entry, got %d", len(result))
	}
	if result[0].CalendarEventId != 20 {
		t.Errorf("CalendarEventId = %d, want 20", result[0].CalendarEventId)
	}
}

func TestDeduplicateCalendarEntries_DifferentEntitiesSameTitle(t *testing.T) {
	entries := []CalendarEventAPI{
		{
			CalendarEventId: 30,
			OrgUnitId:       12348,
			Title:           "Quiz 2",
			Description:     "Entity A",
			StartDateTime:   "2026-09-15T10:00:00Z",
			EndDateTime:     "2026-09-15T11:00:00Z",
			OrgUnitCode:     "MOD1004",
			OrgUnitName:     "MOD1004 Module",
			EventType:       EventTypeDueDate,
			AssociatedEntity: &AssociatedEntity{
				AssociatedEntityType: "D2L.LE.Quizzing.Quiz",
				AssociatedEntityId:   99020,
			},
		},
		{
			CalendarEventId: 31,
			OrgUnitId:       12348,
			Title:           "Quiz 2",
			Description:     "Entity B",
			StartDateTime:   "2026-09-15T12:00:00Z",
			EndDateTime:     "2026-09-15T13:00:00Z",
			OrgUnitCode:     "MOD1004",
			OrgUnitName:     "MOD1004 Module",
			EventType:       EventTypeDueDate,
			AssociatedEntity: &AssociatedEntity{
				AssociatedEntityType: "D2L.LE.Quizzing.Quiz",
				AssociatedEntityId:   99021,
			},
		},
	}

	result := DeduplicateCalendarEntries(entries)

	if len(result) != 2 {
		t.Fatalf("Expected 2 entries (different entities), got %d", len(result))
	}
}

func TestDeduplicateCalendarEntries_SameEntityDifferentCourses(t *testing.T) {
	entries := []CalendarEventAPI{
		{
			CalendarEventId: 40,
			OrgUnitId:       12349,
			Title:           "Assignment",
			Description:     "Course A",
			StartDateTime:   "2026-09-15T10:00:00Z",
			EndDateTime:     "2026-09-15T11:00:00Z",
			OrgUnitCode:     "MOD1005",
			OrgUnitName:     "MOD1005 Course A",
			EventType:       EventTypeDueDate,
			AssociatedEntity: &AssociatedEntity{
				AssociatedEntityType: "D2L.LE.Dropbox.Folder",
				AssociatedEntityId:   99030,
			},
		},
		{
			CalendarEventId: 41,
			OrgUnitId:       12350,
			Title:           "Assignment",
			Description:     "Course B",
			StartDateTime:   "2026-09-15T10:00:00Z",
			EndDateTime:     "2026-09-15T11:00:00Z",
			OrgUnitCode:     "MOD1006",
			OrgUnitName:     "MOD1006 Course B",
			EventType:       EventTypeDueDate,
			AssociatedEntity: &AssociatedEntity{
				AssociatedEntityType: "D2L.LE.Dropbox.Folder",
				AssociatedEntityId:   99030,
			},
		},
	}

	result := DeduplicateCalendarEntries(entries)

	if len(result) != 2 {
		t.Fatalf("Expected 2 entries (different courses), got %d", len(result))
	}
}

func TestDeduplicateCalendarEntries_NonEntityTimeProximity(t *testing.T) {
	entries := []CalendarEventAPI{
		{
			CalendarEventId: 50,
			OrgUnitId:       12351,
			Title:           "Lecture",
			Description:     "Event A",
			StartDateTime:   "2026-09-15T10:00:00Z",
			EndDateTime:     "2026-09-15T10:30:00Z",
			OrgUnitCode:     "MOD1007",
			OrgUnitName:     "MOD1007 Lecture Module",
			EventType:       EventTypeReminder,
		},
		{
			CalendarEventId: 51,
			OrgUnitId:       12351,
			Title:           "Lecture",
			Description:     "Event B",
			StartDateTime:   "2026-09-15T10:10:00Z",
			EndDateTime:     "2026-09-15T10:40:00Z",
			OrgUnitCode:     "MOD1007",
			OrgUnitName:     "MOD1007 Lecture Module",
			EventType:       EventTypeAvailabilityStarts,
		},
	}

	result := DeduplicateCalendarEntries(entries)

	if len(result) != 1 {
		t.Fatalf("Expected 1 entry (time proximity within 15min), got %d", len(result))
	}

	ev := result[0]
	if ev.StartDateTime != "2026-09-15T10:00:00Z" {
		t.Errorf("StartDateTime = %q, want %q", ev.StartDateTime, "2026-09-15T10:00:00Z")
	}
	if ev.EndDateTime != "2026-09-15T10:40:00Z" {
		t.Errorf("EndDateTime = %q, want %q", ev.EndDateTime, "2026-09-15T10:40:00Z")
	}
	if ev.EventType != EventTypeAvailabilityStarts {
		t.Errorf("EventType = %d, want %d (AvailabilityStarts)", ev.EventType, EventTypeAvailabilityStarts)
	}
}

func TestDeduplicateCalendarEntries_NonEntityDifferentTitles(t *testing.T) {
	entries := []CalendarEventAPI{
		{
			CalendarEventId: 60,
			OrgUnitId:       12352,
			Title:           "Lecture A",
			Description:     "Title A",
			StartDateTime:   "2026-09-15T10:00:00Z",
			EndDateTime:     "2026-09-15T11:00:00Z",
			OrgUnitCode:     "MOD1008",
			OrgUnitName:     "MOD1008 Module",
			EventType:       EventTypeReminder,
		},
		{
			CalendarEventId: 61,
			OrgUnitId:       12352,
			Title:           "Lecture B",
			Description:     "Title B",
			StartDateTime:   "2026-09-15T10:00:00Z",
			EndDateTime:     "2026-09-15T11:00:00Z",
			OrgUnitCode:     "MOD1008",
			OrgUnitName:     "MOD1008 Module",
			EventType:       EventTypeReminder,
		},
	}

	result := DeduplicateCalendarEntries(entries)

	if len(result) != 2 {
		t.Fatalf("Expected 2 entries (different titles), got %d", len(result))
	}
}

func TestDeduplicateCalendarEntries_NonEntityTimesFarApart(t *testing.T) {
	entries := []CalendarEventAPI{
		{
			CalendarEventId: 70,
			OrgUnitId:       12353,
			Title:           "Meeting",
			Description:     "Morning",
			StartDateTime:   "2026-09-15T09:00:00Z",
			EndDateTime:     "2026-09-15T10:00:00Z",
			OrgUnitCode:     "MOD1009",
			OrgUnitName:     "MOD1009 Module",
			EventType:       EventTypeReminder,
		},
		{
			CalendarEventId: 71,
			OrgUnitId:       12353,
			Title:           "Meeting",
			Description:     "Evening",
			StartDateTime:   "2026-09-15T18:00:00Z",
			EndDateTime:     "2026-09-15T19:00:00Z",
			OrgUnitCode:     "MOD1009",
			OrgUnitName:     "MOD1009 Module",
			EventType:       EventTypeReminder,
		},
	}

	result := DeduplicateCalendarEntries(entries)

	if len(result) != 2 {
		t.Fatalf("Expected 2 entries (times > 15min apart), got %d", len(result))
	}
}

func TestDeduplicateCalendarEntries_EmptyInputNil(t *testing.T) {
	var entries []CalendarEventAPI = nil
	result := DeduplicateCalendarEntries(entries)
	if result != nil {
		t.Errorf("Expected nil for nil input, got %v", result)
	}
}

func TestDeduplicateCalendarEntries_EmptySlice(t *testing.T) {
	entries := []CalendarEventAPI{}
	result := DeduplicateCalendarEntries(entries)
	if result == nil {
		t.Error("Expected empty slice for empty input, got nil")
	}
	if len(result) != 0 {
		t.Errorf("Expected 0 entries, got %d", len(result))
	}
}

func TestDeduplicateCalendarEntries_MixedEntityAndNonEntity(t *testing.T) {
	entries := []CalendarEventAPI{
		{
			CalendarEventId: 80,
			OrgUnitId:       12354,
			Title:           "Quiz Event",
			Description:     "Entity event",
			StartDateTime:   "2026-09-15T10:00:00Z",
			EndDateTime:     "2026-09-15T11:00:00Z",
			OrgUnitCode:     "MOD1010",
			OrgUnitName:     "MOD1010 Module",
			EventType:       EventTypeDueDate,
			AssociatedEntity: &AssociatedEntity{
				AssociatedEntityType: "D2L.LE.Quizzing.Quiz",
				AssociatedEntityId:   99040,
			},
		},
		{
			CalendarEventId: 81,
			OrgUnitId:       12354,
			Title:           "Reminder",
			Description:     "Non-entity event",
			StartDateTime:   "2026-09-15T14:00:00Z",
			EndDateTime:     "2026-09-15T15:00:00Z",
			OrgUnitCode:     "MOD1010",
			OrgUnitName:     "MOD1010 Module",
			EventType:       EventTypeReminder,
		},
	}

	result := DeduplicateCalendarEntries(entries)

	if len(result) != 2 {
		t.Fatalf("Expected 2 entries (1 entity + 1 non-entity), got %d", len(result))
	}
}

func TestDeduplicateCalendarEntries_EventTypePriority(t *testing.T) {
	entries := []CalendarEventAPI{
		{
			CalendarEventId: 90,
			OrgUnitId:       12355,
			Title:           "Priority Test",
			Description:     "UnlockEnds desc",
			StartDateTime:   "2026-09-15T10:00:00Z",
			EndDateTime:     "2026-09-15T11:00:00Z",
			OrgUnitCode:     "MOD1011",
			OrgUnitName:     "MOD1011 Module",
			EventType:       EventTypeUnlockEnds,
			AssociatedEntity: &AssociatedEntity{
				AssociatedEntityType: "D2L.LE.Quizzing.Quiz",
				AssociatedEntityId:   99050,
			},
		},
		{
			CalendarEventId: 91,
			OrgUnitId:       12355,
			Title:           "Priority Test",
			Description:     "UnlockStarts desc",
			StartDateTime:   "2026-09-15T10:00:00Z",
			EndDateTime:     "2026-09-15T11:00:00Z",
			OrgUnitCode:     "MOD1011",
			OrgUnitName:     "MOD1011 Module",
			EventType:       EventTypeUnlockStarts,
			AssociatedEntity: &AssociatedEntity{
				AssociatedEntityType: "D2L.LE.Quizzing.Quiz",
				AssociatedEntityId:   99050,
			},
		},
		{
			CalendarEventId: 92,
			OrgUnitId:       12355,
			Title:           "Priority Test",
			Description:     "AvailabilityEnds desc",
			StartDateTime:   "2026-09-15T10:00:00Z",
			EndDateTime:     "2026-09-15T11:00:00Z",
			OrgUnitCode:     "MOD1011",
			OrgUnitName:     "MOD1011 Module",
			EventType:       EventTypeAvailabilityEnds,
			AssociatedEntity: &AssociatedEntity{
				AssociatedEntityType: "D2L.LE.Quizzing.Quiz",
				AssociatedEntityId:   99050,
			},
		},
		{
			CalendarEventId: 93,
			OrgUnitId:       12355,
			Title:           "Priority Test",
			Description:     "AvailabilityStarts desc",
			StartDateTime:   "2026-09-15T10:00:00Z",
			EndDateTime:     "2026-09-15T11:00:00Z",
			OrgUnitCode:     "MOD1011",
			OrgUnitName:     "MOD1011 Module",
			EventType:       EventTypeAvailabilityStarts,
			AssociatedEntity: &AssociatedEntity{
				AssociatedEntityType: "D2L.LE.Quizzing.Quiz",
				AssociatedEntityId:   99050,
			},
		},
		{
			CalendarEventId: 94,
			OrgUnitId:       12355,
			Title:           "Priority Test",
			Description:     "Reminder desc",
			StartDateTime:   "2026-09-15T10:00:00Z",
			EndDateTime:     "2026-09-15T11:00:00Z",
			OrgUnitCode:     "MOD1011",
			OrgUnitName:     "MOD1011 Module",
			EventType:       EventTypeReminder,
			AssociatedEntity: &AssociatedEntity{
				AssociatedEntityType: "D2L.LE.Quizzing.Quiz",
				AssociatedEntityId:   99050,
			},
		},
		{
			CalendarEventId: 95,
			OrgUnitId:       12355,
			Title:           "Priority Test",
			Description:     "DueDate desc",
			StartDateTime:   "2026-09-15T10:00:00Z",
			EndDateTime:     "2026-09-15T11:00:00Z",
			OrgUnitCode:     "MOD1011",
			OrgUnitName:     "MOD1011 Module",
			EventType:       EventTypeDueDate,
			AssociatedEntity: &AssociatedEntity{
				AssociatedEntityType: "D2L.LE.Quizzing.Quiz",
				AssociatedEntityId:   99050,
			},
		},
	}

	result := DeduplicateCalendarEntries(entries)

	if len(result) != 1 {
		t.Fatalf("Expected 1 entry, got %d", len(result))
	}

	ev := result[0]
	if ev.EventType != EventTypeDueDate {
		t.Errorf("EventType = %d, want %d (DueDate - highest priority)", ev.EventType, EventTypeDueDate)
	}
	if ev.Description != "DueDate desc" {
		t.Errorf("Description = %q, want %q", ev.Description, "DueDate desc")
	}
}

func TestDeduplicateCalendarEntries_TimeRangeExpansion(t *testing.T) {
	entries := []CalendarEventAPI{
		{
			CalendarEventId: 100,
			OrgUnitId:       12356,
			Title:           "Time Range Test",
			Description:     "Middle event",
			StartDateTime:   "2026-09-15T10:20:00Z",
			EndDateTime:     "2026-09-15T11:00:00Z",
			OrgUnitCode:     "MOD1012",
			OrgUnitName:     "MOD1012 Module",
			EventType:       EventTypeReminder,
		},
		{
			CalendarEventId: 101,
			OrgUnitId:       12356,
			Title:           "Time Range Test",
			Description:     "Earliest start",
			StartDateTime:   "2026-09-15T10:00:00Z",
			EndDateTime:     "2026-09-15T10:30:00Z",
			OrgUnitCode:     "MOD1012",
			OrgUnitName:     "MOD1012 Module",
			EventType:       EventTypeReminder,
		},
		{
			CalendarEventId: 102,
			OrgUnitId:       12356,
			Title:           "Time Range Test",
			Description:     "Latest end",
			StartDateTime:   "2026-09-15T10:10:00Z",
			EndDateTime:     "2026-09-15T14:00:00Z",
			OrgUnitCode:     "MOD1012",
			OrgUnitName:     "MOD1012 Module",
			EventType:       EventTypeReminder,
		},
	}

	result := DeduplicateCalendarEntries(entries)

	if len(result) != 1 {
		t.Fatalf("Expected 1 entry (all within 15min transitively), got %d", len(result))
	}

	ev := result[0]
	if ev.StartDateTime != "2026-09-15T10:00:00Z" {
		t.Errorf("StartDateTime = %q, want %q", ev.StartDateTime, "2026-09-15T10:00:00Z")
	}
	if ev.EndDateTime != "2026-09-15T14:00:00Z" {
		t.Errorf("EndDateTime = %q, want %q", ev.EndDateTime, "2026-09-15T14:00:00Z")
	}
}

func TestDeduplicateCalendarEntries_CaseInsensitiveTitle(t *testing.T) {
	entries := []CalendarEventAPI{
		{
			CalendarEventId: 110,
			OrgUnitId:       12357,
			Title:           "QUIZ ONE",
			Description:     "Uppercase",
			StartDateTime:   "2026-09-15T10:00:00Z",
			EndDateTime:     "2026-09-15T11:00:00Z",
			OrgUnitCode:     "MOD1013",
			OrgUnitName:     "MOD1013 Module",
			EventType:       EventTypeDueDate,
			AssociatedEntity: &AssociatedEntity{
				AssociatedEntityType: "D2L.LE.Quizzing.Quiz",
				AssociatedEntityId:   99060,
			},
		},
		{
			CalendarEventId: 111,
			OrgUnitId:       12357,
			Title:           "quiz one",
			Description:     "Lowercase",
			StartDateTime:   "2026-09-15T11:00:00Z",
			EndDateTime:     "2026-09-15T12:00:00Z",
			OrgUnitCode:     "MOD1013",
			OrgUnitName:     "MOD1013 Module",
			EventType:       EventTypeAvailabilityEnds,
			AssociatedEntity: &AssociatedEntity{
				AssociatedEntityType: "D2L.LE.Quizzing.Quiz",
				AssociatedEntityId:   99060,
			},
		},
	}

	result := DeduplicateCalendarEntries(entries)

	if len(result) != 1 {
		t.Fatalf("Expected 1 entry (case-insensitive title match), got %d", len(result))
	}
}

func TestDeduplicateCalendarEntries_TrimmedTitle(t *testing.T) {
	entries := []CalendarEventAPI{
		{
			CalendarEventId: 120,
			OrgUnitId:       12358,
			Title:           "  Trimmed Quiz  ",
			Description:     "With spaces",
			StartDateTime:   "2026-09-15T10:00:00Z",
			EndDateTime:     "2026-09-15T11:00:00Z",
			OrgUnitCode:     "MOD1014",
			OrgUnitName:     "MOD1014 Module",
			EventType:       EventTypeDueDate,
			AssociatedEntity: &AssociatedEntity{
				AssociatedEntityType: "D2L.LE.Quizzing.Quiz",
				AssociatedEntityId:   99070,
			},
		},
		{
			CalendarEventId: 121,
			OrgUnitId:       12358,
			Title:           "Trimmed Quiz",
			Description:     "Without spaces",
			StartDateTime:   "2026-09-15T11:00:00Z",
			EndDateTime:     "2026-09-15T12:00:00Z",
			OrgUnitCode:     "MOD1014",
			OrgUnitName:     "MOD1014 Module",
			EventType:       EventTypeAvailabilityEnds,
			AssociatedEntity: &AssociatedEntity{
				AssociatedEntityType: "D2L.LE.Quizzing.Quiz",
				AssociatedEntityId:   99070,
			},
		},
	}

	result := DeduplicateCalendarEntries(entries)

	if len(result) != 1 {
		t.Fatalf("Expected 1 entry (trimmed title match), got %d", len(result))
	}
}

func TestEventTypePriority(t *testing.T) {
	tests := []struct {
		eventType int
		want      int
	}{
		{EventTypeDueDate, 5},
		{EventTypeAvailabilityStarts, 4},
		{EventTypeAvailabilityEnds, 3},
		{EventTypeReminder, 2},
		{EventTypeUnlockStarts, 1},
		{EventTypeUnlockEnds, 0},
		{0, -1},
		{99, -1},
	}

	for _, tt := range tests {
		got := eventTypePriority(tt.eventType)
		if got != tt.want {
			t.Errorf("eventTypePriority(%d) = %d, want %d", tt.eventType, got, tt.want)
		}
	}
}

func TestNormalizeTitle(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"  Hello World  ", "hello world"},
		{"QUIZ", "quiz"},
		{"Quiz One", "quiz one"},
		{"", ""},
	}

	for _, tt := range tests {
		got := normalizeTitle(tt.input)
		if got != tt.want {
			t.Errorf("normalizeTitle(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestIsTimeProximity(t *testing.T) {
	loc, _ := time.LoadLocation("UTC")
	base := time.Date(2026, 9, 15, 10, 0, 0, 0, loc)

	tests := []struct {
		name    string
		t1      time.Time
		t2      time.Time
		want    bool
	}{
		{"exact same", base, base, true},
		{"5 min apart", base, base.Add(5 * time.Minute), true},
		{"15 min apart", base, base.Add(15 * time.Minute), true},
		{"16 min apart", base, base.Add(16 * time.Minute), false},
		{"10 min before", base, base.Add(-10 * time.Minute), true},
		{"20 min before", base, base.Add(-20 * time.Minute), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isTimeProximity(tt.t1, tt.t2)
			if got != tt.want {
				t.Errorf("isTimeProximity(%v, %v) = %v, want %v", tt.t1, tt.t2, got, tt.want)
			}
		})
	}
}

func TestDeduplicateCalendarEntries_NoAssociatedEntityEventsOnly(t *testing.T) {
	entries := []CalendarEventAPI{
		{
			CalendarEventId: 200,
			OrgUnitId:       12360,
			Title:           "Standalone Event",
			Description:     "No entity",
			StartDateTime:   "2026-09-15T10:00:00Z",
			EndDateTime:     "2026-09-15T11:00:00Z",
			OrgUnitCode:     "MOD1020",
			OrgUnitName:     "MOD1020 Module",
			EventType:       EventTypeReminder,
		},
	}

	result := DeduplicateCalendarEntries(entries)

	if len(result) != 1 {
		t.Fatalf("Expected 1 entry, got %d", len(result))
	}
	if result[0].AssociatedEntity != nil {
		t.Error("AssociatedEntity should be nil")
	}
}

func TestDeduplicateCalendarEntries_MultipleEntityGroups(t *testing.T) {
	entries := []CalendarEventAPI{
		{
			CalendarEventId: 210,
			OrgUnitId:       12361,
			Title:           "Quiz A",
			Description:     "Group A event 1",
			StartDateTime:   "2026-09-15T10:00:00Z",
			EndDateTime:     "2026-09-15T10:30:00Z",
			OrgUnitCode:     "MOD1021",
			OrgUnitName:     "MOD1021 Module",
			EventType:       EventTypeDueDate,
			AssociatedEntity: &AssociatedEntity{
				AssociatedEntityType: "D2L.LE.Quizzing.Quiz",
				AssociatedEntityId:   99080,
			},
		},
		{
			CalendarEventId: 211,
			OrgUnitId:       12361,
			Title:           "Quiz A",
			Description:     "Group A event 2",
			StartDateTime:   "2026-09-15T10:30:00Z",
			EndDateTime:     "2026-09-15T11:00:00Z",
			OrgUnitCode:     "MOD1021",
			OrgUnitName:     "MOD1021 Module",
			EventType:       EventTypeAvailabilityEnds,
			AssociatedEntity: &AssociatedEntity{
				AssociatedEntityType: "D2L.LE.Quizzing.Quiz",
				AssociatedEntityId:   99080,
			},
		},
		{
			CalendarEventId: 212,
			OrgUnitId:       12361,
			Title:           "Quiz B",
			Description:     "Group B event 1",
			StartDateTime:   "2026-09-15T14:00:00Z",
			EndDateTime:     "2026-09-15T14:30:00Z",
			OrgUnitCode:     "MOD1021",
			OrgUnitName:     "MOD1021 Module",
			EventType:       EventTypeDueDate,
			AssociatedEntity: &AssociatedEntity{
				AssociatedEntityType: "D2L.LE.Quizzing.Quiz",
				AssociatedEntityId:   99081,
			},
		},
	}

	result := DeduplicateCalendarEntries(entries)

	if len(result) != 2 {
		t.Fatalf("Expected 2 entries (2 entity groups), got %d", len(result))
	}

	if result[0].AssociatedEntity.AssociatedEntityId != 99080 && result[1].AssociatedEntity.AssociatedEntityId != 99080 {
		t.Error("One result should have EntityID 99080")
	}
	if result[0].AssociatedEntity.AssociatedEntityId != 99081 && result[1].AssociatedEntity.AssociatedEntityId != 99081 {
		t.Error("One result should have EntityID 99081")
	}
}

func TestDeduplicateCalendarEntries_DedupReducesCount(t *testing.T) {
	entries := []CalendarEventAPI{
		{
			CalendarEventId: 300,
			OrgUnitId:       12362,
			Title:           "Dedup Quiz",
			StartDateTime:   "2026-09-15T10:00:00Z",
			EndDateTime:     "2026-09-15T10:05:00Z",
			OrgUnitCode:     "MOD1022",
			OrgUnitName:     "MOD1022 Module",
			EventType:       EventTypeAvailabilityStarts,
			AssociatedEntity: &AssociatedEntity{
				AssociatedEntityType: "D2L.LE.Quizzing.Quiz",
				AssociatedEntityId:   99090,
			},
		},
		{
			CalendarEventId: 301,
			OrgUnitId:       12362,
			Title:           "Dedup Quiz",
			StartDateTime:   "2026-09-15T10:05:00Z",
			EndDateTime:     "2026-09-15T10:30:00Z",
			OrgUnitCode:     "MOD1022",
			OrgUnitName:     "MOD1022 Module",
			EventType:       EventTypeDueDate,
			AssociatedEntity: &AssociatedEntity{
				AssociatedEntityType: "D2L.LE.Quizzing.Quiz",
				AssociatedEntityId:   99090,
			},
		},
		{
			CalendarEventId: 302,
			OrgUnitId:       12362,
			Title:           "Dedup Quiz",
			StartDateTime:   "2026-09-15T10:30:00Z",
			EndDateTime:     "2026-09-15T11:00:00Z",
			OrgUnitCode:     "MOD1022",
			OrgUnitName:     "MOD1022 Module",
			EventType:       EventTypeAvailabilityEnds,
			AssociatedEntity: &AssociatedEntity{
				AssociatedEntityType: "D2L.LE.Quizzing.Quiz",
				AssociatedEntityId:   99090,
			},
		},
		{
			CalendarEventId: 303,
			OrgUnitId:       12362,
			Title:           "Standalone",
			StartDateTime:   "2026-09-15T14:00:00Z",
			EndDateTime:     "2026-09-15T15:00:00Z",
			OrgUnitCode:     "MOD1022",
			OrgUnitName:     "MOD1022 Module",
			EventType:       EventTypeReminder,
		},
	}

	result := DeduplicateCalendarEntries(entries)

	if len(result) != 2 {
		t.Fatalf("Expected 2 entries (3 deduplicated to 1 + 1 standalone = 2), got %d", len(result))
	}
}
