package app

import (
	"time"

	"github.com/ciel-shieru/sit-ics-go/internal/calendar"
)

func saveAllOutputs(cache *calendar.ICSCache, mainPath, onlinePath, campusPath, bsEventsPath, bsDropboxPath, bsQuizzesPath, tz string, refreshInterval time.Duration, mainAlerts, onlineAlerts, campusAlerts, bsEventsAlerts, bsDropboxAlerts, bsQuizzesAlerts []calendar.Alert) error {
	return cache.SaveOutputs(calendar.Outputs{
		Main:      mainPath,
		Online:    onlinePath,
		Campus:    campusPath,
		BSEvents:  bsEventsPath,
		BSDropbox: bsDropboxPath,
		BSQuizzes: bsQuizzesPath,
	}, tz, refreshInterval, mainAlerts, onlineAlerts, campusAlerts, bsEventsAlerts, bsDropboxAlerts, bsQuizzesAlerts)
}
