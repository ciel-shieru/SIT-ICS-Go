package peoplesoft

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/html"
)

var dayAbbrevRe = regexp.MustCompile(`^(Mo|Tu|We|Th|Fr|Sa|Su)\b`)
var timeRe = regexp.MustCompile(`(?i)^(\d{1,2}):(\d{2})(AM|PM)?$`)

// timetableRowPlaceholders lists cell values that mean the timetable row has no
// schedulable data. Keep this list explicit so additional PeopleSoft placeholders
// can be added without changing the row parser.
var timetableRowPlaceholders = []string{"TBA"}

// isTimetableRowPlaceholder reports whether a cell contains a recognized
// placeholder value. Matching is trimmed and case-insensitive.
func isTimetableRowPlaceholder(value string) bool {
	value = strings.TrimSpace(value)
	for _, placeholder := range timetableRowPlaceholders {
		if strings.EqualFold(value, placeholder) {
			return true
		}
	}
	return false
}

func ParseTimetableHTML(htmlContent string, year int, loc *time.Location) ([]Entry, error) {
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return nil, fmt.Errorf("parse HTML: %w", err)
	}

	var entries []Entry
	var find func(*html.Node) error
	find = func(n *html.Node) error {
		if n.Type == html.ElementNode && n.Data == "table" {
			rows := getRows(n)
			if len(rows) > 0 {
				headerCells := getCells(rows[0])
				if len(headerCells) >= 7 {
					firstCellText := getTextContent(headerCells[0])
					if strings.Contains(firstCellText, "Class Nbr") {
						tableEntries, err := parseMeetingTable(n, loc)
						if err != nil {
							return fmt.Errorf("parse timetable table: %w", err)
						}
						entries = append(entries, tableEntries...)
					}
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if err := find(c); err != nil {
				return err
			}
		}
		return nil
	}
	if err := find(doc); err != nil {
		return nil, err
	}

	return entries, nil
}

func getRows(table *html.Node) []*html.Node {
	var rows []*html.Node
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "tr" {
			rows = append(rows, n)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(table)
	return rows
}

func parseMeetingTable(table *html.Node, loc *time.Location) ([]Entry, error) {
	var entries []Entry

	var rows []*html.Node
	for c := table.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == "tbody" {
			for rc := c.FirstChild; rc != nil; rc = rc.NextSibling {
				if rc.Type == html.ElementNode && rc.Data == "tr" {
					rows = append(rows, rc)
				}
			}
		}
		if c.Type == html.ElementNode && c.Data == "tr" {
			rows = append(rows, c)
		}
	}

	if len(rows) == 0 {
		return nil, nil
	}

	var currentCourseCode, currentClassName string
	var currentSection, currentComp string
	for rowIndex, row := range rows {
		cells := getCells(row)
		if len(cells) < 7 {
			if currentCourseCode != "" && len(cells) > 0 {
				return nil, fmt.Errorf("row %d has %d cells; expected at least 7", rowIndex+1, len(cells))
			}
			continue
		}

		firstCellText := getTextContent(cells[0])
		if strings.Contains(firstCellText, "Class Nbr") {
			findCourseHeader(row, &currentCourseCode, &currentClassName)
			if currentCourseCode == "" {
				return nil, fmt.Errorf("row %d has a timetable header but no recognizable course header", rowIndex+1)
			}
			continue
		}

		if currentCourseCode == "" {
			continue
		}

		// Placeholder values make the entire class-meeting row non-schedulable.
		// Check the data cells before updating carried-forward section/component
		// values or trying strict date/schedule parsing.
		placeholderFound := false
		for _, cell := range cells {
			if isTimetableRowPlaceholder(getTextContent(cell)) {
				placeholderFound = true
				break
			}
		}
		if placeholderFound {
			continue
		}

		section := getTextContent(cells[1])
		comp := getTextContent(cells[2])
		if section == "" {
			section = currentSection
		}
		if comp == "" {
			comp = currentComp
		}
		if comp == "" {
			continue
		}
		currentSection = section
		currentComp = comp
		sched := getTextContent(cells[3])
		location := getTextContent(cells[4])
		_ = getTextContent(cells[5])
		dateText := getTextContent(cells[6])

		meetingDate, err := parseMeetingDate(dateText, loc)
		if err != nil {
			return nil, fmt.Errorf("row %d has invalid meeting date %q: %w", rowIndex+1, dateText, err)
		}

		dayName, timeRange, err := parseSchedule(sched)
		if err != nil {
			return nil, fmt.Errorf("row %d has invalid schedule %q: %w", rowIndex+1, sched, err)
		}

		startTime, endTime, err := parseTimeRange(timeRange)
		if err != nil {
			return nil, fmt.Errorf("row %d has invalid time range %q: %w", rowIndex+1, timeRange, err)
		}

		entryDay, err := computeEntryDate(meetingDate, dayName)
		if err != nil {
			return nil, fmt.Errorf("row %d has invalid weekday %q: %w", rowIndex+1, dayName, err)
		}

		entries = append(entries, Entry{
			CourseCode: currentCourseCode,
			ClassName:  currentClassName,
			Section:    section,
			Type:       comp,
			Day:        entryDay,
			StartTime:  startTime,
			EndTime:    endTime,
			Location:   location,
		})
	}

	return entries, nil
}

func findCourseHeader(row *html.Node, courseCode, className *string) {
	var courseGroup *html.Node
	for p := row.Parent; p != nil; p = p.Parent {
		if p.Type == html.ElementNode && p.Data == "div" {
			for _, a := range p.Attr {
				if a.Key == "id" && strings.HasPrefix(a.Val, "win0divDERIVED_REGFRM1_DESCR20$") {
					courseGroup = p
					break
				}
			}
			if courseGroup != nil {
				break
			}
		}
	}
	if courseGroup == nil {
		return
	}

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if *courseCode != "" {
			return
		}
		if n.Type == html.ElementNode && n.Data == "td" {
			hasClass := false
			for _, a := range n.Attr {
				if a.Key == "class" && strings.Contains(a.Val, "PAGROUPDIVIDER") {
					hasClass = true
					break
				}
			}
			if !hasClass {
				return
			}
			text := getTextContent(n)
			idx := strings.Index(text, " - ")
			if idx > 0 {
				*courseCode = strings.TrimSpace(text[:idx])
				*className = strings.TrimSpace(text[idx+3:])
			} else {
				*courseCode = strings.TrimSpace(text)
			}
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(courseGroup)
}

func parseMeetingDate(s string, loc *time.Location) (time.Time, error) {
	parts := strings.SplitN(s, " - ", 2)
	if len(parts) < 1 {
		return time.Time{}, fmt.Errorf("empty date: %s", s)
	}
	datePart := strings.TrimSpace(parts[0])
	return parseDate(datePart, loc)
}

func parseDate(s string, loc *time.Location) (time.Time, error) {
	parts := strings.Split(s, "/")
	if len(parts) != 3 {
		return time.Time{}, fmt.Errorf("invalid date format: %s", s)
	}
	day, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid day in date %q: %w", s, err)
	}
	month, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid month in date %q: %w", s, err)
	}
	year, err := strconv.Atoi(strings.TrimSpace(parts[2]))
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid year in date %q: %w", s, err)
	}
	if year < 1 || month < 1 || month > 12 || day < 1 || day > 31 {
		return time.Time{}, fmt.Errorf("date out of range: %s", s)
	}
	if loc == nil {
		loc = time.Local
	}
	parsed := time.Date(year, time.Month(month), day, 0, 0, 0, 0, loc)
	if parsed.Year() != year || int(parsed.Month()) != month || parsed.Day() != day {
		return time.Time{}, fmt.Errorf("invalid calendar date: %s", s)
	}
	return parsed, nil
}

func parseSchedule(s string) (dayName string, timeRange string, err error) {
	matches := dayAbbrevRe.FindStringSubmatch(s)
	if len(matches) < 2 {
		return "", "", fmt.Errorf("invalid day in schedule: %s", s)
	}
	dayName = matches[1]
	rest := strings.TrimSpace(s[len(matches[0]):])
	timeRange = rest
	return dayName, timeRange, nil
}

func computeEntryDate(meetingDate time.Time, dayAbbr string) (string, error) {
	if dayToWeekday(dayAbbr) == time.Monday && dayAbbr != "Mo" && dayAbbr != "Mon" {
		return "", fmt.Errorf("unknown weekday abbreviation: %s", dayAbbr)
	}
	targetWeekday := dayToWeekday(dayAbbr)
	currentWeekday := meetingDate.Weekday()
	daysDiff := int(targetWeekday) - int(currentWeekday)
	if daysDiff < 0 {
		daysDiff += 7
	}
	entryDate := meetingDate.AddDate(0, 0, daysDiff)
	return entryDate.Format("02/01/2006"), nil
}

func dayToWeekday(abbr string) time.Weekday {
	switch abbr {
	case "Mo", "Mon":
		return time.Monday
	case "Tu", "Tue":
		return time.Tuesday
	case "We", "Wed":
		return time.Wednesday
	case "Th", "Thu":
		return time.Thursday
	case "Fr", "Fri":
		return time.Friday
	case "Sa", "Sat":
		return time.Saturday
	case "Su", "Sun":
		return time.Sunday
	}
	return time.Monday
}

func parseTimeRange(s string) (string, string, error) {
	parts := strings.SplitN(s, "-", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid time range: %s", s)
	}
	start, err := parseTime(strings.TrimSpace(parts[0]))
	if err != nil {
		return "", "", err
	}
	end, err := parseTime(strings.TrimSpace(parts[1]))
	if err != nil {
		return "", "", err
	}
	return start, end, nil
}

func parseTime(s string) (string, error) {
	s = strings.TrimSpace(s)
	matches := timeRe.FindStringSubmatch(s)
	if len(matches) != 4 {
		return "", fmt.Errorf("invalid time: %s", s)
	}
	hour, err := strconv.Atoi(matches[1])
	if err != nil {
		return "", fmt.Errorf("invalid hour in time %q: %w", s, err)
	}
	minute, err := strconv.Atoi(matches[2])
	if err != nil {
		return "", fmt.Errorf("invalid minute in time %q: %w", s, err)
	}
	period := strings.ToUpper(matches[3])
	if minute > 59 {
		return "", fmt.Errorf("minute out of range in time: %s", s)
	}
	if period != "" {
		if hour < 1 || hour > 12 {
			return "", fmt.Errorf("hour out of range in 12-hour time: %s", s)
		}
		if period == "PM" && hour != 12 {
			hour += 12
		} else if period == "AM" && hour == 12 {
			hour = 0
		}
	} else if hour > 23 {
		return "", fmt.Errorf("hour out of range in 24-hour time: %s", s)
	}
	return fmt.Sprintf("%02d:%02d", hour, minute), nil
}

func getCells(tr *html.Node) []*html.Node {
	var cells []*html.Node
	for c := tr.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && (c.Data == "td" || c.Data == "th") {
			cells = append(cells, c)
		}
	}
	return cells
}

func getTextContent(n *html.Node) string {
	var sb strings.Builder
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.TextNode {
			sb.WriteString(node.Data)
		} else if node.Type == html.ElementNode && node.Data == "br" {
			sb.WriteString("\n")
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return strings.TrimSpace(sb.String())
}

func ExtractYear(s string) int {
	parts := strings.Split(s, "/")
	if len(parts) != 3 {
		return time.Now().Year()
	}
	var year int
	fmt.Sscanf(parts[2], "%d", &year)
	if year == 0 {
		return time.Now().Year()
	}
	return year
}
