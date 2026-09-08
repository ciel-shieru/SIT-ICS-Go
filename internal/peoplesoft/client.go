package peoplesoft

import (
	"context"
	"fmt"
	"time"
)

// Fetcher abstracts the browser page interaction for fetching PeopleSoft timetable HTML.
type Fetcher interface {
	NavigateTimetable(ctx context.Context) (string, error)
}

// Client owns PeopleSoft URL, navigation, and HTML parsing.
type Client struct {
	fetcher Fetcher
	loc     *time.Location
}

// NewClient creates a new Client that uses the given Fetcher for navigation.
func NewClient(f Fetcher, loc *time.Location) *Client {
	return &Client{fetcher: f, loc: loc}
}

// FetchTimetable navigates to the PeopleSoft timetable endpoint and returns the HTML.
func (c *Client) FetchTimetable(ctx context.Context) (string, error) {
	html, err := c.fetcher.NavigateTimetable(ctx)
	if err != nil {
		return "", fmt.Errorf("navigate timetable: %w", err)
	}
	return html, nil
}

// FetchEntries fetches the timetable and parses it into Entries.
func (c *Client) FetchEntries(ctx context.Context) ([]Entry, error) {
	html, err := c.FetchTimetable(ctx)
	if err != nil {
		return nil, err
	}
	entries, err := ParseTimetableHTML(html, 0, c.loc)
	if err != nil {
		return nil, fmt.Errorf("parse timetable: %w", err)
	}
	return entries, nil
}
