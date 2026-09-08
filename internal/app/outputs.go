package app

import (
	"time"

	"github.com/ciel-shieru/sit-ics-go/internal/calendar"
)

func saveAllOutputs(cache *calendar.ICSCache, mainPath, onlinePath, campusPath, xsiteEventsPath, xsiteDropboxPath, tz string, refreshInterval time.Duration) error {
	return cache.SaveOutputs(calendar.Outputs{
		Main:      mainPath,
		Online:    onlinePath,
		Campus:    campusPath,
		BSEvents:  xsiteEventsPath,
		BSDropbox: xsiteDropboxPath,
	}, tz, refreshInterval)
}
