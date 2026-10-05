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

internal/credretry/              # Interactive credential retry on auth failure (desktop only)
  credretry.go     # PromptIfAuthFailed(), isCredentialError()
  credretry_container.go # No-op stub for container builds

internal/peoplesoft/             # Entry struct + HTML parser
  peoplesoft.go  # Entry: CourseCode, ClassName, Section, Type, Day, StartTime, EndTime, Location
  parser.go      # ParseTimetableHTML(html, year, loc) → []Entry — parses SSR_SSENRL_LIST.GBL HTML
  datetime.go    # ParseEntryDateTime(entry, loc) → (DTStart, DTEnd, error)

internal/calendar/               # RFC 5545 writer + in-memory cache with disk upsert
  event.go     # Event struct: UID, DTStart, DTEnd, Summary, Location, Source, etc.
  render.go    # Render(), EventID() — custom ICS writer, deterministic UID via SHA-256
  cache.go     # ICSCache: RWMutex, Update (non-destructive merge), SaveOutputs (7 files)
  persistence.go # writeAtomic (tmp+rename), parseICS (round-trip reader)
  projection.go  # IsOnline, IsNotOnline, IsCampus (excludes BrightSpace), filterEvents

internal/server/                 # stdlib net/http, 7 ICS endpoints
  server.go    # NewServer() — registers /timetable.ics, /timetable-online.ics, /timetable-campus.ics, /xsite-events.ics, /xsite-dropbox.ics, /xsite-quizzes.ics, /xsite.ics, /wakey-sitizen/timetable.json
  handlers.go  # newFilteredHandler() — nil filter = all events

internal/scheduler/              # robfig/cron/v3 with configured TZ
  scheduler.go # Scheduler: New(tz) → error, Start(), Stop(), AddJob()

internal/smartmerge/             # Smart merge of BrightSpace into PeopleSoft events
  smartmerge.go  # MergeEvents(), DedupQuizzes() — module+time+location matching, Zoom extraction

internal/brightspace/            # BrightSpace D2L integration
  models.go    # API response structs, BrightSpaceEntry, BrightSpaceStringEntry, QuizAPI
  fetch.go     # Client: CheckVersion, FetchCourses, FetchCalendarEvents, FetchDropboxFolders
  filter.go    # Blocklist: course name/id/title/location/quiz pattern matching
  timestamp.go # Time parsing utilities
  dedup.go     # Deduplication helpers
  recurrence.go # Recurring event handling
  quiz_removal.go # Quiz removal based on attempt tracking
  quiz_attempt.go # Quiz submission page HTML parsing
```

Module path: `github.com/ciel-shieru/sit-ics-go`

## Fetch flow
```
Run() → scheduler.Start() → goroutine: Fetch() → runFetch():
  1. provider.Authenticate(ctx) → browser.AuthBrowser.Authenticate()
     — auth has separate 5-min deadline; timetable fetch gets fresh deadline (context isolation)
  2. provider.FetchTimetable(ctx, "", loc) → browser.FetchTimetable() → SSR_SSENRL_LIST.GBL HTML (recoverable — BrightSpace still runs on failure)
  3. peoplesoft.ParseTimetableHTML(html, year, loc) → []Entry (recoverable — BrightSpace still runs on failure)
  4. peoplesoft.ParseEntryDateTime(entry, loc) → DTStart, DTEnd → calendar.Event (recoverable — BrightSpace still runs on failure)
  5. (if xsite enabled) cache.RemoveWhere(blocklist.Matches) — delete blocked brightspace events
  6. (if xsite enabled) provider.FetchBrightSpace(ctx, baseURL) → []BrightSpaceEntry → calendar.Event
  7. (if smart merge enabled) smartmerge.MergeEvents(psEvents, bsEvents) — module+time+location match, extracts Zoom details into PS event description
  8. (if xsite enabled) provider.FetchBrightSpaceQuizzesAPI(ctx, baseURL) → []QuizAPI
  9. (if xsite enabled) brightspace.QuizEntriesToEvents() → quiz calendar events
 10. (if smart merge enabled) smartmerge.DedupQuizzes(calendarEvents, quizEvents) — replaces quiz-associated calendar events with quiz entries
 11. (if quiz attempt tracking enabled) FetchQuizSubmissionPage() + ParseQuizSubmissionHTML() — remove completed quizzes
 12. cache.Update(icsEvents, tz) — non-destructive merge by UID
 13. cache.SaveOutputs(7 paths) — atomic write main, online, campus, xsite-events, xsite-dropbox, xsite-quizzes, xsite (with alerts)
```

## Configuration
All via env vars with CLI flag override (flags take priority):
| Env var | Default | Description |
|---------|---------|-------------|
| `USERNAME` | — | ADFS username (desktop: keyring fallback, container: env var only) |
| `PASSWORD` | — | ADFS password (desktop: keyring fallback, container: env var only) |
| `TOTP_SECRET` | — | Azure MFA TOTP secret (desktop: keyring fallback, container: env var only) |
| `OVERRIDE_CREDENTIALS` | false | Force credential prompt even if credentials exist in keyring (desktop builds only) |
| `TZ` | Asia/Singapore | IANA timezone |
| `FETCH_CRON` | `0 1 * * *` | Cron schedule for fetches |
| `SERVER_PORT` | 42748 | HTTP listen port |
| `SERVER_ADDR` | `127.0.0.1` (desktop) / `0.0.0.0` (container) | HTTP server bind address |
| `ICS_STORAGE_PATH` | ./timetable.ics | Main ICS file (for disk load) |
| `ICS_ONLINE_PATH` | ./timetable-online.ics | Online-only events ICS |
| `ICS_CAMPUS_PATH` | ./timetable-campus.ics | Campus-only events ICS (excludes BrightSpace) |
| `XSITE_EVENTS_PATH` | ./xsite-events.ics | xsite calendar events ICS |
| `XSITE_DROPBOX_PATH` | ./xsite-dropbox.ics | xsite dropbox due dates ICS |
| `XSITE_QUIZZES_PATH` | ./xsite-quizzes.ics | xsite quizzes ICS |
| `XSITE_PATH` | ./xsite.ics | xsite combo ICS (all brightspace sources) |
| `BROWSER_MODE` | auto | auto/system/rod/remote |
| `BROWSER_EXECUTABLE` | — | Explicit browser binary path |
| `BROWSER_REMOTE_HOST` | — | Remote browser host (IP or FQDN) |
| `BROWSER_REMOTE_PORT` | 9222 | Remote browser HTTP port |
| `BROWSER_RETRY_INTERVAL` | 5s | Retry interval between browser auth attempts |
| `BROWSER_MAX_RETRIES` | 3 | Max retries for browser auth attempts |
| `BROWSER_HEADLESS` | true | Headless mode |
| `BROWSER_DEBUG` | false | Enable debug logging for browser actions |
| `PROXY_URL` | — | SOCKS5 proxy URL |
| `ICS_REFRESH_INTERVAL` | 1h | ICS REFRESH-INTERVAL property (RFC 7986 DURATION) |
| `XSITE_ENABLED` | true | Enable xsite D2L extraction (opt-out) |
| `XSITE_COURSE_NAME_BLOCKLIST` | — | Comma-separated course name patterns to block |
| `XSITE_COURSE_CODE_BLOCKLIST` | — | Comma-separated module code patterns to block |
| `XSITE_COURSE_ID_BLOCKLIST` | — | Comma-separated OrgUnitIds to block |
| `XSITE_EVENT_TITLE_BLOCKLIST` | — | Comma-separated event title patterns to block |
| `XSITE_EVENT_LOCATION_BLOCKLIST` | — | Comma-separated event location patterns to block |
| `XSITE_SMART_MERGE_ENABLED` | `true` | Enable smart merge of BrightSpace events into PeopleSoft timetable events |
| `XSITE_QUIZ_TITLE_BLOCKLIST` | — | Comma-separated quiz title patterns to block |
| `XSITE_QUIZ_ATTEMPT_TRACKING_ENABLED` | false | Enable quiz attempt tracking to auto-remove completed quizzes |
| `TIMETABLE_ALERTS` | — | Alert config for main timetable ICS |
| `TIMETABLE_ONLINE_ALERTS` | — | Alert config for online-only ICS |
| `TIMETABLE_CAMPUS_ALERTS` | — | Alert config for campus-only ICS |
| `XSITE_EVENTS_ALERTS` | — | Alert config for xsite events ICS |
| `XSITE_DROPBOX_ALERTS` | — | Alert config for xsite dropbox ICS |
| `XSITE_QUIZZES_ALERTS` | — | Alert config for xsite quizzes ICS |
| `SERVER_TRUSTED_PROXIES` | — | Comma-separated trusted proxy CIDRs for logging |
| `SERVER_DISABLE_CACHING` | false | Disable HTTP caching headers on server responses |

`START_DATE` and `END_DATE` are parsed by config but no longer drive the fetch loop.

## Key constraints
- **Go 1.26.5** — pinned in go.mod. Do not upgrade without verifying rod compatibility.
- **No third-party ICS library** — custom writer in `internal/calendar/render.go` for full RFC 5545 control.
- **Rod is the only browser dep** — rest of codebase never imports `github.com/go-rod/rod`.
- **Credentials sourcing**: Desktop builds (`!container`) source credentials from OS keyring (`sit-ics-go` service) with env var fallback. Container builds (`-tags container`) source from env vars only. CLI flags `--username`, `--password`, `--totp-secret` are removed.
- **Override credentials**: --override-credentials CLI flag (or OVERRIDE_CREDENTIALS env var) forces interactive credential prompt on desktop builds regardless of keyring state. Container builds ignore this flag.
- **Credential retry (desktop only)**: On desktop builds (`!container`), when `runFetch()` encounters a credential-related auth error (invalid password, wrong TOTP), it prompts the user with `[Y/n]` to re-enter credentials. If confirmed, calls `credprompt.PromptIfNeeded()` to overwrite stored credentials, then retries authentication once. Container builds are no-op. Network/browser errors do not trigger credential prompt.
- **ICS upsert semantics** — non-destructive merge by UID. Events absent from new data are retained. Changing the UID formula breaks idempotency.
- **Timezone** — all time ops use `TZ` env var (default `Asia/Singapore`). `time.Local` is set at startup.
- **Browser modes** — `auto`, `system`, `managed` all use `LocalBrowser`; `remote` uses `RemoteBrowser`. Controlled by `BROWSER_MODE`.
- **Auth flow** — browser follows ADFS redirect naturally after MFA; no SAMLResponse extraction (ADR-0010).
- **Cookie extraction** — extracts cookies from `*.singaporetech.edu.sg` domains. Tries current page first, falls back to `in4sit.singaporetech.edu.sg` and `fs.singaporetech.edu.sg`.
- **Browser-based fetch** — `fetchTimetable()` navigates to `SSR_SSENRL_LIST.GBL`, waits for stable, extracts full HTML via `page.HTML()`. Returns all scheduled classes in one response (ADR-0012).
- **Browser lifecycle** — browser stays open after `Authenticate()` for `FetchTimetable()` and `FetchBrightSpace()`. Always call `browser.Close()` on shutdown.
- **7 ICS output files** — main, online, campus, xsite-events, xsite-dropbox, xsite-quizzes, xsite. `IsCampus` excludes BrightSpace events (ADR-0013).
- **Server endpoints** — 8 HTTP handlers: 7 ICS handlers matching the 7 output files (ADR-0014) + `/wakey-sitizen/timetable.json` (ADR-0021).
- **Wakey-SITizen JSON endpoint** (ADR-0021) — `GET /wakey-sitizen/timetable.json` serves the timetable as JSON for the companion mobile app. Excludes `brightspace-*` sources; `event_type` is `"online"` (case-insensitive `Online` location) else `"campus"`; timestamps are RFC 3339 in `TZ` (e.g. `+08:00`); `schema_version` starts at 1 (bump on schema change); `generated_at` is the cache last-modified time; always 200 OK — empty result is `"events":[]`, never 204.
- **Server bind address**: Desktop builds (`!container`) default to `127.0.0.1` (loopback only). Container builds (`container`) default to `0.0.0.0` (all interfaces). Override via `SERVER_ADDR` env var or `--server-addr` CLI flag.
- **Smart merge** — when enabled (`XSITE_SMART_MERGE_ENABLED=true`), BrightSpace events with matching module code, overlapping time, and "Online"/"TBD" PS location + "Zoom Online Meeting" BS location are merged into PeopleSoft events. Zoom meeting details (link, meeting ID, passcode) are extracted from BrightSpace HTML description and appended to PS event description. Matched BrightSpace calendar events are removed from cache (not retained by upsert).
- **Quiz attempt tracking** — when enabled (`XSITE_QUIZ_ATTEMPT_TRACKING_ENABLED=true`), fetches each quiz's submission page, parses attempt count and best score, removes quizzes that are fully attempted (reached max attempts or unlimited with a completed attempt).

## Gotchas
- **"BrightSpace" == "Xsite"** — Codebase internals (package names, config keys, function names like `FetchBrightSpace`) use "BrightSpace". User-facing names (env var prefixes like `XSITE_*`, HTTP endpoints like `/xsite-events.ics`, file names like `xsite.ics`) use "Xsite". They refer to the same thing: BrightSpace D2L content extraction.
- **Rod API**: `page.Element()` returns `(*Element, error)` — not chainable. `element.Click(proto.InputMouseButtonLeft, 1)` uses proto params. `element.Input()`, `element.Visible()` return `(error)` or `(bool, error)`. Never use `page.MustQuery()` (does not exist).
- **Rod navigation**: Uses `page.WaitNavigation(proto.PageLifecycleEventNameNetworkAlmostIdle)()` followed by `page.WaitStable(5000)`. Stable wait failure is logged but not fatal.
- **Rod cookies**: Extracted via `page.Cookies([]string{})` with fallback to explicit domain queries for `in4sit.singaporetech.edu.sg` and `fs.singaporetech.edu.sg`.
- **Sentinel errors** — defined in `internal/browser/session.go`: `ErrBrowserUnavailable`, `ErrBrowserLaunch`, `ErrBrowserConnect`, `ErrNavigation`, `ErrAuthentication`, `ErrAuthenticationTimeout`, `ErrCredentialExtraction`.
- **peoplesoft.ParseTimetableHTML** returns `[]Entry{}` (empty slice, not nil), `nil` when no entries found.
- **peoplesoft.ExtractYear("")** returns current year — the `year` param is effectively unused.
- **Scheduler.New(tz)** returns `(*Scheduler, error)` — validates timezone.
- **Remote browser discovery**: For `remote` mode, WebSocket URL discovered at runtime via `http://host:port/json/version/`. Per-session UUID changes on Chromium restart — discover fresh on every `Authenticate()` call.
- **Shutdown**: Signal handler waits for in-flight fetch (30s timeout), stops scheduler, closes browser, then calls `cache.SaveOutputs()` for all 7 paths.
- **ICS UID format**: `SHA-256(summary-location-date-start-end)` for PeopleSoft events; `SHA-256(source-summary-location-date-start-end)` for BrightSpace events (source prefix). Changing any component breaks idempotency.
- **App.Fetch() is reentrant-guarded**: Uses mutex + `fetching` flag to prevent concurrent fetches.
- **Events sorted deterministically**: Primary by DTStart, secondary by Location, tiebreaker by UID.
- **Credential retry vs automatic retry**: `isRetriable()` returns `false` for auth errors, so the browser-level retry loop does not retry auth failures. The user-facing retry in `runFetch()` is a separate, single-attempt interactive flow (desktop only).
- **Degraded fetch cycle (PeopleSoft failure, ADR-0020)**: A non-fatal PeopleSoft timetable fetch failure does not abort `runFetch()`. BrightSpace fetches, quiz dedup/attempts, `cache.Update()`, and `SaveOutputs()` still run; existing PeopleSoft events are retained by non-destructive UID upsert. During a degraded cycle the blocklist `RemoveWhere` predicate additionally skips non-BrightSpace events (source not prefixed `brightspace-`), and smart merge is skipped (BrightSpace events appended standalone). Only errors rooted at the six browser sentinels (`ErrBrowserUnavailable`, `ErrBrowserLaunch`, `ErrBrowserConnect`, `ErrAuthentication`, `ErrAuthenticationTimeout`, `ErrCredentialExtraction`) remain fatal.
- **Smart merge**: Only matches events where PS location is "Online"/"TBD"/"To Be Advised"/"TBA" AND BS location contains "Zoom Online Meeting". Extracts Zoom link, meeting ID, and passcode from BS HTML description via regex. Matched BS calendar events are explicitly removed from cache before upsert (otherwise non-destructive merge would retain them).
- **Quiz attempt tracking**: Fetches each quiz's D2L submission page, parses attempt count and best score. Removes quizzes where attempts reached max or unlimited with a completed attempt. Uses `brightspace.ParseQuizSubmissionHTML()` and `brightspace.ShouldRemoveQuiz()`.
- **IsCampus filter**: Returns true for non-online events that are NOT sourced from BrightSpace (`source` does not start with `brightspace-`). Used for the campus-only ICS projection.

## Data privacy — no sensitive data in the repo

All data committed to the git repo MUST NOT contain any identifiable, sensitive, revealing, personal, organizational, or company information. Examples include but are not limited to:

- Module names and module codes
- Person names (students, staff, faculty)
- Student information (matriculation numbers, student IDs)
- Staff information (employee IDs, staff numbers)
- Location details (building names, room numbers, physical addresses)
- Zoom tenant identifiers
- Zoom meeting names, meeting IDs, meeting passcodes (including in encoded or encrypted forms)
- Zoom meeting PINs (including in encoded or encrypted forms)
- Any similar personally identifiable or organizationally sensitive information

**Exception**: Domain names (e.g. `singaporetech.edu.sg`) are permitted since they are public-facing and required for the core functionality of this application, which is purpose-built for this specific institution.

When data is needed for tests, configuration examples, or documentation, use **fake data** or well-known sample data only (e.g. `JOHN DOE` for names, fictional module codes like `ALT2501`, fabricated IDs, dummy meeting IDs). Never copy real data from production systems, live exports, screenshots, or actual user records.

## Testing
- `go test -v -count=1 -race ./...` — runs all unit tests, no browser or Chromium required.
- `internal/browser/browser_test.go` — sentinel errors, wrapping, `MockAuthBrowser`, `ErrAuthenticationTimeout`.
- `internal/calendar/cache_test.go` — upsert, load/save, dirty tracking, filter online/campus, BrightSpace exclusion, deterministic output, event retention, idempotency.
- `internal/calendar/alerts_test.go` — alert rendering in ICS output.
- `internal/calendar/persistence_test.go` — atomic write, ICS round-trip parsing.
- `internal/peoplesoft/parser_test.go` — HTML parsing for the list endpoint layout, time parsing, date computation.
- `internal/totp/totp_test.go` — TOTP generation determinism and time-window behavior.
- `internal/config/validate_test.go` — config validation.
- `internal/smartmerge/smartmerge_test.go` — module matching, time overlap, location conditions, Zoom extraction.
- `internal/brightspace/` — filter, dedup, recurrence, quiz removal, timestamp tests.
- `internal/server/middleware_test.go`, `handlers_test.go`, `wakey_test.go` — HTTP middleware, handler, and Wakey-SITizen JSON endpoint tests.
- Browser integration and E2E tests are excluded (require Chromium / staging ADFS).

## CI/CD
- **CI** (`.github/workflows/ci.yaml`): External PR gate (requires `ci-approved` label or collaborator), `go vet`, `go test -v -count=1 -race ./...`, cross-arch build (linux/amd64 + linux/arm64 with `-ldflags="-s -w"` + `CGO_ENABLED="0"`), debug Docker image push to GHCR.
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
- 0016: Secure domain validation for cookie and redirect filtering
- 0017: xsite rebranding (BrightSpace → Xsite user-facing names)
- 0018: JSON request logging and trusted proxy support
- 0019: Interactive credential retry on desktop when authentication fails with invalid credentials or TOTP error
- 0020: Resilient fetch — BrightSpace still runs when PeopleSoft fetch fails
- 0021: Wakey-SITizen JSON timetable endpoint for the companion mobile app
