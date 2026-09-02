package browser

import "errors"

var (
	ErrBrowserUnavailable   = errors.New("browser unavailable")
	ErrBrowserLaunch        = errors.New("browser launch failed")
	ErrBrowserConnect       = errors.New("browser connection failed")
	ErrNavigation           = errors.New("navigation failed")
	ErrAuthentication       = errors.New("authentication failed")
	ErrAuthenticationTimeout = errors.New("authentication timed out")
	ErrCredentialExtraction = errors.New("credential extraction failed")
)
