package brightspace

import (
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/net/html"
)

type QuizAttemptInfo struct {
	QuizID         int
	OrgUnitID      string
	AttemptsMade   int
	BestScore      float64
	BestScoreTotal float64
	IsUnattempted  bool
}

func visitAll(n *html.Node, visit func(*html.Node)) {
	visit(n)
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		visitAll(c, visit)
	}
}

func rowHasThElement(n *html.Node) bool {
	hasTh := false
	visitAll(n, func(child *html.Node) {
		if !hasTh && child.Type == html.ElementNode && child.Data == "th" {
			hasTh = true
		}
	})
	return hasTh
}

func extractNumericText(node *html.Node) []float64 {
	var result []float64
	visitAll(node, func(n *html.Node) {
		if n.Type == html.ElementNode && (n.Data == "label" || n.Data == "span") && n.FirstChild != nil {
			if v, err := strconv.ParseFloat(n.FirstChild.Data, 64); err == nil {
				result = append(result, v)
			}
		}
	})
	return result
}

func ParseQuizSubmissionHTML(htmlStr string, quizID int, orgUnitID string) (QuizAttemptInfo, error) {
	if strings.Contains(htmlStr, "You have not attempted this quiz.") {
		return QuizAttemptInfo{QuizID: quizID, OrgUnitID: orgUnitID, IsUnattempted: true}, nil
	}

	doc, err := html.Parse(strings.NewReader(htmlStr))
	if err != nil {
		return QuizAttemptInfo{}, fmt.Errorf("parse HTML: %w", err)
	}

	var table *html.Node
	visitAll(doc, func(n *html.Node) {
		if table == nil && n.Type == html.ElementNode && n.Data == "table" {
			for _, a := range n.Attr {
				if a.Key == "class" && strings.Contains(a.Val, "d2l-table") && strings.Contains(a.Val, "d2l-grid") {
					table = n
					return
				}
			}
		}
	})
	if table == nil {
		return QuizAttemptInfo{QuizID: quizID, OrgUnitID: orgUnitID}, nil
	}

	var rows []*html.Node
	visitAll(table, func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "tr" {
			rows = append(rows, n)
		}
	})
	if len(rows) == 0 {
		return QuizAttemptInfo{QuizID: quizID, OrgUnitID: orgUnitID}, nil
	}

	bestScore := 0.0
	bestTotal := 0.0
	attemptsMade := 0

	for _, row := range rows {
		if rowHasThElement(row) {
			continue
		}

		var rowText strings.Builder
		visitAll(row, func(n *html.Node) {
			if n.Type == html.TextNode {
				rowText.WriteString(n.Data)
			}
		})
		rowStr := rowText.String()

		if strings.Contains(rowStr, "Overall Grade") {
			numerics := extractNumericText(row)
			if len(numerics) >= 2 {
				bestScore = numerics[0]
				bestTotal = numerics[1]
			}
			continue
		}

		attemptsMade++
	}

	if bestTotal > 0 {
		bestScore = (bestScore / bestTotal) * 100.0
	}

	return QuizAttemptInfo{
		QuizID:         quizID,
		OrgUnitID:      orgUnitID,
		AttemptsMade:   attemptsMade,
		BestScore:      bestScore,
		BestScoreTotal: bestTotal,
	}, nil
}
