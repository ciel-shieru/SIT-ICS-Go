package ics

import (
	"os"
	"path/filepath"
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

	if err := cache.SaveToFile(icsPath); err != nil {
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
	if err := cache.LoadFromFile(icsPath); err != nil {
		t.Fatalf("LoadFromFile() error = %v", err)
	}

	data := cache.Get("Asia/Singapore")
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
	if err := cache.SaveToFile(icsPath); err != nil {
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

	data := cache.Get("Asia/Singapore")
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
	if err := cache.LoadFromFile(icsPath); err != nil {
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

	cache.LoadFromFile(icsPath)

	if err := cache.SaveToFile(icsPath); err != nil {
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

	onlineData := cache.GetFiltered("Asia/Singapore", IsOnline)
	onlineContent := string(onlineData)

	if !contains(onlineContent, "SUMMARY:Online Class") {
		t.Error("Filtered online ICS missing online event")
	}
	if contains(onlineContent, "SUMMARY:Campus Class") {
		t.Error("Filtered online ICS should not contain campus event")
	}

	campusData := cache.GetFiltered("Asia/Singapore", IsNotOnline)
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

	if err := cache.SaveAllToFiles(mainPath, onlinePath, campusPath, "Asia/Singapore"); err != nil {
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
