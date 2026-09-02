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

	data := cache.Get()
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

	data := cache.Get()
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

	data := cache.Get()
	if len(data) != 0 {
		t.Error("LoadFromFile() should return empty data for non-existent file")
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
