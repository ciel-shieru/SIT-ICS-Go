package calendar

import (
	"strings"
	"testing"
	"time"
)

func TestIsConcreteVenue(t *testing.T) {
	tests := []struct {
		location string
		want     bool
	}{
		{"Room 101", true},
		{"FAK BLDG Hall A", true},
		// An already-addressed location is still a concrete venue.
		{"Room 101 " + SITCampusAddress, true},
		// Documented D2 edge (ADR-0024): only exactly "online" (trimmed,
		// case-insensitive) is excluded; "Online (Zoom)"-style strings
		// count as concrete venues.
		{"Online (Zoom)", true},
		{"", false},
		{"   ", false},
		{"Online", false},
		{"online", false},
		{" ONLINE ", false},
		{"TBD", false},
		{"tbd", false},
		{"tba", false},
		{"TBA", false},
		{"To Be Advised", false},
		{"to be advised ", false},
	}
	for _, tt := range tests {
		t.Run(tt.location, func(t *testing.T) {
			if got := IsConcreteVenue(tt.location); got != tt.want {
				t.Errorf("IsConcreteVenue(%q) = %v, want %v", tt.location, got, tt.want)
			}
		})
	}
}

func TestWithSITCampusAddress(t *testing.T) {
	tests := []struct {
		source   string
		location string
		want     string
	}{
		{"", "FAK BLDG Hall A", "FAK BLDG Hall A" + SITCampusAddress},
		{"peoplesoft-exams", "FAK HALL 2", "FAK HALL 2" + SITCampusAddress},
		{"brightspace-calendar", "FAK HALL 9", "FAK HALL 9"},
		{"brightspace-quizzes", "Online", "Online"},
		{"", "", ""},
		// Non-concrete locations are returned as-is, NOT trimmed.
		{"", "   ", "   "},
		{"", "Online", "Online"},
		{"", "TBD", "TBD"},
		{"", "To Be Advised", "To Be Advised"},
		{"", " Room 101 ", "Room 101" + SITCampusAddress},
		// Idempotency guard: an already-addressed location is unchanged.
		{"", "Room 101" + SITCampusAddress, "Room 101" + SITCampusAddress},
	}
	for _, tt := range tests {
		t.Run(tt.source+"/"+tt.location, func(t *testing.T) {
			if got := WithSITCampusAddress(tt.source, tt.location); got != tt.want {
				t.Errorf("WithSITCampusAddress(%q, %q) = %q, want %q", tt.source, tt.location, got, tt.want)
			}
		})
	}
}

func TestWithSITCampusAddressRoundTrip(t *testing.T) {
	loc, err := time.LoadLocation("Asia/Singapore")
	if err != nil {
		t.Fatalf("load location: %v", err)
	}

	original := Event{
		Summary:  "TEST2501 - A (Lecture)",
		Location: "Room 101" + SITCampusAddress,
		Source:   "",
		DTStart:  time.Date(2027, 11, 2, 10, 0, 0, 0, loc),
		DTEnd:    time.Date(2027, 11, 2, 12, 0, 0, 0, loc),
		DTStamp:  time.Date(2026, 11, 27, 12, 0, 0, 0, time.UTC),
	}
	original.UID = EventID(original)

	rendered, err := Render([]Event{original}, RenderOptions{Timezone: loc, RefreshInterval: time.Hour})
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	raw := string(rendered)
	if !strings.Contains(raw, "LOCATION:") {
		t.Error("rendered ICS missing LOCATION property")
	}
	// The address must be escaped: leading newline as backslash-n, comma
	// as backslash-comma. The street line is RFC 5545 section 3.1 folded
	// at 75 octets, so assert on fold-safe fragments; full content
	// equality is proven by the round-trip below.
	if !strings.Contains(raw, "\\nSingapore") {
		t.Error("rendered ICS missing escaped leading newline before institution name")
	}
	if !strings.Contains(raw, "Singapore Institute of Technology") {
		t.Error("rendered ICS missing institution name")
	}
	if !strings.Contains(raw, "1 Punggol Coast Rd\\,") {
		t.Error("rendered ICS missing comma-escaped street address")
	}

	parsed := parseICS(rendered, loc)
	if len(parsed) != 1 {
		t.Fatalf("parsed %d events, want 1", len(parsed))
	}
	got := parsed[0]
	if got.Location != original.Location {
		t.Errorf("round-trip Location = %q, want %q", got.Location, original.Location)
	}
	// UIDs must be stable across the disk round-trip: EventID hashes
	// Summary/Location/DTStart/DTEnd, all of which must survive verbatim.
	got.UID = ""
	if id := EventID(got); id != original.UID {
		t.Errorf("round-trip EventID = %s, want %s", id, original.UID)
	}
}
