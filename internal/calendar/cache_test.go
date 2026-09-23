package calendar

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestICSCacheUpdateAndSave(t *testing.T) {
	tmpDir := t.TempDir()
	icsPath := filepath.Join(tmpDir, "test.ics")

	cache := NewICSCache()
	events := []Event{
		{
			DTStart: time.Date(2026, 9, 7, 9, 0, 0, 0, time.UTC),
			DTEnd:   time.Date(2026, 9, 7, 11, 0, 0, 0, time.UTC),
			Summary: "Test Event",
			Location: "Room 101",
		},
	}

	if err := cache.Update(events, "Asia/Singapore"); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	if err := cache.SaveToFile(icsPath, time.Hour); err != nil {
		t.Fatalf("SaveToFile() error = %v", err)
	}

	data, err := os.ReadFile(icsPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	content := string(data)
	if !contains(content, "BEGIN:VCALENDAR") {
		t.Error("Saved ICS missing BEGIN:VCALENDAR")
	}
	if !contains(content, "SUMMARY:Test Event") {
		t.Error("Saved ICS missing SUMMARY")
	}
}

func TestICSCacheLoadFromFile(t *testing.T) {
	tmpDir := t.TempDir()
	icsPath := filepath.Join(tmpDir, "test.ics")

	initialData := []byte("BEGIN:VCALENDAR\r\nVERSION:2.0\r\nBEGIN:VEVENT\r\nUID:existing-uid\r\nSUMMARY:Existing Event\r\nEND:VEVENT\r\nEND:VCALENDAR\r\n")
	if err := os.WriteFile(icsPath, initialData, 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cache := NewICSCache()
	if err := cache.LoadFromFile(icsPath, time.UTC); err != nil {
		t.Fatalf("LoadFromFile() error = %v", err)
	}

	data := cache.Get("Asia/Singapore", time.Hour)
	if !contains(string(data), "SUMMARY:Existing Event") {
		t.Error("Loaded ICS missing existing event")
	}
}

func TestICSCacheUpsert(t *testing.T) {
	tmpDir := t.TempDir()
	icsPath := filepath.Join(tmpDir, "test.ics")

	cache := NewICSCache()

	events1 := []Event{
		{
			DTStart: time.Date(2026, 9, 7, 9, 0, 0, 0, time.UTC),
			DTEnd:   time.Date(2026, 9, 7, 11, 0, 0, 0, time.UTC),
			Summary: "Event 1",
			Location: "Room 101",
		},
	}

	if err := cache.Update(events1, "Asia/Singapore"); err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if err := cache.SaveToFile(icsPath, time.Hour); err != nil {
		t.Fatalf("SaveToFile() error = %v", err)
	}

	events2 := []Event{
		{
			DTStart: time.Date(2026, 9, 7, 14, 0, 0, 0, time.UTC),
			DTEnd:   time.Date(2026, 9, 7, 16, 0, 0, 0, time.UTC),
			Summary: "Event 2",
			Location: "Room 102",
		},
	}

	if err := cache.Update(events2, "Asia/Singapore"); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	data := cache.Get("Asia/Singapore", time.Hour)
	content := string(data)

	if !contains(content, "SUMMARY:Event 1") {
		t.Error("Upsert removed existing event (should preserve)")
	}
	if !contains(content, "SUMMARY:Event 2") {
		t.Error("Upsert failed to add new event")
	}
}

func TestICSCacheLoadNonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	icsPath := filepath.Join(tmpDir, "nonexistent.ics")

	cache := NewICSCache()
	if err := cache.LoadFromFile(icsPath, time.UTC); err != nil {
		t.Fatalf("LoadFromFile() should not error for non-existent file: %v", err)
	}

	if cache.EventCount() != 0 {
		t.Error("LoadFromFile() should return 0 events for non-existent file")
	}
}

func TestICSCacheNoSaveWhenNotDirty(t *testing.T) {
	tmpDir := t.TempDir()
	icsPath := filepath.Join(tmpDir, "test.ics")

	cache := NewICSCache()

	initialData := []byte("BEGIN:VCALENDAR\r\nVERSION:2.0\r\nEND:VCALENDAR\r\n")
	os.WriteFile(icsPath, initialData, 0600)

	cache.LoadFromFile(icsPath, time.UTC)

	if err := cache.SaveToFile(icsPath, time.Hour); err != nil {
		t.Fatalf("SaveToFile() error = %v", err)
	}

	data, _ := os.ReadFile(icsPath)
	if string(data) != string(initialData) {
		t.Error("SaveToFile() should not modify file when not dirty")
	}
}

func TestICSCacheFilterOnline(t *testing.T) {
	cache := NewICSCache()
	events := []Event{
		{
			Summary:  "Online Class",
			Location: "Online",
			DTStart:  time.Date(2026, 9, 7, 9, 0, 0, 0, time.UTC),
			DTEnd:    time.Date(2026, 9, 7, 11, 0, 0, 0, time.UTC),
		},
		{
			Summary:  "Campus Class",
			Location: "W1-05-07",
			DTStart:  time.Date(2026, 9, 7, 14, 0, 0, 0, time.UTC),
			DTEnd:    time.Date(2026, 9, 7, 16, 0, 0, 0, time.UTC),
		},
	}

	if err := cache.Update(events, "Asia/Singapore"); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	onlineData := cache.GetFiltered("Asia/Singapore", IsOnline, time.Hour)
	onlineContent := string(onlineData)

	if !contains(onlineContent, "SUMMARY:Online Class") {
		t.Error("Filtered online ICS missing online event")
	}
	if contains(onlineContent, "SUMMARY:Campus Class") {
		t.Error("Filtered online ICS should not contain campus event")
	}

	campusData := cache.GetFiltered("Asia/Singapore", IsNotOnline, time.Hour)
	campusContent := string(campusData)

	if !contains(campusContent, "SUMMARY:Campus Class") {
		t.Error("Filtered campus ICS missing campus event")
	}
	if contains(campusContent, "SUMMARY:Online Class") {
		t.Error("Filtered campus ICS should not contain online event")
	}
}

func TestICSCacheSaveAllToFiles(t *testing.T) {
	tmpDir := t.TempDir()
	mainPath := filepath.Join(tmpDir, "timetable.ics")
	onlinePath := filepath.Join(tmpDir, "timetable-online.ics")
	campusPath := filepath.Join(tmpDir, "timetable-campus.ics")

	cache := NewICSCache()
	events := []Event{
		{
			Summary:  "Online Class",
			Location: "Online",
			DTStart:  time.Date(2026, 9, 7, 9, 0, 0, 0, time.UTC),
			DTEnd:    time.Date(2026, 9, 7, 11, 0, 0, 0, time.UTC),
		},
		{
			Summary:  "Campus Class",
			Location: "W1-05-07",
			DTStart:  time.Date(2026, 9, 7, 14, 0, 0, 0, time.UTC),
			DTEnd:    time.Date(2026, 9, 7, 16, 0, 0, 0, time.UTC),
		},
	}

	if err := cache.Update(events, "Asia/Singapore"); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	if err := cache.SaveAllToFiles(mainPath, onlinePath, campusPath, "Asia/Singapore", time.Hour); err != nil {
		t.Fatalf("SaveAllToFiles() error = %v", err)
	}

	mainData, _ := os.ReadFile(mainPath)
	if !contains(string(mainData), "SUMMARY:Online Class") {
		t.Error("Main ICS missing online event")
	}
	if !contains(string(mainData), "SUMMARY:Campus Class") {
		t.Error("Main ICS missing campus event")
	}

	onlineData, _ := os.ReadFile(onlinePath)
	if !contains(string(onlineData), "SUMMARY:Online Class") {
		t.Error("Online ICS missing online event")
	}
	if contains(string(onlineData), "SUMMARY:Campus Class") {
		t.Error("Online ICS should not contain campus event")
	}

	campusData, _ := os.ReadFile(campusPath)
	if !contains(string(campusData), "SUMMARY:Campus Class") {
		t.Error("Campus ICS missing campus event")
	}
	if contains(string(campusData), "SUMMARY:Online Class") {
		t.Error("Campus ICS should not contain online event")
	}
}

func TestICSCacheEventCount(t *testing.T) {
	cache := NewICSCache()

	if count := cache.EventCount(); count != 0 {
		t.Errorf("Expected 0 events, got %d", count)
	}

	events := []Event{
		{Summary: "Event 1", Location: "Room 101"},
		{Summary: "Event 2", Location: "Online"},
		{Summary: "Event 3", Location: "Room 102"},
	}

	if err := cache.Update(events, "Asia/Singapore"); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	if count := cache.EventCount(); count != 3 {
		t.Errorf("Expected 3 events, got %d", count)
	}
}

func TestICSCacheSaveOutputs_ExcludesBrightSpaceFromCampus(t *testing.T) {
	tmpDir := t.TempDir()
	mainPath := filepath.Join(tmpDir, "timetable.ics")
	onlinePath := filepath.Join(tmpDir, "timetable-online.ics")
	campusPath := filepath.Join(tmpDir, "timetable-campus.ics")
	xsiteEventsPath := filepath.Join(tmpDir, "xsite-events.ics")
	xsiteDropboxPath := filepath.Join(tmpDir, "xsite-dropbox.ics")
	xsiteQuizzesPath := filepath.Join(tmpDir, "xsite-quizzes.ics")
	xsitePath := filepath.Join(tmpDir, "xsite.ics")

	cache := NewICSCache()
	events := []Event{
		{
			Summary:  "Campus Class",
			Location: "W1-05-07",
			DTStart:  time.Date(2026, 9, 7, 14, 0, 0, 0, time.UTC),
			DTEnd:    time.Date(2026, 9, 7, 16, 0, 0, 0, time.UTC),
		},
		{
			Summary:  "[COR2001] Assignment 1",
			Location: "Online",
			DTStart:  time.Date(2026, 9, 8, 9, 0, 0, 0, time.UTC),
			DTEnd:    time.Date(2026, 9, 8, 10, 0, 0, 0, time.UTC),
			Source:   "brightspace-calendar",
		},
		{
			Summary:  "[COR2002] Lab 3 Due",
			Location: "",
			DTStart:  time.Date(2026, 9, 9, 23, 59, 0, 0, time.UTC),
			DTEnd:    time.Date(2026, 9, 10, 23, 59, 0, 0, time.UTC),
			Source:   "brightspace-dropbox",
		},
	}

	if err := cache.Update(events, "Asia/Singapore"); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	if err := cache.SaveOutputs(Outputs{
		Main:         mainPath,
		Online:       onlinePath,
		Campus:       campusPath,
		XsiteEvents:  xsiteEventsPath,
		XsiteDropbox: xsiteDropboxPath,
		XsiteQuizzes: xsiteQuizzesPath,
		Xsite:        xsitePath,
	}, "Asia/Singapore", time.Hour, nil, nil, nil, nil, nil, nil); err != nil {
		t.Fatalf("SaveOutputs() error = %v", err)
	}

	mainData, _ := os.ReadFile(mainPath)
	if !contains(string(mainData), "SUMMARY:Campus Class") {
		t.Error("Main ICS missing campus event")
	}
	if !contains(string(mainData), "SUMMARY:[COR2001] Assignment 1") {
		t.Error("Main ICS missing brightspace-calendar event")
	}
	if !contains(string(mainData), "SUMMARY:[COR2002] Lab 3 Due") {
		t.Error("Main ICS missing brightspace-dropbox event")
	}

	campusData, _ := os.ReadFile(campusPath)
	if !contains(string(campusData), "SUMMARY:Campus Class") {
		t.Error("Campus ICS missing campus event")
	}
	if contains(string(campusData), "SUMMARY:[COR2001] Assignment 1") {
		t.Error("Campus ICS should not contain brightspace-calendar event")
	}
	if contains(string(campusData), "SUMMARY:[COR2002] Lab 3 Due") {
		t.Error("Campus ICS should not contain brightspace-dropbox event")
	}

	xsiteEventsData, _ := os.ReadFile(xsiteEventsPath)
	if !contains(string(xsiteEventsData), "SUMMARY:[COR2001] Assignment 1") {
		t.Error("xsite events ICS missing brightspace-calendar event")
	}
	if contains(string(xsiteEventsData), "SUMMARY:[COR2002] Lab 3 Due") {
		t.Error("xsite events ICS should not contain brightspace-dropbox event")
	}

	xsiteDropboxData, _ := os.ReadFile(xsiteDropboxPath)
	if !contains(string(xsiteDropboxData), "SUMMARY:[COR2002] Lab 3 Due") {
		t.Error("xsite dropbox ICS missing brightspace-dropbox event")
	}
	if contains(string(xsiteDropboxData), "SUMMARY:[COR2001] Assignment 1") {
		t.Error("xsite dropbox ICS should not contain brightspace-calendar event")
	}

	xsiteComboData, _ := os.ReadFile(xsitePath)
	if !contains(string(xsiteComboData), "SUMMARY:[COR2001] Assignment 1") {
		t.Error("xsite combo ICS missing brightspace-calendar event")
	}
	if !contains(string(xsiteComboData), "SUMMARY:[COR2002] Lab 3 Due") {
		t.Error("xsite combo ICS missing brightspace-dropbox event")
	}
	if contains(string(xsiteComboData), "SUMMARY:Campus Class") {
		t.Error("xsite combo ICS should not contain non-brightspace event")
	}
}

func TestICSCacheSaveOutputs_RoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	mainPath := filepath.Join(tmpDir, "timetable.ics")
	onlinePath := filepath.Join(tmpDir, "timetable-online.ics")
	campusPath := filepath.Join(tmpDir, "timetable-campus.ics")
	xsiteEventsPath := filepath.Join(tmpDir, "xsite-events.ics")
	xsiteDropboxPath := filepath.Join(tmpDir, "xsite-dropbox.ics")
	xsiteQuizzesPath := filepath.Join(tmpDir, "xsite-quizzes.ics")
	xsitePath := filepath.Join(tmpDir, "xsite.ics")

	cache := NewICSCache()
	events := []Event{
		{
			Summary:  "Campus Class",
			Location: "W1-05-07",
			DTStart:  time.Date(2026, 9, 7, 14, 0, 0, 0, time.UTC),
			DTEnd:    time.Date(2026, 9, 7, 16, 0, 0, 0, time.UTC),
		},
		{
			Summary:  "[COR2001] Assignment 1",
			Location: "Online",
			DTStart:  time.Date(2026, 9, 8, 9, 0, 0, 0, time.UTC),
			DTEnd:    time.Date(2026, 9, 8, 10, 0, 0, 0, time.UTC),
			Source:   "brightspace-calendar",
		},
	}

	if err := cache.Update(events, "Asia/Singapore"); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	if err := cache.SaveOutputs(Outputs{
		Main:         mainPath,
		Online:       onlinePath,
		Campus:       campusPath,
		XsiteEvents:  xsiteEventsPath,
		XsiteDropbox: xsiteDropboxPath,
		XsiteQuizzes: xsiteQuizzesPath,
		Xsite:        xsitePath,
	}, "Asia/Singapore", time.Hour, nil, nil, nil, nil, nil, nil); err != nil {
		t.Fatalf("SaveOutputs() error = %v", err)
	}

	campusData1, _ := os.ReadFile(campusPath)
	if contains(string(campusData1), "SUMMARY:[COR2001] Assignment 1") {
		t.Error("First run: Campus ICS should not contain brightspace-calendar event")
	}

	cache2 := NewICSCache()
	if err := cache2.LoadFromFile(mainPath, time.UTC); err != nil {
		t.Fatalf("LoadFromFile() error = %v", err)
	}

	filtered := cache2.GetFiltered("Asia/Singapore", func(e Event) bool {
		return e.Summary == "[COR2001] Assignment 1"
	}, time.Hour)
	content := string(filtered)
	if !contains(content, "X-SOURCE:brightspace-calendar") {
		t.Error("Reloaded brightspace event missing Source=brightspace-calendar")
	}

	newEvents := []Event{
		{
			Summary:  "New Campus Class",
			Location: "W2-03-04",
			DTStart:  time.Date(2026, 9, 10, 10, 0, 0, 0, time.UTC),
			DTEnd:    time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC),
		},
	}

	if err := cache2.Update(newEvents, "Asia/Singapore"); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	if err := cache2.SaveOutputs(Outputs{
		Main:         mainPath,
		Online:       onlinePath,
		Campus:       campusPath,
		XsiteEvents:  xsiteEventsPath,
		XsiteDropbox: xsiteDropboxPath,
		XsiteQuizzes: xsiteQuizzesPath,
		Xsite:        xsitePath,
	}, "Asia/Singapore", time.Hour, nil, nil, nil, nil, nil, nil); err != nil {
		t.Fatalf("SaveOutputs() error = %v", err)
	}

	campusData2, _ := os.ReadFile(campusPath)
	if !contains(string(campusData2), "SUMMARY:Campus Class") {
		t.Error("Second run: Campus ICS missing original campus event")
	}
	if !contains(string(campusData2), "SUMMARY:New Campus Class") {
		t.Error("Second run: Campus ICS missing new campus event")
	}
	if contains(string(campusData2), "SUMMARY:[COR2001] Assignment 1") {
		t.Error("Second run: Campus ICS should not contain brightspace-calendar event")
	}
}

func TestICSCacheDeleteByPredicate(t *testing.T) {
	cache := NewICSCache()
	events := []Event{
		{
			Summary:  "Online Class",
			Location: "Online",
			Source:   "brightspace-calendar",
			Title:    "Online Class",
			DTStart:  time.Date(2026, 9, 7, 9, 0, 0, 0, time.UTC),
			DTEnd:    time.Date(2026, 9, 7, 11, 0, 0, 0, time.UTC),
		},
		{
			Summary:  "Campus Class",
			Location: "W1-05-07",
			DTStart:  time.Date(2026, 9, 7, 14, 0, 0, 0, time.UTC),
			DTEnd:    time.Date(2026, 9, 7, 16, 0, 0, 0, time.UTC),
		},
		{
			Summary:  "Zoom Meeting",
			Location: "Zoom Online Meeting",
			Source:   "brightspace-calendar",
			Title:    "Zoom Meeting",
			DTStart:  time.Date(2026, 9, 8, 10, 0, 0, 0, time.UTC),
			DTEnd:    time.Date(2026, 9, 8, 11, 0, 0, 0, time.UTC),
		},
	}

	if err := cache.Update(events, "Asia/Singapore"); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	if count := cache.EventCount(); count != 3 {
		t.Fatalf("Expected 3 events, got %d", count)
	}

	deleted := cache.RemoveWhere(func(e Event) bool {
		return e.Location == "Zoom Online Meeting"
	})

	if deleted != 1 {
		t.Errorf("Expected 1 deleted event, got %d", deleted)
	}

	if count := cache.EventCount(); count != 2 {
		t.Errorf("Expected 2 events after delete, got %d", count)
	}

	data := cache.Get("Asia/Singapore", time.Hour)
	content := string(data)
	if contains(content, "Zoom Meeting") {
		t.Error("Deleted event should not be in cache")
	}
	if !contains(content, "Campus Class") {
		t.Error("Non-matching event should still be in cache")
	}
	if !contains(content, "Online Class") {
		t.Error("Non-matching event should still be in cache")
	}
}

func TestICSCacheDeleteByPredicate_NoMatch(t *testing.T) {
	cache := NewICSCache()
	events := []Event{
		{Summary: "Event 1", Location: "Room 101"},
		{Summary: "Event 2", Location: "Room 102"},
	}

	if err := cache.Update(events, "Asia/Singapore"); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	deleted := cache.RemoveWhere(func(e Event) bool {
		return e.Location == "Nonexistent"
	})

	if deleted != 0 {
		t.Errorf("Expected 0 deleted events, got %d", deleted)
	}

	if count := cache.EventCount(); count != 2 {
		t.Errorf("Expected 2 events, got %d", count)
	}
}

func TestICSCacheEventRetention(t *testing.T) {
	cache := NewICSCache()

	eventsAB := []Event{
		{
			Summary:  "Event A",
			Location: "Room 101",
			DTStart:  time.Date(2026, 9, 7, 9, 0, 0, 0, time.UTC),
			DTEnd:    time.Date(2026, 9, 7, 11, 0, 0, 0, time.UTC),
		},
		{
			Summary:  "Event B",
			Location: "Room 102",
			DTStart:  time.Date(2026, 9, 7, 14, 0, 0, 0, time.UTC),
			DTEnd:    time.Date(2026, 9, 7, 16, 0, 0, 0, time.UTC),
		},
	}

	if err := cache.Update(eventsAB, "Asia/Singapore"); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	if count := cache.EventCount(); count != 2 {
		t.Fatalf("Expected 2 events after first merge, got %d", count)
	}

	eventsA := []Event{
		{
			Summary:  "Event A",
			Location: "Room 101",
			DTStart:  time.Date(2026, 9, 7, 9, 0, 0, 0, time.UTC),
			DTEnd:    time.Date(2026, 9, 7, 11, 0, 0, 0, time.UTC),
		},
	}

	if err := cache.Update(eventsA, "Asia/Singapore"); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	if count := cache.EventCount(); count != 2 {
		t.Errorf("Expected 2 events after second merge (B retained), got %d", count)
	}

	data := cache.Get("Asia/Singapore", time.Hour)
	content := string(data)

	if !contains(content, "SUMMARY:Event A") {
		t.Error("Event A should still be in cache")
	}
	if !contains(content, "SUMMARY:Event B") {
		t.Error("Event B should still be in cache (event retention)")
	}
}

func TestICSCacheRepeatedMergeIdempotency(t *testing.T) {
	cache := NewICSCache()

	events := []Event{
		{
			Summary:  "Repeated Event",
			Location: "Room 101",
			DTStart:  time.Date(2026, 9, 7, 9, 0, 0, 0, time.UTC),
			DTEnd:    time.Date(2026, 9, 7, 11, 0, 0, 0, time.UTC),
		},
	}

	for i := 0; i < 3; i++ {
		if err := cache.Update(events, "Asia/Singapore"); err != nil {
			t.Fatalf("Update() iteration %d error = %v", i+1, err)
		}
	}

	if count := cache.EventCount(); count != 1 {
		t.Errorf("Expected 1 event after 3 identical merges (no duplicates), got %d", count)
	}

	data := cache.Get("Asia/Singapore", time.Hour)
	content := string(data)

	summaryCount := strings.Count(content, "SUMMARY:Repeated Event")
	if summaryCount != 1 {
		t.Errorf("Expected 1 SUMMARY, got %d (duplicates detected)", summaryCount)
	}
}

func TestICSCacheDeleteByPredicate_BrightSpaceBlocklist(t *testing.T) {
	cache := NewICSCache()

	events := []Event{
		{
			Summary:  "Campus Class",
			Location: "W1-05-07",
			DTStart:  time.Date(2026, 9, 7, 14, 0, 0, 0, time.UTC),
			DTEnd:    time.Date(2026, 9, 7, 16, 0, 0, 0, time.UTC),
		},
		{
			Summary:  "[COR2001] Assignment 1",
			Location: "Online",
			DTStart:  time.Date(2026, 9, 8, 9, 0, 0, 0, time.UTC),
			DTEnd:    time.Date(2026, 9, 8, 10, 0, 0, 0, time.UTC),
			Source:   "brightspace-calendar",
		},
		{
			Summary:  "[COR2002] Lab 3 Due",
			Location: "",
			DTStart:  time.Date(2026, 9, 9, 23, 59, 0, 0, time.UTC),
			DTEnd:    time.Date(2026, 9, 10, 23, 59, 0, 0, time.UTC),
			Source:   "brightspace-dropbox",
		},
	}

	if err := cache.Update(events, "Asia/Singapore"); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	if count := cache.EventCount(); count != 3 {
		t.Fatalf("Expected 3 events, got %d", count)
	}

	deleted := cache.RemoveWhere(func(e Event) bool {
		return strings.HasPrefix(e.Source, "brightspace-")
	})

	if deleted != 2 {
		t.Errorf("Expected 2 deleted BrightSpace events, got %d", deleted)
	}

	if count := cache.EventCount(); count != 1 {
		t.Errorf("Expected 1 event after deletion, got %d", count)
	}

	data := cache.Get("Asia/Singapore", time.Hour)
	content := string(data)

	if !contains(content, "SUMMARY:Campus Class") {
		t.Error("Non-BrightSpace event should still be in cache")
	}
	if contains(content, "SUMMARY:[COR2001] Assignment 1") {
		t.Error("BrightSpace-calendar event should be deleted")
	}
	if contains(content, "SUMMARY:[COR2002] Lab 3 Due") {
		t.Error("BrightSpace-dropbox event should be deleted")
	}
}

func TestICSCacheDeterministicOutput(t *testing.T) {
	cache := NewICSCache()

	events := []Event{
		{
			Summary:  "Event A",
			Location: "Room 101",
			DTStart:  time.Date(2026, 9, 7, 9, 0, 0, 0, time.UTC),
			DTEnd:    time.Date(2026, 9, 7, 11, 0, 0, 0, time.UTC),
		},
		{
			Summary:  "Event B",
			Location: "Room 102",
			DTStart:  time.Date(2026, 9, 7, 11, 0, 0, 0, time.UTC),
			DTEnd:    time.Date(2026, 9, 7, 13, 0, 0, 0, time.UTC),
		},
	}

	if err := cache.Update(events, "Asia/Singapore"); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	data1 := cache.Get("Asia/Singapore", time.Hour)
	data2 := cache.Get("Asia/Singapore", time.Hour)

	if !bytes.Equal(data1, data2) {
		t.Error("Identical cache state should produce identical ICS output")
	}

	cache2 := NewICSCache()
	if err := cache2.Update(events, "Asia/Singapore"); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	data3 := cache2.Get("Asia/Singapore", time.Hour)
	if !bytes.Equal(data1, data3) {
		t.Error("Separate caches with same events should produce identical output")
	}
}

func TestICSCacheLastModifiedInitialZero(t *testing.T) {
	cache := NewICSCache()
	if !cache.GetLastModified().IsZero() {
		t.Errorf("Expected zero lastModified for new cache, got %v", cache.GetLastModified())
	}
}

func TestICSCacheLastModifiedSetOnUpdate(t *testing.T) {
	cache := NewICSCache()

	if !cache.GetLastModified().IsZero() {
		t.Fatal("Expected zero lastModified before update")
	}

	events := []Event{
		{Summary: "Test", DTStart: time.Date(2026, 9, 7, 9, 0, 0, 0, time.UTC), DTEnd: time.Date(2026, 9, 7, 11, 0, 0, 0, time.UTC)},
	}

	if err := cache.Update(events, "Asia/Singapore"); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	lastMod := cache.GetLastModified()
	if lastMod.IsZero() {
		t.Fatal("Expected non-zero lastModified after Update()")
	}
	if lastMod.After(time.Now().UTC()) {
		t.Error("lastModified should not be in the future")
	}
}

func TestICSCacheLastModifiedSetOnLoad(t *testing.T) {
	tmpDir := t.TempDir()
	icsPath := filepath.Join(tmpDir, "test.ics")

	initialData := []byte("BEGIN:VCALENDAR\r\nVERSION:2.0\r\nBEGIN:VEVENT\r\nUID:test-uid\r\nSUMMARY:Loaded Event\r\nEND:VEVENT\r\nEND:VCALENDAR\r\n")
	if err := os.WriteFile(icsPath, initialData, 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	fi, err := os.Stat(icsPath)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	fileModTime := fi.ModTime()

	cache := NewICSCache()
	if err := cache.LoadFromFile(icsPath, time.UTC); err != nil {
		t.Fatalf("LoadFromFile() error = %v", err)
	}

	lastMod := cache.GetLastModified()
	if lastMod.IsZero() {
		t.Fatal("Expected non-zero lastModified after LoadFromFile()")
	}
	if lastMod.Sub(fileModTime).Abs() > time.Second {
		t.Errorf("lastModified = %v, want ~%v (file mod time)", lastMod, fileModTime)
	}
}

func TestICSCacheComputeETagDeterministic(t *testing.T) {
	data := []byte("BEGIN:VCALENDAR\r\nVERSION:2.0\r\nEND:VCALENDAR\r\n")

	etag1 := ComputeETag(data)
	etag2 := ComputeETag(data)

	if etag1 != etag2 {
		t.Errorf("Same data produced different ETags: %q vs %q", etag1, etag2)
	}
	if etag1 == "" {
		t.Error("ETag should not be empty")
	}
}

func TestICSCacheComputeETagDifferentData(t *testing.T) {
	data1 := []byte("BEGIN:VCALENDAR\r\nVERSION:2.0\r\nEND:VCALENDAR\r\n")
	data2 := []byte("BEGIN:VCALENDAR\r\nVERSION:2.0\r\nBEGIN:VEVENT\r\nUID:test\r\nEND:VEVENT\r\nEND:VCALENDAR\r\n")

	etag1 := ComputeETag(data1)
	etag2 := ComputeETag(data2)

	if etag1 == etag2 {
		t.Errorf("Different data produced same ETag: %q", etag1)
	}
}

func TestICSCacheComputeETagWeakFormat(t *testing.T) {
	data := []byte("test")
	etag := ComputeETag(data)

	if !strings.HasPrefix(etag, `W/"`) {
		t.Errorf("ETag should start with W/\", got: %q", etag)
	}
	if !strings.HasSuffix(etag, `"`) {
		t.Errorf("ETag should end with \", got: %q", etag)
	}
}

func TestICSCachePersistenceFailure(t *testing.T) {
	cache := NewICSCache()

	events := []Event{
		{
			Summary:  "Test Event",
			Location: "Room 101",
			DTStart:  time.Date(2026, 9, 7, 9, 0, 0, 0, time.UTC),
			DTEnd:    time.Date(2026, 9, 7, 11, 0, 0, 0, time.UTC),
		},
	}

	if err := cache.Update(events, "Asia/Singapore"); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	invalidPath := filepath.Join(t.TempDir(), "nonexistent_dir", "test.ics")
	err := cache.SaveToFile(invalidPath, time.Hour)
	if err == nil {
		t.Fatal("SaveToFile() should return error when parent directory doesn't exist")
	}

	if !strings.Contains(err.Error(), "write ICS file") {
		t.Errorf("Error should mention write failure, got: %v", err)
	}

	baseDir := t.TempDir()
	mainPath := filepath.Join(baseDir, "nonexistent_dir", "main.ics")
	onlinePath := filepath.Join(baseDir, "nonexistent_dir", "online.ics")
	campusPath := filepath.Join(baseDir, "nonexistent_dir", "campus.ics")

	err = cache.SaveAllToFiles(mainPath, onlinePath, campusPath, "Asia/Singapore", time.Hour)
	if err == nil {
		t.Fatal("SaveAllToFiles() should return error when parent directory doesn't exist")
	}

	if !strings.Contains(err.Error(), "write ICS file") {
		t.Errorf("Error should mention write failure, got: %v", err)
	}
}

func TestICSCacheBrightSpaceFieldsRoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	icsPath := filepath.Join(tmpDir, "test.ics")

	cache := NewICSCache()
	events := []Event{
		{
			Summary:     "[ALT2501] Assignment 1",
			Location:    "Online",
			DTStart:     time.Date(2026, 9, 15, 23, 59, 0, 0, time.UTC),
			DTEnd:       time.Date(2026, 9, 16, 23, 59, 0, 0, time.UTC),
			Source:      "brightspace-calendar",
			OrgUnitID:   "12345678",
			OrgUnitName: "ALT2501 - Software Engineering",
			OrgUnitCode: "ALT2501",
			Title:       "Assignment 1: Requirements Analysis",
		},
	}

	if err := cache.Update(events, "Asia/Singapore"); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	if err := cache.SaveToFile(icsPath, time.Hour); err != nil {
		t.Fatalf("SaveToFile() error = %v", err)
	}

	cache2 := NewICSCache()
	if err := cache2.LoadFromFile(icsPath, time.UTC); err != nil {
		t.Fatalf("LoadFromFile() error = %v", err)
	}

	if cache2.EventCount() != 1 {
		t.Fatalf("Expected 1 event after reload, got %d", cache2.EventCount())
	}

	data := cache2.Get("Asia/Singapore", time.Hour)
	content := string(data)

	if !contains(content, "X-OrgUnitID:12345678") {
		t.Error("Reloaded event missing X-OrgUnitID:12345678")
	}
	if !contains(content, "X-OrgUnitName:ALT2501 - Software Engineering") {
		t.Error("Reloaded event missing X-OrgUnitName")
	}
	if !contains(content, "X-OrgUnitCode:ALT2501") {
		t.Error("Reloaded event missing X-OrgUnitCode:ALT2501")
	}
	if !contains(content, "X-Title:Assignment 1: Requirements Analysis") {
		t.Error("Reloaded event missing X-Title")
	}
}

func TestRemoveWhere_BrightSpaceBlocklistRoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	icsPath := filepath.Join(tmpDir, "test.ics")

	cache := NewICSCache()
	events := []Event{
		{
			Summary:     "Campus Class",
			Location:    "W1-05-07",
			DTStart:     time.Date(2026, 9, 7, 14, 0, 0, 0, time.UTC),
			DTEnd:       time.Date(2026, 9, 7, 16, 0, 0, 0, time.UTC),
		},
		{
			Summary:     "[COR2001] Assignment 1",
			Location:    "Online",
			DTStart:     time.Date(2026, 9, 8, 9, 0, 0, 0, time.UTC),
			DTEnd:       time.Date(2026, 9, 8, 10, 0, 0, 0, time.UTC),
			Source:      "brightspace-calendar",
			OrgUnitID:   "98765432",
			OrgUnitCode: "COR2001",
			Title:       "Assignment 1",
		},
		{
			Summary:     "[COR2002] Lab 3 Due",
			Location:    "",
			DTStart:     time.Date(2026, 9, 9, 23, 59, 0, 0, time.UTC),
			DTEnd:       time.Date(2026, 9, 10, 23, 59, 0, 0, time.UTC),
			Source:      "brightspace-dropbox",
			OrgUnitID:   "11223344",
			OrgUnitCode: "COR2002",
			Title:       "Lab 3",
		},
	}

	if err := cache.Update(events, "Asia/Singapore"); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	if err := cache.SaveToFile(icsPath, time.Hour); err != nil {
		t.Fatalf("SaveToFile() error = %v", err)
	}

	cache2 := NewICSCache()
	if err := cache2.LoadFromFile(icsPath, time.UTC); err != nil {
		t.Fatalf("LoadFromFile() error = %v", err)
	}

	deleted := cache2.RemoveWhere(func(e Event) bool {
		return e.OrgUnitCode == "COR2001" || e.Title == "Lab 3"
	})

	if deleted != 2 {
		t.Errorf("Expected 2 deleted events, got %d", deleted)
	}

	if count := cache2.EventCount(); count != 1 {
		t.Errorf("Expected 1 event after deletion, got %d", count)
	}

	data := cache2.Get("Asia/Singapore", time.Hour)
	content := string(data)

	if !contains(content, "SUMMARY:Campus Class") {
		t.Error("Campus Class should still be in cache")
	}
	if contains(content, "SUMMARY:[COR2001] Assignment 1") {
		t.Error("[COR2001] Assignment 1 should be deleted")
	}
	if contains(content, "SUMMARY:[COR2002] Lab 3 Due") {
		t.Error("[COR2002] Lab 3 Due should be deleted")
	}
}

func TestParseICS_BackwardCompatibility(t *testing.T) {
	icsContent := "BEGIN:VCALENDAR\r\nVERSION:2.0\r\nPRODID:-//SIT Timetable//EN\r\nCALSCALE:GREGORIAN\r\nMETHOD:PUBLISH\r\nREFRESH-INTERVAL;VALUE=DURATION:PT1H\r\nBEGIN:VEVENT\r\nUID:test-uid-12345\r\nDTSTART;TZID=Asia/Singapore:20260907T090000\r\nDTEND;TZID=Asia/Singapore:20260907T110000\r\nSUMMARY:Test Event\r\nLOCATION:Room 101\r\nDESCRIPTION:Test Description\r\nX-SOURCE:brightspace-calendar\r\nEND:VEVENT\r\nEND:VCALENDAR\r\n"

	loc, err := time.LoadLocation("Asia/Singapore")
	if err != nil {
		t.Fatalf("Failed to load timezone: %v", err)
	}

	events := parseICS([]byte(icsContent), loc)

	if len(events) != 1 {
		t.Fatalf("Expected 1 event, got %d", len(events))
	}

	e := events[0]
	if e.Summary != "Test Event" {
		t.Errorf("Summary = %q, want %q", e.Summary, "Test Event")
	}
	if e.Location != "Room 101" {
		t.Errorf("Location = %q, want %q", e.Location, "Room 101")
	}
	if e.Description != "Test Description" {
		t.Errorf("Description = %q, want %q", e.Description, "Test Description")
	}
	if e.Source != "brightspace-calendar" {
		t.Errorf("Source = %q, want %q", e.Source, "brightspace-calendar")
	}
	if e.OrgUnitID != "" {
		t.Errorf("OrgUnitID = %q, want empty", e.OrgUnitID)
	}
	if e.OrgUnitName != "" {
		t.Errorf("OrgUnitName = %q, want empty", e.OrgUnitName)
	}
	if e.OrgUnitCode != "" {
		t.Errorf("OrgUnitCode = %q, want empty", e.OrgUnitCode)
	}
	if e.Title != "" {
		t.Errorf("Title = %q, want empty", e.Title)
	}
}

func TestParseICS_XPropertySpecialCharacters(t *testing.T) {
	icsContent := "BEGIN:VCALENDAR\r\nVERSION:2.0\r\nPRODID:-//SIT Timetable//EN\r\nCALSCALE:GREGORIAN\r\nMETHOD:PUBLISH\r\nREFRESH-INTERVAL;VALUE=DURATION:PT1H\r\nBEGIN:VEVENT\r\nUID:test-uid-special\r\nDTSTART;TZID=Asia/Singapore:20260907T090000\r\nDTEND;TZID=Asia/Singapore:20260907T110000\r\nSUMMARY:Test Event\r\nLOCATION:Room 101\r\nDESCRIPTION:Test Description\r\nX-SOURCE:brightspace-calendar\r\nX-OrgUnitID:12345\\;6789\r\nX-OrgUnitName:ALT2501\\, Software Engineering\r\nX-OrgUnitCode:ALT2501\\;Sec01\r\nX-Title:Assignment with backslash\\\\ and newline\\nhere\r\nEND:VEVENT\r\nEND:VCALENDAR\r\n"

	loc, err := time.LoadLocation("Asia/Singapore")
	if err != nil {
		t.Fatalf("Failed to load timezone: %v", err)
	}

	events := parseICS([]byte(icsContent), loc)

	if len(events) != 1 {
		t.Fatalf("Expected 1 event, got %d", len(events))
	}

	e := events[0]

	if e.OrgUnitID != "12345;6789" {
		t.Errorf("OrgUnitID = %q, want %q", e.OrgUnitID, "12345;6789")
	}
	if e.OrgUnitName != "ALT2501, Software Engineering" {
		t.Errorf("OrgUnitName = %q, want %q", e.OrgUnitName, "ALT2501, Software Engineering")
	}
	if e.OrgUnitCode != "ALT2501;Sec01" {
		t.Errorf("OrgUnitCode = %q, want %q", e.OrgUnitCode, "ALT2501;Sec01")
	}
	if e.Title != "Assignment with backslash\\ and newline\nhere" {
		t.Errorf("Title = %q, want %q", e.Title, "Assignment with backslash\\ and newline\nhere")
	}
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
