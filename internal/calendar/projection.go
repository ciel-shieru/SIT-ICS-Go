package calendar

import (
	"strings"
)

func filterEvents(events []Event, filterFn func(Event) bool) []Event {
	filtered := make([]Event, 0, len(events))
	for _, event := range events {
		if filterFn(event) {
			filtered = append(filtered, event)
		}
	}
	return filtered
}

func filterEventsBySource(events []Event, source string) []Event {
	filtered := make([]Event, 0, len(events))
	for _, event := range events {
		if event.Source == source {
			filtered = append(filtered, event)
		}
	}
	return filtered
}

func IsOnline(event Event) bool {
	return strings.EqualFold(event.Location, "Online")
}

func IsNotOnline(event Event) bool {
	return !IsOnline(event)
}

// IsCampus returns true for events that belong to the campus projection:
// non-online events that are not sourced from BrightSpace.
func IsCampus(event Event) bool {
	return IsNotOnline(event) && !isBrightSpaceEvent(event)
}

func isBrightSpaceEvent(event Event) bool {
	return strings.HasPrefix(event.Source, "brightspace-")
}
