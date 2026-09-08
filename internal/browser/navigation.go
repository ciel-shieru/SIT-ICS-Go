package browser

import (
	"github.com/go-rod/rod"
)

func getPageURL(page *rod.Page) string {
	var url string
	page.Eval("() => window.location.href", &url)
	return url
}
