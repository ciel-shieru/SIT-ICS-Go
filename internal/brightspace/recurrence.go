package brightspace

import (
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/ciel-shieru/sit-ics-go/internal/calendar"
)

const (
	maxOccurrences = 100
	maxWeeks       = 200
	maxMonths      = 2000
	maxYears       = 200
)

// ExpandRecurrence expands a recurring BrightSpace entry into individual calendar events,
// one per occurrence. Returns events sorted by start time with RecurrenceIndex 0, 1, 2, ...
func ExpandRecurrence(entry BrightSpaceStringEntry, loc *time.Location) ([]calendar.Event, error) {
	if !entry.IsRecurring || entry.RepeatType == 1 {
		return nil, fmt.Errorf("not recurring")
	}

	baseStart, err := ParseTimestamp(entry.DTStart, loc)
	if err != nil {
		return nil, fmt.Errorf("parse start: %w", err)
	}

	baseEnd, err := ParseTimestamp(entry.DTEnd, loc)
	if err != nil {
		return nil, fmt.Errorf("parse end: %w", err)
	}

	duration := baseEnd.Sub(baseStart)

	var untilDate time.Time
	if entry.RepeatUntilDateString != "" {
		untilDate, err = ParseTimestamp(entry.RepeatUntilDateString, time.UTC)
		if err != nil {
			return nil, fmt.Errorf("parse repeat until: %w", err)
		}
		untilDate = untilDate.In(loc)
	}

	var occurrences []time.Time

	switch entry.RepeatType {
	case 2: // Daily
		occurrences = expandDaily(baseStart, duration, entry.RepeatEvery, untilDate, loc)
	case 3: // Weekly
		occurrences, err = expandWeekly(baseStart, duration, entry.RepeatEvery, entry.RepeatOnInfo, untilDate, loc)
		if err != nil {
			return nil, err
		}
	case 4: // Monthly
		occurrences = expandMonthly(baseStart, duration, entry.RepeatEvery, untilDate, loc)
	case 5: // Yearly
		occurrences = expandYearly(baseStart, duration, entry.RepeatEvery, untilDate, loc)
	default:
		return nil, fmt.Errorf("unknown repeat type: %d", entry.RepeatType)
	}

	if len(occurrences) > maxOccurrences {
		log.Printf("brightspace: recurring event %q capped at %d occurrences (had %d)", entry.Title, maxOccurrences, len(occurrences))
		occurrences = occurrences[:maxOccurrences]
	}

	events := make([]calendar.Event, 0, len(occurrences))
	for i, start := range occurrences {
		events = append(events, buildEvent(entry, start, start.Add(duration), i))
	}

	return events, nil
}

func expandDaily(baseStart time.Time, duration time.Duration, repeatEvery int, untilDate time.Time, loc *time.Location) []time.Time {
	var occurrences []time.Time
	current := baseStart
	for current.Before(untilDate) {
		occurrences = append(occurrences, current)
		current = current.Add(time.Duration(repeatEvery) * 24 * time.Hour)
	}
	return occurrences
}

func expandWeekly(baseStart time.Time, duration time.Duration, repeatEvery int, repeatOnInfo *RecurrenceInfo, untilDate time.Time, loc *time.Location) ([]time.Time, error) {
	if repeatOnInfo == nil || repeatOnInfo.RepeatOnInfo == nil {
		var occurrences []time.Time
		current := baseStart
		for current.Before(untilDate) {
			occurrences = append(occurrences, current)
			current = current.Add(time.Duration(repeatEvery) * 7 * 24 * time.Hour)
		}
		return occurrences, nil
	}

	ro := repeatOnInfo.RepeatOnInfo
	var activeDays []time.Weekday
	if ro.Monday {
		activeDays = append(activeDays, time.Monday)
	}
	if ro.Tuesday {
		activeDays = append(activeDays, time.Tuesday)
	}
	if ro.Wednesday {
		activeDays = append(activeDays, time.Wednesday)
	}
	if ro.Thursday {
		activeDays = append(activeDays, time.Thursday)
	}
	if ro.Friday {
		activeDays = append(activeDays, time.Friday)
	}
	if ro.Saturday {
		activeDays = append(activeDays, time.Saturday)
	}
	if ro.Sunday {
		activeDays = append(activeDays, time.Sunday)
	}

	if len(activeDays) == 0 {
		return nil, fmt.Errorf("weekly recurrence has no active days")
	}

	sort.Slice(activeDays, func(i, j int) bool { return activeDays[i] < activeDays[j] })

	var allOccurrences []time.Time

	baseDay := baseStart.Weekday()
	for weekOffset := 0; weekOffset < maxWeeks; weekOffset++ {
		weekStart := baseStart.Add(time.Duration(weekOffset*repeatEvery) * 7 * 24 * time.Hour)

		for _, day := range activeDays {
			daysUntil := int(day) - int(baseDay)
			if daysUntil < 0 {
				daysUntil += 7
			}

			occ := weekStart.Add(time.Duration(daysUntil) * 24 * time.Hour)
			if !occ.Before(untilDate) {
				if weekOffset == 0 {
					return nil, fmt.Errorf("weekly recurrence: no active days before repeat until date")
				}
				continue
			}
			allOccurrences = append(allOccurrences, occ)
		}

		lastActiveDay := activeDays[len(activeDays)-1]
		daysUntil := int(lastActiveDay) - int(baseDay)
		if daysUntil < 0 {
			daysUntil += 7
		}
		lastOcc := weekStart.Add(time.Duration(daysUntil) * 24 * time.Hour)
		if !lastOcc.Before(untilDate) && weekOffset > 0 {
			break
		}
	}

	sort.Slice(allOccurrences, func(i, j int) bool { return allOccurrences[i].Before(allOccurrences[j]) })

	var deduped []time.Time
	for i, occ := range allOccurrences {
		if i == 0 || !occ.Equal(allOccurrences[i-1]) {
			deduped = append(deduped, occ)
		}
	}

	return deduped, nil
}

func expandMonthly(baseStart time.Time, duration time.Duration, repeatEvery int, untilDate time.Time, loc *time.Location) []time.Time {
	var occurrences []time.Time
	_, _, day := baseStart.Date()
	hour, min, sec := baseStart.Clock()

	for i := 0; i < maxMonths; i++ {
		totalMonths := int(baseStart.Month()-time.January) + i*repeatEvery
		targetYear := baseStart.Year() + totalMonths/12
		targetMonth := time.Month(totalMonths%12 + 1)

		occ := time.Date(targetYear, targetMonth, day, hour, min, sec, 0, loc)

		// Check if overflow occurred (day was clamped/overflowed into next month)
		_, m, d := occ.Date()
		if m != targetMonth || d != day {
			// Overflow occurred, clamp to last day of target month
			occ = time.Date(targetYear, targetMonth+1, 0, hour, min, sec, 0, loc)
		}

		if !occ.Before(untilDate) {
			break
		}
		occurrences = append(occurrences, occ)
	}

	return occurrences
}

func expandYearly(baseStart time.Time, duration time.Duration, repeatEvery int, untilDate time.Time, loc *time.Location) []time.Time {
	var occurrences []time.Time
	_, month, day := baseStart.Date()
	hour, min, sec := baseStart.Clock()

	for i := 0; i < maxYears; i++ {
		targetYear := baseStart.Year() + i*repeatEvery
		occ := time.Date(targetYear, month, day, hour, min, sec, 0, loc)
		if !occ.Before(untilDate) {
			break
		}
		occurrences = append(occurrences, occ)
	}

	return occurrences
}
