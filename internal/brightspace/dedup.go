package brightspace

import (
	"strings"
	"time"
)

// EventType constants from BrightSpace Valence Calendar API.
const (
	EventTypeReminder       = 1
	EventTypeAvailabilityStarts = 2
	EventTypeAvailabilityEnds   = 3
	EventTypeUnlockStarts       = 4
	EventTypeUnlockEnds         = 5
	EventTypeDueDate            = 6
)

// eventTypePriority returns the priority for an EventType (higher = more important).
// DueDate(6) > AvailabilityStarts(2) > AvailabilityEnds(3) > Reminder(1) > UnlockStarts(4) > UnlockEnds(5)
func eventTypePriority(eventType int) int {
	switch eventType {
	case EventTypeDueDate:
		return 5
	case EventTypeAvailabilityStarts:
		return 4
	case EventTypeAvailabilityEnds:
		return 3
	case EventTypeReminder:
		return 2
	case EventTypeUnlockStarts:
		return 1
	case EventTypeUnlockEnds:
		return 0
	default:
		return -1
	}
}

// normalizeTitle lowercases and trims the title for grouping.
func normalizeTitle(title string) string {
	return strings.ToLower(strings.TrimSpace(title))
}

// isTimeProximity checks if two times are within 15 minutes of each other.
func isTimeProximity(t1, t2 time.Time) bool {
	diff := t1.Sub(t2)
	return diff >= -15*time.Minute && diff <= 15*time.Minute
}

// calendarEventAPIToStringEntry converts a CalendarEventAPI to a temporary entry
// for deduplication purposes.
func calendarEventAPIToStringEntry(ev CalendarEventAPI) BrightSpaceStringEntry {
	title := ev.Title
	if title == "" {
		title = ev.OrgUnitName
	}

	descParts := []string{}
	if ev.Description != "" {
		if strings.Contains(strings.ToLower(ev.Description), "zoom.us") {
			descParts = append(descParts, ev.Description)
		} else {
			plainDesc := htmlToPlainText(ev.Description)
			if plainDesc != "" {
				descParts = append(descParts, plainDesc)
			}
		}
	}
	if ev.LocationName != "" {
		descParts = append(descParts, "Location: "+ev.LocationName)
	}

	var quizID int
	if ev.AssociatedEntity != nil && ev.AssociatedEntity.AssociatedEntityType == "D2L.LE.Quizzing.Quiz" {
		quizID = ev.AssociatedEntity.AssociatedEntityId
	}

	return BrightSpaceStringEntry{
		Title:           title,
		OrgUnitId:       ev.OrgUnitCode,
		OrgUnitName:     ev.OrgUnitName,
		OrgUnitCode:     ev.OrgUnitCode,
		Location:        ev.LocationName,
		Description:     strings.Join(descParts, "\n"),
		DTStart:         ev.StartDateTime,
		DTEnd:           ev.EndDateTime,
		IsAllDay:        ev.IsAllDayEvent,
		Source:          "calendar",
		CalendarEventID: ev.CalendarEventId,
		QuizID:          quizID,
		EventType:       ev.EventType,
	}
}

// entityKey represents the grouping key for entity-associated events.
type entityKey struct {
	EntityID int
	OrgUnitID string
	Title    string
}

// nonEntityKey represents the grouping key for non-entity events.
type nonEntityKey struct {
	OrgUnitID string
	Title     string
	TimeHash  string
}

// consolidatedEntry holds the merged result of a deduplication group.
type consolidatedEntry struct {
	merged      BrightSpaceStringEntry
	bestEventType int
	groupSize   int
}

// DeduplicateCalendarEntries consolidates multiple CalendarEventAPI entries
// that represent the same logical event into single entries.
//
// Entity-associated events (AssociatedEntity != nil) are grouped by
// (AssociatedEntityId, OrgUnitId, normalized Title).
//
// Non-entity events are grouped by (OrgUnitId, normalized Title, time proximity).
func DeduplicateCalendarEntries(entries []CalendarEventAPI) []CalendarEventAPI {
	if entries == nil {
		return nil
	}
	if len(entries) == 0 {
		return []CalendarEventAPI{}
	}

	// First pass: separate entity-associated and non-entity events.
	var entityEntries []CalendarEventAPI
	var nonEntityEntries []CalendarEventAPI

	for _, ev := range entries {
		if ev.AssociatedEntity != nil {
			entityEntries = append(entityEntries, ev)
		} else {
			nonEntityEntries = append(nonEntityEntries, ev)
		}
	}

	// Deduplicate entity-associated events.
	var entityResult []CalendarEventAPI
	if len(entityEntries) > 0 {
		entityResult = deduplicateEntityEvents(entityEntries)
	}

	// Deduplicate non-entity events.
	var nonEntityResult []CalendarEventAPI
	if len(nonEntityEntries) > 0 {
		nonEntityResult = deduplicateNonEntityEvents(nonEntityEntries)
	}

	// Combine results.
	result := make([]CalendarEventAPI, 0, len(entityResult)+len(nonEntityResult))
	result = append(result, entityResult...)
	result = append(result, nonEntityResult...)

	return result
}

// deduplicateEntityEvents groups entity-associated events by (EntityID, OrgUnitID, normalized Title)
// and merges each group into a single entry.
func deduplicateEntityEvents(entries []CalendarEventAPI) []CalendarEventAPI {
	groups := make(map[entityKey][]CalendarEventAPI)

	for _, ev := range entries {
		key := entityKey{
			EntityID:  ev.AssociatedEntity.AssociatedEntityId,
			OrgUnitID: ev.OrgUnitCode,
			Title:     normalizeTitle(ev.Title),
		}
		groups[key] = append(groups[key], ev)
	}

	result := make([]CalendarEventAPI, 0, len(groups))
	for _, group := range groups {
		result = append(result, mergeCalendarEvents(group)...)
	}

	return result
}

// deduplicateNonEntityEvents groups non-entity events by (OrgUnitID, normalized Title, time proximity).
func deduplicateNonEntityEvents(entries []CalendarEventAPI) []CalendarEventAPI {
	if len(entries) == 0 {
		return []CalendarEventAPI{}
	}

	// Convert to string entries for time parsing.
	stringEntries := make([]BrightSpaceStringEntry, 0, len(entries))
	for _, ev := range entries {
		se := calendarEventAPIToStringEntry(ev)
		stringEntries = append(stringEntries, se)
	}

	// Group by (OrgUnitID, normalized Title).
	titleGroups := make(map[nonEntityKey][]int) // key -> indices into stringEntries
	for i, se := range stringEntries {
		title := normalizeTitle(se.Title)
		key := nonEntityKey{
			OrgUnitID: se.OrgUnitCode,
			Title:     title,
		}
		titleGroups[key] = append(titleGroups[key], i)
	}

	// Within each title group, cluster by time proximity.
	var result []CalendarEventAPI
	for _, indices := range titleGroups {
		// Sort indices by start time.
		sortIndicesByTime(indices, stringEntries)

		// Cluster: events within 15 minutes of any member in the cluster (transitive).
		var clusters [][]int
		for _, idx := range indices {
			placed := false
			for ci := range clusters {
				for _, jdx := range clusters[ci] {
					if isTimeProximity(stringEntries[idx].DTStartParsed(), stringEntries[jdx].DTStartParsed()) {
						clusters[ci] = append(clusters[ci], idx)
						placed = true
						break
					}
				}
				if placed {
					break
				}
			}
			if !placed {
				clusters = append(clusters, []int{idx})
			}
		}

		// Merge each cluster.
		for _, cluster := range clusters {
			var calendarEvents []CalendarEventAPI
			for _, idx := range cluster {
				calendarEvents = append(calendarEvents, entries[idx])
			}
			result = append(result, mergeCalendarEvents(calendarEvents)...)
		}
	}

	return result
}

// sortIndicesByTime sorts indices based on the parsed start time of stringEntries.
func sortIndicesByTime(indices []int, stringEntries []BrightSpaceStringEntry) {
	for i := 1; i < len(indices); i++ {
		for j := i; j > 0 && isEarlier(stringEntries[indices[j]], stringEntries[indices[j-1]]); j-- {
			indices[j], indices[j-1] = indices[j-1], indices[j]
		}
	}
}

// isEarlier returns true if a starts before b.
func isEarlier(a, b BrightSpaceStringEntry) bool {
	ta := a.DTStartParsed()
	tb := b.DTStartParsed()
	return ta.Before(tb)
}

// mergeCalendarEvents merges a group of CalendarEventAPI entries into one.
// DTStart = minimum start, DTEnd = maximum end, description from highest-priority EventType.
func mergeCalendarEvents(entries []CalendarEventAPI) []CalendarEventAPI {
	if len(entries) == 1 {
		return entries
	}

	best := entries[0]
	bestPriority := eventTypePriority(entries[0].EventType)

	for _, ev := range entries[1:] {
		priority := eventTypePriority(ev.EventType)
		if priority > bestPriority {
			bestPriority = priority
			best = ev
		}
	}

	// Find earliest start and latest end across ALL entries.
	loc, _ := time.LoadLocation("UTC")
	earliestStart := entries[0].StartDateTime
	latestEnd := entries[0].EndDateTime

	for _, ev := range entries {
		startTime, err1 := parseTimestampForDedup(ev.StartDateTime, loc)
		endTime, err2 := parseTimestampForDedup(ev.EndDateTime, loc)

		if err1 == nil && !startTime.IsZero() {
			earliestStartTime, _ := parseTimestampForDedup(earliestStart, loc)
			if startTime.Before(earliestStartTime) || earliestStart == "" {
				earliestStart = ev.StartDateTime
			}
		}
		if err2 == nil && !endTime.IsZero() {
			latestEndTime, _ := parseTimestampForDedup(latestEnd, loc)
			if endTime.After(latestEndTime) || latestEnd == "" {
				latestEnd = ev.EndDateTime
			}
		}
	}

	best.StartDateTime = earliestStart
	best.EndDateTime = latestEnd

	return []CalendarEventAPI{best}
}

// parseTimestampForDedup parses a timestamp string for deduplication comparisons.
func parseTimestampForDedup(s string, loc *time.Location) (time.Time, error) {
	if s == "" {
		return time.Time{}, nil
	}
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		t, err2 := time.Parse("2006-01-02T15:04:05.000Z", s)
		if err2 != nil {
			return time.Time{}, err
		}
		t = t.In(loc)
	}
	if loc != nil {
		t = t.In(loc)
	}
	return t, nil
}

// DTStartParsed parses the DTStart field as time.Time.
func (e BrightSpaceStringEntry) DTStartParsed() time.Time {
	t, _ := parseTimestampForDedup(e.DTStart, time.UTC)
	return t
}


