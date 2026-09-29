package config

import "github.com/ciel-shieru/sit-ics-go/internal/browser"

type BrowserMode = browser.BrowserMode

const (
	BrowserAuto   BrowserMode = browser.BrowserModeAuto
	BrowserSystem BrowserMode = browser.BrowserModeSystem
	BrowserManaged BrowserMode = browser.BrowserModeManaged
	BrowserRemote BrowserMode = browser.BrowserModeRemote
)
