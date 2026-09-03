package auth

import (
	"context"
	"fmt"
	"strings"

	"github.com/ciel-shieru/sit-ics-go/internal/browser"
	"github.com/ciel-shieru/sit-ics-go/internal/peoplesoft"
	"golang.org/x/net/html"
)

type PeoplesoftClient interface {
	FetchTimetable(ctx context.Context, weekDate string) ([]peoplesoft.Entry, error)
}

type AuthRequest struct {
	Username      string
	Password      string
	TOTPSecret    string
	PeopleSoftURL string
}

type AuthResult struct {
	Cookies []browser.Cookie
	Token   string
}

type ADFSProvider struct {
	browser browser.AuthBrowser
}

func NewADFSProvider(b browser.AuthBrowser) *ADFSProvider {
	return &ADFSProvider{
		browser: b,
	}
}

func (a *ADFSProvider) Authenticate(ctx context.Context, req AuthRequest) (AuthResult, error) {
	browserReq := browser.AuthRequest{
		URL:        "https://in4sit.singaporetech.edu.sg",
		Username:   req.Username,
		Password:   req.Password,
		TOTPSecret: req.TOTPSecret,
	}

	authResult, err := a.browser.Authenticate(ctx, browserReq)
	if err != nil {
		return AuthResult{}, fmt.Errorf("adfs authenticate: %w", err)
	}

	token := extractPS_TOKEN(authResult.Cookies)

	return AuthResult{
		Cookies: authResult.Cookies,
		Token:   token,
	}, nil
}

func (a *ADFSProvider) FetchTimetable(ctx context.Context, weekDate string) ([]peoplesoft.Entry, error) {
	html, err := a.browser.FetchTimetable(ctx, weekDate)
	if err != nil {
		return nil, fmt.Errorf("browser fetch timetable: %w", err)
	}
	return parseTimetableHTML(html)
}

func extractPS_TOKEN(cookies []browser.Cookie) string {
	for _, c := range cookies {
		if c.Name == "PS_TOKEN" {
			return c.Value
		}
	}
	return ""
}

func parseTimetableHTML(htmlContent string) ([]peoplesoft.Entry, error) {
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return nil, fmt.Errorf("parse HTML: %w", err)
	}

	var entries []peoplesoft.Entry
	var currentCourse, currentSection, currentType, currentLocation string
	var currentTimeStart, currentTimeEnd string

	var visit func(*html.Node)
	visit = func(n *html.Node) {
		if n.Type == html.ElementNode {
			if n.Data == "td" {
				for _, attr := range n.Attr {
					if attr.Key == "class" && strings.Contains(attr.Val, "PSLEVEL1GRIDHTM") {
						text := getTextContent(n)
						if text != "" {
							entries = append(entries, peoplesoft.Entry{
								CourseCode: currentCourse,
								Section:    currentSection,
								Type:       currentType,
								Day:        text,
								StartTime:  currentTimeStart,
								EndTime:    currentTimeEnd,
								Location:   currentLocation,
							})
						}
					}
				}
			}

			if n.Data == "span" {
				for _, attr := range n.Attr {
					if attr.Key == "class" {
						text := getTextContent(n)
						switch {
						case strings.Contains(attr.Val, "PSCOURSENAME"):
							currentCourse = text
						case strings.Contains(attr.Val, "PSPNLGROUPNAME"):
							currentSection = text
						case strings.Contains(attr.Val, "PS_SUBJECT"):
							currentType = text
						case strings.Contains(attr.Val, "PSEFF_FROM_TM_STMP"):
							currentTimeStart = text
						case strings.Contains(attr.Val, "PSEFF_TO_TM_STMP"):
							currentTimeEnd = text
						case strings.Contains(attr.Val, "LOCATION"):
							currentLocation = text
						}
					}
				}
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			visit(c)
		}
	}

	visit(doc)
	return entries, nil
}

func getTextContent(n *html.Node) string {
	var sb strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.TextNode {
			sb.WriteString(c.Data)
		}
	}
	return strings.TrimSpace(sb.String())
}
