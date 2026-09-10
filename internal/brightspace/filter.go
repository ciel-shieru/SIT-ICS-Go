package brightspace

import (
	"strings"
)

// Blocklist provides configurable filtering for BrightSpace courses and events.
type Blocklist struct {
	// CourseNamePatterns are case-insensitive substring patterns to match against course names.
	// A course is blocked if any pattern is a substring of its name.
	CourseNamePatterns []string

	// CourseIDs are exact OrgUnitId values to block.
	CourseIDs []string

	// EventTitlePatterns are case-insensitive substring patterns to match against event titles.
	// An event is blocked if any pattern is a substring of its title.
	EventTitlePatterns []string

	// EventLocationPatterns are case-insensitive substring patterns to match against event locations.
	// An event is blocked if any pattern is a substring of its location.
	EventLocationPatterns []string
}

// IsCourseBlocked checks whether a course should be filtered out.
func (b *Blocklist) IsCourseBlocked(orgUnitID string, name string) bool {
	for _, id := range b.CourseIDs {
		if strings.EqualFold(strings.TrimSpace(id), strings.TrimSpace(orgUnitID)) {
			return true
		}
	}
	for _, pattern := range b.CourseNamePatterns {
		if strings.Contains(strings.ToLower(name), strings.ToLower(strings.TrimSpace(pattern))) {
			return true
		}
	}
	return false
}

// IsEventBlocked checks whether an event should be filtered out.
func (b *Blocklist) IsEventBlocked(title string) bool {
	for _, pattern := range b.EventTitlePatterns {
		if strings.Contains(strings.ToLower(title), strings.ToLower(strings.TrimSpace(pattern))) {
			return true
		}
	}
	return false
}

// IsLocationBlocked checks whether an event should be filtered out based on its location.
func (b *Blocklist) IsLocationBlocked(location string) bool {
	for _, pattern := range b.EventLocationPatterns {
		if strings.Contains(strings.ToLower(location), strings.ToLower(strings.TrimSpace(pattern))) {
			return true
		}
	}
	return false
}

// Matches checks whether an event should be blocked based on all blocklist criteria.
func (b *Blocklist) Matches(orgUnitID, orgUnitName, title, location string) bool {
	if b.IsCourseBlocked(orgUnitID, orgUnitName) {
		return true
	}
	if b.IsEventBlocked(title) {
		return true
	}
	if b.IsLocationBlocked(location) {
		return true
	}
	return false
}

// ParseCommaSeparated splits a comma-separated string into trimmed, non-empty patterns.
func ParseCommaSeparated(s string) []string {
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}
