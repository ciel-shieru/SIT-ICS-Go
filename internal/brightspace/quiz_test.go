package brightspace

import (
	"testing"
	"time"
)

func TestQuizToStringEntry(t *testing.T) {
	quiz := QuizAPI{
		Name:        "Midterm Quiz",
		StartDate:   "2026-09-15T10:00:00Z",
		EndDate:     "2026-09-15T11:00:00Z",
		DueDate:     "2026-09-15T10:30:00Z",
		Description: DescField{Text: DescText{Text: "Quiz description", Html: "<p>Quiz description</p>"}},
		SubmissionTimeLimit: TimeLimit{IsEnforced: true, TimeLimitValue: 30},
		AttemptsAllowed:     Attempts{IsUnlimited: false, NumberOfAttemptsAllowed: 3},
		OrgUnitId:   "12345",
		OrgUnitName: "Introduction to Computer Science",
		OrgUnitCode: "SIT1001",
	}

	entry := QuizToStringEntry(quiz)

	if entry.Title != "Midterm Quiz" {
		t.Errorf("Title = %q, want %q", entry.Title, "Midterm Quiz")
	}
	if entry.Source != "brightspace-quizzes" {
		t.Errorf("Source = %q, want %q", entry.Source, "brightspace-quizzes")
	}
	if entry.OrgUnitId != "12345" {
		t.Errorf("OrgUnitId = %q, want %q", entry.OrgUnitId, "12345")
	}
	if entry.OrgUnitName != "Introduction to Computer Science" {
		t.Errorf("OrgUnitName = %q, want %q", entry.OrgUnitName, "Introduction to Computer Science")
	}
	if entry.OrgUnitCode != "SIT1001" {
		t.Errorf("OrgUnitCode = %q, want %q", entry.OrgUnitCode, "SIT1001")
	}
	if entry.DTStart != "2026-09-15T10:00:00Z" {
		t.Errorf("DTStart = %q, want %q", entry.DTStart, "2026-09-15T10:00:00Z")
	}
	if entry.DTEnd != "2026-09-15T11:00:00Z" {
		t.Errorf("DTEnd = %q, want %q (EndDate should be used over DueDate)", entry.DTEnd, "2026-09-15T11:00:00Z")
	}
	if entry.Location != "" {
		t.Errorf("Location = %q, want empty", entry.Location)
	}
	if entry.IsAllDay {
		t.Error("IsAllDay should be false")
	}

	if entry.Description == "" {
		t.Error("Description should not be empty")
	}
	if !containsStr(entry.Description, "Quiz description") {
		t.Errorf("Description should contain 'Quiz description', got: %s", entry.Description)
	}
	if !containsStr(entry.Description, "Duration: 30 minutes") {
		t.Errorf("Description should contain 'Duration: 30 minutes', got: %s", entry.Description)
	}
	if !containsStr(entry.Description, "3 attempts allowed") {
		t.Errorf("Description should contain '3 attempts allowed', got: %s", entry.Description)
	}
}

func TestQuizToStringEntry_UnlimitedAttempts(t *testing.T) {
	quiz := QuizAPI{
		Name:        "Practice Quiz",
		StartDate:   "2026-09-15T10:00:00Z",
		DueDate:     "2026-09-15T10:30:00Z",
		AttemptsAllowed: Attempts{IsUnlimited: true},
	}

	entry := QuizToStringEntry(quiz)

	if !containsStr(entry.Description, "Unlimited attempts") {
		t.Errorf("Description should contain 'Unlimited attempts', got: %s", entry.Description)
	}
}

func TestQuizToStringEntry_NoTimeLimit(t *testing.T) {
	quiz := QuizAPI{
		Name:                "Simple Quiz",
		StartDate:           "2026-09-15T10:00:00Z",
		DueDate:             "2026-09-15T10:30:00Z",
		SubmissionTimeLimit: TimeLimit{IsEnforced: false},
	}

	entry := QuizToStringEntry(quiz)

	if containsStr(entry.Description, "Duration:") {
		t.Errorf("Description should not contain 'Duration:' when not enforced, got: %s", entry.Description)
	}
}

func TestQuizEntriesToEvents(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Singapore")
	blocklist := &Blocklist{}

	entries := []BrightSpaceStringEntry{
		{
			Title:       "Quiz 1",
			OrgUnitId:   "12345",
			OrgUnitName: "SIT1001",
			OrgUnitCode: "SIT1001",
			DTStart:     "2026-09-15T10:00:00Z",
			DTEnd:       "2026-09-15T11:00:00Z",
			Source:      "brightspace-quizzes",
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
	if event.Title != "Quiz 1" {
		t.Errorf("Title = %q, want %q", event.Title, "Quiz 1")
	}
}

func TestQuizEntriesToEvents_Blocked(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Singapore")
	blocklist := &Blocklist{
		QuizTitlePatterns: []string{"Quiz 1"},
	}

	entries := []BrightSpaceStringEntry{
		{
			Title:       "Quiz 1",
			OrgUnitId:   "12345",
			OrgUnitName: "SIT1001",
			OrgUnitCode: "SIT1001",
			DTStart:     "2026-09-15T10:00:00Z",
			DTEnd:       "2026-09-15T11:00:00Z",
			Source:      "brightspace-quizzes",
		},
		{
			Title:       "Quiz 2",
			OrgUnitId:   "12345",
			OrgUnitName: "SIT1001",
			OrgUnitCode: "SIT1001",
			DTStart:     "2026-09-16T10:00:00Z",
			DTEnd:       "2026-09-16T11:00:00Z",
			Source:      "brightspace-quizzes",
		},
	}

	events := QuizEntriesToEvents(entries, blocklist, loc)

	if len(events) != 1 {
		t.Fatalf("Expected 1 event (Quiz 1 blocked), got %d", len(events))
	}
	if events[0].Title != "Quiz 2" {
		t.Errorf("Expected remaining event to be 'Quiz 2', got %q", events[0].Title)
	}
}

func TestBlocklist_IsQuizBlocked(t *testing.T) {
	blocklist := &Blocklist{
		QuizTitlePatterns: []string{"midterm", "final"},
	}

	tests := []struct {
		title   string
		blocked bool
	}{
		{"Midterm Exam", true},
		{"MIDTERM PRACTICE", true},
		{"Final Project", true},
		{"Quiz 1", false},
		{"Homework 2", false},
	}

	for _, tt := range tests {
		result := blocklist.IsQuizBlocked(tt.title)
		if result != tt.blocked {
			t.Errorf("IsQuizBlocked(%q) = %v, want %v", tt.title, result, tt.blocked)
		}
	}
}

func TestBlocklist_Matches_Quiz(t *testing.T) {
	blocklist := &Blocklist{
		QuizTitlePatterns: []string{"practice"},
	}

	tests := []struct {
		orgUnitID, orgUnitName, title, location string
		want                                   bool
	}{
		{"12345", "SIT1001", "Practice Quiz 1", "", true},
		{"12345", "SIT1001", "Quiz 1", "", false},
	}

	for _, tt := range tests {
		result := blocklist.Matches(tt.orgUnitID, tt.orgUnitName, tt.title, tt.location)
		if result != tt.want {
			t.Errorf("Matches(%q, %q, %q, %q) = %v, want %v",
				tt.orgUnitID, tt.orgUnitName, tt.title, tt.location, result, tt.want)
		}
	}
}

func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
