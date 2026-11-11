package app

import (
	"time"

	"github.com/ciel-shieru/sit-ics-go/internal/calendar"
)

func saveAllOutputs(cache *calendar.ICSCache, mainPath, onlinePath, campusPath, examsPath, xsiteEventsPath, xsiteDropboxPath, xsiteQuizzesPath, xsitePath, tz string, refreshInterval time.Duration, mainAlerts, onlineAlerts, campusAlerts, examsAlerts, xsiteEventsAlerts, xsiteDropboxAlerts, xsiteQuizzesAlerts []calendar.Alert) error {
	return cache.SaveOutputs(calendar.Outputs{
		Main:         mainPath,
		Online:       onlinePath,
		Campus:       campusPath,
		Exams:        examsPath,
		XsiteEvents:  xsiteEventsPath,
		XsiteDropbox: xsiteDropboxPath,
		XsiteQuizzes: xsiteQuizzesPath,
		Xsite:        xsitePath,
	}, tz, refreshInterval, mainAlerts, onlineAlerts, campusAlerts, examsAlerts, xsiteEventsAlerts, xsiteDropboxAlerts, xsiteQuizzesAlerts)
}
