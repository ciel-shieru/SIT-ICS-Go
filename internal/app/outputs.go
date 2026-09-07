package app

import (
	"time"

	"github.com/ciel-shieru/sit-ics-go/internal/ics"
)

func saveAll(cache *ics.ICSCache, mainPath, onlinePath, campusPath, xsiteEventsPath, xsiteDropboxPath, tz string, refreshInterval time.Duration) error {
	return cache.SaveAllWithXsiteFiles(mainPath, onlinePath, campusPath, xsiteEventsPath, xsiteDropboxPath, tz, refreshInterval)
}
