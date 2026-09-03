package peoplesoft

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	netcookiejar "net/http/cookiejar"
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
	httpClient        *http.Client
	baseURL           string
	debug             bool
	logResponse       bool
	logRequest        bool
	jar               *netcookiejar.Jar
}

func NewClient(baseURL, proxyURL string, debug bool, logResponse bool, logRequest bool) *Client {
	jar, err := netcookiejar.New(nil)
	if err != nil {
		log.Printf("peoplesoft: failed to create cookie jar: %v", err)
	}

	var httpClient *http.Client

	if proxyURL != "" {
		parsed, err := url.Parse(proxyURL)
		if err != nil {
			httpClient = &http.Client{
				Timeout: 30 * time.Second,
				Jar:     jar,
			}
		} else {
			proxyAddr := parsed.Host
			proxyDialer, err := proxy.SOCKS5("tcp", proxyAddr, nil, proxy.Direct)
			if err != nil {
				httpClient = &http.Client{
					Timeout: 30 * time.Second,
					Jar:     jar,
				}
			} else {
				httpClient = &http.Client{
					Timeout: 30 * time.Second,
					Transport: &http.Transport{
						DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
							return proxyDialer.Dial(network, addr)
						},
					},
					Jar: jar,
				}
			}
		}
	} else {
		httpClient = &http.Client{
			Timeout: 30 * time.Second,
			Jar:     jar,
		}
	}

	return &Client{
		httpClient:  httpClient,
		baseURL:     baseURL,
		debug:       debug,
		logResponse: logResponse,
		logRequest:  logRequest,
		jar:         jar,
	}
}

func (c *Client) debugLog(msg string, args ...any) {
	if c.debug {
		log.Printf("peoplesoft: "+msg, args...)
	}
}

func (c *Client) SetCookies(cookies []browser.Cookie) {
	peoplesoftURL, _ := url.Parse(c.baseURL)
	peoplesoftHost := peoplesoftURL.Host

	httpCookies := make([]*http.Cookie, 0, len(cookies))
	for _, c := range cookies {
		hc := &http.Cookie{
			Name:   c.Name,
			Value:  c.Value,
			Domain: c.Domain,
			Path:   c.Path,
		}
		if c.Expiry > 0 {
			hc.Expires = time.Unix(c.Expiry, 0)
		}
		httpCookies = append(httpCookies, hc)
	}

	if c.jar != nil {
		c.jar.SetCookies(peoplesoftURL, httpCookies)
	}

	for _, domain := range []string{
		"https://in4sit.singaporetech.edu.sg",
		"https://fs.singaporetech.edu.sg",
	} {
		u, err := url.Parse(domain)
		if err != nil {
			continue
		}
		if u.Host != peoplesoftHost {
			domainCookies := make([]*http.Cookie, 0)
			for _, cookie := range cookies {
				if strings.Contains(cookie.Domain, u.Host) {
					hc := &http.Cookie{
						Name:   cookie.Name,
						Value:  cookie.Value,
						Domain: cookie.Domain,
						Path:   cookie.Path,
					}
					if cookie.Expiry > 0 {
						hc.Expires = time.Unix(cookie.Expiry, 0)
					}
					domainCookies = append(domainCookies, hc)
				}
			}
			if len(domainCookies) > 0 && c.jar != nil {
				c.jar.SetCookies(u, domainCookies)
			}
		}
	}
}

func (c *Client) FetchTimetable(ctx context.Context, weekDate string) ([]Entry, error) {
	parsedDate, err := time.Parse("2006-01-02", weekDate)
	if err != nil {
		return nil, fmt.Errorf("parse week date: %w", err)
	}

	formattedDate := parsedDate.Format("02/01/2006")
	encodedDate := url.QueryEscape(formattedDate)

	queryParams := url.Values{}
	queryParams.Add("ICAJAX", "1")
	queryParams.Add("ICAction", "DERIVED_CLASS_S_SSR_REFRESH_CAL$8$")
	queryParams.Add("DERIVED_CLASS_S_START_DT", encodedDate)

	fullURL := strings.TrimSuffix(c.baseURL, "/") + "/EMPLOYEE/SA/SA_LEARNER_SERVICES.SSR_SSENRL_SCHD_W.GBL?" + queryParams.Encode()

	formBody := fmt.Sprintf("_PANEL_MODE=VIEW&_PANELS=0&_PROCESS=SSR_SSENRL_SCHD_W&_ACTION=VIEW&_ADVPRTFLG=N&_DISPLAYPAGELINKS=Y&WEEK_DATE=%s", weekDate)
	formData := strings.NewReader(formBody)

	req, err := http.NewRequestWithContext(ctx, "POST", fullURL, formData)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	if c.logRequest {
		var cookies []*http.Cookie
		if c.jar != nil {
			cookies = c.jar.Cookies(req.URL)
		}
		cookieStr := ""
		for _, cookie := range cookies {
			if cookieStr != "" {
				cookieStr += "; "
			}
			cookieStr += cookie.String()
		}
		log.Printf("peoplesoft request: url=%s method=%s headers=%v body=%s cookies=%s", req.URL.String(), req.Method, req.Header, formBody, cookieStr)
	} else {
		c.debugLog("POST %s", req.URL.String())
	}

	// The cookiejar attached to httpClient automatically processes Set-Cookie
	// response headers after Do() returns. It updates/replaces cookies whose
	// name, domain, and path match those in the header while preserving every
	// other cookie already stored in the jar.
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch timetable: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if c.logResponse {
		log.Printf("peoplesoft response: status=%d headers=%v body=%s", resp.StatusCode, resp.Header, string(body))
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
