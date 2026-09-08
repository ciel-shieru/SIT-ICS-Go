package ics

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/ciel-shieru/sit-ics-go/internal/calendar"
)

func GenerateUID(event calendar.Event) string {
	var uidStr string
	if event.Source != "" {
		uidStr = fmt.Sprintf("%s-%s-%s-%s-%s-%s",
			event.Source,
			event.Summary,
			event.Location,
			event.DTStart.Format("2006-01-02"),
			event.DTStart.Format("15:04"),
			event.DTEnd.Format("15:04"),
		)
	} else {
		uidStr = fmt.Sprintf("%s-%s-%s-%s-%s",
			event.Summary,
			event.Location,
			event.DTStart.Format("2006-01-02"),
			event.DTStart.Format("15:04"),
			event.DTEnd.Format("15:04"),
		)
	}
	hash := sha256.Sum256([]byte(uidStr))
	return hex.EncodeToString(hash[:])
}

func EscapeText(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, ";", "\\;")
	s = strings.ReplaceAll(s, ",", "\\,")
	s = strings.ReplaceAll(s, "\n", "\\n")
	return s
}
