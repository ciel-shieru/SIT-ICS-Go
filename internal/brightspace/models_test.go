package brightspace

import (
	"encoding/json"
	"testing"
)

func TestCalendarEventAPI_WithAssociatedEntity(t *testing.T) {
	jsonStr := `{
		"CalendarEventId": 99001,
		"OrgUnitId": 12345,
		"Title": "Quiz 1",
		"Description": "Test description",
		"IsAllDayEvent": false,
		"StartDateTime": "2026-09-15T10:00:00Z",
		"EndDateTime": "2026-09-15T11:00:00Z",
		"IsRecurring": false,
		"LocationName": "Zoom Online Meeting",
		"OrgUnitName": "MOD1001-Sample Module",
		"OrgUnitCode": "MOD1001",
		"EventType": 1,
		"QuizId": 99010,
		"AssociatedEntity": {
			"AssociatedEntityType": "D2L.LE.Quizzing.Quiz",
			"AssociatedEntityId": 99002,
			"Link": "/d2l/le/quizzing/99002"
		}
	}`

	var ev CalendarEventAPI
	err := json.Unmarshal([]byte(jsonStr), &ev)
	if err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if ev.CalendarEventId != 99001 {
		t.Errorf("CalendarEventId = %d, want 99001", ev.CalendarEventId)
	}
	if ev.QuizId != 99010 {
		t.Errorf("QuizId = %d, want 99010", ev.QuizId)
	}
	if ev.AssociatedEntity == nil {
		t.Fatal("AssociatedEntity should not be nil")
	}
	if ev.AssociatedEntity.AssociatedEntityType != "D2L.LE.Quizzing.Quiz" {
		t.Errorf("AssociatedEntityType = %q, want %q", ev.AssociatedEntity.AssociatedEntityType, "D2L.LE.Quizzing.Quiz")
	}
	if ev.AssociatedEntity.AssociatedEntityId != 99002 {
		t.Errorf("AssociatedEntityId = %d, want 99002", ev.AssociatedEntity.AssociatedEntityId)
	}
	if ev.AssociatedEntity.Link != "/d2l/le/quizzing/99002" {
		t.Errorf("Link = %q, want %q", ev.AssociatedEntity.Link, "/d2l/le/quizzing/99002")
	}
}

func TestCalendarEventAPI_WithoutAssociatedEntity(t *testing.T) {
	jsonStr := `{
		"CalendarEventId": 99003,
		"OrgUnitId": 12346,
		"Title": "Lecture 1",
		"StartDateTime": "2026-09-15T09:00:00Z",
		"EndDateTime": "2026-09-15T10:00:00Z",
		"OrgUnitName": "MOD1002-Lecture Module",
		"OrgUnitCode": "MOD1002",
		"QuizId": 0
	}`

	var ev CalendarEventAPI
	err := json.Unmarshal([]byte(jsonStr), &ev)
	if err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if ev.CalendarEventId != 99003 {
		t.Errorf("CalendarEventId = %d, want 99003", ev.CalendarEventId)
	}
	if ev.QuizId != 0 {
		t.Errorf("QuizId = %d, want 0", ev.QuizId)
	}
	if ev.AssociatedEntity != nil {
		t.Errorf("AssociatedEntity should be nil, got %+v", ev.AssociatedEntity)
	}
}

func TestCalendarEventAPI_AssociatedEntity_OtherTypes(t *testing.T) {
	jsonStr := `{
		"CalendarEventId": 99004,
		"Title": "Assignment",
		"StartDateTime": "2026-09-15T09:00:00Z",
		"EndDateTime": "2026-09-15T10:00:00Z",
		"AssociatedEntity": {
			"AssociatedEntityType": "D2L.LE.Dropbox.Folder",
			"AssociatedEntityId": 99005,
			"Link": "/d2l/le/dropbox/99005"
		}
	}`

	var ev CalendarEventAPI
	err := json.Unmarshal([]byte(jsonStr), &ev)
	if err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if ev.AssociatedEntity.AssociatedEntityType != "D2L.LE.Dropbox.Folder" {
		t.Errorf("AssociatedEntityType = %q, want %q", ev.AssociatedEntity.AssociatedEntityType, "D2L.LE.Dropbox.Folder")
	}
	if ev.AssociatedEntity.AssociatedEntityId != 99005 {
		t.Errorf("AssociatedEntityId = %d, want 99005", ev.AssociatedEntity.AssociatedEntityId)
	}
}
