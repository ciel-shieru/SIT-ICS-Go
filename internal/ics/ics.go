package ics

import (
	"strings"

	"github.com/ciel-shieru/sit-ics-go/internal/calendar"
)

func GenerateUID(event calendar.Event) string {
	return calendar.EventID(event)
}

func EscapeText(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, ";", "\\;")
	s = strings.ReplaceAll(s, ",", "\\,")
	s = strings.ReplaceAll(s, "\n", "\\n")
	return s
}
