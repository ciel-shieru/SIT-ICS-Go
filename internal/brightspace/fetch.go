package brightspace

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
)

// Fetcher abstracts browser page operations for BrightSpace fetching.
type Fetcher interface {
	Navigate(url string) error
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

// FetchConfig holds the configuration for a BrightSpace fetch.
type FetchConfig struct {
	CourseNameBlocklist    []string
	CourseIDBlocklist      []string
	EventTitleBlocklist    []string
	EventLocationBlocklist []string
}

// Fetch extracts BrightSpace D2L calendar events and dropbox due dates,
// returning them as BrightSpaceStringEntry values for downstream ICS conversion.
func Fetch(ctx context.Context, client *Client, cfg *FetchConfig) ([]BrightSpaceStringEntry, error) {
	blocklist := &Blocklist{
		CourseNamePatterns:    cfg.CourseNameBlocklist,
		CourseIDs:             cfg.CourseIDBlocklist,
		EventTitlePatterns:    cfg.EventTitleBlocklist,
		EventLocationPatterns: cfg.EventLocationBlocklist,
	}

	version, err := client.CheckVersion(ctx)
	if err != nil {
		return nil, fmt.Errorf("check version: %w", err)
	}
	log.Printf("brightspace: API version %s", version)

	courses, err := client.FetchCourses(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetch courses: %w", err)
	}
	log.Printf("brightspace: found %d courses", len(courses))

	var entries []BrightSpaceStringEntry

	for _, course := range courses {
		if blocklist.IsCourseBlocked(course.OrgUnitId, course.Name) {
			log.Printf("brightspace: blocked course %s (%s)", course.Name, course.OrgUnitId)
			continue
		}

		// Fetch calendar events
		events, err := client.FetchCalendarEvents(ctx, version, course.OrgUnitId)
		if err != nil {
			log.Printf("brightspace: failed to fetch calendar events for %s: %v", course.Name, err)
			continue
		}
		for _, ev := range events {
			title := ev.Title
			if title == "" {
				title = ev.OrgUnitName
			}

			descParts := []string{}
			if ev.Description != "" {
				plainDesc := htmlToPlainText(ev.Description)
				if plainDesc != "" {
					descParts = append(descParts, plainDesc)
				}
			}
			if ev.LocationName != "" {
				descParts = append(descParts, "Location: "+ev.LocationName)
			}

			entry := BrightSpaceStringEntry{
				Title:       title,
				OrgUnitId:   strconv.Itoa(ev.OrgUnitId),
				OrgUnitName: course.Name,
				OrgUnitCode: course.Code,
				Location:    ev.LocationName,
				Description: strings.Join(descParts, "\n"),
				DTStart:     ev.StartDateTime,
				DTEnd:       ev.EndDateTime,
				IsAllDay:    ev.IsAllDayEvent,
				Source:      "brightspace-calendar",
			}
			if entry.Title == "" {
				continue
			}
			if blocklist.IsEventBlocked(entry.Title) {
				log.Printf("brightspace: blocked event %q in %s", entry.Title, course.Name)
				continue
			}
			if blocklist.IsLocationBlocked(entry.Location) {
				log.Printf("brightspace: blocked event %q in %s (location %q)", entry.Title, course.Name, entry.Location)
				continue
			}
			entries = append(entries, entry)
		}

		// Fetch dropbox folders
		folders, err := client.FetchDropboxFolders(ctx, version, course.OrgUnitId)
		if err != nil {
			log.Printf("brightspace: failed to fetch dropbox folders for %s: %v", course.Name, err)
			continue
		}
		for _, folder := range folders {
			entry := BrightSpaceStringEntry{
				Title:       fmt.Sprintf("[Submission Due] %s", folder.Name),
				OrgUnitId:   strconv.Itoa(folder.Id),
				OrgUnitName: course.Name,
				OrgUnitCode: course.Code,
				Description: "Dropbox: " + folder.Name,
				DTStart:     folder.DueDate,
				DTEnd:       folder.DueDate,
				IsAllDay:    false,
				Source:      "brightspace-dropbox",
			}
			if entry.Title == "" {
				continue
			}
			if blocklist.IsEventBlocked(entry.Title) {
				log.Printf("brightspace: blocked dropbox %q in %s", entry.Title, course.Name)
				continue
			}
			entries = append(entries, entry)
		}
	}

	log.Printf("brightspace: extracted %d entries", len(entries))
	return entries, nil
}
