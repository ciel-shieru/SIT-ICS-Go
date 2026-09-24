package calendar

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

func AlertsFromConfig(main, online, campus, bsEvents, bsDropbox, bsQuizzes string) (
	mainAlerts, onlineAlerts, campusAlerts, bsEventsAlerts, bsDropboxAlerts, bsQuizzesAlerts []Alert,
) {
	mainAlerts = parseAlerts(main)
	onlineAlerts = parseAlerts(online)
	campusAlerts = parseAlerts(campus)
	bsEventsAlerts = parseAlerts(bsEvents)
	bsDropboxAlerts = parseAlerts(bsDropbox)
	bsQuizzesAlerts = parseAlerts(bsQuizzes)
	return
}

func parseAlerts(s string) []Alert {
	durations, err := parseAlertDurations(s)
	if err != nil || len(durations) == 0 {
		return nil
	}
	alerts := make([]Alert, 0, len(durations))
	for _, d := range durations {
		alerts = append(alerts, Alert{
			Duration:    -d,
			Action:      AlertDisplay,
			Description: "Reminder",
		})
	}
	return alerts
}

func parseAlertDurations(s string) ([]time.Duration, error) {
	if strings.TrimSpace(s) == "" {
		return nil, nil
	}
	parts := strings.Split(s, ",")
	durations := make([]time.Duration, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		d, err := parseAlertDuration(part)
		if err != nil {
			return nil, fmt.Errorf("parse alert duration %q: %w", part, err)
		}
		durations = append(durations, d)
	}
	return durations, nil
}

func parseAlertDuration(s string) (time.Duration, error) {
	if !strings.HasPrefix(s, "-") {
		return 0, fmt.Errorf("alert duration must be negative (start with -): %s", s)
	}
	ics := s[1:]
	return parseICSDurationForAlert(ics)
}

func parseICSDurationForAlert(s string) (time.Duration, error) {
	if !strings.HasPrefix(s, "P") {
		return 0, fmt.Errorf("invalid ICS duration format (must start with P): %s", s)
	}
	s = s[1:]
	var days int
	remaining := s

	if strings.Contains(remaining, "D") {
		parts := strings.SplitN(remaining, "D", 2)
		d, err := strconv.Atoi(parts[0])
		if err != nil {
			return 0, fmt.Errorf("invalid days in duration: %w", err)
		}
		days = d
		remaining = parts[1]
	}

	if strings.HasPrefix(remaining, "T") {
		remaining = remaining[1:]
		var h, m, sec int

		for len(remaining) > 0 {
			found := false
			for i, c := range remaining {
				if c == 'H' || c == 'M' || c == 'S' {
					val := remaining[:i]
					n, err := strconv.Atoi(val)
					if err != nil {
						return 0, fmt.Errorf("invalid number in duration: %w", err)
					}
					switch c {
					case 'H':
						h = n
					case 'M':
						m = n
					case 'S':
						sec = n
					}
					remaining = remaining[i+1:]
					found = true
					break
				}
			}
			if !found {
				break
			}
		}
		return time.Duration(days)*24*time.Hour + time.Duration(h)*time.Hour + time.Duration(m)*time.Minute + time.Duration(sec)*time.Second, nil
	}

	return time.Duration(days) * 24 * time.Hour, nil
}
