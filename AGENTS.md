# AGENTS.md — SIT ICS Go

## Quick start
```bash
go build ./cmd/sit-ics/          # build binary
go test ./...                     # run all tests (no browser required)
go mod tidy                       # after adding dependencies
```

No linter, formatter, or typecheck tool — `go build` + `go test` are the verification gates.

## Architecture
```
cmd/sit-ics/main.go              # single binary entrypoint
internal/config/                  # env vars (caarlos0/env) + CLI flags (spf13/pflag), flags override env
internal/browser/                 # AuthBrowser interface + Rod impl
  browser.go  # AuthBrowser interface, MockAuthBrowser, Cookie/AuthRequest/AuthResult types
  local.go    # headless Chromium via rod launcher (launchBrowser)
  remote.go   # CDP connection to external browser (discovers WebSocket URL at runtime)
  shared.go   # fetchTimetable() — navigates to SSR_SSENRL_LIST.GBL, extracts HTML
  errors.go   # sentinel errors (ErrBrowserLaunch, ErrBrowserConnect, ErrAuthentication, etc.)
internal/auth/                    # ADFS auth + PeoplesoftClient interface
internal/totp/                    # RFC 6238 TOTP via pquerna/otp
internal/peoplesoft/              # Entry struct + HTML parser
internal/ics/                     # RFC 5545 writer + in-memory cache with disk upsert
  ics.go      # custom writer (~70 lines), deterministic UID via SHA-256
  cache.go    # RWMutex cache, atomic writes (tmp+rename), merge by UID, filter online/campus
internal/server/                  # stdlib net/http, serves /timetable.ics
internal/scheduler/               # robfig/cron/v3 with configured TZ
```

Module path: `github.com/ciel-shieru/sit-ics-go`

## Key constraints
- **Go 1.26.5** — pinned in go.mod. Do not upgrade without verifying rod compatibility.
- **No third-party ICS library** — custom writer for full RFC 5545 control.
- **Rod is the only browser dep** — rest of codebase never imports `github.com/go-rod/rod`.
- **Credentials via env only** — `USERNAME`, `PASSWORD`, `TOTP_SECRET`. Never logged, never persisted.
- **ICS upsert semantics** — never delete old events; merge by deterministic UID (SHA-256 of summary+location+date+start+end). Changing the UID formula breaks idempotency.
- **Timezone** — all time ops use `TZ` env var (default `Asia/Singapore`). `time.Local` is set at startup.
- **Browser modes** — `auto`, `system`, `rod`, `remote`. Controlled by `BROWSER_MODE`.
- **Auth flow** — browser follows ADFS redirect naturally after MFA; no SAMLResponse extraction (ADR-0010).
- **Cookie extraction** — extracts all cookies from `*.singaporetech.edu.sg` domains (in4sit + fs.singaporetech).
- **Browser-based fetch** — `fetchTimetable()` navigates to `SSR_SSENRL_LIST.GBL` (no params), waits for stable, extracts full HTML via `page.HTML()`. Returns all scheduled classes in one response (ADR-0012).
- **Browser lifecycle** — browser stays open after `Authenticate()` for `FetchTimetable()`. Always call `browser.Close()` on shutdown.

## Fetch flow
```
Authenticate() → FetchTimetable(ctx, "") → navigate SSR_SSENRL_LIST.GBL → parse HTML → []Entry
```

`FetchTimetable` in `auth.ADFSProvider` accepts a `weekDate` parameter but it is now always called with `""`. The parser extracts the year from HTML content via `peoplesoft.ExtractYear()`.

## Configuration
All via env vars with CLI flag override (flags take priority):
| Env var | Default | Description |
|---------|---------|-------------|
| `USERNAME` | — | ADFS username |
| `PASSWORD` | — | ADFS password |
| `TOTP_SECRET` | — | Azure MFA TOTP secret |
| `TZ` | Asia/Singapore | IANA timezone |
| `FETCH_CRON` | `0 1 * * *` | Cron schedule for fetches |
| `SERVER_PORT` | 8080 | HTTP listen port |
| `ICS_STORAGE_PATH` | ./timetable.ics | Main ICS file path |
| `ICS_ONLINE_PATH` | ./timetable-online.ics | Online-only events ICS |
| `ICS_CAMPUS_PATH` | ./timetable-campus.ics | Non-online events ICS |
| `BROWSER_MODE` | auto | auto/system/rod/remote |
| `BROWSER_EXECUTABLE` | — | Explicit browser binary path |
| `BROWSER_REMOTE_HOST` | — | Remote browser host (IP or FQDN) |
| `BROWSER_REMOTE_PORT` | 9222 | Remote browser HTTP port |
| `BROWSER_HEADLESS` | true | Headless mode |
| `BROWSER_DEBUG` | false | Enable debug logging for Rod events |
| `PROXY_URL` | — | SOCKS5 proxy URL |
| `ICS_REFRESH_INTERVAL` | 1h | ICS refresh interval (RFC 7986 DURATION, e.g. 1h, 30m) |

`START_DATE` and `END_DATE` are parsed by config but no longer drive the fetch loop — the single endpoint returns all classes.

## Gotchas
- **Rod API**: `page.MustQuery()` does not exist — use `page.Element()` which returns `(*Element, error)`. Element methods like `.Input()`, `.Click()`, `.Visible()` do not chain with `.Context()` — they return `(bool, error)` or `error` directly.
- **Rod cookies**: Use `page.MustCookies(url)` — not raw proto calls. For multi-domain extraction, query both `in4sit.singaporetech.edu.sg` and `fs.singaporetech.edu.sg`.
- **Rod stable wait**: `fetchTimetable()` uses `page.WaitStable(5000)` after `WaitNavigation(NetworkAlmostIdle)`. Stable wait failure is logged but not fatal.
- **Cron.New** returns `*Cron` only (no error). The `Scheduler` wraps it and returns errors from `AddJob`. Logger interface requires `Error(error, string, ...)` method.
- **peoplesoft.Entry** lives in `internal/peoplesoft`, not `internal/auth`. Has fields: `CourseCode`, `ClassName`, `Section`, `Type`, `Day`, `StartTime`, `EndTime`, `Location`.
- **AuthResult** — no longer includes `SAMLResponse`. Browser handles SAML redirect naturally; cookies extracted from all singaporetech domains.
- **peoplesoft.ParseTimetableHTML** returns `nil, nil` (not error) when no entries found.
- **peoplesoft.ExtractYear("")** returns current year — the `year` param in `ParseTimetableHTML` is effectively unused.
- **Remote browser discovery**: For `remote` mode, the WebSocket URL is discovered at runtime via `http://host:port/json/version/`. Per-session UUID changes on Chromium restart — discover fresh on every `Authenticate()` call.
- **Shutdown**: Signal handler stops scheduler, closes browser, then calls `cache.SaveAllToFiles()`. Always flush all three ICS paths.
- **ICS UID format**: `SHA-256(summary-location-date(YYYY-MM-DD)-start(15:04)-end(15:04))`. Changing any component breaks idempotency.

## Testing
- `go test ./...` runs all unit tests — no browser or Chromium required.
- `internal/browser/browser_test.go` — sentinel errors, wrapping, `MockAuthBrowser`.
- `internal/ics/cache_test.go` — upsert, load/save, dirty tracking, filter online/campus.
- `internal/peoplesoft/parser_test.go` — HTML parsing for the list endpoint layout.
- `internal/totp/totp_test.go` — TOTP generation determinism and time-window behavior.
- Browser integration and E2E tests are excluded (require Chromium / staging ADFS).

## ADRs
All decisions in `docs/adr/`. Index with status in `docs/adr/README.md`.
- 0001: ADFS auth (superseded by 0005+0006)
- 0002: Config via env + CLI
- 0003: ICS cache with upsert
- 0004: Cron scheduling
- 0005: Browser abstraction (AuthBrowser interface)
- 0006: Auth flow with Rod waiting primitives
- 0007: Error model with sentinel errors
- 0008: Three-layer testing strategy
- 0009: SAMLResponse submission (superseded by 0010)
- 0010: Natural browser redirect for ADFS→PeopleSoft SAML
- 0011: Browser-based fetch instead of HTTP client (superseded by 0012)
- 0012: Switch to SSR_SSENRL_LIST endpoint (single fetch, list layout)
