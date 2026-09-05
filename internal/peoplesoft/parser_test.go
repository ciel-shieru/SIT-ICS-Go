package peoplesoft

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"golang.org/x/net/html"
)

func TestParseTimetableHTML(t *testing.T) {
	htmlContent := `<!DOCTYPE html>
<html>
<body>
<div id='win0divSTDNT_ENRL_SSV2$0'>
<table>
<tr><td>
<div id='win0divDERIVED_REGFRM1_DESCR20$0'>
<table>
<tr><td class='PAGROUPDIVIDER' align='left'>DEF 0001 - Computer Organization and Architecture</td></tr>
<tr><td>
<div id='win0divCLASS_MTG_VW$0'>
<table cellspacing='0' class='PSLEVEL3GRIDWBO' id='CLASS_MTG_VW$scroll$0'>
<tr><td>
<table border='0' cellpadding='2' cellspacing='0' cols='7' width='100%' class='PSLEVEL3GRID'>
<tr>
<th scope='col' abbr='Class Nbr' width='29' align='CENTER' class='PSLEVEL3GRIDCOLUMNHDR PSGRIDFIRSTCOLUMN'>Class Nbr</th>
<th scope='col' abbr='Section' width='26' align='CENTER' class='PSLEVEL3GRIDCOLUMNHDR'>Section</th>
<th scope='col' abbr='Component' width='51' align='left' class='PSLEVEL3GRIDCOLUMNHDR'>Component</th>
<th scope='col' abbr='Days &amp; Times' width='90' align='left' class='PSLEVEL3GRIDCOLUMNHDR'>Days &amp; Times</th>
<th scope='col' abbr='Room' width='75' align='left' class='PSLEVEL3GRIDCOLUMNHDR'>Room</th>
<th scope='col' abbr='Instructor' width='87' align='left' class='PSLEVEL3GRIDCOLUMNHDR'>Instructor</th>
<th scope='col' abbr='Start/End Date' width='80' align='left' class='PSLEVEL3GRIDCOLUMNHDR'>Start/End Date</th>
</tr>
<tr id='trCLASS_MTG_VW$0_row1' valign='center'>
<td><DIV id='win0divDERIVED_CLS_DTL_CLASS_NBR$0'><span class='PSEDITBOX_DISPONLY' id='DERIVED_CLS_DTL_CLASS_NBR$0'>1160</span></DIV></td>
<td class='PSLEVEL2GRIDODDROW'><DIV id='win0divMTG_SECTION$0'><span id='MTG_SECTION$span$0' class='PSHYPERLINK'><a name='MTG_SECTION$0' id='MTG_SECTION$0'>P1</a></span></DIV></td>
<td class='PSLEVEL2GRIDODDROW'><DIV id='win0divMTG_COMP$0'><span class='PSEDITBOX_DISPONLY' id='MTG_COMP$0'>Laboratory</span></DIV></td>
<td class='PSLEVEL2GRIDODDROW'><DIV id='win0divMTG_SCHED$0'><span class='PSEDITBOX_DISPONLY' id='MTG_SCHED$0'>Th 9:00AM - 11:00AM</span></DIV></td>
<td class='PSLEVEL2GRIDODDROW'><DIV id='win0divMTG_LOC$0'><span class='PSEDITBOX_DISPONLY' id='MTG_LOC$0'>W1-06-18</span></DIV></td>
<td class='PSLEVEL2GRIDODDROW'><DIV id='win0divDERIVED_CLS_DTL_SSR_INSTR_LONG$0'><span class='PSLONGEDITBOX' id='DERIVED_CLS_DTL_SSR_INSTR_LONG$0'>JOEL ALIGAEN GONZALES</span></DIV></td>
<td class='PSLEVEL2GRIDODDROW'><DIV id='win0divMTG_DATES$0'><span class='PSEDITBOX_DISPONLY' id='MTG_DATES$0'>17/09/2026 - 17/09/2026</span></DIV></td>
</tr>
<tr id='trCLASS_MTG_VW$0_row2' valign='center'>
<td><DIV id='win0divDERIVED_CLS_DTL_CLASS_NBR$1'><span class='PSEDITBOX_DISPONLY' id='DERIVED_CLS_DTL_CLASS_NBR$1'>&nbsp;</span></DIV></td>
<td class='PSLEVEL2GRIDEVENROW'><DIV id='win0divMTG_SECTION$1'>&nbsp;</DIV></td>
<td class='PSLEVEL2GRIDEVENROW'><DIV id='win0divMTG_COMP$1'><span class='PSEDITBOX_DISPONLY' id='MTG_COMP$1'>&nbsp;</span></DIV></td>
<td class='PSLEVEL2GRIDEVENROW'><DIV id='win0divMTG_SCHED$1'><span class='PSEDITBOX_DISPONLY' id='MTG_SCHED$1'>Th 9:00AM - 11:00AM</span></DIV></td>
<td class='PSLEVEL2GRIDEVENROW'><DIV id='win0divMTG_LOC$1'><span class='PSEDITBOX_DISPONLY' id='MTG_LOC$1'>W1-06-18</span></DIV></td>
<td class='PSLEVEL2GRIDEVENROW'><DIV id='win0divDERIVED_CLS_DTL_SSR_INSTR_LONG$1'><span class='PSLONGEDITBOX' id='DERIVED_CLS_DTL_SSR_INSTR_LONG$1'>JOEL ALIGAEN GONZALES</span></DIV></td>
<td class='PSLEVEL2GRIDEVENROW'><DIV id='win0divMTG_DATES$1'><span class='PSEDITBOX_DISPONLY' id='MTG_DATES$1'>24/09/2026 - 24/09/2026</span></DIV></td>
</tr>
<tr id='trCLASS_MTG_VW$0_row3' valign='center'>
<td><DIV id='win0divDERIVED_CLS_DTL_CLASS_NBR$2'><span class='PSEDITBOX_DISPONLY' id='DERIVED_CLS_DTL_CLASS_NBR$2'>1163</span></DIV></td>
<td class='PSLEVEL2GRIDODDROW'><DIV id='win0divMTG_SECTION$2'><span id='MTG_SECTION$span$2' class='PSHYPERLINK'><a name='MTG_SECTION$2' id='MTG_SECTION$2'>ALL</a></span></DIV></td>
<td class='PSLEVEL2GRIDODDROW'><DIV id='win0divMTG_COMP$2'><span class='PSEDITBOX_DISPONLY' id='MTG_COMP$2'>Lecture</span></DIV></td>
<td class='PSLEVEL2GRIDODDROW'><DIV id='win0divMTG_SCHED$2'><span class='PSEDITBOX_DISPONLY' id='MTG_SCHED$2'>Fr 9:00AM - 11:00AM</span></DIV></td>
<td class='PSLEVEL2GRIDODDROW'><DIV id='win0divMTG_LOC$2'><span class='PSEDITBOX_DISPONLY' id='MTG_LOC$2'>Online</span></DIV></td>
<td class='PSLEVEL2GRIDODDROW'><DIV id='win0divDERIVED_CLS_DTL_SSR_INSTR_LONG$2'><span class='PSLONGEDITBOX' id='DERIVED_CLS_DTL_SSR_INSTR_LONG$2'>WONG KAI JUAN STEVEN</span></DIV></td>
<td class='PSLEVEL2GRIDODDROW'><DIV id='win0divMTG_DATES$2'><span class='PSEDITBOX_DISPONLY' id='MTG_DATES$2'>18/09/2026 - 18/09/2026</span></DIV></td>
</tr>
</table>
</div>
</div>
</td></tr>
</table>
</div>
<div id='win0divDERIVED_REGFRM1_DESCR20$1'>
<table>
<tr><td class='PAGROUPDIVIDER' align='left'>ABC 0002 - Introduction to Computer Systems</td></tr>
<tr><td>
<div id='win0divCLASS_MTG_VW$1'>
<table cellspacing='0' class='PSLEVEL3GRIDWBO' id='CLASS_MTG_VW$scroll$1'>
<tr><td>
<table border='0' cellpadding='2' cellspacing='0' cols='7' width='100%' class='PSLEVEL3GRID'>
<tr>
<th scope='col' abbr='Class Nbr' width='29' align='CENTER' class='PSLEVEL3GRIDCOLUMNHDR PSGRIDFIRSTCOLUMN'>Class Nbr</th>
<th scope='col' abbr='Section' width='26' align='CENTER' class='PSLEVEL3GRIDCOLUMNHDR'>Section</th>
<th scope='col' abbr='Component' width='51' align='left' class='PSLEVEL3GRIDCOLUMNHDR'>Component</th>
<th scope='col' abbr='Days &amp; Times' width='90' align='left' class='PSLEVEL3GRIDCOLUMNHDR'>Days &amp; Times</th>
<th scope='col' abbr='Room' width='75' align='left' class='PSLEVEL3GRIDCOLUMNHDR'>Room</th>
<th scope='col' abbr='Instructor' width='87' align='left' class='PSLEVEL3GRIDCOLUMNHDR'>Instructor</th>
<th scope='col' abbr='Start/End Date' width='80' align='left' class='PSLEVEL3GRIDCOLUMNHDR'>Start/End Date</th>
</tr>
<tr id='trCLASS_MTG_VW$1_row1' valign='center'>
<td><DIV id='win0divDERIVED_CLS_DTL_CLASS_NBR$3'><span class='PSEDITBOX_DISPONLY' id='DERIVED_CLS_DTL_CLASS_NBR$3'>2812</span></DIV></td>
<td class='PSLEVEL2GRIDODDROW'><DIV id='win0divMTG_SECTION$3'><span id='MTG_SECTION$span$3' class='PSHYPERLINK'><a name='MTG_SECTION$3' id='MTG_SECTION$3'>ALL</a></span></DIV></td>
<td class='PSLEVEL2GRIDODDROW'><DIV id='win0divMTG_COMP$3'><span class='PSEDITBOX_DISPONLY' id='MTG_COMP$3'>Lecture</span></DIV></td>
<td class='PSLEVEL2GRIDODDROW'><DIV id='win0divMTG_SCHED$3'><span class='PSEDITBOX_DISPONLY' id='MTG_SCHED$3'>Mo 9:00AM - 11:00AM</span></DIV></td>
<td class='PSLEVEL2GRIDODDROW'><DIV id='win0divMTG_LOC$3'><span class='PSEDITBOX_DISPONLY' id='MTG_LOC$3'>Online</span></DIV></td>
<td class='PSLEVEL2GRIDODDROW'><DIV id='win0divDERIVED_CLS_DTL_SSR_INSTR_LONG$3'><span class='PSLONGEDITBOX' id='DERIVED_CLS_DTL_SSR_INSTR_LONG$3'>IAN VINCE MCLOUGHLIN</span></DIV></td>
<td class='PSLEVEL2GRIDODDROW'><DIV id='win0divMTG_DATES$3'><span class='PSEDITBOX_DISPONLY' id='MTG_DATES$3'>31/08/2026 - 31/08/2026</span></DIV></td>
</tr>
</table>
</div>
</div>
</td></tr>
</table>
</div>
<div id='win0divDERIVED_REGFRM1_DESCR20$2'>
<table>
<tr><td class='PAGROUPDIVIDER' align='left'>GHI 1111 - Digital Competency Essentials</td></tr>
<tr><td>
<div id='win0divCLASS_MTG_VW$2'>
<table cellspacing='0' class='PSLEVEL3GRIDWBO' id='CLASS_MTG_VW$scroll$2'>
<tr><td>
<table border='0' cellpadding='2' cellspacing='0' cols='7' width='100%' class='PSLEVEL3GRID'>
<tr>
<th scope='col' abbr='Class Nbr' width='29' align='CENTER' class='PSLEVEL3GRIDCOLUMNHDR PSGRIDFIRSTCOLUMN'>Class Nbr</th>
<th scope='col' abbr='Section' width='26' align='CENTER' class='PSLEVEL3GRIDCOLUMNHDR'>Section</th>
<th scope='col' abbr='Component' width='51' align='left' class='PSLEVEL3GRIDCOLUMNHDR'>Component</th>
<th scope='col' abbr='Days &amp; Times' width='90' align='left' class='PSLEVEL3GRIDCOLUMNHDR'>Days &amp; Times</th>
<th scope='col' abbr='Room' width='75' align='left' class='PSLEVEL3GRIDCOLUMNHDR'>Room</th>
<th scope='col' abbr='Instructor' width='87' align='left' class='PSLEVEL3GRIDCOLUMNHDR'>Instructor</th>
<th scope='col' abbr='Start/End Date' width='80' align='left' class='PSLEVEL3GRIDCOLUMNHDR'>Start/End Date</th>
</tr>
<tr id='trCLASS_MTG_VW$2_row1' valign='center'>
<td><DIV id='win0divDERIVED_CLS_DTL_CLASS_NBR$4'><span class='PSEDITBOX_DISPONLY' id='DERIVED_CLS_DTL_CLASS_NBR$4'>2816</span></DIV></td>
<td class='PSLEVEL2GRIDODDROW'><DIV id='win0divMTG_SECTION$4'><span id='MTG_SECTION$span$4' class='PSHYPERLINK'><a name='MTG_SECTION$4' id='MTG_SECTION$4'>ALL</a></span></DIV></td>
<td class='PSLEVEL2GRIDODDROW'><DIV id='win0divMTG_COMP$4'><span class='PSEDITBOX_DISPONLY' id='MTG_COMP$4'>Lecture</span></DIV></td>
<td class='PSLEVEL2GRIDODDROW'><DIV id='win0divMTG_SCHED$4'><span class='PSEDITBOX_DISPONLY' id='MTG_SCHED$4'>Mo 2:00PM - 4:00PM</span></DIV></td>
<td class='PSLEVEL2GRIDODDROW'><DIV id='win0divMTG_LOC$4'><span class='PSEDITBOX_DISPONLY' id='MTG_LOC$4'>Online</span></DIV></td>
<td class='PSLEVEL2GRIDODDROW'><DIV id='win0divDERIVED_CLS_DTL_SSR_INSTR_LONG$4'><span class='PSLONGEDITBOX' id='DERIVED_CLS_DTL_SSR_INSTR_LONG$4'>NISHA JAIN</span></DIV></td>
<td class='PSLEVEL2GRIDODDROW'><DIV id='win0divMTG_DATES$4'><span class='PSEDITBOX_DISPONLY' id='MTG_DATES$4'>31/08/2026 - 31/08/2026</span></DIV></td>
</tr>
</table>
</div>
</div>
</td></tr>
</table>
</div>
</td></tr>
</table>
</div>
</body>
</html>`

	entries, err := ParseTimetableHTML(htmlContent, 2026)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(entries) != 4 {
		t.Fatalf("expected 4 entries, got %d", len(entries))
	}

	type expectedEntry struct {
		courseCode  string
		className   string
		section     string
		classType   string
		day         string
		startTime   string
		endTime     string
		location    string
	}

	expected := []expectedEntry{
		{courseCode: "DEF 0001", className: "Computer Organization and Architecture", section: "P1", classType: "Laboratory", day: "17/09/2026", startTime: "09:00", endTime: "11:00", location: "W1-06-18"},
		{courseCode: "DEF 0001", className: "Computer Organization and Architecture", section: "ALL", classType: "Lecture", day: "18/09/2026", startTime: "09:00", endTime: "11:00", location: "Online"},
		{courseCode: "ABC 0002", className: "Introduction to Computer Systems", section: "ALL", classType: "Lecture", day: "31/08/2026", startTime: "09:00", endTime: "11:00", location: "Online"},
		{courseCode: "GHI 1111", className: "Digital Competency Essentials", section: "ALL", classType: "Lecture", day: "31/08/2026", startTime: "14:00", endTime: "16:00", location: "Online"},
	}

	for i, exp := range expected {
		e := entries[i]
		if e.CourseCode != exp.courseCode {
			t.Errorf("entry[%d] CourseCode: got %q, want %q", i, e.CourseCode, exp.courseCode)
		}
		if e.ClassName != exp.className {
			t.Errorf("entry[%d] ClassName: got %q, want %q", i, e.ClassName, exp.className)
		}
		if e.Section != exp.section {
			t.Errorf("entry[%d] Section: got %q, want %q", i, e.Section, exp.section)
		}
		if e.Type != exp.classType {
			t.Errorf("entry[%d] Type: got %q, want %q", i, e.Type, exp.classType)
		}
		if e.Day != exp.day {
			t.Errorf("entry[%d] Day: got %q, want %q", i, e.Day, exp.day)
		}
		if e.StartTime != exp.startTime {
			t.Errorf("entry[%d] StartTime: got %q, want %q", i, e.StartTime, exp.startTime)
		}
		if e.EndTime != exp.endTime {
			t.Errorf("entry[%d] EndTime: got %q, want %q", i, e.EndTime, exp.endTime)
		}
		if e.Location != exp.location {
			t.Errorf("entry[%d] Location: got %q, want %q", i, e.Location, exp.location)
		}
	}
}

func TestParseTimetableHTML_NoTable(t *testing.T) {
	htmlContent := `<html><body><p>No timetable here</p></body></html>`
	entries, err := ParseTimetableHTML(htmlContent, 2026)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if entries != nil {
		t.Errorf("expected nil entries for missing table, got %d entries", len(entries))
	}
}

func TestParseTimeRange(t *testing.T) {
	tests := []struct {
		input       string
		wantStart   string
		wantEnd     string
		wantErr     bool
	}{
		{"9:00AM - 11:00AM", "09:00", "11:00", false},
		{"11:00AM - 12:00PM", "11:00", "12:00", false},
		{"11:00AM - 1:00PM", "11:00", "13:00", false},
		{"2:00PM - 4:00PM", "14:00", "16:00", false},
		{"4:00PM - 6:00PM", "16:00", "18:00", false},
		{"12:00AM - 12:00PM", "00:00", "12:00", false},
		{"12:00PM - 12:00AM", "12:00", "00:00", false},
		{"invalid", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			start, end, err := parseTimeRange(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error for %q, got nil", tt.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error for %q: %v", tt.input, err)
			}
			if start != tt.wantStart {
				t.Errorf("start: got %q, want %q", start, tt.wantStart)
			}
			if end != tt.wantEnd {
				t.Errorf("end: got %q, want %q", end, tt.wantEnd)
			}
		})
	}
}

func TestParseTime(t *testing.T) {
	tests := []struct {
		input   string
		want    string
		wantErr bool
	}{
		{"9:00AM", "09:00", false},
		{"11:00AM", "11:00", false},
		{"12:00PM", "12:00", false},
		{"1:00PM", "13:00", false},
		{"2:00PM", "14:00", false},
		{"4:00PM", "16:00", false},
		{"6:00PM", "18:00", false},
		{"12:00AM", "00:00", false},
		{"12:00am", "00:00", false},
		{"invalid", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := parseTime(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error for %q, got nil", tt.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error for %q: %v", tt.input, err)
			}
			if got != tt.want {
				t.Errorf("parseTime(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseTimeRange_Empty(t *testing.T) {
	_, _, err := parseTimeRange("")
	if err == nil {
		t.Error("expected error for empty string, got nil")
	}
}

func TestParseTimeRange_InvalidFormat(t *testing.T) {
	_, _, err := parseTimeRange("9:00AM")
	if err == nil {
		t.Error("expected error for single time, got nil")
	}
}

func TestGetCells(t *testing.T) {
	trHTML := `<table><tr><th>Header</th><td>Data</th><th>Header2</th></tr></table>`
	doc, err := html.Parse(strings.NewReader(trHTML))
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}
	var tr *html.Node
	var visit func(*html.Node)
	visit = func(n *html.Node) {
		if tr != nil {
			return
		}
		if n.Type == html.ElementNode && n.Data == "tr" {
			tr = n
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			visit(c)
		}
	}
	visit(doc)
	cells := getCells(tr)
	if len(cells) != 3 {
		t.Errorf("expected 3 cells, got %d", len(cells))
	}
}

func TestExtractYear(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"26/09/2026", 2026},
		{"01/01/2025", 2025},
		{"15/12/2030", 2030},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := ExtractYear(tt.input)
			if got != tt.expected {
				t.Errorf("ExtractYear(%q) = %d, want %d", tt.input, got, tt.expected)
			}
		})
	}
}

func TestGetTextContent(t *testing.T) {
	htmlStr := `<span>Hello<br>World</span>`
	doc, err := html.Parse(strings.NewReader(htmlStr))
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}
	got := getTextContent(doc.FirstChild)
	if got != "Hello\nWorld" {
		t.Errorf("got %q, want %q", got, "Hello\nWorld")
	}
}

func TestParseSchedule(t *testing.T) {
	tests := []struct {
		input       string
		wantDay     string
		wantTime    string
		wantErr     bool
	}{
		{"Mo 9:00AM - 11:00AM", "Mo", "9:00AM - 11:00AM", false},
		{"Th 9:00AM - 11:00AM", "Th", "9:00AM - 11:00AM", false},
		{"Fr 2:00PM - 4:00PM", "Fr", "2:00PM - 4:00PM", false},
		{"We 6:00PM - 8:00PM", "We", "6:00PM - 8:00PM", false},
		{"invalid", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			day, timeRange, err := parseSchedule(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error for %q, got nil", tt.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error for %q: %v", tt.input, err)
			}
			if day != tt.wantDay {
				t.Errorf("day: got %q, want %q", day, tt.wantDay)
			}
			if timeRange != tt.wantTime {
				t.Errorf("timeRange: got %q, want %q", timeRange, tt.wantTime)
			}
		})
	}
}

func TestComputeEntryDay(t *testing.T) {
	tests := []struct {
		date      string
		dayAbbr   string
		wantDay   string
		wantErr   bool
	}{
		{"17/09/2026", "Thu", "17/09/2026", false},
		{"24/09/2026", "Th", "24/09/2026", false},
		{"18/09/2026", "Fr", "18/09/2026", false},
		{"31/08/2026", "Mo", "31/08/2026", false},
		{"31/08/2026", "Th", "03/09/2026", false},
	}

	for _, tt := range tests {
		t.Run(tt.date+"-"+tt.dayAbbr, func(t *testing.T) {
			date, err := parseDate(tt.date)
			if err != nil {
				t.Fatalf("failed to parse date: %v", err)
			}
			got, err := computeEntryDay(date, tt.dayAbbr)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.wantDay {
				t.Errorf("got %q, want %q", got, tt.wantDay)
			}
		})
	}
}

func TestParseMeetingDate(t *testing.T) {
	tests := []struct {
		input   string
		want    string
		wantErr bool
	}{
		{"17/09/2026 - 17/09/2026", "17/09/2026", false},
		{"01/10/2026 - 01/10/2026", "01/10/2026", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := parseMeetingDate(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error for %q, got nil", tt.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error for %q: %v", tt.input, err)
			}
			want := tt.want
			parts := strings.Split(want, "/")
			if len(parts) == 3 {
				var day, month int
				fmt.Sscanf(parts[0], "%d", &day)
				fmt.Sscanf(parts[1], "%d", &month)
				expected := time.Date(2026, time.Month(month), day, 0, 0, 0, 0, time.Local)
				if got.Year() != expected.Year() || got.Month() != expected.Month() || got.Day() != expected.Day() {
					t.Errorf("got %v, want %v", got, expected)
				}
			}
		})
	}
}
