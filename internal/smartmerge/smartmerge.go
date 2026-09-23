package smartmerge

import (
	"log"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/ciel-shieru/sit-ics-go/internal/calendar"
	"golang.org/x/net/html"
)

// ZoomDetails extracted from a BrightSpace event description.
type ZoomDetails struct {
	Link        string
	MeetingID   string
	Passcode    string
}

var zoomMeetingPathRe = regexp.MustCompile(`/j/(\d+)`)
	var zoomMeetingIDRe = regexp.MustCompile(`(?:[Mm]eeting[[:space:]]*[:]*[[:space:]]*)(\d[\d ]*\d)`)
	var zoomURLTextRe = regexp.MustCompile(`https?://[^ <>"'\n]*zoom\.us/j/(\d+)([^ <>"'\n]*)`)

// NormalizeModuleCode normalizes a module code by trimming whitespace,
// removing all spaces, and converting to uppercase.
func NormalizeModuleCode(code string) string {
	code = strings.TrimSpace(code)
	code = strings.ReplaceAll(code, " ", "")
	return strings.ToUpper(code)
}

// moduleCodeRe validates the PeopleSoft module code format: exactly 3 uppercase
// letters followed by exactly 4 digits, optionally followed by any suffix.
var moduleCodeRe = regexp.MustCompile(`^[A-Z]{3}[0-9]{4}([^0-9]|$)`)

// IsValidModuleCode returns true if the code matches the expected PeopleSoft
// module code format (3 letters + 4 digits, optional suffix).
func IsValidModuleCode(code string) bool {
	return moduleCodeRe.MatchString(code)
}

// ExtractModuleCodeFromOrgUnitName extracts the leading alphanumeric module code
// from a BrightSpace OrgUnitName.
func ExtractModuleCodeFromOrgUnitName(orgUnitName string) string {
	re := regexp.MustCompile(`^([A-Za-z0-9]+)`)
	match := re.FindStringSubmatch(orgUnitName)
	if match == nil || len(match) < 2 {
		return ""
	}
	return NormalizeModuleCode(match[1])
}

// ModulesMatch checks whether a PeopleSoft module code and a BrightSpace
// event refer to the same module by comparing their OrgUnitCode fields.
func ModulesMatch(psCode string, bsEvent calendar.Event) bool {
	normalizedPS := NormalizeModuleCode(psCode)
	normalizedBS := NormalizeModuleCode(bsEvent.OrgUnitCode)
	if normalizedPS == "" || normalizedBS == "" {
		return false
	}
	if !IsValidModuleCode(normalizedPS) {
		log.Printf("smartmerge: invalid module code format: %q", normalizedPS)
		return false
	}
	match := strings.Contains(normalizedBS, normalizedPS)
	return match
}

// TimesOverlap checks whether two time ranges overlap.
// Uses strict "before" to avoid matching adjacent events.
func TimesOverlap(psStart, psEnd, bsStart, bsEnd time.Time) bool {
	return psStart.Before(bsEnd) && bsStart.Before(psEnd)
}

// MatchesLocationConditions checks whether the location conditions for smart
// merge are satisfied: PeopleSoft location is "Online", "TBD", "To Be Advised",
// or "TBA" and BrightSpace location contains "Zoom Online Meeting".
func MatchesLocationConditions(psEvent, bsEvent calendar.Event) bool {
	psLocation := strings.ToLower(strings.TrimSpace(psEvent.Location))
	bsLocation := strings.ToLower(bsEvent.Location)

	psValid := psLocation == "online" || psLocation == "tbd" || psLocation == "to be advised" || psLocation == "tba"

	bsValid := strings.Contains(bsLocation, "zoom online meeting")

	return psValid && bsValid
}

// parseZoomURL extracts meeting ID and passcode from a Zoom URL.
func parseZoomURL(rawURL string, details *ZoomDetails) {
	if matches := zoomMeetingPathRe.FindStringSubmatch(rawURL); len(matches) > 1 {
		digits := matches[1]
		if len(digits) >= 11 {
			details.MeetingID = digits[0:3] + " " + digits[3:6] + " " + digits[6:11]
		} else {
			details.MeetingID = digits
		}
	}
	u, err := url.Parse(rawURL)
	if err == nil {
		details.Passcode = u.Query().Get("pwd")
	}
}

// getTextContent extracts the text content from an HTML node and its children.
func getTextContent(n *html.Node) string {
	var text strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.TextNode {
			text.WriteString(c.Data)
		}
	}
	return text.String()
}

// ExtractZoomDetails parses an HTML description to extract Zoom meeting details.
func ExtractZoomDetails(description string) ZoomDetails {
	doc, err := html.Parse(strings.NewReader(description))
	if err != nil {
		return ZoomDetails{}
	}

	var details ZoomDetails

	// First pass: look for <a href="...zoom.us..."> tags
	var visit func(*html.Node)
	visit = func(n *html.Node) {
		if details.Link != "" {
			return
		}
		if n.Type == html.ElementNode && n.Data == "a" {
			for _, attr := range n.Attr {
				if attr.Key == "href" && strings.Contains(strings.ToLower(attr.Val), "zoom.us") {
					details.Link = attr.Val
					parseZoomURL(attr.Val, &details)
					break
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			visit(c)
		}
	}
	visit(doc)

	// Second pass: if no <a> tag found, scan all text nodes for zoom URLs
	if details.Link == "" {
		var visitText func(*html.Node)
		visitText = func(n *html.Node) {
			if n.Type == html.TextNode {
				text := n.Data
				if matches := zoomURLTextRe.FindStringSubmatch(text); len(matches) >= 3 {
					meetingDigits := matches[1]
					rest := matches[2]
					details.Link = "http://zoom.us/j/" + meetingDigits + rest
					parseZoomURL(details.Link, &details)
					return
				}
			}
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				visitText(c)
			}
		}
		visitText(doc)
	}

	// Third pass: try to extract meeting ID from link text if not yet found
	if details.Link != "" && details.MeetingID == "" {
		var visitLinkText func(*html.Node)
		visitLinkText = func(n *html.Node) {
			if details.MeetingID != "" {
				return
			}
			if n.Type == html.ElementNode && n.Data == "a" {
				text := getTextContent(n)
				if matches := zoomMeetingIDRe.FindStringSubmatch(text); len(matches) > 1 {
					digits := matches[1]
					cleaned := strings.ReplaceAll(strings.TrimSpace(digits), " ", "")
					if len(cleaned) >= 11 {
						details.MeetingID = cleaned[0:3] + " " + cleaned[3:6] + " " + cleaned[6:11]
					} else {
						details.MeetingID = strings.TrimSpace(digits)
					}
				}
			}
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				visitLinkText(c)
			}
		}
		visitLinkText(doc)
	}

	return details
}

// MergeEvents performs smart merging of PeopleSoft timetable events with
// BrightSpace calendar events. When a match is found, the BrightSpace event
// is discarded and its Zoom meeting details are appended to the PeopleSoft
// event's description.
//
// A match requires all three conditions:
// 1. Module code match
// 2. Time overlap
// 3. Location conditions (PS "Online" + BS "Zoom Online Meeting")
func MergeEvents(psEvents, bsEvents []calendar.Event) ([]calendar.Event, int) {
	merged := 0
	mergedPS := make([]calendar.Event, len(psEvents))
	copy(mergedPS, psEvents)
	matchedPS := make([]bool, len(psEvents))

	unmatchedBS := make([]calendar.Event, 0, len(bsEvents))

	for _, bsEvent := range bsEvents {
		found := false
		for i, psEvent := range mergedPS {
			if matchedPS[i] {
				continue
			}

			if !ModulesMatch(psEvent.CourseCode, bsEvent) {
				continue
			}

			if !TimesOverlap(psEvent.DTStart, psEvent.DTEnd, bsEvent.DTStart, bsEvent.DTEnd) {
				continue
			}

			if !MatchesLocationConditions(psEvent, bsEvent) {
				continue
			}

		zoomDetails := ExtractZoomDetails(bsEvent.Description)
		if zoomDetails.Link == "" {
			continue
		}

		psEvent.Description += "\n\nZoom Meeting Details:\nLink: " + zoomDetails.Link
		if zoomDetails.MeetingID != "" {
			psEvent.Description += "\nMeeting ID: " + zoomDetails.MeetingID
		}
		if zoomDetails.Passcode != "" {
			psEvent.Description += "\nPasscode: " + zoomDetails.Passcode
		}

		mergedPS[i] = psEvent
		matchedPS[i] = true
		merged++
		found = true
		break
		}
		if !found {
			unmatchedBS = append(unmatchedBS, bsEvent)
		}
	}

	result := make([]calendar.Event, 0, len(mergedPS)+len(unmatchedBS))
	result = append(result, mergedPS...)
	result = append(result, unmatchedBS...)

	return result, merged
}

// DedupQuizzes replaces calendar events that are associated with quizzes
// with their quiz equivalents.
//
// BrightSpace calendar events associated with quizzes have QuizID == 0.
// Matching is done by cross-referencing: a quiz event with QuizID == X
// replaces a calendar event with CalendarEventID == X.
//
// When a match is found:
//   - The calendar event is removed from the result
//   - The quiz event is kept as-is (CalendarEventID remains 0)
//   - Unmatched quiz events are kept as-is
//   - Non-quiz calendar events (no matching quiz) are kept as-is
//
// Duplicate calendar events (same CalendarEventID) are deduplicated —
// only the first occurrence is kept.
//
// Returns the merged event slice and the count of calendar events
// replaced by quiz events.
func DedupQuizzes(calendarEvents, quizEvents []calendar.Event) ([]calendar.Event, int) {
	// Build set of CalendarEventIDs that have matching quiz events.
	// BrightSpace calendar events associated with quizzes have QuizID == 0,
	// so we match by cross-referencing calendar CalendarEventID with quiz QuizID.
	quizCalendarEventIDs := make(map[int]bool)
	for _, qEv := range quizEvents {
		if qEv.QuizID > 0 {
			quizCalendarEventIDs[qEv.QuizID] = true
		}
	}

	// Deduplicate calendar events by CalendarEventID (keep first occurrence),
	// and filter out those that have matching quiz events.
	replacements := 0
	seenCalendarEventIDs := make(map[int]bool)
	var result []calendar.Event
	for _, ev := range calendarEvents {
		if ev.CalendarEventID > 0 {
			if seenCalendarEventIDs[ev.CalendarEventID] {
				continue // duplicate calendar event, skip
			}
			if quizCalendarEventIDs[ev.CalendarEventID] {
				replacements++ // calendar event replaced by quiz event
				continue       // skip (quiz will be added below)
			}
			seenCalendarEventIDs[ev.CalendarEventID] = true
		}
		result = append(result, ev)
	}

	// Add quiz events.
	for _, quizEv := range quizEvents {
		result = append(result, quizEv)
	}

	return result, replacements
}
