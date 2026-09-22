package brightspace

import (
	"fmt"
	"log"
	"regexp"
	"strings"
)

// Blocklist provides configurable filtering for BrightSpace courses and events.
type Blocklist struct {
	// CourseNamePatterns are case-insensitive substring patterns to match against course names.
	// A course is blocked if any pattern is a substring of its name.
	CourseNamePatterns []string

	// CourseIDs are exact OrgUnitId values to block.
	CourseIDs []string

	// CourseCodePatterns are case-insensitive patterns to match against course module codes.
	CourseCodePatterns []string

	// EventTitlePatterns are case-insensitive substring patterns to match against event titles.
	// An event is blocked if any pattern is a substring of its title.
	EventTitlePatterns []string

	// EventLocationPatterns are case-insensitive substring patterns to match against event locations.
	// An event is blocked if any pattern is a substring of its location.
	EventLocationPatterns []string

	// QuizTitlePatterns are case-insensitive substring patterns to match against quiz titles.
	// A quiz is blocked if any pattern is a substring of its title.
	QuizTitlePatterns []string

	CourseNameRegexes    []*regexp.Regexp
	CourseCodeRegexes    []*regexp.Regexp
	EventTitleRegexes    []*regexp.Regexp
	EventLocationRegexes []*regexp.Regexp
	QuizTitleRegexes     []*regexp.Regexp

	// Each regex corresponds to the pattern at the same index in the
	// matching Patterns field. A nil entry means that pattern has no
	// wildcard and uses substring matching instead.
}

// IsCourseBlocked checks whether a course should be filtered out.
func (b *Blocklist) IsCourseBlocked(orgUnitID string, name string, code string) bool {
	for _, id := range b.CourseIDs {
		if strings.EqualFold(strings.TrimSpace(id), strings.TrimSpace(orgUnitID)) {
			return true
		}
	}
	for i, pattern := range b.CourseNamePatterns {
		trimmed := strings.ToLower(strings.TrimSpace(pattern))
		if strings.Contains(strings.ToLower(name), trimmed) {
			return true
		}
		if i < len(b.CourseNameRegexes) {
			if re := b.CourseNameRegexes[i]; re != nil && re.MatchString(name) {
				return true
			}
		}
	}
	if b.IsCourseCodeBlocked(code) {
		return true
	}
	return false
}

// IsCourseCodeBlocked checks whether a course code should be filtered out.
func (b *Blocklist) IsCourseCodeBlocked(code string) bool {
	for i, pattern := range b.CourseCodePatterns {
		trimmed := strings.ToLower(strings.TrimSpace(pattern))
		if strings.Contains(trimmed, "*") {
			if i < len(b.CourseCodeRegexes) {
				if re := b.CourseCodeRegexes[i]; re != nil && re.MatchString(code) {
					return true
				}
			}
		} else {
			if strings.Contains(strings.ToLower(code), trimmed) {
				return true
			}
		}
	}
	return false
}

// IsEventBlocked checks whether an event should be filtered out.
func (b *Blocklist) IsEventBlocked(title string) bool {
	for i, pattern := range b.EventTitlePatterns {
		trimmed := strings.ToLower(strings.TrimSpace(pattern))
		if strings.Contains(strings.ToLower(title), trimmed) {
			return true
		}
		if i < len(b.EventTitleRegexes) {
			if re := b.EventTitleRegexes[i]; re != nil && re.MatchString(title) {
				return true
			}
		}
	}
	return false
}

// IsLocationBlocked checks whether an event should be filtered out based on its location.
func (b *Blocklist) IsLocationBlocked(location string) bool {
	for i, pattern := range b.EventLocationPatterns {
		trimmed := strings.ToLower(strings.TrimSpace(pattern))
		if strings.Contains(strings.ToLower(location), trimmed) {
			return true
		}
		if i < len(b.EventLocationRegexes) {
			if re := b.EventLocationRegexes[i]; re != nil && re.MatchString(location) {
				return true
			}
		}
	}
	return false
}

// IsQuizBlocked checks whether a quiz should be filtered out.
func (b *Blocklist) IsQuizBlocked(title string) bool {
	for i, pattern := range b.QuizTitlePatterns {
		trimmed := strings.ToLower(strings.TrimSpace(pattern))
		if strings.Contains(strings.ToLower(title), trimmed) {
			return true
		}
		if i < len(b.QuizTitleRegexes) {
			if re := b.QuizTitleRegexes[i]; re != nil && re.MatchString(title) {
				return true
			}
		}
	}
	return false
}

// Matches checks whether an event should be blocked based on all blocklist criteria.
func (b *Blocklist) Matches(orgUnitID, orgUnitName, orgUnitCode, title, location string) bool {
	if b.IsCourseBlocked(orgUnitID, orgUnitName, orgUnitCode) {
		return true
	}
	if b.IsEventBlocked(title) {
		return true
	}
	if b.IsLocationBlocked(location) {
		return true
	}
	if b.IsQuizBlocked(title) {
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

// PatternToRegex converts a wildcard pattern to a *regexp.Regexp.
// A literal asterisk '*' matches zero or more of any character.
// Consecutive asterisks are collapsed into a single wildcard.
// Spaces immediately adjacent to '*' (on either side) are omitted from the
// resulting regex, allowing natural-language patterns like "MOD0001 * quiz"
// to match "MOD0001week2quiz", "MOD0001 quiz", or "MOD0001 week 2 quiz".
// All other regex special characters (. + ? ^ $ ( ) [ ] { } | \) are escaped.
// The resulting regex is anchored with ^...$ and is case-insensitive.
// Returns nil if the pattern contains no asterisk.
func PatternToRegex(pattern string) (*regexp.Regexp, error) {
	if !strings.Contains(pattern, "*") {
		return nil, nil
	}

	var b strings.Builder
	runes := []rune(pattern)
	for i, ch := range runes {
		switch ch {
		case ' ':
			prevIsStar := i > 0 && runes[i-1] == '*'
			nextIsStar := i+1 < len(runes) && runes[i+1] == '*'
			if prevIsStar || nextIsStar {
				continue
			}
			b.WriteRune(ch)
		case '*':
			if i > 0 && b.Len() >= 2 && b.String()[b.Len()-2:] == ".*" {
				continue
			}
			b.WriteString(".*")
		case '.', '+', '?', '^', '$', '(', ')', '[', ']', '{', '}', '|', '\\':
			b.WriteString(`\`)
			b.WriteRune(ch)
		default:
			b.WriteRune(ch)
		}
	}

	re, err := regexp.Compile("(?i)^" + b.String() + "$")
	if err != nil {
		return nil, fmt.Errorf("invalid wildcard pattern %q: %w", pattern, err)
	}
	return re, nil
}

// CompilePatterns separates wildcard patterns from non-wildcard patterns,
// compiles all wildcard patterns into regexes, and stores them in the
// corresponding regex slice fields. This should be called once after
// the Blocklist is initialized.
func (b *Blocklist) CompilePatterns() {
	b.CourseNameRegexes = make([]*regexp.Regexp, len(b.CourseNamePatterns))
	for i, p := range b.CourseNamePatterns {
		re, err := PatternToRegex(p)
		if err != nil {
			log.Printf("brightspace: failed to compile course name pattern %q: %v", p, err)
		}
		b.CourseNameRegexes[i] = re
	}

	b.CourseCodeRegexes = make([]*regexp.Regexp, len(b.CourseCodePatterns))
	for i, p := range b.CourseCodePatterns {
		re, err := PatternToRegex(p)
		if err != nil {
			log.Printf("brightspace: failed to compile course code pattern %q: %v", p, err)
		}
		b.CourseCodeRegexes[i] = re
	}

	b.EventTitleRegexes = make([]*regexp.Regexp, len(b.EventTitlePatterns))
	for i, p := range b.EventTitlePatterns {
		re, err := PatternToRegex(p)
		if err != nil {
			log.Printf("brightspace: failed to compile event title pattern %q: %v", p, err)
		}
		b.EventTitleRegexes[i] = re
	}

	b.EventLocationRegexes = make([]*regexp.Regexp, len(b.EventLocationPatterns))
	for i, p := range b.EventLocationPatterns {
		re, err := PatternToRegex(p)
		if err != nil {
			log.Printf("brightspace: failed to compile event location pattern %q: %v", p, err)
		}
		b.EventLocationRegexes[i] = re
	}

	b.QuizTitleRegexes = make([]*regexp.Regexp, len(b.QuizTitlePatterns))
	for i, p := range b.QuizTitlePatterns {
		re, err := PatternToRegex(p)
		if err != nil {
			log.Printf("brightspace: failed to compile quiz title pattern %q: %v", p, err)
		}
		b.QuizTitleRegexes[i] = re
	}
}
