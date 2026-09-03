package peoplesoft

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
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

	client := NewClient(server.URL, "", false, false)

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

	client := NewClient(server.URL, "", false, false)

	_, err := client.FetchTimetable(context.Background(), "2026-09-07")
	if err != nil {
	}

	_, err = client.FetchTimetable(context.Background(), "2026-09-14")
	if err != nil {
	}
}
