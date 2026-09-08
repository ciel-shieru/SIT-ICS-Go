package ics

import "github.com/ciel-shieru/sit-ics-go/internal/calendar"

func GenerateUID(event calendar.Event) string {
	return calendar.EventID(event)
}

func EscapeText(s string) string {
	return calendar.EscapeText(s)
}
