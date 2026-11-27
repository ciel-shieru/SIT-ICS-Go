package peoplesoft

import (
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"
)

// icajaxExamsFragment is a synthetic PeopleSoft ICAJAX exam page response.
// The tag/id/class structure mirrors the production markup, while every
// value (module codes, module names, dates, locations, seats) is fake.
const icajaxExamsFragment = `<?xml version='1.0' encoding='utf-8' ?>
<PAGE id='blank'>
  <GENSCRIPT id='script'><![CDATA[window.icajaxTest = 1;]]></GENSCRIPT>
  <GENCSS id='css'><![CDATA[.ps_box { color: red; }]]></GENCSS>
  <GENJS id='script'><![CDATA[window.icajaxTest2 = 2;]]></GENJS>
  <FIELD id='win0divPSHIDDENFIELDS'><![CDATA[<input type='hidden' name='H$Key' value='0'/>]]></FIELD>
  <FIELD id='win0divPSPANELTABS' class='psc_hidden'><![CDATA[]]></FIELD>
  <FIELD id='divPAGECONTAINER_TGT' class='ps_pagecontainer'><![CDATA[
<DIV class='ps_pspagecontainer' id='win0divPSPAGECONTAINER'>
  <div class='ps_box-group psc_layout ps_apps_content' id='win0div$ICField1'>
    <div class='ps_box-scrollarea' id='win0div$ICField16$0'>
      <div class='ps_box-scrollarea-row' id='win0div$ICField16_row$0'>
        <div class='ps_box-group psc_column-3' id='win0divN_ES_XMTT_WRK_GROUP1$0'>
          <DIV class='ps_box-edit psc_disabled psc_has_value psc_strong' id='win0divN_ES_XMTT_WRK_DESCR100$0'>
            <DIV class='ps_box-label' id='win0divN_ES_XMTT_WRK_DESCR100lbl$0'><span class='ps-label'>&nbsp;</span></DIV>
            <span class='ps_box-value' id='N_ES_XMTT_WRK_DESCR100$0'>Monday Nov 30,2026</span>
          </DIV>
        </div>
        <div class='ps_box-group psc_column-3' id='win0divN_ES_XMTT_WRK_GROUP1$48$$0'><DIV class='ps_box-hr' id='win0div$ICField15$0'><hr class='ps-hr' /></DIV></div>
        <div class='ps_box-group psc_column-2' id='win0divN_ES_XMTT_WRK_$19$$0'>
          <div class='ps_box-group psc_layout psc_columnitem-3of10' id='win0divN_ES_XMTT_WRK_$51$$0'>
            <DIV id='win0divN_ES_XMTT_WRK_DESCR$0'><span id='N_ES_XMTT_WRK_DESCR$0'>&nbsp;</span></DIV>
            <DIV id='win0divN_ES_XMTT_WRK_DESCR100_2$0'><span id='N_ES_XMTT_WRK_DESCR100_2$0'>9.30 AM to 11.40 AM</span></DIV>
            <DIV id='win0divN_ES_XMTT_WRK_DESCR$69$$0'><span id='N_ES_XMTT_WRK_DESCR$69$$0'>&nbsp;</span></DIV>
            <DIV id='win0divN_ES_XMTT_WRK_DESCR$76$$0'><span id='N_ES_XMTT_WRK_DESCR$76$$0'>&nbsp;</span></DIV>
          </div>
          <div class='ps_box-group' id='win0divN_ES_XMTT_WRK_$50$$0'>
            <DIV id='win0divN_ES_XMTT_WRK_DESCR100_3$0'><span id='N_ES_XMTT_WRK_DESCR100_3$0'>FAK 1001 Sample Mathematics Course</span></DIV>
            <DIV id='win0divN_ES_XMTT_WRK_DESCR100_4$0'><span id='N_ES_XMTT_WRK_DESCR100_4$0'>Location :&nbsp;&nbsp;&nbsp;&nbsp; FAK BLDG Hall A</span></DIV>
            <DIV id='win0divN_ES_XMTT_WRK_DESCR100_5$0'><span id='N_ES_XMTT_WRK_DESCR100_5$0'>Seat Number :&nbsp;&nbsp; 001</span></DIV>
            <DIV id='win0divN_ES_XMTT_WRK_DESCR1$0'><span id='N_ES_XMTT_WRK_DESCR1$0'>Final Exam</span></DIV>
          </div>
        </div>
        <div class='ps_box-group psc_layout psc_columnitem-3of10' id='win0divN_ES_XMTT_WRK_$75$$0'>
          <DIV id='win0divN_ES_XMTT_WRK_N_SPEC_ARRGMT$0'><span id='N_ES_XMTT_WRK_N_SPEC_ARRGMT$0'>&nbsp;</span></DIV>
        </div>
        <div class='ps_box-group psc_column-3' id='win0divN_ES_XMTT_WRK_GROUP1$66$$0'><DIV class='ps_box-hr' id='win0div$ICField67$0'><hr class='ps-hr' /></DIV></div>
      </div>
      <div class='ps_box-scrollarea-row' id='win0div$ICField16_row$1'>
        <div class='ps_box-group psc_column-3' id='win0divN_ES_XMTT_WRK_GROUP1$1'>
          <DIV class='ps_box-edit psc_disabled psc_has_value psc_strong' id='win0divN_ES_XMTT_WRK_DESCR100$1'>
            <span class='ps_box-value' id='N_ES_XMTT_WRK_DESCR100$1'>Monday Dec 07,2026</span>
          </DIV>
        </div>
        <div class='ps_box-group psc_column-2' id='win0divN_ES_XMTT_WRK_$19$$1'>
          <div class='ps_box-group psc_layout psc_columnitem-3of10' id='win0divN_ES_XMTT_WRK_$51$$1'>
            <DIV id='win0divN_ES_XMTT_WRK_DESCR100_2$1'><span id='N_ES_XMTT_WRK_DESCR100_2$1'>9.30 AM to 11.40 AM</span></DIV>
          </div>
          <div class='ps_box-group' id='win0divN_ES_XMTT_WRK_$50$$1'>
            <DIV id='win0divN_ES_XMTT_WRK_DESCR100_3$1'><span id='N_ES_XMTT_WRK_DESCR100_3$1'>FAK 2002 Synthetic Statistics Module</span></DIV>
            <DIV id='win0divN_ES_XMTT_WRK_DESCR100_4$1'><span id='N_ES_XMTT_WRK_DESCR100_4$1'>Location : FAK HALL 2</span></DIV>
            <DIV id='win0divN_ES_XMTT_WRK_DESCR100_5$1'><span id='N_ES_XMTT_WRK_DESCR100_5$1'>&nbsp;</span></DIV>
            <DIV id='win0divN_ES_XMTT_WRK_DESCR1$1'><span id='N_ES_XMTT_WRK_DESCR1$1'>Final Exam</span></DIV>
          </div>
        </div>
        <div class='ps_box-group psc_layout psc_columnitem-3of10' id='win0divN_ES_XMTT_WRK_$75$$1'>
          <DIV id='win0divN_ES_XMTT_WRK_N_SPEC_ARRGMT$1'><span id='N_ES_XMTT_WRK_N_SPEC_ARRGMT$1'>&nbsp;</span></DIV>
        </div>
      </div>
    </div>
    <DIV class='ps_box-edit psc_disabled psc_wrappable psc_has_value psc_strong' id='win0divN_ES_XMTT_WRK_DESCR100_6'><span id='N_ES_XMTT_WRK_DESCR100_6'>&nbsp;</span></DIV>
  </div>
</DIV>
]]></FIELD>
  <FIELD id='PT_SIDE2'><![CDATA[<DIV class='ps_pspagecontainer_side2' id='win0divPSPAGECONTAINER_SIDE2'></DIV>]]></FIELD>
  <FIELD id='PT_FOOTER'><![CDATA[]]></FIELD>
  <SYSVAR id='sysvar'><![CDATA[window.sysvarTest = 3;]]></SYSVAR>
</PAGE>`

// plainExamsHTML is the same inner page HTML without the ICAJAX XML wrapper
// and CDATA sections (backward-compatibility path).
const plainExamsHTML = `<DIV class='ps_pspagecontainer' id='win0divPSPAGECONTAINER'>
  <div class='ps_box-group psc_layout ps_apps_content' id='win0div$ICField1'>
    <div class='ps_box-scrollarea' id='win0div$ICField16$0'>
      <div class='ps_box-scrollarea-row' id='win0div$ICField16_row$0'>
        <DIV id='win0divN_ES_XMTT_WRK_DESCR100$0'><span id='N_ES_XMTT_WRK_DESCR100$0'>Monday Nov 30,2026</span></DIV>
        <DIV id='win0divN_ES_XMTT_WRK_DESCR100_2$0'><span id='N_ES_XMTT_WRK_DESCR100_2$0'>9.30 AM to 11.40 AM</span></DIV>
        <DIV id='win0divN_ES_XMTT_WRK_DESCR100_3$0'><span id='N_ES_XMTT_WRK_DESCR100_3$0'>FAK 1001 Sample Mathematics Course</span></DIV>
        <DIV id='win0divN_ES_XMTT_WRK_DESCR100_4$0'><span id='N_ES_XMTT_WRK_DESCR100_4$0'>Location :&nbsp;&nbsp;&nbsp;&nbsp; FAK BLDG Hall A</span></DIV>
        <DIV id='win0divN_ES_XMTT_WRK_DESCR100_5$0'><span id='N_ES_XMTT_WRK_DESCR100_5$0'>Seat Number :&nbsp;&nbsp; 001</span></DIV>
        <DIV id='win0divN_ES_XMTT_WRK_DESCR1$0'><span id='N_ES_XMTT_WRK_DESCR1$0'>Final Exam</span></DIV>
      </div>
    </div>
  </div>
</DIV>`

var expectedICAJAXExams = []Exam{
	{
		Date:       "Monday Nov 30,2026",
		Time:       "9.30 AM to 11.40 AM",
		CourseCode: "FAK 1001",
		CourseName: "Sample Mathematics Course",
		Location:   "FAK BLDG Hall A",
		SeatNumber: "001",
		ExamType:   "Final Exam",
	},
	{
		Date:       "Monday Dec 07,2026",
		Time:       "9.30 AM to 11.40 AM",
		CourseCode: "FAK 2002",
		CourseName: "Synthetic Statistics Module",
		Location:   "FAK HALL 2",
		SeatNumber: "",
		ExamType:   "Final Exam",
	},
}

func examTestLoc(t *testing.T) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation("Asia/Singapore")
	if err != nil {
		t.Fatalf("load location: %v", err)
	}
	return loc
}

func checkExams(t *testing.T, exams []Exam, want []Exam) {
	t.Helper()
	if len(exams) != len(want) {
		t.Fatalf("got %d exams, want %d: %+v", len(exams), len(want), exams)
	}
	for i := range want {
		if exams[i] != want[i] {
			t.Errorf("exam %d = %+v, want %+v", i, exams[i], want[i])
		}
	}
}

func TestParseExamHTMLICAJAXFragment(t *testing.T) {
	exams, err := ParseExamHTML(icajaxExamsFragment, examTestLoc(t))
	if err != nil {
		t.Fatalf("ParseExamHTML: %v", err)
	}
	checkExams(t, exams, expectedICAJAXExams)
}

func TestParseExamHTMLWithoutXMLDeclaration(t *testing.T) {
	const xmlDecl = "<?xml version='1.0' encoding='utf-8' ?>"
	fragment := icajaxExamsFragment[len(xmlDecl)+1:]
	if strings.HasPrefix(fragment, "<?xml") {
		t.Fatal("test fixture did not lose its xml declaration")
	}
	exams, err := ParseExamHTML(fragment, examTestLoc(t))
	if err != nil {
		t.Fatalf("ParseExamHTML: %v", err)
	}
	checkExams(t, exams, expectedICAJAXExams)
}

func TestParseExamHTMLPlainInnerHTML(t *testing.T) {
	exams, err := ParseExamHTML(plainExamsHTML, examTestLoc(t))
	if err != nil {
		t.Fatalf("ParseExamHTML: %v", err)
	}
	checkExams(t, exams, expectedICAJAXExams[:1])
}

func TestParseExamHTMLMultipleExamsSingleScrollarea(t *testing.T) {
	// Live PeopleSoft layout: one scrollarea div (win0div$ICField16$0) wraps
	// every exam as a ps_box-scrollarea-row sibling (win0div$ICField16_row$N).
	// All three exams must come back in document order, not just the first.
	input := wrapSyntheticICAJAX(syntheticExamRow(0,
		"Monday Nov 30,2026", "9.30 AM to 11.40 AM",
		"FAK 1001 Sample Mathematics Course",
		"Location : FAK BLDG Hall A", "Seat Number : 001", "Final Exam") +
		rowSep +
		syntheticExamRow(1,
			"Tuesday Dec 1,2026", "1.00 PM to 3.10 PM",
			"FAK 2002 Synthetic Architecture Course",
			"Location : FAK HALL 2", "Seat Number : 007", "Final Exam") +
		rowSep +
		syntheticExamRow(5, // non-contiguous row index, as seen in production
			"Wednesday Dec 2,2026", "9.30 AM to 11.40 AM",
			"FAK 3003 Synthetic Physics Course",
			"Location : FAK HALL 3", "&nbsp;", "Final Exam"))
	exams, err := ParseExamHTML(input, examTestLoc(t))
	if err != nil {
		t.Fatalf("ParseExamHTML: %v", err)
	}
	checkExams(t, exams, []Exam{
		{Date: "Monday Nov 30,2026", Time: "9.30 AM to 11.40 AM", CourseCode: "FAK 1001",
			CourseName: "Sample Mathematics Course", Location: "FAK BLDG Hall A",
			SeatNumber: "001", ExamType: "Final Exam"},
		{Date: "Tuesday Dec 1,2026", Time: "1.00 PM to 3.10 PM", CourseCode: "FAK 2002",
			CourseName: "Synthetic Architecture Course", Location: "FAK HALL 2",
			SeatNumber: "007", ExamType: "Final Exam"},
		{Date: "Wednesday Dec 2,2026", Time: "9.30 AM to 11.40 AM", CourseCode: "FAK 3003",
			CourseName: "Synthetic Physics Course", Location: "FAK HALL 3",
			SeatNumber: "", ExamType: "Final Exam"},
	})
}

func TestParseExamHTMLMissingContentFieldErrors(t *testing.T) {
	inputs := map[string]string{
		"xml declaration present": `<?xml version='1.0' encoding='utf-8' ?>
<PAGE id='blank'>
  <FIELD id='PT_SIDE2'><![CDATA[]]></FIELD>
</PAGE>`,
		"page element present": `<PAGE id='blank'>
  <FIELD id='win0divPSPANELTABS' class='psc_hidden'><![CDATA[]]></FIELD>
</PAGE>`,
		"content field without cdata": `<PAGE id='blank'>
  <FIELD id='divPAGECONTAINER_TGT' class='ps_pagecontainer'></FIELD>
  <FIELD id='PT_SIDE2'><![CDATA[<div id='side'></div>]]></FIELD>
</PAGE>`,
	}
	for name, input := range inputs {
		t.Run(name, func(t *testing.T) {
			exams, err := ParseExamHTML(input, examTestLoc(t))
			if err == nil {
				t.Fatalf("ParseExamHTML = %d exams, nil error; want error", len(exams))
			}
			if !strings.Contains(err.Error(), "divPAGECONTAINER_TGT") {
				t.Errorf("error %q does not mention divPAGECONTAINER_TGT", err)
			}
		})
	}
}

// syntheticExamRow renders one exam container (row index) with the given
// field values; pass "&nbsp;" for a placeholder value.
func syntheticExamRow(row int, date, timeRange, course, location, seat, examType string) string {
	return fmt.Sprintf(
		`<div class='ps_box-scrollarea-row' id='win0div$ICField16_row$%d'>
    <DIV id='win0divN_ES_XMTT_WRK_DESCR100$%d'><span id='N_ES_XMTT_WRK_DESCR100$%d'>%s</span></DIV>
    <DIV id='win0divN_ES_XMTT_WRK_DESCR100_2$%d'><span id='N_ES_XMTT_WRK_DESCR100_2$%d'>%s</span></DIV>
    <DIV id='win0divN_ES_XMTT_WRK_DESCR100_3$%d'><span id='N_ES_XMTT_WRK_DESCR100_3$%d'>%s</span></DIV>
    <DIV id='win0divN_ES_XMTT_WRK_DESCR100_4$%d'><span id='N_ES_XMTT_WRK_DESCR100_4$%d'>%s</span></DIV>
    <DIV id='win0divN_ES_XMTT_WRK_DESCR100_5$%d'><span id='N_ES_XMTT_WRK_DESCR100_5$%d'>%s</span></DIV>
    <DIV id='win0divN_ES_XMTT_WRK_DESCR1$%d'><span id='N_ES_XMTT_WRK_DESCR1$%d'>%s</span></DIV>
    <DIV id='win0divN_ES_XMTT_WRK_N_SPEC_ARRGMT$%d'><span id='N_ES_XMTT_WRK_N_SPEC_ARRGMT$%d'>&nbsp;</span></DIV>
</div>`, row, row, row, date, row, row, timeRange, row, row, course,
		row, row, location, row, row, seat, row, row, examType, row, row)
}

// wrapSyntheticICAJAX wraps exam row HTML in a minimal ICAJAX fragment.
func wrapSyntheticICAJAX(rows string) string {
	return `<?xml version='1.0' encoding='utf-8' ?>
<PAGE id='blank'>
  <FIELD id='divPAGECONTAINER_TGT' class='ps_pagecontainer'><![CDATA[
<DIV class='ps_pspagecontainer' id='win0divPSPAGECONTAINER'>
  <div class='ps_box-group psc_layout ps_apps_content' id='win0div$ICField1'>
    <div class='ps_box-scrollarea' id='win0div$ICField16$0'>
` + rows + `
    </div>
  </div>
</DIV>
]]></FIELD>
</PAGE>`
}

const rowSep = "\n"

func TestParseExamHTMLStrictness(t *testing.T) {
	loc := examTestLoc(t)

	validRow := func(row int) string {
		return syntheticExamRow(row,
			"Monday Nov 30,2026",
			"9.30 AM to 11.40 AM",
			"FAK 1001 Sample Mathematics Course",
			"Location : FAK BLDG Hall A",
			"Seat Number : 001",
			"Final Exam",
		)
	}
	emptyRow := func(row int) string {
		return syntheticExamRow(row, "&nbsp;", "&nbsp;", "&nbsp;", "&nbsp;", "&nbsp;", "&nbsp;")
	}

	errorCases := map[string]string{
		"missing date": wrapSyntheticICAJAX(validRow(0) + rowSep + syntheticExamRow(1,
			"&nbsp;", "9.30 AM to 11.40 AM", "FAK 1001 Sample Mathematics Course",
			"Location : FAK HALL 2", "Seat Number : 001", "Final Exam")),
		"missing time": wrapSyntheticICAJAX(validRow(0) + rowSep + syntheticExamRow(1,
			"Monday Nov 30,2026", "&nbsp;", "FAK 1001 Sample Mathematics Course",
			"Location : FAK HALL 2", "Seat Number : 001", "Final Exam")),
		"missing course": wrapSyntheticICAJAX(validRow(0) + rowSep + syntheticExamRow(1,
			"Monday Nov 30,2026", "9.30 AM to 11.40 AM", "&nbsp;",
			"Location : FAK HALL 2", "Seat Number : 001", "Final Exam")),
		"malformed date": wrapSyntheticICAJAX(validRow(0) + rowSep + syntheticExamRow(1,
			"Nov 30,2026", "9.30 AM to 11.40 AM", "FAK 1001 Sample Mathematics Course",
			"Location : FAK HALL 2", "Seat Number : 001", "Final Exam")),
		"malformed time": wrapSyntheticICAJAX(validRow(0) + rowSep + syntheticExamRow(1,
			"Monday Nov 30,2026", "9.30 AM", "FAK 1001 Sample Mathematics Course",
			"Location : FAK HALL 2", "Seat Number : 001", "Final Exam")),
	}
	for name, input := range errorCases {
		t.Run(name, func(t *testing.T) {
			if _, err := ParseExamHTML(input, loc); err == nil {
				t.Fatal("ParseExamHTML succeeded; want error")
			}
		})
	}

	t.Run("all-empty placeholder row skipped", func(t *testing.T) {
		input := wrapSyntheticICAJAX(validRow(0) + rowSep + emptyRow(1))
		exams, err := ParseExamHTML(input, loc)
		if err != nil {
			t.Fatalf("ParseExamHTML: %v", err)
		}
		checkExams(t, exams, expectedICAJAXExams[:1])
	})
}

func TestSplitCourseCode(t *testing.T) {
	tests := []struct {
		name string
		in   string
		code string
		want string
	}{
		{"full", "FAK 1001 Sample Mathematics Course", "FAK 1001", "Sample Mathematics Course"},
		{"two tokens", "FAK 1001", "FAK 1001", ""},
		{"single token", "FAK", "FAK", ""},
		{"empty", "", "", ""},
		{"extra whitespace", "  FAK 1001   Sample  Course  ", "FAK 1001", "Sample Course"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, name := splitCourseCode(tt.in)
			if code != tt.code || name != tt.want {
				t.Errorf("splitCourseCode(%q) = (%q, %q), want (%q, %q)", tt.in, code, name, tt.code, tt.want)
			}
		})
	}
}

func TestParseExamDate(t *testing.T) {
	loc := examTestLoc(t)
	tests := []struct {
		name    string
		in      string
		want    time.Time
		wantErr bool
	}{
		{"november", "Monday Nov 30,2026", time.Date(2026, 11, 30, 0, 0, 0, 0, loc), false},
		{"december", "Monday Dec 07,2026", time.Date(2026, 12, 7, 0, 0, 0, 0, loc), false},
		{"single digit day", "Friday Sep 4,2027", time.Date(2027, 9, 4, 0, 0, 0, 0, loc), false},
		{"abbreviated weekday", "Fri Sep 4,2027", time.Time{}, true},
		{"missing weekday", "Nov 30,2026", time.Time{}, true},
		{"full month name", "Monday November 30,2026", time.Time{}, true},
		{"missing year", "Monday Nov 30", time.Time{}, true},
		{"empty", "", time.Time{}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseExamDate(tt.in, loc)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("ParseExamDate(%q) = %v, nil error; want error", tt.in, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseExamDate(%q): %v", tt.in, err)
			}
			if !got.Equal(tt.want) {
				t.Errorf("ParseExamDate(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestParseExamTime(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    [4]int
		wantErr bool
	}{
		{"morning", "9.30 AM to 11.40 AM", [4]int{9, 30, 11, 40}, false},
		{"afternoon", "2.00 PM to 4.30 PM", [4]int{14, 0, 16, 30}, false},
		{"midnight start", "12.00 AM to 1.30 PM", [4]int{0, 0, 13, 30}, false},
		{"late evening", "11.00 AM to 11.40 PM", [4]int{11, 0, 23, 40}, false},
		{"colon format", "9:30 AM to 9:45 AM", [4]int{9, 30, 9, 45}, false},
		{"missing to", "9.30 AM", [4]int{}, true},
		{"empty", "", [4]int{}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sh, sm, eh, em, err := ParseExamTime(tt.in)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("ParseExamTime(%q) = %d:%02d-%d:%02d, nil error; want error", tt.in, sh, sm, eh, em)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseExamTime(%q): %v", tt.in, err)
			}
			if got := [4]int{sh, sm, eh, em}; got != tt.want {
				t.Errorf("ParseExamTime(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestParseExamDateTime(t *testing.T) {
	loc := examTestLoc(t)
	tests := []struct {
		name  string
		exam  Exam
		start time.Time
		end   time.Time
	}{
		{
			name:  "morning exam",
			exam:  Exam{Date: "Monday Nov 30,2026", Time: "9.30 AM to 11.40 AM"},
			start: time.Date(2026, 11, 30, 9, 30, 0, 0, loc),
			end:   time.Date(2026, 11, 30, 11, 40, 0, 0, loc),
		},
		{
			name:  "midnight to evening",
			exam:  Exam{Date: "Friday Dec 04,2026", Time: "12.00 AM to 1.30 PM"},
			start: time.Date(2026, 12, 4, 0, 0, 0, 0, loc),
			end:   time.Date(2026, 12, 4, 13, 30, 0, 0, loc),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start, end, err := ParseExamDateTime(tt.exam, loc)
			if err != nil {
				t.Fatalf("ParseExamDateTime: %v", err)
			}
			if !start.Equal(tt.start) || !end.Equal(tt.end) {
				t.Errorf("got [%v, %v], want [%v, %v]", start, end, tt.start, tt.end)
			}
		})
	}

	if _, _, err := ParseExamDateTime(Exam{Date: "not a date", Time: "9.30 AM to 11.40 AM"}, loc); err == nil {
		t.Error("ParseExamDateTime with invalid date succeeded; want error")
	}
	if _, _, err := ParseExamDateTime(Exam{Date: "Monday Nov 30,2026", Time: "whenever"}, loc); err == nil {
		t.Error("ParseExamDateTime with invalid time succeeded; want error")
	}
}

func TestExamLabelStripping(t *testing.T) {
	tests := []struct {
		name string
		in   string
		re   *regexp.Regexp
		want string
	}{
		{"location nbsp run", "Location :\u00a0\u00a0 FAK BLDG Hall A", examLocationLabelRe, "FAK BLDG Hall A"},
		{"location plain spaces", "Location :    FAK HALL 2", examLocationLabelRe, "FAK HALL 2"},
		{"location no label", "FAK BLDG Hall A", examLocationLabelRe, "FAK BLDG Hall A"},
		{"seat nbsp run", "Seat Number :\u00a0\u00a0 001", examSeatLabelRe, "001"},
		{"seat plain spaces", "Seat Number :  001", examSeatLabelRe, "001"},
		{"seat no label", "007", examSeatLabelRe, "007"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.re.ReplaceAllString(normalizeExamValue(tt.in), "")
			if got != tt.want {
				t.Errorf("strip label from %q = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
