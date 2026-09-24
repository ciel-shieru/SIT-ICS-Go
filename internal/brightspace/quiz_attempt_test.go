package brightspace

import (
	"strings"
	"testing"

	"golang.org/x/net/html"
)

func TestParseQuizSubmissionHTML_NotAttempted(t *testing.T) {
	htmlStr := `<html><body><div>You have not attempted this quiz.</div></body></html>`
	info, err := ParseQuizSubmissionHTML(htmlStr, 123, "ORG-001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !info.IsUnattempted {
		t.Errorf("expected IsUnattempted=true, got false")
	}
	if info.AttemptsMade != 0 {
		t.Errorf("expected AttemptsMade=0, got %d", info.AttemptsMade)
	}
	if info.QuizID != 123 {
		t.Errorf("expected QuizID=123, got %d", info.QuizID)
	}
	if info.OrgUnitID != "ORG-001" {
		t.Errorf("expected OrgUnitID=ORG-001, got %q", info.OrgUnitID)
	}
}

func TestParseQuizSubmissionHTML_SingleAttempt(t *testing.T) {
	htmlStr := `<html><body>
	<table class="d2l-table d2l-grid">
		<tr>
			<td><a class="d2l-link d2l-link-inline" href="#">Attempt 1</a></td>
			<td class="d_gn"><div><label id="z_eb">14</label><label id="z_ec"> / </label><label id="z_ed">15</label></div></td>
		</tr>
		<tr>
			<td class="d_gr"><label>Overall Grade (highest attempt):</label></td>
			<td class="d_gn"><div><label id="z_en">14</label><label id="z_o"> / </label><label id="z_ep">15</label></div></td>
		</tr>
	</table>
	</body></html>`
	info, err := ParseQuizSubmissionHTML(htmlStr, 456, "ORG-002")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.IsUnattempted {
		t.Error("expected IsUnattempted=false, got true")
	}
	if info.AttemptsMade != 1 {
		t.Errorf("expected AttemptsMade=1, got %d", info.AttemptsMade)
	}
	if info.BestScoreTotal != 15 {
		t.Errorf("expected BestScoreTotal=15, got %f", info.BestScoreTotal)
	}
	expectedScore := (14.0 / 15.0) * 100.0
	if info.BestScore < expectedScore-0.01 || info.BestScore > expectedScore+0.01 {
		t.Errorf("expected BestScore≈%f, got %f", expectedScore, info.BestScore)
	}
}

func TestParseQuizSubmissionHTML_MultipleAttempts(t *testing.T) {
	htmlStr := `<html><body>
	<table class="d2l-table d2l-grid">
		<tr>
			<td><a class="d2l-link d2l-link-inline" href="#">Attempt 1</a></td>
			<td class="d_gn"><div><label id="z_eb">5</label><label id="z_ec"> / </label><label id="z_ed">10</label></div></td>
		</tr>
		<tr>
			<td><a class="d2l-link d2l-link-inline" href="#">Attempt 2</a></td>
			<td class="d_gn"><div><label id="z_eb">8</label><label id="z_ec"> / </label><label id="z_ed">10</label></div></td>
		</tr>
		<tr>
			<td><a class="d2l-link d2l-link-inline" href="#">Attempt 3</a></td>
			<td class="d_gn"><div><label id="z_eb">10</label><label id="z_ec"> / </label><label id="z_ed">10</label></div></td>
		</tr>
		<tr>
			<td class="d_gr"><label>Overall Grade (highest attempt):</label></td>
			<td class="d_gn"><div><label id="z_en">10</label><label id="z_o"> / </label><label id="z_ep">10</label></div></td>
		</tr>
	</table>
	</body></html>`
	info, err := ParseQuizSubmissionHTML(htmlStr, 789, "ORG-003")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.AttemptsMade != 3 {
		t.Errorf("expected AttemptsMade=3, got %d", info.AttemptsMade)
	}
	if info.BestScore != 100.0 {
		t.Errorf("expected BestScore=100.0, got %f", info.BestScore)
	}
	if info.BestScoreTotal != 10 {
		t.Errorf("expected BestScoreTotal=10, got %f", info.BestScoreTotal)
	}
}

func TestParseQuizSubmissionHTML_NoTable(t *testing.T) {
	htmlStr := `<html><body><div>No table here</div></body></html>`
	info, err := ParseQuizSubmissionHTML(htmlStr, 999, "ORG-004")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.IsUnattempted {
		t.Error("expected IsUnattempted=false, got true")
	}
	if info.AttemptsMade != 0 {
		t.Errorf("expected AttemptsMade=0, got %d", info.AttemptsMade)
	}
}

func TestParseQuizSubmissionHTML_HeaderRowSkipped(t *testing.T) {
	htmlStr := `<html><body>
	<table class="d2l-table d2l-grid">
		<tr><th>Individual Attempts</th><th>Grade</th></tr>
		<tr>
			<td><a class="d2l-link d2l-link-inline" href="#">Attempt 1</a></td>
			<td class="d_gn"><div><label>8</label><label> / </label><label>10</label></div></td>
		</tr>
		<tr>
			<td class="d_gr"><label>Overall Grade (highest attempt):</label></td>
			<td class="d_gn"><div><label>8</label><label> / </label><label>10</label></div></td>
		</tr>
	</table>
	</body></html>`
	info, err := ParseQuizSubmissionHTML(htmlStr, 100, "ORG-010")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.AttemptsMade != 1 {
		t.Errorf("expected AttemptsMade=1 (header row skipped), got %d", info.AttemptsMade)
	}
}

func TestParseQuizSubmissionHTML_SpanElementScore(t *testing.T) {
	htmlStr := `<html><body>
	<table class="d2l-table d2l-grid">
		<tr>
			<td><a class="d2l-link d2l-link-inline" href="#">Attempt 1</a></td>
			<td class="d_gn"><div><span>9</span><span> / </span><span>10</span></div></td>
		</tr>
		<tr>
			<td class="d_gr"><label>Overall Grade (highest attempt):</label></td>
			<td class="d_gn"><div><span>9</span><span> / </span><span>10</span></div></td>
		</tr>
	</table>
	</body></html>`
	info, err := ParseQuizSubmissionHTML(htmlStr, 200, "ORG-020")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.AttemptsMade != 1 {
		t.Errorf("expected AttemptsMade=1, got %d", info.AttemptsMade)
	}
	expectedScore := (9.0 / 10.0) * 100.0
	if info.BestScore < expectedScore-0.01 || info.BestScore > expectedScore+0.01 {
		t.Errorf("expected BestScore≈%f, got %f", expectedScore, info.BestScore)
	}
}

func TestRowHasThElement(t *testing.T) {
	tests := []struct {
		name string
		html string
		want bool
	}{
		{"has th", `<table><tr><th>Header</th><td>Data</td></tr></table>`, true},
		{"no th", `<table><tr><td>Attempt 1</td><td>8 / 10</td></tr></table>`, false},
		{"empty row", `<table><tr></tr></table>`, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := html.Parse(strings.NewReader(tt.html))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			var row *html.Node
			visitAll(doc, func(n *html.Node) {
				if n.Type == html.ElementNode && n.Data == "tr" && row == nil {
					row = n
				}
			})
			if row == nil {
				t.Fatal("no <tr> found in parsed HTML")
			}
			got := rowHasThElement(row)
			if got != tt.want {
				t.Errorf("rowHasThElement() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtractNumericText(t *testing.T) {
	tests := []struct {
		name string
		html string
		want []float64
	}{
		{
			name: "label elements",
			html: `<div><label>5</label><label> / </label><label>10</label></div>`,
			want: []float64{5, 10},
		},
		{
			name: "span elements",
			html: `<div><span>7</span><span> / </span><span>10</span></div>`,
			want: []float64{7, 10},
		},
		{
			name: "mixed label and span",
			html: `<div><label>3</label><span> / </span><span>5</span></div>`,
			want: []float64{3, 5},
		},
		{
			name: "no numeric text",
			html: `<div><label>abc</label><label> / </label><label>xyz</label></div>`,
			want: []float64{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := html.Parse(strings.NewReader(tt.html))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			got := extractNumericText(doc)
			if len(got) != len(tt.want) {
				t.Errorf("extractNumericText() = %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("extractNumericText()[%d] = %v, want %v", i, got[i], tt.want[i])
				}
			}
		})
	}
}
