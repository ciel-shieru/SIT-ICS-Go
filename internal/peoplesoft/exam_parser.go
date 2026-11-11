package peoplesoft

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/html"
)

// examContainerIDRe matches PeopleSoft exam container div ids like
// "div$ICField16$0", "div$ICField16$1", etc. The trailing number indexes
// individual exam entries on the page.
var examContainerIDRe = regexp.MustCompile(`^div\$ICField16\$(\d+)$`)

// examFieldRe matches PeopleSoft exam field div ids within a container,
// e.g. "N_ES_XMTT_WRK_DESCR100$0", "N_ES_XMTT_WRK_DESCR100_2$0", etc.
var examFieldRe = regexp.MustCompile(`^N_ES_XMTT_WRK_(?:DESCR100(?:_\d+)?|N_SPEC_ARRGMT|DESCR1)\$(\d+)$`)

// ParseExamHTML parses exam timetable data from the PeopleSoft exam page HTML.
// The exam page returns HTML wrapped in XML-like tags; this function extracts
// the inner HTML and parses exam entries from it.
func ParseExamHTML(htmlContent string, loc *time.Location) ([]Exam, error) {
	// Strip PeopleSoft XML wrapper if present.
	inner := extractHTMLFromXML(htmlContent)

	doc, err := html.Parse(strings.NewReader(inner))
	if err != nil {
		return nil, fmt.Errorf("parse exam HTML: %w", err)
	}

	var exams []Exam
	findExamContainers(doc, &exams, loc)

	return exams, nil
}

// extractHTMLFromXML strips PeopleSoft XML wrapping (<PAGE><FIELD>…</FIELD></PAGE>)
// and returns the raw HTML fragment.
func extractHTMLFromXML(raw string) string {
	// Look for the first < and last > to strip XML wrapper.
	if idx := strings.Index(raw, "<"); idx >= 0 {
		// Find matching closing tag of the outermost element.
		outerTag := extractTagName(raw[idx:])
		closeTag := "</" + strings.TrimSuffix(outerTag, ">")
		if end := strings.LastIndex(raw, closeTag); end > idx {
			return strings.TrimSpace(raw[idx+len(outerTag) : end])
		}
	}
	return raw
}

func extractTagName(s string) string {
	end := strings.Index(s, ">")
	if end < 0 {
		return s
	}
	return s[:end+1]
}

func getAttr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}

func findExamContainers(n *html.Node, exams *[]Exam, loc *time.Location) {
	if n.Type == html.ElementNode && n.Data == "div" {
		classAttr := getAttr(n, "class")

		// Look for the PAGECONTAINER_TGT div.
		if classAttr == "divPAGECONTAINER_TGT" {
			// Walk children to find exam containers.
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				if cid := getAttr(c, "id"); examContainerIDRe.MatchString(cid) {
					exam := parseExamContainer(c, loc)
					if exam != nil {
						*exams = append(*exams, *exam)
					}
				}
			}
			return
		}
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		findExamContainers(c, exams, loc)
	}
}

func parseExamContainer(n *html.Node, loc *time.Location) *Exam {
	var dateStr, timeStr, courseStr, locationStr, seatStr, examTypeStr string
	var containerIndex int

	// First pass: collect all field values and find the container index.
	var foundFields int
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.ElementNode && node.Data == "div" {
			idAttr := getAttr(node, "id")
			if idAttr == "" {
				return
			}

			// Check if this is an exam field div.
			if matches := examFieldRe.FindStringSubmatch(idAttr); matches != nil {
				idx, _ := strconv.Atoi(matches[1])
				if containerIndex == 0 {
					containerIndex = idx
				} else if containerIndex != idx {
					return // belongs to a different container
				}
				foundFields++

				text := strings.TrimSpace(getTextContent(node))
				switch {
				case strings.HasSuffix(idAttr, "_2$") || strings.Contains(idAttr, "DESCR100_2$"):
					timeStr = text
				case strings.Contains(idAttr, "DESCR100_3$"):
					courseStr = text
				case strings.Contains(idAttr, "DESCR100_4$"):
					locationStr = text
				case strings.Contains(idAttr, "N_SPEC_ARRGMT$"):
					seatStr = text
				case strings.Contains(idAttr, "DESCR100$") && !strings.Contains(idAttr, "_"):
					dateStr = text
				case strings.Contains(idAttr, "DESCR1$"):
					examTypeStr = text
				}
			}
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)

	if foundFields == 0 {
		return nil
	}

	if dateStr == "" || timeStr == "" || courseStr == "" {
		return nil
	}

	// Parse course code and name.
	courseCode, courseName := splitCourseCode(courseStr)

	return &Exam{
		Date:       dateStr,
		Time:       timeStr,
		CourseCode: courseCode,
		CourseName: courseName,
		Location:   locationStr,
		SeatNumber: seatStr,
		ExamType:   examTypeStr,
	}
}

// splitCourseCode splits "FAK 1001 Sample Course Title" into
// code "FAK 1001" and name "Sample Course Title".
func splitCourseCode(s string) (code, name string) {
	s = strings.TrimSpace(s)
	parts := strings.SplitN(s, " ", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return s, ""
}

// examDateRe matches date strings like "Monday Jan 15,2027".
var examDateRe = regexp.MustCompile(`^(Mon|Tue|Wed|Thu|Fri|Sat|Sun)\s+(\w+)\s+(\d{1,2}),(\d{4})$`)

// ParseExamDate converts a PeopleSoft exam date string to a time.Time.
func ParseExamDate(dateStr string, loc *time.Location) (time.Time, error) {
	matches := examDateRe.FindStringSubmatch(dateStr)
	if matches == nil {
		return time.Time{}, fmt.Errorf("invalid exam date format: %q", dateStr)
	}

	// Map month abbreviation to month number.
	monthMap := map[string]time.Month{
		"Jan": time.January, "Feb": time.February, "Mar": time.March,
		"Apr": time.April, "May": time.May, "Jun": time.June,
		"Jul": time.July, "Aug": time.August, "Sep": time.September,
		"Oct": time.October, "Nov": time.November, "Dec": time.December,
	}

	month, ok := monthMap[matches[2]]
	if !ok {
		return time.Time{}, fmt.Errorf("unknown month in exam date: %q", matches[2])
	}

	var day, year int
	fmt.Sscanf(matches[3], "%d", &day)
	fmt.Sscanf(matches[4], "%d", &year)

	return time.Date(year, month, day, 0, 0, 0, 0, loc), nil
}

// ParseExamTime parses a PeopleSoft exam time range like "2.00 PM to 4.30 PM"
// and returns (startHour, startMin, endHour, endMin).
func ParseExamTime(timeStr string) (int, int, int, int, error) {
	// Split on " to ".
	parts := strings.SplitN(timeStr, " to ", 2)
	if len(parts) != 2 {
		return 0, 0, 0, 0, fmt.Errorf("invalid exam time format: %q", timeStr)
	}

	startHour, startMin, err := parseExamTimePart(parts[0])
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("parse start time %q: %w", parts[0], err)
	}

	endHour, endMin, err := parseExamTimePart(parts[1])
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("parse end time %q: %w", parts[1], err)
	}

	return startHour, startMin, endHour, endMin, nil
}

func parseExamTimePart(s string) (int, int, error) {
	s = strings.TrimSpace(s)

	// Match patterns like "9.30 AM", "9.30AM", "9:30 AM", "9:30AM"
	var hour, min int
	var ampm string

	if strings.Contains(s, ".") {
		// Format: "9.30 AM" or "9.30AM"
		parts := strings.Split(s, ".")
		if len(parts) != 2 {
			return 0, 0, fmt.Errorf("invalid time format: %q", s)
		}
		hour = parseInt(parts[0])
		// Remove AM/PM from the minute part.
		minPart := strings.TrimSpace(parts[1])
		if strings.HasPrefix(minPart, "AM") || strings.HasPrefix(minPart, "am") {
			ampm = "AM"
			min = parseInt(minPart[2:])
		} else if strings.HasPrefix(minPart, "PM") || strings.HasPrefix(minPart, "pm") {
			ampm = "PM"
			min = parseInt(minPart[2:])
		} else {
			min = parseInt(minPart)
		}
	} else {
		// Format: "9:30 AM" or "9:30AM"
		parts := strings.Split(s, ":")
		if len(parts) != 2 {
			return 0, 0, fmt.Errorf("invalid time format: %q", s)
		}
		hour = parseInt(parts[0])
		minPart := strings.TrimSpace(parts[1])
		if strings.HasPrefix(minPart, "AM") || strings.HasPrefix(minPart, "am") {
			ampm = "AM"
			min = parseInt(minPart[2:])
		} else if strings.HasPrefix(minPart, "PM") || strings.HasPrefix(minPart, "pm") {
			ampm = "PM"
			min = parseInt(minPart[2:])
		} else {
			min = parseInt(minPart)
		}
	}

	if ampm == "PM" && hour < 12 {
		hour += 12
	}
	if ampm == "AM" && hour == 12 {
		hour = 0
	}

	return hour, min, nil
}

func parseInt(s string) int {
	result := 0
	for _, c := range s {
		if c >= '0' && c <= '9' {
			result = result*10 + int(c-'0')
		}
	}
	return result
}

// ParseExamDateTime converts an Exam to its calendar start and end times.
func ParseExamDateTime(exam Exam, loc *time.Location) (time.Time, time.Time, error) {
	dtStart, err := ParseExamDate(exam.Date, loc)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("parse exam date %q: %w", exam.Date, err)
	}

	startHour, startMin, endHour, endMin, err := ParseExamTime(exam.Time)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("parse exam time %q: %w", exam.Time, err)
	}

	dtStart = dtStart.Add(time.Duration(startHour)*time.Hour + time.Duration(startMin)*time.Minute)
	dtEnd := dtStart.Add(time.Duration(endHour-startHour)*time.Hour + time.Duration(endMin-startMin)*time.Minute)

	return dtStart, dtEnd, nil
}
