package brightspace

import (
	"testing"
	"time"
)

func TestExpandRecurrence_YearlyLeapYear(t *testing.T) {
	entry := BrightSpaceStringEntry{
		Title:                 "Test Yearly Leap",
		DTStart:               "2024-02-29T00:00:00.000Z",
		DTEnd:                 "2024-02-29T01:00:00.000Z",
		IsRecurring:           true,
		RepeatType:            5,
		RepeatEvery:           1,
		RepeatUntilDateString: "2032-01-01T00:00:00.000Z",
		Source:                "brightspace-calendar",
	}

	loc, _ := time.LoadLocation("UTC")
	events, err := ExpandRecurrence(entry, loc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(events) != 8 {
		t.Fatalf("expected 8 events, got %d", len(events))
	}

	if events[0].DTStart.Month() != time.February || events[0].DTStart.Day() != 29 {
		t.Errorf("first event: expected Feb 29, got %s", events[0].DTStart.Format("2006-01-02"))
	}

	if events[2].DTStart.Month() != time.March || events[2].DTStart.Day() != 1 {
		t.Errorf("third event: expected Mar 1, got %s", events[2].DTStart.Format("2006-01-02"))
	}
}

func TestExpandRecurrence_WeeklySingleDay(t *testing.T) {
	entry := BrightSpaceStringEntry{
		Title:                 "Test Weekly",
		OrgUnitId:             "123",
		OrgUnitName:           "Test Course",
		OrgUnitCode:           "TEST101",
		DTStart:               "2026-09-04T01:00:00.000Z",
		DTEnd:                 "2026-09-04T03:00:00.000Z",
		IsRecurring:           true,
		RepeatType:            3,
		RepeatEvery:           1,
		RepeatOnInfo: &RecurrenceInfo{
			RepeatOnInfo: &RepeatOnInfo{Friday: true},
		},
		RepeatUntilDateString: "2026-10-02T01:00:00.000Z",
		Source:                "brightspace-calendar",
	}

	loc, _ := time.LoadLocation("Asia/Singapore")
	events, err := ExpandRecurrence(entry, loc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(events) != 4 {
		t.Fatalf("expected 4 events, got %d", len(events))
	}

	for i, ev := range events {
		if ev.RecurrenceIndex != i {
			t.Errorf("event %d: expected RecurrenceIndex %d, got %d", i, i, ev.RecurrenceIndex)
		}
	}

	expectedFirst := time.Date(2026, 9, 4, 9, 0, 0, 0, loc)
	if !events[0].DTStart.Equal(expectedFirst) {
		t.Errorf("first event DTStart: expected %v, got %v", expectedFirst, events[0].DTStart)
	}
}

func TestExpandRecurrence_Daily(t *testing.T) {
	entry := BrightSpaceStringEntry{
		Title:                 "Test Daily",
		DTStart:               "2026-09-04T00:00:00.000Z",
		DTEnd:                 "2026-09-04T01:00:00.000Z",
		IsRecurring:           true,
		RepeatType:            2,
		RepeatEvery:           1,
		RepeatUntilDateString: "2026-09-07T00:00:00.000Z",
		Source:                "brightspace-calendar",
	}

	loc, _ := time.LoadLocation("UTC")
	events, err := ExpandRecurrence(entry, loc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(events))
	}
}

func TestExpandRecurrence_NotRecurring(t *testing.T) {
	entry := BrightSpaceStringEntry{
		Title:        "Test Non-Recurring",
		DTStart:      "2026-09-04T00:00:00.000Z",
		DTEnd:        "2026-09-04T01:00:00.000Z",
		IsRecurring:  false,
		RepeatType:   1,
		Source:       "brightspace-calendar",
	}

	loc, _ := time.LoadLocation("UTC")
	_, err := ExpandRecurrence(entry, loc)
	if err == nil {
		t.Fatal("expected error for non-recurring event")
	}
}

func TestExpandRecurrence_Monthly(t *testing.T) {
	entry := BrightSpaceStringEntry{
		Title:                 "Test Monthly",
		DTStart:               "2026-01-31T00:00:00.000Z",
		DTEnd:                 "2026-01-31T01:00:00.000Z",
		IsRecurring:           true,
		RepeatType:            4,
		RepeatEvery:           1,
		RepeatUntilDateString: "2026-05-01T00:00:00.000Z",
		Source:                "brightspace-calendar",
	}

	loc, _ := time.LoadLocation("UTC")
	events, err := ExpandRecurrence(entry, loc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(events) != 4 {
		t.Fatalf("expected 4 events, got %d", len(events))
	}
}

func TestExpandRecurrence_MaxCap(t *testing.T) {
	entry := BrightSpaceStringEntry{
		Title:                 "Test Daily Long",
		DTStart:               "2026-01-01T00:00:00.000Z",
		DTEnd:                 "2026-01-01T01:00:00.000Z",
		IsRecurring:           true,
		RepeatType:            2,
		RepeatEvery:           1,
		RepeatUntilDateString: "2027-12-31T00:00:00.000Z",
		Source:                "brightspace-calendar",
	}

	loc, _ := time.LoadLocation("UTC")
	events, err := ExpandRecurrence(entry, loc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(events) != maxOccurrences {
		t.Fatalf("expected %d events (capped), got %d", maxOccurrences, len(events))
	}
}

func TestExpandRecurrence_WeeklyMultipleDays(t *testing.T) {
	entry := BrightSpaceStringEntry{
		Title:                 "Test Weekly Multi",
		DTStart:               "2026-09-01T00:00:00.000Z",
		DTEnd:                 "2026-09-01T01:00:00.000Z",
		IsRecurring:           true,
		RepeatType:            3,
		RepeatEvery:           1,
		RepeatOnInfo: &RecurrenceInfo{
			RepeatOnInfo: &RepeatOnInfo{
				Tuesday:  true,
				Thursday: true,
			},
		},
		RepeatUntilDateString: "2026-09-15T00:00:00.000Z",
		Source:                "brightspace-calendar",
	}

	loc, _ := time.LoadLocation("UTC")
	events, err := ExpandRecurrence(entry, loc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(events) != 4 {
		t.Fatalf("expected 4 events, got %d", len(events))
	}

	if !events[0].DTStart.Before(events[1].DTStart) {
		t.Error("events not sorted by start time")
	}
}

func TestExpandRecurrence_Yearly(t *testing.T) {
	entry := BrightSpaceStringEntry{
		Title:                 "Test Yearly",
		DTStart:               "2026-03-15T00:00:00.000Z",
		DTEnd:                 "2026-03-15T01:00:00.000Z",
		IsRecurring:           true,
		RepeatType:            5,
		RepeatEvery:           1,
		RepeatUntilDateString: "2029-01-01T00:00:00.000Z",
		Source:                "brightspace-calendar",
	}

	loc, _ := time.LoadLocation("UTC")
	events, err := ExpandRecurrence(entry, loc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(events))
	}
}

func TestExpandRecurrence_DurationPreserved(t *testing.T) {
	entry := BrightSpaceStringEntry{
		Title:                 "Test Duration",
		DTStart:               "2026-09-04T01:30:00.000Z",
		DTEnd:                 "2026-09-04T02:45:00.000Z",
		IsRecurring:           true,
		RepeatType:            2,
		RepeatEvery:           1,
		RepeatUntilDateString: "2026-09-06T00:00:00.000Z",
		Source:                "brightspace-calendar",
	}

	loc, _ := time.LoadLocation("UTC")
	events, err := ExpandRecurrence(entry, loc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedDuration := 75 * time.Minute
	for _, ev := range events {
		actualDuration := ev.DTEnd.Sub(ev.DTStart)
		if actualDuration != expectedDuration {
			t.Errorf("duration mismatch: expected %v, got %v", expectedDuration, actualDuration)
		}
	}
}

func TestExpandRecurrence_WeeklyNilRepeatOnInfo(t *testing.T) {
	entry := BrightSpaceStringEntry{
		Title:                 "Test Weekly Nil",
		DTStart:               "2026-09-04T00:00:00.000Z",
		DTEnd:                 "2026-09-04T01:00:00.000Z",
		IsRecurring:           true,
		RepeatType:            3,
		RepeatEvery:           1,
		RepeatOnInfo:          nil,
		RepeatUntilDateString: "2026-09-20T00:00:00.000Z",
		Source:                "brightspace-calendar",
	}

	loc, _ := time.LoadLocation("UTC")
	events, err := ExpandRecurrence(entry, loc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedCount := 3
	if len(events) != expectedCount {
		t.Fatalf("expected %d events, got %d", expectedCount, len(events))
	}
}

func TestExpandRecurrence_WeeklyNilRepeatOnInfoInner(t *testing.T) {
	entry := BrightSpaceStringEntry{
		Title:                 "Test Weekly Nil Inner",
		DTStart:               "2026-09-04T00:00:00.000Z",
		DTEnd:                 "2026-09-04T01:00:00.000Z",
		IsRecurring:           true,
		RepeatType:            3,
		RepeatEvery:           1,
		RepeatOnInfo:          &RecurrenceInfo{},
		RepeatUntilDateString: "2026-09-20T00:00:00.000Z",
		Source:                "brightspace-calendar",
	}

	loc, _ := time.LoadLocation("UTC")
	events, err := ExpandRecurrence(entry, loc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedCount := 3
	if len(events) != expectedCount {
		t.Fatalf("expected %d events, got %d", expectedCount, len(events))
	}
}

func TestExpandRecurrence_WeeklyNoActiveDays(t *testing.T) {
	entry := BrightSpaceStringEntry{
		Title:                 "Test Weekly No Days",
		DTStart:               "2026-09-04T00:00:00.000Z",
		DTEnd:                 "2026-09-04T01:00:00.000Z",
		IsRecurring:           true,
		RepeatType:            3,
		RepeatEvery:           1,
		RepeatOnInfo: &RecurrenceInfo{
			RepeatOnInfo: &RepeatOnInfo{
				Monday: false, Tuesday: false, Wednesday: false,
				Thursday: false, Friday: false, Saturday: false, Sunday: false,
			},
		},
		RepeatUntilDateString: "2026-10-02T00:00:00.000Z",
		Source:                "brightspace-calendar",
	}

	loc, _ := time.LoadLocation("UTC")
	_, err := ExpandRecurrence(entry, loc)
	if err == nil {
		t.Fatal("expected error for weekly recurrence with no active days")
	}
}

func TestExpandRecurrence_WeeklyNoActiveDaysFirstWeek(t *testing.T) {
	entry := BrightSpaceStringEntry{
		Title:                 "Test Weekly No Days First Week",
		DTStart:               "2026-09-01T00:00:00.000Z",
		DTEnd:                 "2026-09-01T01:00:00.000Z",
		IsRecurring:           true,
		RepeatType:            3,
		RepeatEvery:           1,
		RepeatOnInfo: &RecurrenceInfo{
			RepeatOnInfo: &RepeatOnInfo{Monday: true},
		},
		RepeatUntilDateString: "2026-09-01T00:00:00.000Z",
		Source:                "brightspace-calendar",
	}

	loc, _ := time.LoadLocation("UTC")
	_, err := ExpandRecurrence(entry, loc)
	if err == nil {
		t.Fatal("expected error when no active days before repeat until date in first week")
	}
}

func TestExpandRecurrence_RepeatEvery2(t *testing.T) {
	entry := BrightSpaceStringEntry{
		Title:                 "Test Biweekly",
		DTStart:               "2026-09-04T00:00:00.000Z",
		DTEnd:                 "2026-09-04T01:00:00.000Z",
		IsRecurring:           true,
		RepeatType:            2,
		RepeatEvery:           2,
		RepeatUntilDateString: "2026-09-20T00:00:00.000Z",
		Source:                "brightspace-calendar",
	}

	loc, _ := time.LoadLocation("UTC")
	events, err := ExpandRecurrence(entry, loc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Sep 4, 6, 8, 10, 12, 14, 16, 18 = 8 occurrences (Sep 20 excluded)
	expectedCount := 8
	if len(events) != expectedCount {
		t.Fatalf("expected %d events, got %d", expectedCount, len(events))
	}
}

func TestExpandRecurrence_CalendarEventIDPreserved(t *testing.T) {
	entry := BrightSpaceStringEntry{
		Title:                 "Test ID Preserved",
		DTStart:               "2026-09-04T00:00:00.000Z",
		DTEnd:                 "2026-09-04T01:00:00.000Z",
		IsRecurring:           true,
		RepeatType:            3,
		RepeatEvery:           1,
		RepeatOnInfo: &RecurrenceInfo{
			RepeatOnInfo: &RepeatOnInfo{Friday: true},
		},
		RepeatUntilDateString: "2026-09-20T00:00:00.000Z",
		CalendarEventID:       212823,
		Source:                "brightspace-calendar",
	}

	loc, _ := time.LoadLocation("UTC")
	events, err := ExpandRecurrence(entry, loc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for i, ev := range events {
		if ev.CalendarEventID != 212823 {
			t.Errorf("event %d: CalendarEventID = %d, want 212823", i, ev.CalendarEventID)
		}
		if ev.RecurrenceIndex != i {
			t.Errorf("event %d: RecurrenceIndex = %d, want %d", i, ev.RecurrenceIndex, i)
		}
	}
}

func TestExpandRecurrence_MonthlyOverflow(t *testing.T) {
	entry := BrightSpaceStringEntry{
		Title:                 "Test Monthly Overflow",
		DTStart:               "2026-01-31T00:00:00.000Z",
		DTEnd:                 "2026-01-31T01:00:00.000Z",
		IsRecurring:           true,
		RepeatType:            4,
		RepeatEvery:           1,
		RepeatUntilDateString: "2026-06-01T00:00:00.000Z",
		Source:                "brightspace-calendar",
	}

	loc, _ := time.LoadLocation("UTC")
	events, err := ExpandRecurrence(entry, loc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(events) != 5 {
		t.Fatalf("expected 5 events, got %d", len(events))
	}

	if events[1].DTStart.Month() != time.February || events[1].DTStart.Day() != 28 {
		t.Errorf("second event: expected Feb 28, got %s", events[1].DTStart.Format("2006-01-02"))
	}
}

func TestExpandRecurrence_YearlyRepeatEvery2(t *testing.T) {
	entry := BrightSpaceStringEntry{
		Title:                 "Test Yearly Biennial",
		DTStart:               "2024-06-15T00:00:00.000Z",
		DTEnd:                 "2024-06-15T01:00:00.000Z",
		IsRecurring:           true,
		RepeatType:            5,
		RepeatEvery:           2,
		RepeatUntilDateString: "2035-01-01T00:00:00.000Z",
		Source:                "brightspace-calendar",
	}

	loc, _ := time.LoadLocation("UTC")
	events, err := ExpandRecurrence(entry, loc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(events) != 6 {
		t.Fatalf("expected 6 events, got %d", len(events))
	}

	expectedYears := []int{2024, 2026, 2028, 2030, 2032, 2034}
	for i, expectedYear := range expectedYears {
		if events[i].DTStart.Year() != expectedYear {
			t.Errorf("event %d: expected year %d, got %d", i, expectedYear, events[i].DTStart.Year())
		}
	}
}

func TestExpandRecurrence_UIDDeterminism(t *testing.T) {
	entry := BrightSpaceStringEntry{
		Title:                 "Test UID Determinism",
		OrgUnitCode:           "TEST101",
		OrgUnitName:           "Test Course",
		DTStart:               "2026-09-04T01:00:00.000Z",
		DTEnd:                 "2026-09-04T03:00:00.000Z",
		IsRecurring:           true,
		RepeatType:            2,
		RepeatEvery:           1,
		RepeatUntilDateString: "2026-09-07T00:00:00.000Z",
		Source:                "brightspace-calendar",
	}

	loc, _ := time.LoadLocation("UTC")

	events1, err := ExpandRecurrence(entry, loc)
	if err != nil {
		t.Fatalf("first expansion error: %v", err)
	}
	events2, err := ExpandRecurrence(entry, loc)
	if err != nil {
		t.Fatalf("second expansion error: %v", err)
	}

	if len(events1) != len(events2) {
		t.Fatalf("different occurrence counts: %d vs %d", len(events1), len(events2))
	}

	for i := range events1 {
		if !events1[i].DTStart.Equal(events2[i].DTStart) {
			t.Errorf("event %d: DTStart mismatch: %v vs %v", i, events1[i].DTStart, events2[i].DTStart)
		}
		if events1[i].RecurrenceIndex != events2[i].RecurrenceIndex {
			t.Errorf("event %d: RecurrenceIndex mismatch: %d vs %d", i, events1[i].RecurrenceIndex, events2[i].RecurrenceIndex)
		}
	}
}
