package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/ciel-shieru/sit-ics-go/internal/browser"
	"github.com/ciel-shieru/sit-ics-go/internal/peoplesoft"
)

type HTMLFetcher interface {
	FetchTimetable(ctx context.Context, weekDate string) (string, error)
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

func ExtractPS_TOKEN(cookies []browser.Cookie) string {
	for _, c := range cookies {
		if c.Name == "PS_TOKEN" {
			return c.Value
		}
	}
	return ""
}

func FetchTimetable(ctx context.Context, fetcher HTMLFetcher, weekDate string, loc *time.Location) ([]peoplesoft.Entry, error) {
	html, err := fetcher.FetchTimetable(ctx, weekDate)
	if err != nil {
		return nil, fmt.Errorf("fetch timetable: %w", err)
	}
	year := peoplesoft.ExtractYear(weekDate)
	return peoplesoft.ParseTimetableHTML(html, year, loc)
}
