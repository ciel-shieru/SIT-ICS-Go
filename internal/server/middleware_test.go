package server

import (
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestTrustedProxyMatcher_New(t *testing.T) {
	tests := []struct {
		name    string
		cidrs   string
		wantErr bool
	}{
		{"empty", "", false},
		{"single_ipv4", "10.0.0.1", false},
		{"single_ipv6", "::1", false},
		{"ipv4_cidr", "192.168.0.0/16", false},
		{"ipv6_cidr", "2001:db8::/32", false},
		{"mixed", "10.0.0.1,192.168.0.0/16", false},
		{"invalid_ip", "not_an_ip", true},
		{"invalid_cidr", "192.168.1.0/33", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, err := newTrustedProxyMatcher(tt.cidrs)
			if (err != nil) != tt.wantErr {
				t.Errorf("newTrustedProxyMatcher(%q) error = %v, wantErr %v", tt.cidrs, err, tt.wantErr)
			}
			if m == nil && !tt.wantErr {
				t.Error("newTrustedProxyMatcher returned nil without error")
			}
		})
	}
}

func TestTrustedProxyMatcher_Contains(t *testing.T) {
	m, err := newTrustedProxyMatcher("10.0.0.0/8,192.168.0.0/16,::1")
	if err != nil {
		t.Fatalf("newTrustedProxyMatcher() error: %v", err)
	}

	tests := []struct {
		ip    string
		want  bool
	}{
		{"10.1.2.3", true},
		{"10.0.0.1", true},
		{"192.168.1.1", true},
		{"192.168.255.255", true},
		{"::1", true},
		{"1.2.3.4", false},
		{"172.16.0.1", false},
		{"2001:db8::1", false},
	}

	for _, tt := range tests {
		t.Run(tt.ip, func(t *testing.T) {
			got := m.Contains(netParseIP(t, tt.ip))
			if got != tt.want {
				t.Errorf("Contains(%q) = %v, want %v", tt.ip, got, tt.want)
			}
		})
	}
}

func TestTrustedProxyMatcher_EmptyConfig(t *testing.T) {
	m, err := newTrustedProxyMatcher("")
	if err != nil {
		t.Fatalf("newTrustedProxyMatcher(\"\") error: %v", err)
	}
	if m.Contains(netParseIP(t, "10.0.0.1")) {
		t.Error("empty config should trust nothing")
	}
}

func TestLoggingResponseWriter_WriteHeader(t *testing.T) {
	lrw := newLoggingResponseWriter(httptest.NewRecorder())
	lrw.WriteHeader(http.StatusCreated)
	if lrw.Status() != http.StatusCreated {
		t.Errorf("Status() = %d, want %d", lrw.Status(), http.StatusCreated)
	}
}

func TestLoggingResponseWriter_Write_Auto200(t *testing.T) {
	lrw := newLoggingResponseWriter(httptest.NewRecorder())
	lrw.Write([]byte("hello"))
	if lrw.Status() != http.StatusOK {
		t.Errorf("Status() = %d, want %d", lrw.Status(), http.StatusOK)
	}
}

func TestLoggingResponseWriter_Write_AfterWriteHeader(t *testing.T) {
	lrw := newLoggingResponseWriter(httptest.NewRecorder())
	lrw.WriteHeader(http.StatusNotFound)
	lrw.Write([]byte("not found"))
	if lrw.Status() != http.StatusNotFound {
		t.Errorf("Status() = %d, want %d", lrw.Status(), http.StatusNotFound)
	}
}

func TestLoggingResponseWriter_Header_Delegation(t *testing.T) {
	lrw := newLoggingResponseWriter(httptest.NewRecorder())
	lrw.Header().Set("X-Custom", "value")
	if got := lrw.Header().Get("X-Custom"); got != "value" {
		t.Errorf("Header().Get() = %q, want %q", got, "value")
	}
}

func TestLoggingResponseWriter_DefaultStatus(t *testing.T) {
	lrw := newLoggingResponseWriter(httptest.NewRecorder())
	if lrw.Status() != http.StatusOK {
		t.Errorf("default Status() = %d, want %d", lrw.Status(), http.StatusOK)
	}
}

func TestNewLoggingMiddleware(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	tests := []struct {
		name    string
		proxies string
		wantErr bool
	}{
		{"valid", "10.0.0.0/8", false},
		{"empty", "", false},
		{"invalid", "not_an_ip", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewLoggingMiddleware(handler, tt.proxies)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewLoggingMiddleware(%q) error = %v, wantErr %v", tt.proxies, err, tt.wantErr)
			}
		})
	}
}

func TestServeHTTP_LogLine(t *testing.T) {
	var logged string
	oldWriter := requestLogWriter
	requestLogWriter = &singleLineWriter{f: func(line string) { logged = line }}
	defer func() { requestLogWriter = oldWriter }()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("ETag", `"test-etag"`)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	mw, err := NewLoggingMiddleware(handler, "10.0.0.0/8")
	if err != nil {
		t.Fatalf("NewLoggingMiddleware() error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/timetable.ics", nil)
	req.RemoteAddr = "10.0.0.5:12345"
	rec := httptest.NewRecorder()

	mw.ServeHTTP(rec, req)

	if logged == "" {
		t.Fatal("no log line emitted")
	}

	var entry map[string]any
	if err := json.Unmarshal([]byte(logged), &entry); err != nil {
		t.Fatalf("invalid JSON: %v\nline: %s", err, logged)
	}

	expectedKeys := []string{"ts", "remote_ip", "xff_ip", "path", "method", "user_agent", "etag", "last_modified", "status", "xff_modified", "resp_bytes"}
	for _, key := range expectedKeys {
		if _, ok := entry[key]; !ok {
			t.Errorf("missing key %q in log entry", key)
		}
	}

	if entry["remote_ip"] != "10.0.0.5" {
		t.Errorf("remote_ip = %v, want 10.0.0.5", entry["remote_ip"])
	}
	if entry["xff_ip"] != "-" {
		t.Errorf("xff_ip = %v, want -", entry["xff_ip"])
	}
	if entry["path"] != "/timetable.ics" {
		t.Errorf("path = %v, want /timetable.ics", entry["path"])
	}
	if entry["method"] != "GET" {
		t.Errorf("method = %v, want GET", entry["method"])
	}
	if entry["etag"] != `"test-etag"` {
		t.Errorf("etag = %v, want \"test-etag\"", entry["etag"])
	}
	if entry["status"] != float64(200) {
		t.Errorf("status = %v, want 200", entry["status"])
	}
	if entry["xff_modified"] != false {
		t.Errorf("xff_modified = %v, want false", entry["xff_modified"])
	}
	if entry["resp_bytes"] != float64(2) {
		t.Errorf("resp_bytes = %v, want 2", entry["resp_bytes"])
	}
}

func TestServeHTTP_304WithETag(t *testing.T) {
	var logged string
	oldWriter := requestLogWriter
	requestLogWriter = &singleLineWriter{f: func(line string) { logged = line }}
	defer func() { requestLogWriter = oldWriter }()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("ETag", `"wot"` + `/"abc123"`)
		w.WriteHeader(http.StatusNotModified)
	})

	mw, err := NewLoggingMiddleware(handler, "10.0.0.0/8")
	if err != nil {
		t.Fatalf("NewLoggingMiddleware() error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/xsite.ics", nil)
	req.RemoteAddr = "192.168.1.10:54321"
	rec := httptest.NewRecorder()

	mw.ServeHTTP(rec, req)

	if logged == "" {
		t.Fatal("no log line emitted")
	}

	var entry map[string]any
	if err := json.Unmarshal([]byte(logged), &entry); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if entry["status"] != float64(304) {
		t.Errorf("status = %v, want 304", entry["status"])
	}
}

func TestServeHTTP_XFF_UntrustedL4(t *testing.T) {
	var logged string
	oldWriter := requestLogWriter
	requestLogWriter = &singleLineWriter{f: func(line string) { logged = line }}
	defer func() { requestLogWriter = oldWriter }()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mw, err := NewLoggingMiddleware(handler, "10.0.0.0/8")
	if err != nil {
		t.Fatalf("NewLoggingMiddleware() error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/timetable.ics", nil)
	req.RemoteAddr = "1.2.3.4:12345"
	req.Header.Set("X-Forwarded-For", "5.6.7.8")
	rec := httptest.NewRecorder()

	mw.ServeHTTP(rec, req)

	var entry map[string]any
	json.Unmarshal([]byte(logged), &entry)
	if entry["xff_ip"] != "-" {
		t.Errorf("xff_ip = %v, want - (untrusted L4 should ignore XFF)", entry["xff_ip"])
	}
}

func TestServeHTTP_XFF_TrustedL4_SingleHop(t *testing.T) {
	var logged string
	oldWriter := requestLogWriter
	requestLogWriter = &singleLineWriter{f: func(line string) { logged = line }}
	defer func() { requestLogWriter = oldWriter }()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mw, err := NewLoggingMiddleware(handler, "10.0.0.0/8")
	if err != nil {
		t.Fatalf("NewLoggingMiddleware() error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/timetable.ics", nil)
	req.RemoteAddr = "10.0.0.1:12345"
	req.Header.Set("X-Forwarded-For", "5.6.7.8")
	rec := httptest.NewRecorder()

	mw.ServeHTTP(rec, req)

	var entry map[string]any
	json.Unmarshal([]byte(logged), &entry)
	if entry["xff_ip"] != "5.6.7.8" {
		t.Errorf("xff_ip = %v, want 5.6.7.8", entry["xff_ip"])
	}
}

func TestServeHTTP_XFF_TrustedL4_MultiHopAllTrusted(t *testing.T) {
	var logged string
	oldWriter := requestLogWriter
	requestLogWriter = &singleLineWriter{f: func(line string) { logged = line }}
	defer func() { requestLogWriter = oldWriter }()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mw, err := NewLoggingMiddleware(handler, "10.0.0.0/8,172.16.0.0/12")
	if err != nil {
		t.Fatalf("NewLoggingMiddleware() error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/timetable.ics", nil)
	req.RemoteAddr = "10.0.0.1:12345"
	req.Header.Set("X-Forwarded-For", "172.16.0.5, 10.0.0.2")
	rec := httptest.NewRecorder()

	mw.ServeHTTP(rec, req)

	var entry map[string]any
	json.Unmarshal([]byte(logged), &entry)
	if entry["xff_ip"] != "172.16.0.5" {
		t.Errorf("xff_ip = %v, want 172.16.0.5 (leftmost when all trusted)", entry["xff_ip"])
	}
}

func TestServeHTTP_XFF_TrustedL4_NoXFF(t *testing.T) {
	var logged string
	oldWriter := requestLogWriter
	requestLogWriter = &singleLineWriter{f: func(line string) { logged = line }}
	defer func() { requestLogWriter = oldWriter }()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mw, err := NewLoggingMiddleware(handler, "10.0.0.0/8")
	if err != nil {
		t.Fatalf("NewLoggingMiddleware() error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/timetable.ics", nil)
	req.RemoteAddr = "10.0.0.1:12345"
	rec := httptest.NewRecorder()

	mw.ServeHTTP(rec, req)

	var entry map[string]any
	json.Unmarshal([]byte(logged), &entry)
	if entry["xff_ip"] != "-" {
		t.Errorf("xff_ip = %v, want -", entry["xff_ip"])
	}
}

func TestServeHTTP_XFF_MalformedXFF(t *testing.T) {
	var logged string
	oldWriter := requestLogWriter
	requestLogWriter = &singleLineWriter{f: func(line string) { logged = line }}
	defer func() { requestLogWriter = oldWriter }()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mw, err := NewLoggingMiddleware(handler, "10.0.0.0/8")
	if err != nil {
		t.Fatalf("NewLoggingMiddleware() error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/timetable.ics", nil)
	req.RemoteAddr = "10.0.0.1:12345"
	req.Header.Set("X-Forwarded-For", ", , ")
	rec := httptest.NewRecorder()

	mw.ServeHTTP(rec, req)

	var entry map[string]any
	json.Unmarshal([]byte(logged), &entry)
	if entry["xff_ip"] != "-" {
		t.Errorf("xff_ip = %v, want -", entry["xff_ip"])
	}
}

func TestServeHTTP_XFF_TrustedL4_MultiHopMixed(t *testing.T) {
	var logged string
	oldWriter := requestLogWriter
	requestLogWriter = &singleLineWriter{f: func(line string) { logged = line }}
	defer func() { requestLogWriter = oldWriter }()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mw, err := NewLoggingMiddleware(handler, "10.0.0.0/8")
	if err != nil {
		t.Fatalf("NewLoggingMiddleware() error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/timetable.ics", nil)
	req.RemoteAddr = "10.0.0.1:12345"
	req.Header.Set("X-Forwarded-For", "172.16.0.5, 10.0.0.2")
	rec := httptest.NewRecorder()

	mw.ServeHTTP(rec, req)

	var entry map[string]any
	json.Unmarshal([]byte(logged), &entry)
	if entry["xff_ip"] != "172.16.0.5" {
		t.Errorf("xff_ip = %v, want 172.16.0.5 (rightmost untrusted)", entry["xff_ip"])
	}
}

func TestServeHTTP_HeaderCapture_Present(t *testing.T) {
	var logged string
	oldWriter := requestLogWriter
	requestLogWriter = &singleLineWriter{f: func(line string) { logged = line }}
	defer func() { requestLogWriter = oldWriter }()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mw, err := NewLoggingMiddleware(handler, "")
	if err != nil {
		t.Fatalf("NewLoggingMiddleware() error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/timetable.ics", nil)
	req.Header.Set("If-None-Match", `"abc"`)
	req.Header.Set("If-Modified-Since", "Mon, 01 Jan 2024 00:00:00 GMT")
	req.Header.Set("User-Agent", "TestAgent/1.0")
	rec := httptest.NewRecorder()

	mw.ServeHTTP(rec, req)

	var entry map[string]any
	json.Unmarshal([]byte(logged), &entry)
	if entry["user_agent"] != "TestAgent/1.0" {
		t.Errorf("user_agent = %v, want TestAgent/1.0", entry["user_agent"])
	}
}

func TestServeHTTP_HeaderCapture_Absent(t *testing.T) {
	var logged string
	oldWriter := requestLogWriter
	requestLogWriter = &singleLineWriter{f: func(line string) { logged = line }}
	defer func() { requestLogWriter = oldWriter }()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mw, err := NewLoggingMiddleware(handler, "")
	if err != nil {
		t.Fatalf("NewLoggingMiddleware() error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/timetable.ics", nil)
	rec := httptest.NewRecorder()

	mw.ServeHTTP(rec, req)

	var entry map[string]any
	json.Unmarshal([]byte(logged), &entry)
	if entry["user_agent"] != nil {
		t.Errorf("user_agent = %v, want nil", entry["user_agent"])
	}
}

func TestServeHTTP_JSONValidity_AllKeys(t *testing.T) {
	var logged string
	oldWriter := requestLogWriter
	requestLogWriter = &singleLineWriter{f: func(line string) { logged = line }}
	defer func() { requestLogWriter = oldWriter }()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("ETag", `"test"`)
		w.Header().Set("Last-Modified", "Mon, 01 Jan 2024 00:00:00 GMT")
		w.WriteHeader(http.StatusOK)
	})

	mw, err := NewLoggingMiddleware(handler, "")
	if err != nil {
		t.Fatalf("NewLoggingMiddleware() error: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/xsite.ics", nil)
	req.Header.Set("If-None-Match", `"old"`)
	req.Header.Set("If-Modified-Since", "Sun, 31 Dec 2023 00:00:00 GMT")
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.RemoteAddr = "192.168.1.1:9999"
	rec := httptest.NewRecorder()

	mw.ServeHTTP(rec, req)

	var entry map[string]any
	if err := json.Unmarshal([]byte(logged), &entry); err != nil {
		t.Fatalf("invalid JSON: %v\nline: %s", err, logged)
	}

	keys := []string{"ts", "remote_ip", "xff_ip", "path", "method", "user_agent", "etag", "last_modified", "status", "xff_modified", "resp_bytes"}
	for _, k := range keys {
		if _, ok := entry[k]; !ok {
			t.Errorf("missing key %q", k)
		}
	}
	if len(entry) != len(keys) {
		t.Errorf("expected %d keys, got %d", len(keys), len(entry))
	}
}

func TestServeHTTP_LogInjection_UserAgent(t *testing.T) {
	var logged string
	oldWriter := requestLogWriter
	requestLogWriter = &singleLineWriter{f: func(line string) { logged = line }}
	defer func() { requestLogWriter = oldWriter }()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mw, err := NewLoggingMiddleware(handler, "")
	if err != nil {
		t.Fatalf("NewLoggingMiddleware() error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/timetable.ics", nil)
	req.Header.Set("User-Agent", "Inject\r\nFAKE: header")
	rec := httptest.NewRecorder()

	mw.ServeHTTP(rec, req)

	var entry map[string]any
	if err := json.Unmarshal([]byte(logged), &entry); err != nil {
		t.Fatalf("invalid JSON from injected UA: %v", err)
	}
	lines := strings.Split(logged, "\n")
	if len(lines) != 1 {
		t.Fatalf("log line split into %d lines (log injection possible), want 1", len(lines))
	}
}

func TestServeHTTP_LastModified_Absent(t *testing.T) {
	var logged string
	oldWriter := requestLogWriter
	requestLogWriter = &singleLineWriter{f: func(line string) { logged = line }}
	defer func() { requestLogWriter = oldWriter }()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mw, err := NewLoggingMiddleware(handler, "")
	if err != nil {
		t.Fatalf("NewLoggingMiddleware() error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/timetable.ics", nil)
	rec := httptest.NewRecorder()

	mw.ServeHTTP(rec, req)

	var entry map[string]any
	json.Unmarshal([]byte(logged), &entry)
	if entry["last_modified"] != nil {
		t.Errorf("last_modified = %v, want nil", entry["last_modified"])
	}
}

func TestServeHTTP_ETag_Absent(t *testing.T) {
	var logged string
	oldWriter := requestLogWriter
	requestLogWriter = &singleLineWriter{f: func(line string) { logged = line }}
	defer func() { requestLogWriter = oldWriter }()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mw, err := NewLoggingMiddleware(handler, "")
	if err != nil {
		t.Fatalf("NewLoggingMiddleware() error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/timetable.ics", nil)
	rec := httptest.NewRecorder()

	mw.ServeHTTP(rec, req)

	var entry map[string]any
	json.Unmarshal([]byte(logged), &entry)
	if entry["etag"] != nil {
		t.Errorf("etag = %v, want nil", entry["etag"])
	}
}

func TestServeHTTP_XFFModified_TrustedL4WithXFF(t *testing.T) {
	var logged string
	oldWriter := requestLogWriter
	requestLogWriter = &singleLineWriter{f: func(line string) { logged = line }}
	defer func() { requestLogWriter = oldWriter }()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mw, err := NewLoggingMiddleware(handler, "10.0.0.0/8")
	if err != nil {
		t.Fatalf("NewLoggingMiddleware() error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/timetable.ics", nil)
	req.RemoteAddr = "10.0.0.1:12345"
	req.Header.Set("X-Forwarded-For", "5.6.7.8")
	rec := httptest.NewRecorder()

	mw.ServeHTTP(rec, req)

	var entry map[string]any
	if err := json.Unmarshal([]byte(logged), &entry); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if entry["xff_modified"] != true {
		t.Errorf("xff_modified = %v, want true", entry["xff_modified"])
	}
}

func TestServeHTTP_XFFModified_UntrustedL4(t *testing.T) {
	var logged string
	oldWriter := requestLogWriter
	requestLogWriter = &singleLineWriter{f: func(line string) { logged = line }}
	defer func() { requestLogWriter = oldWriter }()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mw, err := NewLoggingMiddleware(handler, "10.0.0.0/8")
	if err != nil {
		t.Fatalf("NewLoggingMiddleware() error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/timetable.ics", nil)
	req.RemoteAddr = "1.2.3.4:12345"
	req.Header.Set("X-Forwarded-For", "5.6.7.8")
	rec := httptest.NewRecorder()

	mw.ServeHTTP(rec, req)

	var entry map[string]any
	if err := json.Unmarshal([]byte(logged), &entry); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if entry["xff_modified"] != false {
		t.Errorf("xff_modified = %v, want false", entry["xff_modified"])
	}
}

func TestServeHTTP_XFFModified_NoXFF(t *testing.T) {
	var logged string
	oldWriter := requestLogWriter
	requestLogWriter = &singleLineWriter{f: func(line string) { logged = line }}
	defer func() { requestLogWriter = oldWriter }()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mw, err := NewLoggingMiddleware(handler, "10.0.0.0/8")
	if err != nil {
		t.Fatalf("NewLoggingMiddleware() error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/timetable.ics", nil)
	req.RemoteAddr = "10.0.0.1:12345"
	rec := httptest.NewRecorder()

	mw.ServeHTTP(rec, req)

	var entry map[string]any
	if err := json.Unmarshal([]byte(logged), &entry); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if entry["xff_modified"] != false {
		t.Errorf("xff_modified = %v, want false", entry["xff_modified"])
	}
}

func TestServeHTTP_RespBytes(t *testing.T) {
	var logged string
	oldWriter := requestLogWriter
	requestLogWriter = &singleLineWriter{f: func(line string) { logged = line }}
	defer func() { requestLogWriter = oldWriter }()

	expectedBody := "hello world"
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(expectedBody))
	})

	mw, err := NewLoggingMiddleware(handler, "")
	if err != nil {
		t.Fatalf("NewLoggingMiddleware() error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/timetable.ics", nil)
	rec := httptest.NewRecorder()

	mw.ServeHTTP(rec, req)

	var entry map[string]any
	if err := json.Unmarshal([]byte(logged), &entry); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	respBytes, ok := entry["resp_bytes"].(float64)
	if !ok {
		t.Fatalf("resp_bytes is not a number: %v", entry["resp_bytes"])
	}
	if int(respBytes) != len(expectedBody) {
		t.Errorf("resp_bytes = %v, want %d", respBytes, len(expectedBody))
	}
}

func TestLoggingResponseWriter_BytesWritten(t *testing.T) {
	lrw := newLoggingResponseWriter(httptest.NewRecorder())
	if lrw.BytesWritten() != 0 {
		t.Errorf("initial BytesWritten() = %d, want 0", lrw.BytesWritten())
	}

	lrw.Write([]byte("hello"))
	if lrw.BytesWritten() != 5 {
		t.Errorf("BytesWritten() after write(5) = %d, want 5", lrw.BytesWritten())
	}

	lrw.Write([]byte(" world"))
	if lrw.BytesWritten() != 11 {
		t.Errorf("BytesWritten() after write(6) = %d, want 11", lrw.BytesWritten())
	}
}

func TestExtractL4IP(t *testing.T) {
	tests := []struct {
		name string
		addr string
		want string
	}{
		{"ipv4", "10.0.0.1:8080", "10.0.0.1"},
		{"ipv6_bracket", "[::1]:8080", "::1"},
		{"ipv4_mapped", "::ffff:192.168.1.1:8080", "192.168.1.1"},
		{"no_port", "10.0.0.1", "10.0.0.1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractL4IP(&http.Request{RemoteAddr: tt.addr})
			if got != tt.want {
				t.Errorf("extractL4IP(%q) = %q, want %q", tt.addr, got, tt.want)
			}
		})
	}
}

func netParseIP(t *testing.T, s string) net.IP {
	t.Helper()
	ip := net.ParseIP(s)
	if ip == nil {
		t.Fatalf("failed to parse IP %q", s)
	}
	return ip
}

type singleLineWriter struct {
	f func(string)
}

func (w *singleLineWriter) Write(p []byte) (int, error) {
	line := strings.TrimSuffix(string(p), "\n")
	w.f(line)
	return len(p), nil
}
