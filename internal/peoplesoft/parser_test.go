package peoplesoft

import (
	"strings"
	"testing"

	"golang.org/x/net/html"
)

func TestParseTimetableHTML(t *testing.T) {
	htmlContent := `<!DOCTYPE html>
<html>
<body>
<div id="win0divPSPAGECONTAINER">
<table cellspacing='0' class='PSLEVEL1GRIDWBO' role='presentation' dir='ltr' cols='1' width='776' cellpadding='0'>
<tr><td class='PSLEVEL1GRIDLABEL' align='left'><div id='win0divSSR_DUMMY_RECGP$0'>Schedule</div></td></tr>
<tr><td>
<table dir='ltr' border='0' cellpadding='2' cellspacing='0' cols='1' width='100%' class='PSLEVEL1GRID' style='border-style:none'>
<tr id='trSSR_DUMMY_REC$0_row1' valign='center'>
<td align='left' width='702' height='18' class='PABACKGROUNDINVISIBLE PSGRIDFIRSTCOLUMN'>
<div id='win0divDERIVED_CLASS_S_HTMLAREA$0'>
<div>
<table cellspacing='0' cellpadding='2' width='100%' class='PSLEVEL3GRIDODDROW' id='WEEKLY_SCHED_HTMLAREA' summary='Weekly Schedule'>
<colgroup span='1' width='9%' align='center' valign='middle'>
<colgroup span='7' width='13%' align='center' valign='middle'>
<tr><th scope='col' align='center' class='PSLEVEL3GRIDODDROW'>Time</th>
<th scope='col' align='center' class='PSLEVEL3GRIDODDROW'>Monday<br>21 Sep</th>
<th scope='col' align='center' class='PSLEVEL3GRIDODDROW'>Tuesday<br>22 Sep</th>
<th scope='col' align='center' class='PSLEVEL3GRIDODDROW'>Wednesday<br>23 Sep</th>
<th scope='col' align='center' class='PSLEVEL3GRIDODDROW'>Thursday<br>24 Sep</th>
<th scope='col' align='center' class='PSLEVEL3GRIDODDROW'>Friday<br>25 Sep</th>
<th scope='col' align='center' class='PSLEVEL3GRIDODDROW'>Saturday<br>26 Sep</th>
<th scope='col' align='center' class='PSLEVEL3GRIDODDROW'>Sunday<br>27 Sep</th></tr>
<tr><td class='PSLEVEL3GRIDODDROW' rowspan='1' scope="row"><span class=''>8:00AM</span></td>
<td class='PSLEVEL3GRIDODDROW'>&nbsp;</td><td class='PSLEVEL3GRIDODDROW'>&nbsp;</td><td class='PSLEVEL3GRIDODDROW'>&nbsp;</td><td class='PSLEVEL3GRIDODDROW'>&nbsp;</td><td class='PSLEVEL3GRIDODDROW'>&nbsp;</td><td class='PSLEVEL3GRIDODDROW'>&nbsp;</td><td class='PSLEVEL3GRIDODDROW'>&nbsp;</td></tr>
<tr><td class='PSLEVEL3GRIDODDROW' rowspan='1' scope="row"><span class=''>9:00AM</span></td>
<td class='PSLEVEL3GRIDODDROW' rowspan='2' STYLE="color:rgb(0,0,0);background-color:rgb(182,209,146);text-align: center;"><span class='' STYLE="color:rgb(0,0,0);background-color:rgb(182,209,146);">ABC 0002 - ALL<br>Lecture<br>9:00AM - 11:00AM<br>Online</span></td>
<td class='PSLEVEL3GRIDODDROW'>&nbsp;</td><td class='PSLEVEL3GRIDODDROW'>&nbsp;</td><td class='PSLEVEL3GRIDODDROW'>&nbsp;</td>
<td class='PSLEVEL3GRIDODDROW' rowspan='2' STYLE="color:rgb(0,0,0);background-color:rgb(182,209,146);text-align: center;"><span class='' STYLE="color:rgb(0,0,0);background-color:rgb(182,209,146);">DEF 0001 - ALL<br>Lecture<br>9:00AM - 11:00AM<br>Online</span></td>
<td class='PSLEVEL3GRIDODDROW'>&nbsp;</td><td class='PSLEVEL3GRIDODDROW'>&nbsp;</td></tr>
<tr><td class='PSLEVEL3GRIDODDROW' rowspan='1' scope="row"><span class=''>10:00AM</span></td>
<td class='PSLEVEL3GRIDODDROW'>&nbsp;</td><td class='PSLEVEL3GRIDODDROW'>&nbsp;</td><td class='PSLEVEL3GRIDODDROW'>&nbsp;</td><td class='PSLEVEL3GRIDODDROW'>&nbsp;</td><td class='PSLEVEL3GRIDODDROW'>&nbsp;</td></tr>
<tr><td class='PSLEVEL3GRIDODDROW' rowspan='1' scope="row"><span class=''>11:00AM</span></td>
<td class='PSLEVEL3GRIDODDROW' rowspan='1' STYLE="color:rgb(0,0,0);background-color:rgb(182,209,146);text-align: center;"><span class='' STYLE="color:rgb(0,0,0);background-color:rgb(182,209,146);">DEF 0001 - T2<br>Tutorial<br>11:00AM - 12:00PM<br>Online</span></td>
<td class='PSLEVEL3GRIDODDROW'>&nbsp;</td><td class='PSLEVEL3GRIDODDROW'>&nbsp;</td>
<td class='PSLEVEL3GRIDODDROW' rowspan='2' STYLE="color:rgb(0,0,0);background-color:rgb(182,209,146);text-align: center;"><span class='' STYLE="color:rgb(0,0,0);background-color:rgb(182,209,146);">ABC 0004 - T7<br>Tutorial<br>11:00AM - 1:00PM<br>W1-03-04-SR222</span></td>
<td class='PSLEVEL3GRIDODDROW'>&nbsp;</td><td class='PSLEVEL3GRIDODDROW'>&nbsp;</td><td class='PSLEVEL3GRIDODDROW'>&nbsp;</td></tr>
<tr><td class='PSLEVEL3GRIDODDROW' rowspan='1' scope="row"><span class=''>12:00PM</span></td>
<td class='PSLEVEL3GRIDODDROW'>&nbsp;</td><td class='PSLEVEL3GRIDODDROW'>&nbsp;</td><td class='PSLEVEL3GRIDODDROW'>&nbsp;</td><td class='PSLEVEL3GRIDODDROW'>&nbsp;</td><td class='PSLEVEL3GRIDODDROW'>&nbsp;</td><td class='PSLEVEL3GRIDODDROW'>&nbsp;</td></tr>
<tr><td class='PSLEVEL3GRIDODDROW' rowspan='1' scope="row"><span class=''>2:00PM</span></td>
<td class='PSLEVEL3GRIDODDROW' rowspan='2' STYLE="color:rgb(0,0,0);background-color:rgb(182,209,146);text-align: center;"><span class='' STYLE="color:rgb(0,0,0);background-color:rgb(182,209,146);">ABC 0003 - ALL<br>Lecture<br>2:00PM - 4:00PM<br>Online</span></td>
<td class='PSLEVEL3GRIDODDROW'>&nbsp;</td>
<td class='PSLEVEL3GRIDODDROW' rowspan='2' STYLE="color:rgb(0,0,0);background-color:rgb(182,209,146);text-align: center;"><span class='' STYLE="color:rgb(0,0,0);background-color:rgb(182,209,146);">DEF 1002A - IS26<br>Workshop<br>2:00PM - 4:00PM<br>Online</span></td>
<td class='PSLEVEL3GRIDODDROW'>&nbsp;</td>
<td class='PSLEVEL3GRIDODDROW' rowspan='2' STYLE="color:rgb(0,0,0);background-color:rgb(182,209,146);text-align: center;"><span class='' STYLE="color:rgb(0,0,0);background-color:rgb(182,209,146);">ABC 0003 - P6<br>Laboratory<br>2:00PM - 4:00PM<br>W1-05-07</span></td>
<td class='PSLEVEL3GRIDODDROW'>&nbsp;</td><td class='PSLEVEL3GRIDODDROW'>&nbsp;</td></tr>
<tr><td class='PSLEVEL3GRIDODDROW' rowspan='1' scope="row"><span class=''>3:00PM</span></td>
<td class='PSLEVEL3GRIDODDROW'>&nbsp;</td><td class='PSLEVEL3GRIDODDROW'>&nbsp;</td><td class='PSLEVEL3GRIDODDROW'>&nbsp;</td><td class='PSLEVEL3GRIDODDROW'>&nbsp;</td></tr>
<tr><td class='PSLEVEL3GRIDODDROW' rowspan='1' scope="row"><span class=''>4:00PM</span></td>
<td class='PSLEVEL3GRIDODDROW' rowspan='2' STYLE="color:rgb(0,0,0);background-color:rgb(182,209,146);text-align: center;"><span class='' STYLE="color:rgb(0,0,0);background-color:rgb(182,209,146);">ABC 0004 - ALL<br>Lecture<br>4:00PM - 6:00PM<br>Online</span></td>
<td class='PSLEVEL3GRIDODDROW'>&nbsp;</td><td class='PSLEVEL3GRIDODDROW'>&nbsp;</td><td class='PSLEVEL3GRIDODDROW'>&nbsp;</td>
<td class='PSLEVEL3GRIDODDROW' rowspan='2' STYLE="color:rgb(0,0,0);background-color:rgb(182,209,146);text-align: center;"><span class='' STYLE="color:rgb(0,0,0);background-color:rgb(182,209,146);">ABC 0002 - P6<br>Laboratory<br>4:00PM - 6:00PM<br>W1-05-05</span></td>
<td class='PSLEVEL3GRIDODDROW'>&nbsp;</td><td class='PSLEVEL3GRIDODDROW'>&nbsp;</td></tr>
<tr><td class='PSLEVEL3GRIDODDROW' rowspan='1' scope="row"><span class=''>5:00PM</span></td>
<td class='PSLEVEL3GRIDODDROW'>&nbsp;</td><td class='PSLEVEL3GRIDODDROW'>&nbsp;</td><td class='PSLEVEL3GRIDODDROW'>&nbsp;</td><td class='PSLEVEL3GRIDODDROW'>&nbsp;</td><td class='PSLEVEL3GRIDODDROW'>&nbsp;</td></tr>
</table>
</div>
</div>
</td></tr>
</table></td></tr>
</table>
</div>
</body>
</html>`

	entries, err := ParseTimetableHTML(htmlContent, 2026)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(entries) != 9 {
		t.Fatalf("expected 9 entries, got %d", len(entries))
	}

	expected := []struct {
		courseCode  string
		section     string
		classType   string
		day         string
		startTime   string
		endTime     string
		location    string
	}{
		{"ABC 0002", "ALL", "Lecture", "21/09/2026", "09:00", "11:00", "Online"},
		{"DEF 0001", "ALL", "Lecture", "25/09/2026", "09:00", "11:00", "Online"},
		{"DEF 0001", "T2", "Tutorial", "21/09/2026", "11:00", "12:00", "Online"},
		{"ABC 0004", "T7", "Tutorial", "24/09/2026", "11:00", "13:00", "W1-03-04-SR222"},
		{"ABC 0003", "ALL", "Lecture", "21/09/2026", "14:00", "16:00", "Online"},
		{"DEF 1002A", "IS26", "Workshop", "23/09/2026", "14:00", "16:00", "Online"},
		{"ABC 0003", "P6", "Laboratory", "25/09/2026", "14:00", "16:00", "W1-05-07"},
		{"ABC 0004", "ALL", "Lecture", "21/09/2026", "16:00", "18:00", "Online"},
		{"ABC 0002", "P6", "Laboratory", "25/09/2026", "16:00", "18:00", "W1-05-05"},
	}

	for i, exp := range expected {
		e := entries[i]
		if e.CourseCode != exp.courseCode {
			t.Errorf("entry[%d] CourseCode: got %q, want %q", i, e.CourseCode, exp.courseCode)
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

func TestParseDayDate(t *testing.T) {
	tests := []struct {
		input   string
		wantDay int
		wantMon string
		wantErr bool
	}{
		{"21 Sep", 21, "September", false},
		{"22 Sep", 22, "September", false},
		{"23 Sep", 23, "September", false},
		{"24 Sep", 24, "September", false},
		{"25 Sep", 25, "September", false},
		{"invalid", 0, "", true},
		{"31 Foo", 0, "", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := parseDayDate(tt.input, 2026)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error for %q, got nil", tt.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error for %q: %v", tt.input, err)
			}
			if got.Year() != 2026 {
				t.Errorf("year: got %d, want 2026", got.Year())
			}
			if got.Day() != tt.wantDay {
				t.Errorf("day: got %d, want %d", got.Day(), tt.wantDay)
			}
			if got.Month().String() != tt.wantMon {
				t.Errorf("month: got %s, want %s", got.Month().String(), tt.wantMon)
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
