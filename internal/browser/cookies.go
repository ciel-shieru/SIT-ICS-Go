package browser

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
)

func extractCookiesForPage(page *rod.Page, ctx context.Context) ([]Cookie, error) {
	cookieCtx, cookieCancel := context.WithTimeout(ctx, 10*time.Second)
	defer cookieCancel()
	page = page.Context(cookieCtx)

	var rodCookies []*proto.NetworkCookie
	func() {
		defer func() { recover() }()
		cookies, err := page.Cookies([]string{})
		if err == nil {
			rodCookies = cookies
		}
	}()

	if rodCookies == nil {
		pageURL := getPageURL(page)
		var domains []string
		if !strings.Contains(pageURL, "in4sit.singaporetech.edu.sg") {
			domains = append(domains, "https://in4sit.singaporetech.edu.sg/")
		}
		if !strings.Contains(pageURL, "fs.singaporetech.edu.sg") {
			domains = append(domains, "https://fs.singaporetech.edu.sg/")
		}
		for _, domain := range domains {
			func() {
				defer func() { recover() }()
				extra, err := page.Context(cookieCtx).Cookies([]string{domain})
				if err == nil {
					rodCookies = append(rodCookies, extra...)
				}
			}()
		}
	}

	if rodCookies == nil {
		return nil, fmt.Errorf("%w: page became invalid during cookie extraction", ErrAuthentication)
	}

	cookieMap := make(map[string]Cookie)
	for _, c := range rodCookies {
		if !isSingaporeTechDomain(c.Domain) {
			continue
		}
		expiry := int64(c.Expires)
		existing, exists := cookieMap[c.Name]
		if !exists || expiry > existing.Expiry {
			cookieMap[c.Name] = Cookie{Name: c.Name, Value: c.Value, Domain: c.Domain, Path: c.Path, Expiry: expiry}
		}
	}

	cookies := make([]Cookie, 0, len(cookieMap))
	for _, c := range cookieMap {
		cookies = append(cookies, c)
	}

	return cookies, nil
}

func isSingaporeTechDomain(domain string) bool {
	return strings.Contains(domain, "singaporetech.edu.sg")
}
