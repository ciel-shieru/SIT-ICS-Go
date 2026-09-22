# ADR-0018: JSON Request Logging and Trusted Proxy Support

## Status

Accepted

## Date

2026-09-21

## Context

The SIT ICS Go HTTP server has no request-level logging. When debugging access patterns, caching behavior, or investigating issues, there is no structured log of what requests were made, by whom, and with what response codes. This makes it difficult to audit access or debug conditional request handling (ETag/If-None-Match, Last-Modified/If-Modified-Since).

When the server runs behind a reverse proxy (e.g., nginx, Traefik, cloud load balancer), the L4 client IP (`r.RemoteAddr`) is the proxy's IP, not the original client. The `X-Forwarded-For` header contains the original client IP, but it must not be trusted unconditionally — a malicious client can set a fake `X-Forwarded-For` header directly.

## Decision Drivers

- **Structured logging**: Logs should be machine-parseable for aggregation and analysis
- **Stdlib-only**: No third-party logging libraries; consistent with existing `log.Printf` usage in the codebase
- **Safe XFF handling**: Only trust `X-Forwarded-For` from known proxy IPs
- **Conditional request visibility**: Capture ETag and Last-Modified response headers to debug caching behavior
- **Minimal overhead**: One JSON marshal per request; no blocking I/O
- **No new dependencies**: Only stdlib (`encoding/json`, `net/http`, `fmt`, `time`)

## Considered Options

### Option A: Hand-rolled JSON via encoding/json (chosen)

Write a middleware that wraps the `*http.ServeMux`, captures request/response details, and emits one JSON line per request to stdout using `encoding/json`.

### Option B: log/slog JSON handler

Use Go's `slog` package with a JSON handler to produce structured logs. Would require migrating from existing `log.Printf` usage throughout the codebase to a new logging paradigm.

### Option C: Third-party logging library

Use a production-grade structured logger (e.g., zap, zerolog, logrus). Adds a dependency and introduces a new logging paradigm inconsistent with the existing codebase.

## Decision

We will choose **Option A: Hand-rolled JSON via encoding/json**.

A `loggingMiddleware` wraps the `*http.ServeMux` in `Server.Start()`. It captures:

1. L4 client IP from `r.RemoteAddr` (port-stripped, IPv4-mapped IPv6 normalized)
2. XFF-derived client IP (only when L4 IP matches a trusted proxy CIDR)
3. Request path and method
4. Conditional request headers (`If-None-Match`, `If-Modified-Since`, `User-Agent`)
5. Response headers (`ETag`, `Last-Modified`) captured after handler completes
6. Response status code (default 200 if handler writes body without `WriteHeader`)

Each request produces exactly one JSON object on one line, printed to stdout via `fmt.Println`.

Trusted proxy configuration is controlled by `SERVER_TRUSTED_PROXIES` env var and `--server-trusted-proxies` CLI flag. Default is empty string (trust nothing). Value format is comma-separated CIDRs or bare IPs (bare IP = /32 or /128). Whitespace is trimmed.

XFF client IP derivation:
- If L4 IP is not in the trusted set → `xff_ip` is always `"-"` (XFF ignored)
- If L4 IP is trusted → parse XFF header, iterate right-to-left, return first entry that is a valid IP AND not in the trusted set
- If all entries are valid and all trusted → return the leftmost valid entry
- If no valid entries → `"-"`

## Consequences

### Positive

- **Structured machine-parseable logs**: Each request emits a single JSON line to stdout, suitable for log aggregation tools (Fluentd, Vector, etc.)
- **Safe XFF handling**: XFF is only honored when the L4 peer is a trusted proxy, preventing IP spoofing
- **No new dependencies**: Only stdlib packages used
- **Conditional request debugging**: ETag and Last-Modified response headers are captured for debugging cache validation
- **Configurable trust**: Deployment can configure trusted proxy CIDRs when running behind a reverse proxy
- **Default-deny**: Trusts nothing by default, requiring explicit configuration

### Negative

- **Per-request marshal overhead**: Each request allocates a map and marshals it to JSON
- **Stdout vs stderr separation**: Logs go to stdout; error messages and server startup messages go to stdout via `fmt.Printf` — no stream separation
- **Manual field management**: Adding new log fields requires updating the map and tests manually
- **No log levels**: All requests are logged at the same level; no filtering by severity
- **No request duration**: Current implementation does not measure request processing time (noted as follow-up)

### Neutral / Operational

- **Deployment configuration**: When running behind a reverse proxy, operators must configure `SERVER_TRUSTED_PROXIES` to see real client IPs
- **JSON null vs string "-""**: Missing optional headers produce JSON `null`; missing XFF produces `"-"` string — consumers should handle both

## Follow-Ups

- Consider migrating to `log/slog` for a more structured logging approach across the codebase
- Add request duration field (measure time from middleware entry to handler exit)
- Consider log sampling for high-traffic deployments
- Consider adding request ID / trace ID support for distributed tracing

## References

- `internal/server/middleware.go` — `loggingMiddleware`, `trustedProxyMatcher`, `loggingResponseWriter`
- `internal/server/server.go` — middleware wiring in `Start()`
- `internal/config/config.go` — `ServerTrustedProxies` config field
- `internal/config/flags.go` — `--server-trusted-proxies` CLI flag
- `internal/config/validate.go` — `validateServerTrustedProxies` validation
- **ADR-0014**: Xsite ICS HTTP Endpoints (existing server infrastructure)
- **ADR-0015**: Codebase Restructuring for Production Readiness (server package location)
