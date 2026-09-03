package peoplesoft

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/ciel-shieru/sit-ics-go/internal/browser"
)

func TestClientUsesCookieJar(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookies := r.Cookies()
		w.Header().Set("Set-Cookie", "test_cookie=hello123; Path=/")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<FIELD><FIELD><NAME>_HTML</NAME><VALUE>test</VALUE></FIELD></FIELD>`))
		_ = cookies
	}))
	defer server.Close()

	client := NewClient(server.URL, "", false, false, false)

	if client.httpClient.Jar == nil {
		t.Fatal("expected cookie jar to be set, got nil")
	}
}

func TestClientCookieJarPersistence(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if requestCount == 1 {
			w.Header().Set("Set-Cookie", "persisted=first_request; Path=/")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`<FIELD><FIELD><NAME>_HTML</NAME><VALUE>test</VALUE></FIELD></FIELD>`))
		} else {
			cookies := r.Cookies()
			found := false
			for _, c := range cookies {
				if c.Name == "persisted" && c.Value == "first_request" {
					found = true
					break
				}
			}
			if !found {
				t.Error("expected persisted cookie to be sent on second request")
			}
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`<FIELD><FIELD><NAME>_HTML</NAME><VALUE>test</VALUE></FIELD></FIELD>`))
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, "", false, false, false)

	_, err := client.FetchTimetable(context.Background(), "2026-09-07")
	if err != nil {
	}

	_, err = client.FetchTimetable(context.Background(), "2026-09-14")
	if err != nil {
	}
}

func TestClientSetCookieUpdatesWithoutDeletingExisting(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if requestCount == 1 {
			w.Header().Set("Set-Cookie", "new_cookie=from_server; Path=/")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`<FIELD><FIELD><NAME>_HTML</NAME><VALUE>test</VALUE></FIELD></FIELD>`))
		} else {
			cookies := r.Cookies()
			hasOriginal := false
			hasNew := false
			for _, c := range cookies {
				if c.Name == "original" && c.Value == "preloaded" {
					hasOriginal = true
				}
				if c.Name == "new_cookie" && c.Value == "from_server" {
					hasNew = true
				}
			}
			if !hasOriginal {
				t.Error("expected original preloaded cookie to still be present after Set-Cookie update")
			}
			if !hasNew {
				t.Error("expected new cookie from Set-Cookie header to be present on second request")
			}
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`<FIELD><FIELD><NAME>_HTML</NAME><VALUE>test</VALUE></FIELD></FIELD>`))
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, "", false, false, false)
	parsedURL, _ := url.Parse(server.URL)
	client.SetCookies([]browser.Cookie{
		{Name: "original", Value: "preloaded", Domain: parsedURL.Hostname(), Path: "/"},
	})

	_, err := client.FetchTimetable(context.Background(), "2026-09-07")
	if err != nil {
	}

	_, err = client.FetchTimetable(context.Background(), "2026-09-14")
	if err != nil {
	}
}
