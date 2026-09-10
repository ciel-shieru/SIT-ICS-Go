# AGENTS.md — SIT ICS Go

## Quick start
```bash
go build ./cmd/sit-ics/          # build binary
go test -v -count=1 -race ./...  # run all tests (no browser required)
go vet ./...                     # lint gate (CI)
go mod tidy                      # after adding dependencies
```

**Verification**: `go vet` + `go test -race ./...` are the gates. No separate linter.

## Architecture
```
cmd/sit-ics/main.go              # single binary entrypoint
  createBrowser(cfg) → LocalBrowser or RemoteBrowser
  app.New(cfg, cache, browser, provider, loc) → a.Run()

internal/app/                    # orchestration layer
  app.go    # App struct: Run(), Fetch(), Shutdown()
  fetch.go  # runFetch(): auth → peoplesoft → brightspace → cache.Update → saveAllOutputs
  outputs.go # saveAllOutputs() → cache.SaveOutputs()

internal/browser/                # AuthBrowser interface + Rod impl
  browser.go  # AuthBrowser interface, MockAuthBrowser, Cookie/AuthRequest/AuthResult types
  session.go  # BrowserMode (auto/system/rod/remote), BrowserConfig, sentinel errors
  auth.go     # AuthenticateADFS() — fills credentials, handles MFA, waits for SAML redirect
  cookies.go  # extractCookiesForPage() — multi-domain extraction
  local.go    # LocalBrowser — rod launcher (launchBrowser), Authenticate, FetchTimetable, FetchBrightSpace
  remote.go   # RemoteBrowser — CDP connection, discovers WebSocket URL at runtime
  fetcher.go  # FetchBrightSpace(), RodFetcher (browser-backed JSON API client)
  shared.go   # fetchTimetable() — navigates SSR_SSENRL_LIST.GBL, extracts HTML via page.HTML()

internal/auth/                   # ADFSProvider (wraps AuthBrowser)
  provider.go  # Authenticate(), FetchTimetable(), FetchBrightSpace()

internal/peoplesoft/             # Entry struct + HTML parser
  peoplesoft.go  # Entry: CourseCode, ClassName, Section, Type, Day, StartTime, EndTime, Location
  parser.go      # ParseTimetableHTML(html, year, loc) → []Entry — parses SSR_SSENRL_LIST.GBL HTML
  datetime.go    # ParseEntryDateTime(entry, loc) → (DTStart, DTEnd, error)

internal/calendar/               # RFC 5545 writer + in-memory cache with disk upsert
  event.go     # Event struct: UID, DTStart, DTEnd, Summary, Location, Source, etc.
  render.go    # Render(), EventID() — custom ICS writer, deterministic UID via SHA-256
  cache.go     # ICSCache: RWMutex, Update (non-destructive merge), SaveOutputs (5 files)
  persistence.go # writeAtomic (tmp+rename), parseICS (round-trip reader)
  projection.go  # IsOnline, IsNotOnline, IsCampus (excludes BrightSpace), filterEvents

internal/server/                 # stdlib net/http, 5 ICS endpoints
  server.go    # NewServer() — registers /timetable.ics, /timetable-online.ics, /timetable-campus.ics, /brightspace-events.ics, /brightspace-dropbox.ics
  handlers.go  # newFilteredHandler() — nil filter = all events

internal/scheduler/              # robfig/cron/v3 with configured TZ
  scheduler.go # Scheduler: New(tz) → error, Start(), Stop(), AddJob()

internal/brightspace/            # BrightSpace D2L integration
  models.go    # API response structs, BrightSpaceEntry
  fetch.go     # Client: CheckVersion, FetchCourses, FetchCalendarEvents, FetchDropboxFolders
  filter.go    # Blocklist: course name/id/title/location pattern matching
  timestamp.go # Time parsing utilities
```

Module path: `github.com/ciel-shieru/sit-ics-go`

## Fetch flow
```
Run() → scheduler.Start() → goroutine: Fetch() → runFetch():
  1. provider.Authenticate(ctx) → browser.AuthBrowser.Authenticate()
  2. provider.FetchTimetable(ctx, "", loc) → browser.FetchTimetable() → SSR_SSENRL_LIST.GBL HTML
  3. peoplesoft.ParseTimetableHTML(html, year, loc) → []Entry
  4. peoplesoft.ParseEntryDateTime(entry, loc) → DTStart, DTEnd → calendar.Event
  5. (if BrightSpace enabled) provider.FetchBrightSpace(ctx, baseURL) → []BrightSpaceEntry → calendar.Event
  6. (if BrightSpace enabled) cache.RemoveWhere(blocklist.Matches) — delete blocked events
  7. cache.Update(icsEvents, tz) — non-destructive merge by UID
  8. cache.SaveOutputs(5 paths) — atomic write main, online, campus, brightspace-events, brightspace-dropbox
```

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
| `ICS_STORAGE_PATH` | ./timetable.ics | Main ICS file (for disk load) |
| `ICS_ONLINE_PATH` | ./timetable-online.ics | Online-only events ICS |
| `ICS_CAMPUS_PATH` | ./timetable-campus.ics | Campus-only events ICS (excludes BrightSpace) |
| `BRIGHTSPACE_EVENTS_PATH` | ./brightspace-events.ics | BrightSpace calendar events ICS |
| `BRIGHTSPACE_DROPBOX_PATH` | ./brightspace-dropbox.ics | BrightSpace dropbox due dates ICS |
| `BROWSER_MODE` | auto | auto/system/rod/remote |
| `BROWSER_EXECUTABLE` | — | Explicit browser binary path |
| `BROWSER_REMOTE_HOST` | — | Remote browser host (IP or FQDN) |
| `BROWSER_REMOTE_PORT` | 9222 | Remote browser HTTP port |
| `BROWSER_HEADLESS` | true | Headless mode |
| `BROWSER_DEBUG` | false | Enable debug logging for browser actions |
| `PROXY_URL` | — | SOCKS5 proxy URL |
| `ICS_REFRESH_INTERVAL` | 1h | ICS REFRESH-INTERVAL property (RFC 7986 DURATION) |
| `BRIGHTSPACE_ENABLED` | false | Enable BrightSpace D2L extraction |
| `BRIGHTSPACE_BASE_URL` | https://xsite.singaporetech.edu.sg | BrightSpace D2L base URL |
| `BRIGHTSPACE_API_KEY` | — | BrightSpace API key (unused for browser fetch) |
| `BRIGHTSPACE_API_SECRET` | — | BrightSpace API secret (unused for browser fetch) |
| `BRIGHTSPACE_COURSE_NAME_BLOCKLIST` | — | Comma-separated course name patterns to block |
| `BRIGHTSPACE_COURSE_ID_BLOCKLIST` | — | Comma-separated OrgUnitIds to block |
| `BRIGHTSPACE_EVENT_TITLE_BLOCKLIST` | — | Comma-separated event title patterns to block |
| `BRIGHTSPACE_EVENT_LOCATION_BLOCKLIST` | — | Comma-separated event location patterns to block |

`START_DATE` and `END_DATE` are parsed by config but no longer drive the fetch loop.

## Key constraints
- **Go 1.26.5** — pinned in go.mod. Do not upgrade without verifying rod compatibility.
- **No third-party ICS library** — custom writer in `internal/calendar/render.go` for full RFC 5545 control.
- **Rod is the only browser dep** — rest of codebase never imports `github.com/go-rod/rod`.
- **Credentials via env only** — `USERNAME`, `PASSWORD`, `TOTP_SECRET`. Never logged, never persisted.
- **ICS upsert semantics** — non-destructive merge by UID. Events absent from new data are retained. Changing the UID formula breaks idempotency.
- **Timezone** — all time ops use `TZ` env var (default `Asia/Singapore`). `time.Local` is set at startup.
- **Browser modes** — `auto`, `system`, `rod` all use `LocalBrowser`; `remote` uses `RemoteBrowser`. Controlled by `BROWSER_MODE`.
- **Auth flow** — browser follows ADFS redirect naturally after MFA; no SAMLResponse extraction (ADR-0010).
- **Cookie extraction** — extracts cookies from `*.singaporetech.edu.sg` domains. Tries current page first, falls back to `in4sit.singaporetech.edu.sg` and `fs.singaporetech.edu.sg`.
- **Browser-based fetch** — `fetchTimetable()` navigates to `SSR_SSENRL_LIST.GBL`, waits for stable, extracts full HTML via `page.HTML()`. Returns all scheduled classes in one response (ADR-0012).
- **Browser lifecycle** — browser stays open after `Authenticate()` for `FetchTimetable()` and `FetchBrightSpace()`. Always call `browser.Close()` on shutdown.
- **5 ICS output files** — main, online, campus, brightspace-events, brightspace-dropbox. `IsCampus` excludes BrightSpace events (ADR-0013).
- **Server endpoints** — 5 HTTP handlers match the 5 output files (ADR-0014).

## Gotchas
- **Rod API**: `page.Element()` returns `(*Element, error)` — not chainable. `element.Click(proto.InputMouseButtonLeft, 1)` uses proto params. `element.Input()`, `element.Visible()` return `(error)` or `(bool, error)`. Never use `page.MustQuery()` (does not exist).
- **Rod navigation**: Uses `page.WaitNavigation(proto.PageLifecycleEventNameNetworkAlmostIdle)()` followed by `page.WaitStable(5000)`. Stable wait failure is logged but not fatal.
- **Rod cookies**: Extracted via `page.Cookies([]string{})` with fallback to explicit domain queries for `in4sit.singaporetech.edu.sg` and `fs.singaporetech.edu.sg`.
- **Sentinel errors** — defined in `internal/browser/session.go`: `ErrBrowserUnavailable`, `ErrBrowserLaunch`, `ErrBrowserConnect`, `ErrNavigation`, `ErrAuthentication`, `ErrAuthenticationTimeout`, `ErrCredentialExtraction`.
- **peoplesoft.ParseTimetableHTML** returns `[]Entry{}` (empty slice, not nil), `nil` when no entries found.
- **peoplesoft.ExtractYear("")** returns current year — the `year` param is effectively unused.
- **Scheduler.New(tz)** returns `(*Scheduler, error)` — validates timezone.
- **Remote browser discovery**: For `remote` mode, WebSocket URL discovered at runtime via `http://host:port/json/version/`. Per-session UUID changes on Chromium restart — discover fresh on every `Authenticate()` call.
- **Shutdown**: Signal handler waits for in-flight fetch (30s timeout), stops scheduler, closes browser, then calls `cache.SaveOutputs()` for all 5 paths.
- **ICS UID format**: `SHA-256(summary-location-date-start-end)` for PeopleSoft events; `SHA-256(source-summary-location-date-start-end)` for BrightSpace events (source prefix). Changing any component breaks idempotency.
- **App.Fetch() is reentrant-guarded**: Uses mutex + `fetching` flag to prevent concurrent fetches.
- **Events sorted deterministically**: Primary by DTStart, secondary by Location, tiebreaker by UID.

## Testing
- `go test -v -count=1 -race ./...` — runs all unit tests, no browser or Chromium required.
- `internal/browser/browser_test.go` — sentinel errors, wrapping, `MockAuthBrowser`, `ErrAuthenticationTimeout`.
- `internal/calendar/cache_test.go` — upsert, load/save, dirty tracking, filter online/campus, BrightSpace exclusion, deterministic output, event retention, idempotency.
- `internal/peoplesoft/parser_test.go` — HTML parsing for the list endpoint layout, time parsing, date computation.
- `internal/totp/totp_test.go` — TOTP generation determinism and time-window behavior.
- `internal/config/validate_test.go` — config validation.
- Browser integration and E2E tests are excluded (require Chromium / staging ADFS).

## CI/CD
- **CI** (`.github/workflows/ci.yaml`): External PR gate (requires `ci-approved` label or collaborator), `go vet`, `go test -v -count=1 -race ./...`, cross-arch build (linux/amd64 + linux/arm64), debug Docker image push to GHCR.
- **Release** (`.github/workflows/release.yaml`): Semver tag gate, builds 5 platforms (windows/amd64, linux/amd64, linux/arm64, darwin/amd64, darwin/arm64), generates SBOM (spdx-json), creates GitHub Release, builds+pushes production Docker image, signs with Cosign.

## ADRs
All decisions in `docs/adr/`. Index with status in `docs/adr/README.md`.
- 0001: ADFS auth + PeopleSoft timetable via HTTP client (superseded by 0005+0006+0012)
- 0002: Config via env + CLI
- 0003: ICS cache with upsert
- 0004: Cron scheduling and timezone handling
- 0005: Browser abstraction (AuthBrowser interface)
- 0006: Auth flow with Rod waiting primitives
- 0007: Error model with sentinel errors
- 0008: Three-layer testing strategy
- 0009: SAMLResponse submission (superseded by 0010)
- 0010: Natural browser redirect for ADFS→PeopleSoft SAML
- 0011: Browser-based fetch instead of HTTP client (superseded by 0012)
- 0012: Switch to SSR_SSENRL_LIST endpoint (single fetch, list layout)
- 0013: BrightSpace D2L event integration
- 0014: xsite ICS HTTP endpoints (5 endpoints)
- 0015: Codebase restructuring for production readiness
