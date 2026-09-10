package calendar

import "time"

type Event struct {
	UID         string
	DTStart     time.Time
	DTEnd       time.Time
	CourseCode  string
	Summary     string
	Title       string
	OrgUnitID   string
	OrgUnitName string
	OrgUnitCode string
	Location    string
	Description string
	Source      string
}
