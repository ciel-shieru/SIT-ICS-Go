package peoplesoft

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/html"
)

var dayDateRe = regexp.MustCompile(`^(Monday|Tuesday|Wednesday|Thursday|Friday|Saturday|Sunday)\s*(\d+)`)

func ParseTimetableHTML(htmlContent string, year int) ([]Entry, error) {
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return nil, fmt.Errorf("parse HTML: %w", err)
	}

	var scheduleTable *html.Node
	var visit func(*html.Node)
	visit = func(n *html.Node) {
		if scheduleTable != nil {
			return
		}
		if n.Type == html.ElementNode && n.Data == "table" {
			for _, attr := range n.Attr {
				if attr.Key == "id" && attr.Val == "WEEKLY_SCHED_HTMLAREA" {
					scheduleTable = n
					return
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			visit(c)
		}
	}
	visit(doc)

	if scheduleTable == nil {
		return nil, nil
	}

	var rows []*html.Node
	for c := scheduleTable.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == "tbody" {
			for rc := c.FirstChild; rc != nil; rc = rc.NextSibling {
				if rc.Type == html.ElementNode && rc.Data == "tr" {
					rows = append(rows, rc)
				}
			}
		}
	}

	if len(rows) < 2 {
		return nil, nil
	}

	headerCells := getCells(rows[0])
	type dayInfo struct {
		day   string
		dateS string
	}
	var days []dayInfo
	for i := 1; i < len(headerCells); i++ {
		text := getTextContent(headerCells[i])
		matches := dayDateRe.FindStringSubmatch(text)
		if len(matches) >= 3 {
			days = append(days, dayInfo{day: matches[1], dateS: matches[2] + " " + extractMonth(text)})
		}
	}

	if len(days) == 0 {
		return nil, nil
	}

	numRows := len(rows)
	numCols := len(headerCells)
	grid := make([][]bool, numRows)
	visited := make([][]bool, numRows)
	for i := 0; i < numRows; i++ {
		grid[i] = make([]bool, numCols)
		visited[i] = make([]bool, numCols)
	}

	var entries []Entry
	for r := 1; r < numRows; r++ {
		cells := getCells(rows[r])
		if len(cells) < 2 {
			continue
		}

		for c := 1; c < len(cells); c++ {
			if visited[r][c] {
				continue
			}

			cellText := getTextContent(cells[c])
			if cellText == "" {
				visited[r][c] = true
				continue
			}

			rowspan := 1
			for _, attr := range cells[c].Attr {
				if attr.Key == "rowspan" {
					if v, err := strconv.Atoi(attr.Val); err == nil && v > 0 {
						rowspan = v
					}
				}
			}

			for i := 0; i < rowspan; i++ {
				if r+i < numRows {
					visited[r+i][c] = true
					grid[r+i][c] = true
				}
			}

			parts := strings.Split(cellText, "\n")
			if len(parts) < 4 {
				continue
			}

			courseCode := strings.TrimSpace(parts[0])
			section := ""
			if strings.Contains(courseCode, " - ") {
				idx := strings.Index(courseCode, " - ")
				section = strings.TrimSpace(courseCode[idx+3:])
				courseCode = strings.TrimSpace(courseCode[:idx])
			}
			classType := strings.TrimSpace(parts[1])
			timeRange := strings.TrimSpace(parts[2])
			location := strings.TrimSpace(parts[3])

			startTime, endTime, err := parseTimeRange(timeRange)
			if err != nil {
				continue
			}

			dayIdx := c - 1
			if dayIdx < 0 || dayIdx >= len(days) {
				continue
			}

			weekStartDay, _ := parseDayDate(days[0].dateS, year)
			entryDate := weekStartDay.AddDate(0, 0, dayIdx)
			dateStr := entryDate.Format("02/01/2006")

			entries = append(entries, Entry{
				CourseCode: courseCode,
				Section:    section,
				Type:       classType,
				Day:        dateStr,
				StartTime:  startTime,
				EndTime:    endTime,
				Location:   location,
			})
		}
	}

	return entries, nil
}

func extractMonth(s string) string {
	months := []string{"Jan", "Feb", "Mar", "Apr", "May", "Jun",
		"Jul", "Aug", "Sep", "Oct", "Nov", "Dec"}
	for _, m := range months {
		if strings.Contains(s, m) {
			return m
		}
	}
	return ""
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

func parseDayDate(s string, year int) (time.Time, error) {
	parts := strings.Split(s, " ")
	if len(parts) < 2 {
		return time.Time{}, fmt.Errorf("invalid date: %s", s)
	}
	monthMap := map[string]int{
		"Jan": 1, "Feb": 2, "Mar": 3, "Apr": 4, "May": 5, "Jun": 6,
		"Jul": 7, "Aug": 8, "Sep": 9, "Oct": 10, "Nov": 11, "Dec": 12,
	}
	month, ok := monthMap[parts[1]]
	if !ok {
		return time.Time{}, fmt.Errorf("unknown month: %s", parts[1])
	}
	var day int
	fmt.Sscanf(parts[0], "%d", &day)
	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.Local), nil
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
