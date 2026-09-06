package peoplesoft

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"golang.org/x/net/html"
)

var dayAbbrevRe = regexp.MustCompile(`^(Mo|Tu|We|Th|Fr|Sa|Su)\b`)

func ParseTimetableHTML(htmlContent string, year int) ([]Entry, error) {
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return nil, fmt.Errorf("parse HTML: %w", err)
	}

	var entries []Entry
	var find func(*html.Node)
	find = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "table" {
			rows := getRows(n)
			if len(rows) > 0 {
				headerCells := getCells(rows[0])
				if len(headerCells) >= 7 {
					firstCellText := getTextContent(headerCells[0])
					if strings.Contains(firstCellText, "Class Nbr") {
						entries = append(entries, parseMeetingTable(n)...)
					}
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			find(c)
		}
	}
	find(doc)

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

func parseMeetingTable(table *html.Node) []Entry {
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
		return nil
	}

	var currentCourseCode, currentClassName string
	var currentSection, currentComp string
	for _, row := range rows {
		cells := getCells(row)
		if len(cells) < 7 {
			continue
		}

		firstCellText := getTextContent(cells[0])
		if strings.Contains(firstCellText, "Class Nbr") {
			findCourseHeader(row, &currentCourseCode, &currentClassName)
			continue
		}

		if currentCourseCode == "" {
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
		loc := getTextContent(cells[4])
		_ = getTextContent(cells[5])
		dateText := getTextContent(cells[6])

		meetingDate, err := parseMeetingDate(dateText)
		if err != nil {
			continue
		}

		dayName, timeRange, err := parseSchedule(sched)
		if err != nil {
			continue
		}

		startTime, endTime, err := parseTimeRange(timeRange)
		if err != nil {
			continue
		}

		entryDay, err := computeEntryDay(meetingDate, dayName)
		if err != nil {
			continue
		}

		entries = append(entries, Entry{
			CourseCode: currentCourseCode,
			ClassName:  currentClassName,
			Section:    section,
			Type:       comp,
			Day:        entryDay,
			StartTime:  startTime,
			EndTime:    endTime,
			Location:   loc,
		})
	}

	return entries
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
				if a.Key == "class" && a.Val == "PAGROUPDIVIDER" {
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

func parseMeetingDate(s string) (time.Time, error) {
	parts := strings.SplitN(s, " - ", 2)
	if len(parts) < 1 {
		return time.Time{}, fmt.Errorf("empty date: %s", s)
	}
	datePart := strings.TrimSpace(parts[0])
	return parseDate(datePart)
}

func parseDate(s string) (time.Time, error) {
	parts := strings.Split(s, "/")
	if len(parts) != 3 {
		return time.Time{}, fmt.Errorf("invalid date format: %s", s)
	}
	var day, month, year int
	fmt.Sscanf(parts[0], "%d", &day)
	fmt.Sscanf(parts[1], "%d", &month)
	fmt.Sscanf(parts[2], "%d", &year)
	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.Local), nil
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

func computeEntryDay(meetingDate time.Time, dayAbbr string) (string, error) {
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
	var hour, minute int
	var period string
	n, err := fmt.Sscanf(s, "%d:%d%s", &hour, &minute, &period)
	if n < 2 || err != nil {
		return "", fmt.Errorf("invalid time: %s", s)
	}
	period = strings.ToUpper(period)
	if period == "PM" {
		if hour != 12 {
			hour += 12
		}
	} else if period == "AM" {
		if hour == 12 {
			hour = 0
		}
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
