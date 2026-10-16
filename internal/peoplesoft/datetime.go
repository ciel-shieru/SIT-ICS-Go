package peoplesoft

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ParseEntryDateTime converts a PeopleSoft Entry's Day/StartTime/EndTime fields
// into concrete time.Time values in the given location.
func ParseEntryDateTime(entry Entry, loc *time.Location) (time.Time, time.Time, error) {
	day, err := parseDate(entry.Day, loc)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid timetable entry date: %w", err)
	}
	startHour, startMin, err := parseHoursMinutes(entry.StartTime)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid start time %q: %w", entry.StartTime, err)
	}
	endHour, endMin, err := parseHoursMinutes(entry.EndTime)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid end time %q: %w", entry.EndTime, err)
	}

	dtStart := time.Date(day.Year(), day.Month(), day.Day(), startHour, startMin, 0, 0, day.Location())
	dtEnd := time.Date(day.Year(), day.Month(), day.Day(), endHour, endMin, 0, 0, day.Location())
	return dtStart, dtEnd, nil
}

func parseHoursMinutes(s string) (int, int, error) {
	parts := strings.Split(s, ":")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid time format: %s", s)
	}
	h, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		return 0, 0, fmt.Errorf("invalid hour: %w", err)
	}
	m, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil {
		return 0, 0, fmt.Errorf("invalid minute: %w", err)
	}
	if h < 0 || h > 23 || m < 0 || m > 59 {
		return 0, 0, fmt.Errorf("time out of range: %s", s)
	}
	return h, m, nil
}
