package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

var requestLogWriter io.Writer = os.Stdout

type trustedProxyMatcher struct {
	cidrs []*net.IPNet
}

func newTrustedProxyMatcher(cidrs string) (*trustedProxyMatcher, error) {
	if cidrs == "" {
		return &trustedProxyMatcher{cidrs: nil}, nil
	}
	var matchers []*net.IPNet
	for _, entry := range strings.Split(cidrs, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		if strings.Contains(entry, "/") {
			_, cidr, err := net.ParseCIDR(entry)
			if err != nil {
				return nil, fmt.Errorf("parse trusted proxy %q: %w", entry, err)
			}
			matchers = append(matchers, cidr)
		} else {
			ip := net.ParseIP(entry)
			if ip == nil {
				return nil, fmt.Errorf("parse trusted proxy %q: not a valid IP", entry)
			}
			bits := 32
			if ip.To4() == nil {
				bits = 128
			}
			_, cidr, err := net.ParseCIDR(ip.String() + "/" + fmt.Sprintf("%d", bits))
			if err != nil {
				return nil, fmt.Errorf("parse trusted proxy %q: %w", entry, err)
			}
			matchers = append(matchers, cidr)
		}
	}
	return &trustedProxyMatcher{cidrs: matchers}, nil
}

func (m *trustedProxyMatcher) Contains(ip net.IP) bool {
	if m == nil || len(m.cidrs) == 0 {
		return false
	}
	for _, cidr := range m.cidrs {
		normalized := ip
		if v4 := ip.To4(); v4 != nil {
			normalized = v4
		}
		if cidr.Contains(normalized) {
			return true
		}
	}
	return false
}

type loggingResponseWriter struct {
	http.ResponseWriter
	status       int
	bytesWritten int64
}

func newLoggingResponseWriter(w http.ResponseWriter) *loggingResponseWriter {
	return &loggingResponseWriter{ResponseWriter: w}
}

func (w *loggingResponseWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *loggingResponseWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	n, err := w.ResponseWriter.Write(b)
	w.bytesWritten += int64(n)
	return n, err
}

func (w *loggingResponseWriter) Status() int {
	if w.status == 0 {
		return http.StatusOK
	}
	return w.status
}

func (w *loggingResponseWriter) BytesWritten() int64 {
	return w.bytesWritten
}

type loggingMiddleware struct {
	handler         http.Handler
	trustedProxies  *trustedProxyMatcher
}

func NewLoggingMiddleware(handler http.Handler, trustedProxies string) (http.Handler, error) {
	matcher, err := newTrustedProxyMatcher(trustedProxies)
	if err != nil {
		return nil, err
	}
	return &loggingMiddleware{
		handler:        handler,
		trustedProxies: matcher,
	}, nil
}

func extractL4IP(r *http.Request) string {
	host := r.RemoteAddr
	if portStripIdx := strings.LastIndex(host, ":"); portStripIdx >= 0 {
		// Check if it's IPv6 with brackets like [::1]:port
		if strings.HasPrefix(host, "[") {
			closeBracket := strings.Index(host, "]")
			if closeBracket > 0 {
				host = host[1:closeBracket]
			}
		} else {
			host = host[:portStripIdx]
		}
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return r.RemoteAddr
	}
	if v4 := ip.To4(); v4 != nil {
		return v4.String()
	}
	return ip.String()
}

func extractXFFIP(r *http.Request, trustedProxies *trustedProxyMatcher) string {
	xff := r.Header.Get("X-Forwarded-For")
	if xff == "" {
		return "-"
	}
	entries := strings.Split(xff, ",")
	validIPs := make([]net.IP, 0, len(entries))
	for _, entry := range entries {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		ip := net.ParseIP(entry)
		if ip != nil {
			validIPs = append(validIPs, ip)
		}
	}
	if len(validIPs) == 0 {
		return "-"
	}
	for i := len(validIPs) - 1; i >= 0; i-- {
		if !trustedProxies.Contains(validIPs[i]) {
			return validIPs[i].String()
		}
	}
	return validIPs[0].String()
}

func (m *loggingMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	l4IP := extractL4IP(r)
	trusted := m.trustedProxies != nil && m.trustedProxies.Contains(net.ParseIP(l4IP))

	var xffIP string
	xffModified := false
	if trusted {
		xffIP = extractXFFIP(r, m.trustedProxies)
		if xffIP != "-" {
			xffModified = true
		}
	} else {
		xffIP = "-"
	}

	lrw := newLoggingResponseWriter(w)

	m.handler.ServeHTTP(lrw, r)

	userAgent := r.Header.Get("User-Agent")
	ifNoneMatch := r.Header.Get("If-None-Match")
	ifModifiedSince := r.Header.Get("If-Modified-Since")

	var etag *string
	etagVal := lrw.Header().Get("ETag")
	if etagVal != "" {
		etag = &etagVal
	}
	var lastModified *string
	lmVal := lrw.Header().Get("Last-Modified")
	if lmVal != "" {
		lastModified = &lmVal
	}

	logEntry := map[string]any{
		"ts":                time.Now().UTC().Format(time.RFC3339Nano),
		"remote_ip":         l4IP,
		"xff_ip":            xffIP,
		"path":              r.URL.Path,
		"method":            r.Method,
		"if_none_match":     userAgentValue(ifNoneMatch),
		"if_modified_since": userAgentValue(ifModifiedSince),
		"user_agent":        userAgentValue(userAgent),
		"etag":              etag,
		"last_modified":     lastModified,
		"status":            lrw.Status(),
		"xff_modified":      xffModified,
		"resp_bytes":        int(lrw.BytesWritten()),
	}

	data, err := json.Marshal(logEntry)
	if err != nil {
		fmt.Fprintf(requestLogWriter, `{"error":"%s","ts":"%s"}`+"\n", err.Error(), time.Now().UTC().Format(time.RFC3339Nano))
		return
	}
	fmt.Fprintln(requestLogWriter, string(data))
}

func userAgentValue(s string) any {
	if s == "" {
		return nil
	}
	return s
}
