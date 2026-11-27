package peoplesoft

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"golang.org/x/net/html"
)

// examContainerIDRe matches PeopleSoft exam container div ids in the exam
// page markup, e.g. "win0div$ICField16$0", "win0div$ICField16$1". The
// trailing number indexes individual exam entries. The anchors ensure row
// divs such as "win0div$ICField16_row$0" never match; the optional "win0"
// prefix is accepted for page variants without it.
var examContainerIDRe = regexp.MustCompile(`^(?:win0)?div\$ICField16\$(\d+)$`)

// examFieldRe matches PeopleSoft exam field ids after stripping the
// "win0div" prefix from the markup ids. Field ids have the form
// N_ES_XMTT_WRK_<ROLE>$<ROW>: the wrapping div ids carry the win0div prefix
// (e.g. win0divN_ES_XMTT_WRK_DESCR100_2$0), the value spans carry the bare
// ids (e.g. N_ES_XMTT_WRK_DESCR100_2$0). Label divs such as
// N_ES_XMTT_WRK_DESCR100lbl$0 do not match.
var examFieldRe = regexp.MustCompile(`^N_ES_XMTT_WRK_(?:DESCR100(?:_\d+)?|N_SPEC_ARRGMT|DESCR1)\$(\d+)$`)

// examContentFieldRe matches the opening tag of the ICAJAX FIELD element that
// carries the exam page content:
// <FIELD id='divPAGECONTAINER_TGT' ...>
// (single or double quotes around the id).
var examContentFieldRe = regexp.MustCompile(`<FIELD[^>]*\bid=["']divPAGECONTAINER_TGT["'][^>]*>`)

// examLocationLabelRe and examSeatLabelRe match the leading labels PeopleSoft
// prepends to location and seat field values, e.g. "Location :" and
// "Seat Number :".
var (
	examLocationLabelRe = regexp.MustCompile(`(?i)^Location\s*:\s*`)
	examSeatLabelRe     = regexp.MustCompile(`(?i)^Seat Number\s*:\s*`)
)

// ParseExamHTML parses exam timetable data from a PeopleSoft exam page
// response. The exams endpoint returns an ICAJAX page fragment: an XML-like
// document in which the real page HTML is embedded in a CDATA section inside
// <FIELD id='divPAGECONTAINER_TGT' ...>. When that structure is present the
// CDATA payload is extracted first; plain HTML input without an XML wrapper
// is parsed unchanged (backward compatibility). An XML-wrapped response that
// lacks the content field is an error, not an empty result, so the caller
// can treat the page as a degraded fetch and retry.
func ParseExamHTML(htmlContent string, loc *time.Location) ([]Exam, error) {
	inner, err := extractExamPageContent(htmlContent)
	if err != nil {
		return nil, err
	}

	doc, err := html.Parse(strings.NewReader(inner))
	if err != nil {
		return nil, fmt.Errorf("parse exam HTML: %w", err)
	}

	var exams []Exam
	for _, container := range findExamContainers(doc) {
		exam, err := parseExamContainer(container, loc)
		if err != nil {
			return nil, err
		}
		if exam != nil {
			exams = append(exams, *exam)
		}
	}
	return exams, nil
}

// extractExamPageContent returns the HTML fragment embedded in the CDATA
// section of the divPAGECONTAINER_TGT FIELD of an ICAJAX page response.
// If the XML wrapper is present (a <?xml declaration or a <PAGE element) but
// the content FIELD is missing, an error is returned so the caller reacts to
// the degraded page instead of parsing it silently. Plain HTML input without
// an XML wrapper is returned unchanged.
func extractExamPageContent(raw string) (string, error) {
	match := examContentFieldRe.FindStringIndex(raw)
	if match != nil {
		rest := raw[match[1]:] // after the FIELD opening tag
		cdataStart := strings.Index(rest, "<![CDATA[")
		if cdataStart < 0 {
			return "", fmt.Errorf("exam page content section (divPAGECONTAINER_TGT) not found")
		}
		// The CDATA section must belong to the content FIELD itself, not to a
		// later FIELD element.
		if nextField := strings.Index(rest, "<FIELD"); nextField >= 0 && nextField < cdataStart {
			return "", fmt.Errorf("exam page content section (divPAGECONTAINER_TGT) not found")
		}
		payload := rest[cdataStart+len("<![CDATA["):]
		cdataEnd := strings.Index(payload, "]]>")
		if cdataEnd < 0 {
			return "", fmt.Errorf("exam page content section (divPAGECONTAINER_TGT) not found")
		}
		return payload[:cdataEnd], nil
	}

	if strings.Contains(raw, "<?xml") || strings.Contains(raw, "<PAGE") {
		return "", fmt.Errorf("exam page content section (divPAGECONTAINER_TGT) not found")
	}
	return raw, nil
}

func getAttr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}

// findExamContainers recursively scans the parsed document and returns, in
// document order, every element whose id matches examContainerIDRe. The exam
// containers are nested several levels below the page container (win0div
// PSPAGECONTAINER -> win0div$ICField1 -> row divs), so the whole tree is
// scanned; no class gate is used because in the real markup the id
// divPAGECONTAINER_TGT identifies a FIELD element (class "ps_pagecontainer"),
// not a div.
func findExamContainers(doc *html.Node) []*html.Node {
	var containers []*html.Node
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "div" && examContainerIDRe.MatchString(getAttr(n, "id")) {
			containers = append(containers, n)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	return containers
}

// parseExamContainer extracts a single exam from an exam container element
// (id win0div$ICField16$N). Field values live in SPAN elements with the bare
// ids N_ES_XMTT_WRK_<ROLE>$N; the wrapping div ids carry the same ids with
// the win0div prefix. The label divs (...lbl$N) are not mapped, and the
// special-arrangements field (N_ES_XMTT_WRK_N_SPEC_ARRGMT$N, usually a
// &nbsp; placeholder) is deliberately not mapped to any exam field.
//
// A container whose fields are all empty (&nbsp; placeholders) is a
// placeholder row and yields (nil, nil). A container with data whose date,
// time, or course is missing or malformed yields an error, consistent with
// the strictness of ParseTimetableHTML, so the fetch layer retries the full
// fetch (ADR-0020) instead of silently dropping the row.
func parseExamContainer(n *html.Node, loc *time.Location) (*Exam, error) {
	idAttr := getAttr(n, "id")
	matches := examContainerIDRe.FindStringSubmatch(idAttr)
	if matches == nil {
		return nil, fmt.Errorf("unrecognized exam container id %q", idAttr)
	}
	rowIdx := matches[1]

	var dateStr, timeStr, courseStr, locationStr, seatStr, examTypeStr string
	var hasValue bool

	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.ElementNode {
			nodeID := getAttr(node, "id")
			if nodeID != "" {
				bareID := strings.TrimPrefix(nodeID, "win0div")
				if fm := examFieldRe.FindStringSubmatch(bareID); fm != nil && fm[1] == rowIdx {
					text := normalizeExamValue(getTextContent(node))
					if text != "" {
						hasValue = true
					}
					role := bareID[len("N_ES_XMTT_WRK_"):]
					switch role {
					case "DESCR100$" + rowIdx:
						// Date. The anchored role match allows no underscore
						// after DESCR100, which also excludes DESCR100_2$..
						// and the label ids.
						dateStr = text
					case "DESCR100_2$" + rowIdx:
						timeStr = text
					case "DESCR100_3$" + rowIdx:
						courseStr = text
					case "DESCR100_4$" + rowIdx:
						locationStr = examLocationLabelRe.ReplaceAllString(text, "")
					case "DESCR100_5$" + rowIdx:
						seatStr = examSeatLabelRe.ReplaceAllString(text, "")
					case "DESCR1$" + rowIdx:
						examTypeStr = text
						// N_SPEC_ARRGMT$<row> is special arrangements, not a seat:
						// intentionally not mapped.
					}
				}
			}
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)

	if !hasValue {
		// Placeholder row: every field is an empty (&nbsp;) value.
		return nil, nil
	}

	if dateStr == "" {
		return nil, fmt.Errorf("exam row %s has an empty date", rowIdx)
	}
	if _, err := ParseExamDate(dateStr, loc); err != nil {
		return nil, fmt.Errorf("exam row %s has invalid date %q: %w", rowIdx, dateStr, err)
	}
	if timeStr == "" {
		return nil, fmt.Errorf("exam row %s has an empty time", rowIdx)
	}
	if _, _, _, _, err := ParseExamTime(timeStr); err != nil {
		return nil, fmt.Errorf("exam row %s has invalid time %q: %w", rowIdx, timeStr, err)
	}
	if courseStr == "" {
		return nil, fmt.Errorf("exam row %s has an empty course", rowIdx)
	}

	courseCode, courseName := splitCourseCode(courseStr)

	return &Exam{
		Date:       dateStr,
		Time:       timeStr,
		CourseCode: courseCode,
		CourseName: courseName,
		Location:   strings.TrimSpace(locationStr),
		SeatNumber: strings.TrimSpace(seatStr),
		ExamType:   examTypeStr,
	}, nil
}

// normalizeExamValue converts non-breaking spaces (PeopleSoft emits &nbsp;
// as placeholders and label padding) to regular spaces and trims surrounding
// whitespace.
func normalizeExamValue(s string) string {
	return strings.TrimSpace(strings.ReplaceAll(s, "\u00a0", " "))
}

// splitCourseCode splits a PeopleSoft course field like
// "FAK 1001 Sample Mathematics Course" into the module code — the first two
// whitespace-separated tokens ("FAK 1001") — and the module name
// ("Sample Mathematics Course"). A single-token input yields (token, "").
func splitCourseCode(s string) (code, name string) {
	fields := strings.Fields(s)
	switch len(fields) {
	case 0:
		return "", ""
	case 1:
		return fields[0], ""
	default:
		return fields[0] + " " + fields[1], strings.Join(fields[2:], " ")
	}
}

// examDateRe matches date strings like "Monday Jan 15,2027".
var examDateRe = regexp.MustCompile(`^(Monday|Tuesday|Wednesday|Thursday|Friday|Saturday|Sunday)\s+(\w+)\s+(\d{1,2}),(\d{4})$`)

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
		minPart := strings.TrimSpace(parts[1])
		// AM/PM trails the minute value ("30 AM", "30AM").
		upperMinPart := strings.ToUpper(minPart)
		if strings.HasSuffix(upperMinPart, "AM") {
			ampm = "AM"
			minPart = strings.TrimSpace(minPart[:len(minPart)-2])
		} else if strings.HasSuffix(upperMinPart, "PM") {
			ampm = "PM"
			minPart = strings.TrimSpace(minPart[:len(minPart)-2])
		}
		min = parseInt(minPart)
	} else {
		// Format: "9:30 AM" or "9:30AM"
		parts := strings.Split(s, ":")
		if len(parts) != 2 {
			return 0, 0, fmt.Errorf("invalid time format: %q", s)
		}
		hour = parseInt(parts[0])
		minPart := strings.TrimSpace(parts[1])
		// AM/PM trails the minute value ("45 AM", "45AM").
		upperMinPart := strings.ToUpper(minPart)
		if strings.HasSuffix(upperMinPart, "AM") {
			ampm = "AM"
			minPart = strings.TrimSpace(minPart[:len(minPart)-2])
		} else if strings.HasSuffix(upperMinPart, "PM") {
			ampm = "PM"
			minPart = strings.TrimSpace(minPart[:len(minPart)-2])
		}
		min = parseInt(minPart)
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
