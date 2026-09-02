# AGENTS.md — SIT ICS Go

## Quick start
```bash
go build ./cmd/sit-ics/          # build binary
go test ./...                     # run all tests
go mod tidy                       # after adding dependencies
```

## Architecture
```
cmd/sit-ics/main.go              # single binary entrypoint
internal/config/                  # env vars + CLI flags (ADR-0002)
internal/browser/                 # AuthBrowser interface + Rod impl (ADR-0005, ADR-0007)
  local.go   — headless Chromium via rod launcher
  remote.go  — CDP connection to external browser
  errors.go  # sentinel errors (ErrBrowserConnect, ErrAuthentication, etc.)
  browser.go # AuthBrowser interface, MockAuthBrowser for tests
internal/auth/                    # ADFS auth flow using AuthBrowser (ADR-0006)
internal/totp/                    # RFC 6238 TOTP (~50 lines, stdlib only)
internal/peoplesoft/              # XML/HTML timetable parsing from PeopleSoft
internal/ics/                     # ICS writer + in-memory cache with disk upsert (ADR-0003)
  ics.go     # RFC 5545 writer, deterministic UID via SHA-256
  cache.go   # RWMutex cache, atomic writes (tmp + rename), preserve historical events
internal/server/                  # stdlib net/http, serves /timetable.ics
internal/scheduler/               # robfig/cron/v3 with configured TZ
```

## Key constraints
- **Go 1.26** — pinned in go.mod. Do not upgrade without verifying rod compatibility.
- **No third-party ICS library** — custom writer (~100 lines) for full RFC 5545 control.
- **Rod is the only browser dep** — rest of codebase never imports `github.com/go-rod/rod`.
- **Credentials via env only** — `USERNAME`, `PASSWORD`, `TOTP_SECRET`. Never logged, never persisted.
- **ICS upsert semantics** — never delete old events; merge by deterministic UID.
- **Timezone** — all time ops use `TZ` env var (default `Asia/Singapore`). Set `time.Local` at startup.
- **Browser modes** — `auto`, `system`, `rod`, `remote`. Controlled by `BROWSER_MODE`.

## Dependencies
| Package | Purpose |
|---------|---------|
| `github.com/caarlos0/env/v10` | Env var → struct parsing |
| `github.com/spf13/pflag` | CLI flags with env fallback |
| `github.com/go-rod/rod` | Headless browser automation |
| `github.com/robfig/cron/v3` | Cron scheduling |
| `golang.org/x/net/html` | HTML table parsing |

## Testing
- Unit tests run with `go test ./...` — no browser required.
- `internal/browser/browser_test.go` — tests sentinel errors, wrapping, `MockAuthBrowser`.
- `internal/auth/auth_test.go` — tests auth flow with mocked browser + peoplesoft client.
- `internal/ics/cache_test.go` — tests upsert, load/save, dirty tracking.
- `internal/totp/totp_test.go` — tests TOTP generation determinism and time-window behavior.
- Browser integration and E2E tests are excluded (require Chromium / staging ADFS).

## Configuration
All via env vars with CLI flag override:
| Env var | Default | Description |
|---------|---------|-------------|
| `USERNAME` | — | ADFS username |
| `PASSWORD` | — | ADFS password |
| `TOTP_SECRET` | — | Azure MFA TOTP secret |
| `START_DATE` | now | First week to fetch (YYYY-MM-DD) |
| `END_DATE` | start+90d | Last week to fetch |
| `TZ` | Asia/Singapore | IANA timezone |
| `FETCH_CRON` | `0 1 * * *` | Cron schedule |
| `SERVER_PORT` | 8080 | HTTP listen port |
| `ICS_STORAGE_PATH` | ./timetable.ics | ICS file path |
| `BROWSER_MODE` | auto | auto/system/rod/remote |
| `BROWSER_EXECUTABLE` | — | Explicit browser binary path |
| `BROWSER_CONTROL_URL` | — | CDP URL for remote mode |
| `BROWSER_HEADLESS` | true | Headless mode |
| `PROXY_URL` | — | SOCKS5 proxy URL (e.g. socks5://localhost:1080) |

## Gotchas
- **Rod API**: `page.MustQuery()` does not exist — use `page.Element()` which returns `(*Element, error)`. Element methods like `.Input()`, `.Click()`, `.Visible()` do not chain with `.Context()` — they return `(bool, error)` or `error` directly.
- **Rod cookies**: Use `page.MustCookies(url)` — not raw proto calls.
- **Cron.New** returns `*Cron` only (no error). Logger interface requires `Error(error, string, ...)` method.
- **ICS UID**: Deterministic SHA-256 hash of `summary+location+date+start+end`. Changing the formula breaks idempotency.
- **peoplesoft.Entry** lives in `internal/peoplesoft`, not `internal/auth`. The `PeoplesoftClient` interface returns `[]peoplesoft.Entry`.
- **Shutdown**: Signal handler stops scheduler and flushes dirty ICS cache to disk. Always call `cache.SaveToFile()` on shutdown.

## ADRs
All decisions are documented in `docs/adr/`:
- 0001: ADFS auth (superseded by 0005+0006)
- 0002: Config via env + CLI
- 0003: ICS cache with upsert
- 0004: Cron scheduling
- 0005: Browser abstraction (AuthBrowser interface)
- 0006: Auth flow with Rod waiting primitives
- 0007: Error model with sentinel errors
- 0008: Three-layer testing strategy
