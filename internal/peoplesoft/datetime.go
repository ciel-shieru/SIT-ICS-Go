package peoplesoft

import (
	"fmt"
	"strings"
	"time"
)

// ParseEntryDateTime converts a PeopleSoft Entry's Day/StartTime/EndTime fields
// into concrete time.Time values in the given location.
func ParseEntryDateTime(entry Entry, loc *time.Location) (time.Time, time.Time, error) {
	parts := strings.Split(entry.Day, "/")
	if len(parts) != 3 {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid day format: %s", entry.Day)
	}
	var day, month, year int
	fmt.Sscanf(parts[0], "%d", &day)
	fmt.Sscanf(parts[1], "%d", &month)
	fmt.Sscanf(parts[2], "%d", &year)

	startHour, startMin := parseHoursMinutes(entry.StartTime)
	endHour, endMin := parseHoursMinutes(entry.EndTime)

	dtStart := time.Date(year, time.Month(month), day, startHour, startMin, 0, 0, loc)
	dtEnd := time.Date(year, time.Month(month), day, endHour, endMin, 0, 0, loc)
	return dtStart, dtEnd, nil
}

func parseHoursMinutes(s string) (int, int) {
	var h, m int
	fmt.Sscanf(s, "%d:%d", &h, &m)
	return h, m
}
