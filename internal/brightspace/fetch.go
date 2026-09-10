package brightspace

import (
	"context"
	"fmt"
	"log"
	"strings"
)

// Fetcher abstracts browser page operations for BrightSpace fetching.
type Fetcher interface {
	DecodeJSON(url string, v interface{}) error
}

// Client owns BrightSpace endpoint paths, URL construction, and API fetching through browser.
type Client struct {
	baseURL string
	fetcher Fetcher
}

// NewClient creates a new BrightSpace client using the given Fetcher for browser-backed HTTP.
func NewClient(baseURL string, fetcher Fetcher) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		fetcher: fetcher,
	}
}

// CheckVersion returns the latest BrightSpace API version.
func (c *Client) CheckVersion(ctx context.Context) (string, error) {
	var vr VersionResponse
	if err := c.fetcher.DecodeJSON(fmt.Sprintf("%s/d2l/api/le/versions/", c.baseURL), &vr); err != nil {
		return "", fmt.Errorf("check version: %w", err)
	}
	if vr.LatestVersion == "" {
		return "", fmt.Errorf("check version: empty LatestVersion in response")
	}
	return vr.LatestVersion, nil
}

// FetchCourses returns all courses accessible to the authenticated user.
func (c *Client) FetchCourses(ctx context.Context) ([]Course, error) {
	var cr MyCoursesResponse
	if err := c.fetcher.DecodeJSON(fmt.Sprintf("%s/d2l/le/manageCourses/api/mycourses", c.baseURL), &cr); err != nil {
		return nil, fmt.Errorf("fetch courses: %w", err)
	}
	return cr.Courses, nil
}

// FetchCalendarEvents returns calendar events for a course.
func (c *Client) FetchCalendarEvents(ctx context.Context, version, orgUnitID string) ([]CalendarEventAPI, error) {
	var events []CalendarEventAPI
	url := fmt.Sprintf("%s/d2l/api/le/%s/%s/calendar/events/", c.baseURL, version, orgUnitID)
	if err := c.fetcher.DecodeJSON(url, &events); err != nil {
		return nil, fmt.Errorf("fetch calendar events: %w", err)
	}
	return events, nil
}

// FetchDropboxFolders returns dropbox folders for a course.
func (c *Client) FetchDropboxFolders(ctx context.Context, version, orgUnitID string) ([]DropboxFolderAPI, error) {
	var folders []DropboxFolderAPI
	url := fmt.Sprintf("%s/d2l/api/le/%s/%s/dropbox/folders/", c.baseURL, version, orgUnitID)
	if err := c.fetcher.DecodeJSON(url, &folders); err != nil {
		return nil, fmt.Errorf("fetch dropbox folders: %w", err)
	}
	return folders, nil
}

// Fetch extracts BrightSpace D2L calendar events and dropbox due dates.
// Returns raw API objects; callers should use EntriesToEvents() for conversion
// and blocklist filtering to avoid duplication.
func Fetch(ctx context.Context, client *Client) ([]CalendarEventAPI, []DropboxFolderAPI, error) {
	version, err := client.CheckVersion(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("check version: %w", err)
	}
	log.Printf("brightspace: API version %s", version)

	courses, err := client.FetchCourses(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("fetch courses: %w", err)
	}
	log.Printf("brightspace: found %d courses", len(courses))

	var allEvents []CalendarEventAPI
	var allFolders []DropboxFolderAPI

	for _, course := range courses {
		events, err := client.FetchCalendarEvents(ctx, version, course.OrgUnitId)
		if err != nil {
			log.Printf("brightspace: failed to fetch calendar events for %s: %v", course.Name, err)
			continue
		}
		for _, ev := range events {
			ev.OrgUnitName = course.Name
			ev.OrgUnitCode = course.Code
			allEvents = append(allEvents, ev)
		}

		folders, err := client.FetchDropboxFolders(ctx, version, course.OrgUnitId)
		if err != nil {
			log.Printf("brightspace: failed to fetch dropbox folders for %s: %v", course.Name, err)
			continue
		}
		for _, folder := range folders {
			folder.OrgUnitId = course.OrgUnitId
			folder.OrgUnitName = course.Name
			folder.OrgUnitCode = course.Code
			allFolders = append(allFolders, folder)
		}
	}

	log.Printf("brightspace: extracted %d events, %d dropbox folders", len(allEvents), len(allFolders))
	return allEvents, allFolders, nil
}
