package app

import (
	"time"

	"github.com/ciel-shieru/sit-ics-go/internal/calendar"
)

func saveAllOutputs(cache *calendar.ICSCache, mainPath, onlinePath, campusPath, bsEventsPath, bsDropboxPath, tz string, refreshInterval time.Duration) error {
	return cache.SaveOutputs(calendar.Outputs{
		Main:      mainPath,
		Online:    onlinePath,
		Campus:    campusPath,
		BSEvents:  bsEventsPath,
		BSDropbox: bsDropboxPath,
	}, tz, refreshInterval)
}
