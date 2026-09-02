package peoplesoft

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/ciel-shieru/sit-ics-go/internal/browser"
	"golang.org/x/net/html"
	"golang.org/x/net/proxy"
	"net/url"
)

type Field struct {
	Name  string `xml:"NAME"`
	Value string `xml:"VALUE"`
}

type Response struct {
	XMLName xml.Name `xml:"FIELD"`
	Fields  []Field  `xml:"FIELD"`
}

type Entry struct {
	CourseCode string
	Section    string
	Type       string
	Day        string
	StartTime  string
	EndTime    string
	Location   string
}

type Client struct {
	httpClient *http.Client
	baseURL    string
}

func NewClient(baseURL, proxyURL string) *Client {
	var httpClient *http.Client

	if proxyURL != "" {
		parsed, err := url.Parse(proxyURL)
		if err != nil {
			httpClient = &http.Client{}
		} else {
			proxyAddr := parsed.Host
			proxyDialer, err := proxy.SOCKS5("tcp", proxyAddr, nil, proxy.Direct)
			if err != nil {
				httpClient = &http.Client{}
			} else {
				httpClient = &http.Client{
					Timeout: 30 * time.Second,
					Transport: &http.Transport{
						DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
							return proxyDialer.Dial(network, addr)
						},
					},
				}
			}
		}
	} else {
		httpClient = &http.Client{
			Timeout: 30 * time.Second,
		}
	}

	return &Client{
		httpClient: httpClient,
		baseURL:    baseURL,
	}
}

func (c *Client) SetCookies(ctx context.Context, cookies []browser.Cookie) ([]browser.Cookie, string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL, nil)
	if err != nil {
		return nil, "", fmt.Errorf("create request: %w", err)
	}

	for _, cookie := range cookies {
		req.AddCookie(&http.Cookie{
			Name:   cookie.Name,
			Value:  cookie.Value,
			Domain: cookie.Domain,
			Path:   cookie.Path,
		})
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("fetch peoplesoft: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("peoplesoft returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("read response: %w", err)
	}

	var psResp Response
	if err := xml.Unmarshal(body, &psResp); err != nil {
		return nil, "", fmt.Errorf("parse peoplesoft response: %w", err)
	}

	token := ""
	for _, field := range psResp.Fields {
		if field.Name == "PS_TOKEN" {
			token = field.Value
			break
		}
	}

	newCookies := make([]browser.Cookie, 0, len(psResp.Fields))
	for _, field := range psResp.Fields {
		newCookies = append(newCookies, browser.Cookie{
			Name:  field.Name,
			Value: field.Value,
		})
	}

	return newCookies, token, nil
}

func (c *Client) FetchTimetable(ctx context.Context, weekDate string) ([]Entry, error) {
	formData := strings.NewReader(fmt.Sprintf("_PANEL_MODE=VIEW&_PANELS=0&_PROCESS=SSR_SSENRL_SCHD_W&_ACTION=VIEW&_ADVPRTFLG=N&_DISPLAYPAGELINKS=Y&WEEK_DATE=%s", weekDate))

	req, err := http.NewRequestWithContext(ctx, "POST", strings.TrimSuffix(c.baseURL, "/")+"/SA_LEARNER_SERVICES.SSR_SSENRL_SCHD_W.GBL", formData)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch timetable: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var psResp Response
	if err := xml.Unmarshal(body, &psResp); err != nil {
		return nil, fmt.Errorf("parse timetable response: %w", err)
	}

	htmlContent := ""
	for _, field := range psResp.Fields {
		if field.Name == "_HTML" {
			htmlContent = field.Value
			break
		}
	}

	if htmlContent == "" {
		return nil, fmt.Errorf("no timetable HTML found in response")
	}

	return parseTimetableHTML(htmlContent)
}

func parseTimetableHTML(htmlContent string) ([]Entry, error) {
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return nil, fmt.Errorf("parse HTML: %w", err)
	}

	var entries []Entry
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
							entries = append(entries, Entry{
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
