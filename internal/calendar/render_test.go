package calendar

import (
	"strings"
	"testing"
	"time"
)

func TestRender_WithAlerts(t *testing.T) {
	events := []Event{
		{
			Summary:  "Test Class",
			Location: "W1-05-07",
			DTStart:  time.Date(2026, 9, 7, 9, 0, 0, 0, time.UTC),
			DTEnd:    time.Date(2026, 9, 7, 11, 0, 0, 0, time.UTC),
		},
	}

	alerts := []Alert{
		{Duration: -24 * time.Hour, Action: AlertDisplay, Description: "Reminder"},
		{Duration: -1 * time.Hour, Action: AlertDisplay, Description: "Reminder"},
	}

	data, err := Render(events, RenderOptions{
		Timezone:        time.UTC,
		RefreshInterval: time.Hour,
		Alerts:          alerts,
	})
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	content := string(data)

	if !strings.Contains(content, "BEGIN:VALARM") {
		t.Error("Rendered ICS missing BEGIN:VALARM")
	}
	if !strings.Contains(content, "ACTION:DISPLAY") {
		t.Error("Rendered ICS missing ACTION:DISPLAY")
	}
	if !strings.Contains(content, "TRIGGER:-P1D") {
		t.Error("Rendered ICS missing TRIGGER:-P1D for 24h alert")
	}
	if !strings.Contains(content, "TRIGGER:-PT1H") {
		t.Error("Rendered ICS missing TRIGGER:-PT1H for 1h alert")
	}
	if !strings.Contains(content, "DESCRIPTION:Reminder") {
		t.Error("Rendered ICS missing DESCRIPTION:Reminder")
	}

	valarmCount := strings.Count(content, "BEGIN:VALARM")
	if valarmCount != 2 {
		t.Errorf("Expected 2 VALARM blocks, got %d", valarmCount)
	}
}

func TestRender_WithAlerts_MultipleEvents(t *testing.T) {
	events := []Event{
		{
			Summary:  "Event 1",
			Location: "Room 101",
			DTStart:  time.Date(2026, 9, 7, 9, 0, 0, 0, time.UTC),
			DTEnd:    time.Date(2026, 9, 7, 10, 0, 0, 0, time.UTC),
		},
		{
			Summary:  "Event 2",
			Location: "Room 102",
			DTStart:  time.Date(2026, 9, 7, 14, 0, 0, 0, time.UTC),
			DTEnd:    time.Date(2026, 9, 7, 15, 0, 0, 0, time.UTC),
		},
	}

	alerts := []Alert{
		{Duration: -2 * time.Hour, Action: AlertDisplay, Description: "Reminder"},
	}

	data, err := Render(events, RenderOptions{
		Timezone:        time.UTC,
		RefreshInterval: time.Hour,
		Alerts:          alerts,
	})
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	content := string(data)
	valarmCount := strings.Count(content, "BEGIN:VALARM")
	if valarmCount != 2 {
		t.Errorf("Expected 2 VALARM blocks (one per event), got %d", valarmCount)
	}
}

func TestRender_WithoutAlerts(t *testing.T) {
	events := []Event{
		{
			Summary:  "Test Class",
			Location: "W1-05-07",
			DTStart:  time.Date(2026, 9, 7, 9, 0, 0, 0, time.UTC),
			DTEnd:    time.Date(2026, 9, 7, 11, 0, 0, 0, time.UTC),
		},
	}

	data, err := Render(events, RenderOptions{
		Timezone:        time.UTC,
		RefreshInterval: time.Hour,
		Alerts:          nil,
	})
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	content := string(data)
	if strings.Contains(content, "VALARM") {
		t.Error("Rendered ICS should not contain VALARM when no alerts configured")
	}
}

func TestRender_WithEmptyAlerts(t *testing.T) {
	events := []Event{
		{
			Summary:  "Test Class",
			Location: "W1-05-07",
			DTStart:  time.Date(2026, 9, 7, 9, 0, 0, 0, time.UTC),
			DTEnd:    time.Date(2026, 9, 7, 11, 0, 0, 0, time.UTC),
		},
	}

	data, err := Render(events, RenderOptions{
		Timezone:        time.UTC,
		RefreshInterval: time.Hour,
		Alerts:          []Alert{},
	})
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	content := string(data)
	if strings.Contains(content, "VALARM") {
		t.Error("Rendered ICS should not contain VALARM when alerts slice is empty")
	}
}

func TestFormatICSDuration(t *testing.T) {
	tests := []struct {
		input    time.Duration
		expected string
	}{
		{-24 * time.Hour, "-P1D"},
		{-48 * time.Hour, "-P2D"},
		{-1 * time.Hour, "-PT1H"},
		{-2 * time.Hour, "-PT2H"},
		{-30 * time.Minute, "-PT30M"},
		{-1 * time.Hour - 30 * time.Minute, "-PT1H30M"},
		{-1 * time.Hour - 1 * time.Minute, "-PT1H1M"},
		{-2*24*time.Hour - 3*time.Hour - 15*time.Minute, "-P2DT3H15M"},
		{-1 * time.Minute, "-PT1M"},
	}

	for _, tt := range tests {
		result := formatICSDuration(tt.input)
		if result != tt.expected {
			t.Errorf("formatICSDuration(%v) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestRenderVALARM(t *testing.T) {
	alert := Alert{
		Duration:    -1 * time.Hour,
		Action:      AlertDisplay,
		Description: "Class starts in 1 hour",
	}

	var sb strings.Builder
	renderVALARM(&sb, alert)

	output := sb.String()
	expected := "BEGIN:VALARM\r\nACTION:DISPLAY\r\nTRIGGER:-PT1H\r\nDESCRIPTION:Class starts in 1 hour\r\nEND:VALARM\r\n"
	if output != expected {
		t.Errorf("renderVALARM() = %q, want %q", output, expected)
	}
}

func TestRenderVALARM_NoDescription(t *testing.T) {
	alert := Alert{
		Duration: -30 * time.Minute,
		Action:   AlertDisplay,
	}

	var sb strings.Builder
	renderVALARM(&sb, alert)

	output := sb.String()
	if strings.Contains(output, "DESCRIPTION") {
		t.Error("renderVALARM() should not include DESCRIPTION when empty")
	}
	if !strings.Contains(output, "TRIGGER:-PT30M") {
		t.Error("renderVALARM() should include TRIGGER")
	}
}

func TestRenderVALARM_EscapesSpecialChars(t *testing.T) {
	alert := Alert{
		Duration:    -1 * time.Hour,
		Action:      AlertDisplay,
		Description: "Test; with , special chars",
	}

	var sb strings.Builder
	renderVALARM(&sb, alert)

	output := sb.String()
	if strings.Contains(output, "Test;") {
		t.Error("DESCRIPTION should escape semicolons")
	}
	if !strings.Contains(output, "Test\\;") || !strings.Contains(output, "\\,") {
		t.Error("DESCRIPTION should contain escaped special chars")
	}
}

func TestRender_WithAlerts_DurationFormats(t *testing.T) {
	tests := []struct {
		name           string
		duration       time.Duration
		expectedTrigger string
	}{
		{"days", -2 * 24 * time.Hour, "-P2D"},
		{"hours", -3 * time.Hour, "-PT3H"},
		{"minutes", -45 * time.Minute, "-PT45M"},
		{"days_and_hours", -2*24*time.Hour - 5*time.Hour, "-P2DT5H"},
		{"hours_and_minutes", -2*time.Hour - 30*time.Minute, "-PT2H30M"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			events := []Event{
				{
					Summary:  "Test",
					Location: "Room",
					DTStart:  time.Date(2026, 9, 7, 9, 0, 0, 0, time.UTC),
					DTEnd:    time.Date(2026, 9, 7, 10, 0, 0, 0, time.UTC),
				},
			}

			alerts := []Alert{
				{Duration: tt.duration, Action: AlertDisplay, Description: "Test"},
			}

			data, err := Render(events, RenderOptions{
				Timezone:        time.UTC,
				RefreshInterval: time.Hour,
				Alerts:          alerts,
			})
			if err != nil {
				t.Fatalf("Render() error = %v", err)
			}

			content := string(data)
			if !strings.Contains(content, "TRIGGER:"+tt.expectedTrigger) {
				t.Errorf("Expected TRIGGER:%s in output", tt.expectedTrigger)
			}
		})
	}
}
