package brightspace

// ShouldRemoveQuiz determines if a quiz event should be removed from ICS data
// based on the quiz's attempt tracking information.
func ShouldRemoveQuiz(isUnlimited bool, numberOfAttemptsAllowed int, attemptsMade int, bestScore float64) bool {
	if attemptsMade == 0 && bestScore == 0 {
		return false
	}

	if isUnlimited {
		return bestScore >= 100.0
	}

	if attemptsMade >= numberOfAttemptsAllowed {
		return true
	}
	if bestScore >= 100.0 {
		return true
	}
	return false
}
