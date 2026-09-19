package calendar

import (
	"strings"
	"testing"
	"time"
)

func TestAlertsFromConfig_EmptyStrings(t *testing.T) {
	mainAlerts, onlineAlerts, campusAlerts, bsEventsAlerts, bsDropboxAlerts, bsQuizzesAlerts := AlertsFromConfig("", "", "", "", "", "")

	if mainAlerts != nil {
		t.Error("mainAlerts should be nil for empty string")
	}
	if onlineAlerts != nil {
		t.Error("onlineAlerts should be nil for empty string")
	}
	if campusAlerts != nil {
		t.Error("campusAlerts should be nil for empty string")
	}
	if bsEventsAlerts != nil {
		t.Error("bsEventsAlerts should be nil for empty string")
	}
	if bsDropboxAlerts != nil {
		t.Error("bsDropboxAlerts should be nil for empty string")
	}
	if bsQuizzesAlerts != nil {
		t.Error("bsQuizzesAlerts should be nil for empty string")
	}
}

func TestAlertsFromConfig_WithDurations(t *testing.T) {
	mainAlerts, onlineAlerts, campusAlerts, bsEventsAlerts, bsDropboxAlerts, bsQuizzesAlerts := AlertsFromConfig(
		"-P2D,-PT1H",
		"-P1D",
		"-PT30M",
		"-P2D,-P1D,-PT1H",
		"-PT1H",
		"-P1D",
	)

	if len(mainAlerts) != 2 {
		t.Errorf("Expected 2 main alerts, got %d", len(mainAlerts))
	}
	if len(onlineAlerts) != 1 {
		t.Errorf("Expected 1 online alert, got %d", len(onlineAlerts))
	}
	if len(campusAlerts) != 1 {
		t.Errorf("Expected 1 campus alert, got %d", len(campusAlerts))
	}
	if len(bsEventsAlerts) != 3 {
		t.Errorf("Expected 3 bsEvents alerts, got %d", len(bsEventsAlerts))
	}
	if len(bsDropboxAlerts) != 1 {
		t.Errorf("Expected 1 bsDropbox alert, got %d", len(bsDropboxAlerts))
	}
	if len(bsQuizzesAlerts) != 1 {
		t.Errorf("Expected 1 bsQuizzes alert, got %d", len(bsQuizzesAlerts))
	}

	for _, alerts := range [][]Alert{mainAlerts, onlineAlerts, campusAlerts, bsEventsAlerts, bsDropboxAlerts, bsQuizzesAlerts} {
		for _, alert := range alerts {
			if alert.Action != AlertDisplay {
				t.Errorf("Expected AlertDisplay action, got %d", alert.Action)
			}
			if alert.Description != "Reminder" {
				t.Errorf("Expected 'Reminder' description, got %q", alert.Description)
			}
		}
	}
}

func TestAlertsFromConfig_Whitespace(t *testing.T) {
	mainAlerts, _, _, _, _, _ := AlertsFromConfig("  -P1D , -PT1H  ", "", "", "", "", "")

	if len(mainAlerts) != 2 {
		t.Errorf("Expected 2 alerts with whitespace, got %d", len(mainAlerts))
	}
}

func TestAlertsFromConfig_MixedEmpty(t *testing.T) {
	_, onlineAlerts, _, _, _, _ := AlertsFromConfig("", "  ,  , -P1D , ", "", "", "", "")

	if len(onlineAlerts) != 1 {
		t.Errorf("Expected 1 alert with mixed empty entries, got %d", len(onlineAlerts))
	}
}

func TestParseAlerts_Empty(t *testing.T) {
	alerts := parseAlerts("")
	if alerts != nil {
		t.Errorf("parseAlerts(\"\") = %v, want nil", alerts)
	}

	alerts = parseAlerts("   ")
	if alerts != nil {
		t.Errorf("parseAlerts(\"   \") = %v, want nil", alerts)
	}
}

func TestParseAlerts_InvalidDuration(t *testing.T) {
	alerts := parseAlerts("PT1H")
	if alerts != nil {
		t.Error("parseAlerts with positive duration should return nil (missing minus sign)")
	}
}

func TestParseAlerts_AllValidFormats(t *testing.T) {
	durations, err := parseAlertDurations("-P2D,-P1D,-PT1H,-PT30M,-PT1H30M")
	if err != nil {
		t.Fatalf("parseAlertDurations() error = %v", err)
	}

	expected := []time.Duration{
		2 * 24 * time.Hour,
		24 * time.Hour,
		time.Hour,
		30 * time.Minute,
		time.Hour + 30*time.Minute,
	}

	if len(durations) != len(expected) {
		t.Fatalf("Expected %d durations, got %d", len(expected), len(durations))
	}

	for i, d := range durations {
		if d != expected[i] {
			t.Errorf("Duration[%d] = %v, want %v", i, d, expected[i])
		}
	}
}

func TestAlertsFromConfig_NegativeDurationsPreserved(t *testing.T) {
	mainAlerts, _, _, _, _, _ := AlertsFromConfig("-P1D,-PT2H", "", "", "", "", "")

	if len(mainAlerts) != 2 {
		t.Fatalf("Expected 2 alerts, got %d", len(mainAlerts))
	}

	for _, alert := range mainAlerts {
		if alert.Duration > 0 {
			t.Errorf("Alert duration should be negative (before event), got %v", alert.Duration)
		}
	}
}

func TestParseAlertDurations_ErrorOnPositiveDuration(t *testing.T) {
	_, err := parseAlertDurations("P1D")
	if err == nil {
		t.Error("Expected error for positive duration (missing minus sign)")
	}
	if err != nil && !strings.Contains(err.Error(), "negative") {
		t.Errorf("Error should mention 'negative', got: %v", err)
	}
}

func TestParseAlertDurations_ErrorOnInvalidFormat(t *testing.T) {
	_, err := parseAlertDurations("-X1D")
	if err == nil {
		t.Error("Expected error for invalid ICS duration format")
	}
	if err != nil && !strings.Contains(err.Error(), "P") {
		t.Errorf("Error should mention 'P' prefix requirement, got: %v", err)
	}
}

func TestParseAlertDurations_SingleDuration(t *testing.T) {
	durations, err := parseAlertDurations("-P1D")
	if err != nil {
		t.Fatalf("parseAlertDurations() error = %v", err)
	}
	if len(durations) != 1 {
		t.Fatalf("Expected 1 duration, got %d", len(durations))
	}
	if durations[0] != 24*time.Hour {
		t.Errorf("Expected 24h, got %v", durations[0])
	}
}

func TestParseAlertDurations_TrailingComma(t *testing.T) {
	durations, err := parseAlertDurations("-P1D,-PT1H,")
	if err != nil {
		t.Fatalf("parseAlertDurations() error = %v", err)
	}
	if len(durations) != 2 {
		t.Errorf("Expected 2 durations (trailing comma ignored), got %d", len(durations))
	}
}

func TestParseAlertDurations_LeadingComma(t *testing.T) {
	durations, err := parseAlertDurations(",,-P1D,,")
	if err != nil {
		t.Fatalf("parseAlertDurations() error = %v", err)
	}
	if len(durations) != 1 {
		t.Errorf("Expected 1 duration (leading/trailing commas ignored), got %d", len(durations))
	}
}
