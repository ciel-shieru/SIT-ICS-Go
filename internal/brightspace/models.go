package brightspace

import "time"

// API response models for BrightSpace D2L.

type VersionResponse struct {
	ProductCode       string   `json:"ProductCode"`
	LatestVersion     string   `json:"LatestVersion"`
	SupportedVersions []string `json:"SupportedVersions"`
}

type MyCoursesResponse struct {
	Courses []Course `json:"Courses"`
	Bookmark *string  `json:"Bookmark"`
	Sort    string   `json:"Sort"`
}

type Course struct {
	OrgUnitId string `json:"OrgUnitId"`
	Name      string `json:"Name"`
	Code      string `json:"Code"`
	IsActive  bool   `json:"IsActive"`
}

type CalendarEvent struct {
	CalendarEventId int       `json:"CalendarEventId"`
	OrgUnitId       int       `json:"OrgUnitId"`
	Title           string    `json:"Title"`
	Description     string    `json:"Description"`
	IsAllDayEvent   bool      `json:"IsAllDayEvent"`
	StartDateTime   time.Time `json:"StartDateTime"`
	EndDateTime     time.Time `json:"EndDateTime"`
	IsRecurring     bool      `json:"IsRecurring"`
	LocationName    string    `json:"LocationName"`
	OrgUnitName     string    `json:"OrgUnitName"`
	OrgUnitCode     string    `json:"OrgUnitCode"`
	EventType       int       `json:"EventType"`
}

type DropboxFolder struct {
	Id            int    `json:"Id"`
	Name          string `json:"Name"`
	DueDate       time.Time `json:"DueDate"`
	DisplayInCalendar bool `json:"DisplayInCalendar"`
	OrgUnitName   string `json:"-"`
	OrgUnitCode   string `json:"-"`
}

// BrightSpaceEntry is the internal representation of a BrightSpace event/due date.
type BrightSpaceEntry struct {
	// Source identifies whether this came from calendar events or dropbox folders.
	Source      SourceType
	Title       string
	OrgUnitId   string
	OrgUnitName string
	OrgUnitCode string
	Location    string
	Description string
	DTStart     time.Time
	DTEnd       time.Time
	IsAllDay    bool
}

type SourceType string

const (
	SourceCalendar SourceType = "calendar"
	SourceDropbox  SourceType = "dropbox"
)
