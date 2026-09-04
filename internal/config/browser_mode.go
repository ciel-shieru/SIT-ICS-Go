package config

type BrowserMode string

const (
	BrowserAuto   BrowserMode = "auto"
	BrowserSystem BrowserMode = "system"
	BrowserRod    BrowserMode = "rod"
	BrowserRemote BrowserMode = "remote"
)
