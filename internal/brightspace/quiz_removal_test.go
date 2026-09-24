package brightspace

import "testing"

func TestShouldRemoveQuiz(t *testing.T) {
	tests := []struct {
		name                    string
		isUnlimited             bool
		numberOfAttemptsAllowed int
		attemptsMade            int
		bestScore               float64
		wantRemove              bool
	}{
		{"unlimited + 100%", true, 0, 5, 100.0, true},
		{"unlimited + 90%", true, 0, 5, 90.0, false},
		{"unlimited + not attempted", true, 0, 0, 0.0, false},
		{"fixed(2) + 2 made + 100%", false, 2, 2, 100.0, true},
		{"fixed(2) + 2 made + 90%", false, 2, 2, 90.0, true},
		{"fixed(2) + 1 made + 90%", false, 2, 1, 90.0, false},
		{"fixed(1) + 1 made + 100%", false, 1, 1, 100.0, true},
		{"fixed(1) + 0 made", false, 1, 0, 0.0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ShouldRemoveQuiz(tt.isUnlimited, tt.numberOfAttemptsAllowed, tt.attemptsMade, tt.bestScore)
			if got != tt.wantRemove {
				t.Errorf("ShouldRemoveQuiz(%v, %d, %d, %.1f) = %v, want %v",
					tt.isUnlimited, tt.numberOfAttemptsAllowed, tt.attemptsMade, tt.bestScore, got, tt.wantRemove)
			}
		})
	}
}
