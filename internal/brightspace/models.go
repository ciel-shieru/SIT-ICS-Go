package brightspace

import "time"

// API response models for BrightSpace D2L.

type VersionResponse struct {
	ProductCode       string   `json:"ProductCode"`
	LatestVersion     string   `json:"LatestVersion"`
	SupportedVersions []string `json:"SupportedVersions"`
}

type MyCoursesResponse struct {
	Courses  []Course `json:"Courses"`
	Bookmark *string  `json:"Bookmark"`
	Sort     string   `json:"Sort"`
}

type Course struct {
	OrgUnitId string `json:"OrgUnitId"`
	Name      string `json:"Name"`
	Code      string `json:"Code"`
	IsActive  bool   `json:"IsActive"`
}

type AssociatedEntity struct {
	AssociatedEntityType string `json:"AssociatedEntityType"`
	AssociatedEntityId   int    `json:"AssociatedEntityId"`
	Link                 string `json:"Link"`
}

type RecurrenceInfo struct {
	RepeatType      int             `json:"RepeatType"`
	RepeatEvery     int             `json:"RepeatEvery"`
	RepeatOnInfo    *RepeatOnInfo   `json:"RepeatOnInfo,omitempty"`
	RepeatUntilDate string          `json:"RepeatUntilDate"`
}

type RepeatOnInfo struct {
	Monday    bool `json:"Monday"`
	Tuesday   bool `json:"Tuesday"`
	Wednesday bool `json:"Wednesday"`
	Thursday  bool `json:"Thursday"`
	Friday    bool `json:"Friday"`
	Saturday  bool `json:"Saturday"`
	Sunday    bool `json:"Sunday"`
}

// CalendarEventAPI mirrors the BrightSpace calendar event JSON structure
// with string-based timestamps for browser-based fetching.
type CalendarEventAPI struct {
	CalendarEventId  int              `json:"CalendarEventId"`
	OrgUnitId        int              `json:"OrgUnitId"`
	Title            string           `json:"Title"`
	Description      string           `json:"Description"`
	IsAllDayEvent    bool             `json:"IsAllDayEvent"`
	StartDateTime    string           `json:"StartDateTime"`
	EndDateTime      string           `json:"EndDateTime"`
	IsRecurring      bool             `json:"IsRecurring"`
	RecurrenceInfo   *RecurrenceInfo  `json:"RecurrenceInfo,omitempty"`
	LocationName     string           `json:"LocationName"`
	OrgUnitName      string           `json:"OrgUnitName"`
	OrgUnitCode      string           `json:"OrgUnitCode"`
	EventType        int              `json:"EventType"`
	QuizId           int              `json:"QuizId"`
	AssociatedEntity *AssociatedEntity `json:"AssociatedEntity,omitempty"`
}

// DropboxFolderAPI mirrors the BrightSpace dropbox folder JSON structure
// with string-based timestamps for browser-based fetching.
type DropboxFolderAPI struct {
	Id          int    `json:"Id"`
	Name        string `json:"Name"`
	DueDate     string `json:"DueDate"`
	OrgUnitId   string `json:"-"`
	OrgUnitName string `json:"-"`
	OrgUnitCode string `json:"-"`
}

// QuizAPI mirrors the BrightSpace quizzes API JSON structure.
type QuizAPI struct {
	QuizId              int       `json:"QuizId"`
	Name                string    `json:"Name"`
	IsActive            bool      `json:"IsActive"`
	StartDate           string    `json:"StartDate"`
	EndDate             string    `json:"EndDate"`
	DueDate             string    `json:"DueDate"`
	Description         DescField `json:"Description"`
	SubmissionTimeLimit TimeLimit `json:"SubmissionTimeLimit"`
	AttemptsAllowed     Attempts  `json:"AttemptsAllowed"`
	ActivityId          string    `json:"ActivityId"`
	OrgUnitId           string    `json:"-"`
	OrgUnitName         string    `json:"-"`
	OrgUnitCode         string    `json:"-"`
}

type DescField struct {
	Text        DescText `json:"Text"`
	IsDisplayed bool     `json:"IsDisplayed"`
}

type DescText struct {
	Text string `json:"Text"`
	Html string `json:"Html"`
}

type TimeLimit struct {
	IsEnforced      bool `json:"IsEnforced"`
	TimeLimitValue  int  `json:"TimeLimitValue"`
}

type Attempts struct {
	IsUnlimited             bool `json:"IsUnlimited"`
	NumberOfAttemptsAllowed int  `json:"NumberOfAttemptsAllowed"`
}

// QuizzesResponse mirrors the BrightSpace quizzes API JSON structure.
type QuizzesResponse struct {
	Objects []QuizAPI `json:"Objects"`
	Next    *string   `json:"Next"`
}

// BrightSpaceEntry is the internal representation of a BrightSpace event/due date.
type BrightSpaceEntry struct {
	// Source identifies whether this came from calendar events or dropbox folders.
	Source        SourceType
	Title         string
	OrgUnitId     string
	OrgUnitName   string
	OrgUnitCode   string
	Location      string
	Description   string
	DTStart       time.Time
	DTEnd         time.Time
	IsAllDay      bool
	CalendarEventID int
	QuizID        int
}

type SourceType string

const (
	SourceCalendar SourceType = "calendar"
	SourceDropbox  SourceType = "dropbox"
)
