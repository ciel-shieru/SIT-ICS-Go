package brightspace

import (
	"testing"
)

func TestPatternToRegex(t *testing.T) {
	tests := []struct {
		name      string
		pattern   string
		wantNil   bool
		input     string
		wantMatch bool
	}{
		// No wildcard → nil
		{
			name:    "no wildcard returns nil",
			pattern: "practice quiz",
			wantNil: true,
		},
		// Single wildcard
		{
			name:      "single wildcard matches middle text",
			pattern:   "MOD0001*quiz",
			wantNil:   false,
			input:     "MOD0001 week 2 quiz",
			wantMatch: true,
		},
		{
			name:      "single wildcard matches zero chars",
			pattern:   "MOD0001*quiz",
			wantNil:   false,
			input:     "MOD0001 quiz",
			wantMatch: true,
		},
		{
			name:      "single wildcard matches multiple words",
			pattern:   "MOD0001*practice quiz",
			wantNil:   false,
			input:     "MOD0001 week 2 practice quiz",
			wantMatch: true,
		},
		{
			name:      "anchored start rejects prefix",
			pattern:   "MOD0001*quiz",
			wantNil:   false,
			input:     "Other MOD0001 quiz",
			wantMatch: false,
		},
		{
			name:      "anchored end rejects suffix",
			pattern:   "MOD0001*quiz",
			wantNil:   false,
			input:     "MOD0001 week 2 other",
			wantMatch: false,
		},
		// Multiple wildcards
		{
			name:      "multiple wildcards match",
			pattern:   "MOD0001*week*quiz",
			wantNil:   false,
			input:     "MOD0001 week 2 weekly quiz",
			wantMatch: true,
		},
		{
			name:      "multiple wildcards with zero chars",
			pattern:   "MOD0001*week*quiz",
			wantNil:   false,
			input:     "MOD0001 week quiz",
			wantMatch: true,
		},
		// Regex special characters escaped
		{
			name:      "parentheses are literal",
			pattern:   "MOD0001*(practice)*",
			wantNil:   false,
			input:     "MOD0001 (practice)",
			wantMatch: true,
		},
		{
			name:      "parentheses do not act as group",
			pattern:   "MOD0001*(practice)*",
			wantNil:   false,
			input:     "MOD0001 practice",
			wantMatch: false,
		},
		{
			name:      "dot is literal",
			pattern:   "test.*value",
			wantNil:   false,
			input:     "test.value",
			wantMatch: true,
		},
		{
			name:      "dot does not match any char",
			pattern:   "test.*value",
			wantNil:   false,
			input:     "testXvalue",
			wantMatch: false,
		},
		// Case insensitive
		{
			name:      "case insensitive matching",
			pattern:   "MOD0001*QUIZ",
			wantNil:   false,
			input:     "mod1002 Week 2 quiz",
			wantMatch: true,
		},
		// Leading/trailing wildcard
		{
			name:      "leading wildcard matches suffix",
			pattern:   "*practice quiz",
			wantNil:   false,
			input:     "MOD0001 week 2 practice quiz",
			wantMatch: true,
		},
		{
			name:      "trailing wildcard matches prefix",
			pattern:   "MOD0001*",
			wantNil:   false,
			input:     "MOD0001 week 2 practice quiz",
			wantMatch: true,
		},
		// Consecutive wildcards collapsed
		{
			name:      "consecutive wildcards collapsed",
			pattern:   "MOD0001**quiz",
			wantNil:   false,
			input:     "MOD0001Xquiz",
			wantMatch: true,
		},
		{
			name:      "consecutive wildcards match many chars",
			pattern:   "MOD0001**quiz",
			wantNil:   false,
			input:     "MOD0001 week 2 practice quiz",
			wantMatch: true,
		},
		// Pattern that is only *
		{
			name:      "single asterisk matches everything",
			pattern:   "*",
			wantNil:   false,
			input:     "anything at all",
			wantMatch: true,
		},
		// Space skipping edge cases
		{
			name:      "space before and after star both skipped",
			pattern:   "hello * world",
			wantNil:   false,
			input:     "helloXworld",
			wantMatch: true,
		},
		{
			name:    "space between non-star words preserved",
			pattern: "hello world",
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			re, err := PatternToRegex(tt.pattern)
			if tt.wantNil {
				if re != nil {
					t.Errorf("PatternToRegex(%q) = %v, want nil", tt.pattern, re)
				}
				return
			}
			if err != nil {
				t.Fatalf("PatternToRegex(%q) error: %v", tt.pattern, err)
			}
			if re == nil {
				t.Fatalf("PatternToRegex(%q) = nil, want non-nil", tt.pattern)
			}
			got := re.MatchString(tt.input)
			if got != tt.wantMatch {
				t.Errorf("regex.MatchString(%q) = %v, want %v", tt.input, got, tt.wantMatch)
			}
		})
	}
}

func TestBlocklist_CompilePatterns_NoPanicOnEmpty(t *testing.T) {
	blocklist := &Blocklist{
		CourseNamePatterns:    []string{},
		CourseCodePatterns:    []string{},
		EventTitlePatterns:    []string{},
		EventLocationPatterns: []string{},
		QuizTitlePatterns:     []string{},
	}
	blocklist.CompilePatterns()

	if blocklist.CourseNameRegexes == nil {
		t.Error("CourseNameRegexes should not be nil")
	}
	if len(blocklist.CourseNameRegexes) != 0 {
		t.Errorf("CourseNameRegexes length = %d, want 0", len(blocklist.CourseNameRegexes))
	}
}

func TestBlocklist_CompilePatterns_NonWildcardStaysNil(t *testing.T) {
	blocklist := &Blocklist{
		EventTitlePatterns: []string{"no wildcard", "has * wildcard"},
	}
	blocklist.CompilePatterns()

	if blocklist.EventTitleRegexes[0] != nil {
		t.Error("EventTitleRegexes[0] should be nil (no wildcard)")
	}
	if blocklist.EventTitleRegexes[1] == nil {
		t.Error("EventTitleRegexes[1] should be non-nil (has wildcard)")
	}
}

func TestBlocklist_IsEventBlocked_Wildcard(t *testing.T) {
	blocklist := &Blocklist{
		EventTitlePatterns: []string{"MOD0001*practice quiz", "practice quiz"},
	}
	blocklist.CompilePatterns()

	tests := []struct {
		title   string
		blocked bool
	}{
		// Wildcard pattern matches
		{"MOD0001 week 2 practice quiz", true},
		{"MOD0001 practice quiz", true}, // * matches zero chars
		// Non-wildcard substring pattern matches
		{"Some practice quiz extra", true},
		// Neither matches
		{"MOD0001 week 2 midterm", false},
		{"Random event", false},
	}

	for _, tt := range tests {
		result := blocklist.IsEventBlocked(tt.title)
		if result != tt.blocked {
			t.Errorf("IsEventBlocked(%q) = %v, want %v", tt.title, result, tt.blocked)
		}
	}
}

func TestBlocklist_IsCourseBlocked_Wildcard(t *testing.T) {
	blocklist := &Blocklist{
		CourseNamePatterns: []string{"* Intro*", "* Advanced*"},
	}
	blocklist.CompilePatterns()

	tests := []struct {
		name    string
		blocked bool
	}{
		{"Introduction to Programming", true},
		{"CS Intro to Computing", true},
		{"Advanced Data Structures", true},
		{"Programming Fundamentals", false},
	}

	for _, tt := range tests {
		result := blocklist.IsCourseBlocked("12345", tt.name, "")
		if result != tt.blocked {
			t.Errorf("IsCourseBlocked(%q) = %v, want %v", tt.name, result, tt.blocked)
		}
	}
}

func TestBlocklist_IsCourseCodeBlocked_Substring(t *testing.T) {
	blocklist := &Blocklist{
		CourseCodePatterns: []string{"ALT2501", "MOD0001"},
	}
	blocklist.CompilePatterns()

	tests := []struct {
		code    string
		blocked bool
	}{
		{"ALT2501", true},
		{"MOD0001", true},
		{"alt2501", true},
		{"mod0001", true},
		{"ALT2501A", true},
		{"XALT2501Y", true},
		{"MOD0002", false},
		{"", false},
	}

	for _, tt := range tests {
		result := blocklist.IsCourseCodeBlocked(tt.code)
		if result != tt.blocked {
			t.Errorf("IsCourseCodeBlocked(%q) = %v, want %v", tt.code, result, tt.blocked)
		}
	}
}

func TestBlocklist_IsCourseCodeBlocked_Wildcard(t *testing.T) {
	blocklist := &Blocklist{
		CourseCodePatterns: []string{"MOD*001", "*2501"},
	}
	blocklist.CompilePatterns()

	tests := []struct {
		code    string
		blocked bool
	}{
		{"MOD0001", true},
		{"MOD2501", true},
		{"MOD1234001", true},
		{"MOD0002", false},
		{"XMOD0001", false},
		{"ALT2501", true},
		{"X2501Y", false},
	}

	for _, tt := range tests {
		result := blocklist.IsCourseCodeBlocked(tt.code)
		if result != tt.blocked {
			t.Errorf("IsCourseCodeBlocked(%q) = %v, want %v", tt.code, result, tt.blocked)
		}
	}
}

func TestBlocklist_IsCourseCodeBlocked_EmptyCode(t *testing.T) {
	blocklist := &Blocklist{
		CourseCodePatterns: []string{"MOD0001"},
	}
	blocklist.CompilePatterns()

	if blocklist.IsCourseCodeBlocked("") {
		t.Error("IsCourseCodeBlocked(\"\") should return false")
	}
}

func TestBlocklist_IsCourseCodeBlocked_NoMatch(t *testing.T) {
	blocklist := &Blocklist{
		CourseCodePatterns: []string{"ALT2501", "MOD0001"},
	}
	blocklist.CompilePatterns()

	tests := []string{"MOD0002", "ALT2502", "CS101", "PHY201"}
	for _, code := range tests {
		if blocklist.IsCourseCodeBlocked(code) {
			t.Errorf("IsCourseCodeBlocked(%q) should return false", code)
		}
	}
}

func TestBlocklist_IsCourseCodeBlocked_MixedPatterns(t *testing.T) {
	blocklist := &Blocklist{
		CourseCodePatterns: []string{"MOD*001", "ALT2501", "*2501"},
	}
	blocklist.CompilePatterns()

	tests := []struct {
		code    string
		blocked bool
	}{
		{"MOD0001", true},   // wildcard
		{"ALT2501", true},   // substring
		{"MOD2501", true},   // wildcard
		{"CS101", false},    // no match
	}

	for _, tt := range tests {
		result := blocklist.IsCourseCodeBlocked(tt.code)
		if result != tt.blocked {
			t.Errorf("IsCourseCodeBlocked(%q) = %v, want %v", tt.code, result, tt.blocked)
		}
	}
}

func TestBlocklist_IsLocationBlocked_Wildcard(t *testing.T) {
	blocklist := &Blocklist{
		EventLocationPatterns: []string{"* Online*", "* Zoom*"},
	}
	blocklist.CompilePatterns()

	tests := []struct {
		location string
		blocked  bool
	}{
		{"Zoom Online Meeting", true},
		{"Online - Virtual", true},
		{"Campus A Room 101", false},
	}

	for _, tt := range tests {
		result := blocklist.IsLocationBlocked(tt.location)
		if result != tt.blocked {
			t.Errorf("IsLocationBlocked(%q) = %v, want %v", tt.location, result, tt.blocked)
		}
	}
}

func TestBlocklist_IsQuizBlocked_Wildcard(t *testing.T) {
	blocklist := &Blocklist{
		QuizTitlePatterns: []string{"* practice quiz*", "* Mock Exam*"},
	}
	blocklist.CompilePatterns()

	tests := []struct {
		title   string
		blocked bool
	}{
		{"week 2 practice quiz", true},
		{"Practice Quiz", true},
		{"Mock Exam 1", true},
		{"Final Quiz", false},
	}

	for _, tt := range tests {
		result := blocklist.IsQuizBlocked(tt.title)
		if result != tt.blocked {
			t.Errorf("IsQuizBlocked(%q) = %v, want %v", tt.title, result, tt.blocked)
		}
	}
}

func TestBlocklist_MixWildcardAndSubstring(t *testing.T) {
	blocklist := &Blocklist{
		EventTitlePatterns: []string{"* quiz", "midterm", "* final"},
	}
	blocklist.CompilePatterns()

	tests := []struct {
		title   string
		blocked bool
	}{
		{"week 2 quiz", true},    // wildcard - ends with quiz
		{"midterm review", true}, // substring - contains "midterm"
		{"final exam", false},    // wildcard * final = ends with final
		{"quiz 1", false},        // wildcard * quiz = ends with quiz
		{"pop quiz", true},       // wildcard - ends with quiz
		{"homework", false},      // no match
	}

	for _, tt := range tests {
		result := blocklist.IsEventBlocked(tt.title)
		if result != tt.blocked {
			t.Errorf("IsEventBlocked(%q) = %v, want %v", tt.title, result, tt.blocked)
		}
	}
}

func TestBlocklist_CompilePatterns_IgnoresNonWildcardErrors(t *testing.T) {
	blocklist := &Blocklist{
		EventTitlePatterns: []string{"no wildcard here"},
	}
	blocklist.CompilePatterns()

	if blocklist.EventTitleRegexes[0] != nil {
		t.Error("EventTitleRegexes[0] should be nil for non-wildcard pattern")
	}

	if !blocklist.IsEventBlocked("this has no wildcard here in it") {
		t.Error("IsEventBlocked should match via substring path")
	}
}
