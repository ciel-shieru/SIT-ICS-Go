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
