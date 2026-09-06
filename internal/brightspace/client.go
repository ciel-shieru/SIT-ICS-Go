package brightspace

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	baseURL    string
	apiKey     string
	apiSecret  string
	httpClient *http.Client
}

func NewClient(baseURL, apiKey, apiSecret string) *Client {
	return &Client{
		baseURL:   strings.TrimRight(baseURL, "/"),
		apiKey:    apiKey,
		apiSecret: apiSecret,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) CheckVersion(ctx context.Context) (string, error) {
	url := fmt.Sprintf("%s/d2l/api/le/versions/", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("create version request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch versions: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("versions API returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var vr VersionResponse
	if err := json.NewDecoder(resp.Body).Decode(&vr); err != nil {
		return "", fmt.Errorf("parse versions response: %w", err)
	}

	return vr.LatestVersion, nil
}

func (c *Client) FetchCourses(ctx context.Context) ([]Course, error) {
	url := fmt.Sprintf("%s/d2l/le/manageCourses/api/mycourses", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create courses request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch courses: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("courses API returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var cr MyCoursesResponse
	if err := json.NewDecoder(resp.Body).Decode(&cr); err != nil {
		return nil, fmt.Errorf("parse courses response: %w", err)
	}

	return cr.Courses, nil
}

func (c *Client) FetchCalendarEvents(ctx context.Context, version, orgUnitID string) ([]CalendarEvent, error) {
	url := fmt.Sprintf("%s/d2l/api/le/%s/%s/calendar/events/", c.baseURL, version, orgUnitID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create calendar events request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch calendar events: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNoContent {
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("calendar events API returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var events []CalendarEvent
	if err := json.NewDecoder(resp.Body).Decode(&events); err != nil {
		return nil, fmt.Errorf("parse calendar events response: %w", err)
	}

	return events, nil
}

func (c *Client) FetchDropboxFolders(ctx context.Context, version, orgUnitID string) ([]DropboxFolder, error) {
	url := fmt.Sprintf("%s/d2l/api/le/%s/%s/dropbox/folders/", c.baseURL, version, orgUnitID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create dropbox folders request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch dropbox folders: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNoContent {
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("dropbox folders API returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var folders []DropboxFolder
	if err := json.NewDecoder(resp.Body).Decode(&folders); err != nil {
		return nil, fmt.Errorf("parse dropbox folders response: %w", err)
	}

	return folders, nil
}
