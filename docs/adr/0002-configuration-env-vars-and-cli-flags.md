# ADR-0002: Configuration via Environment Variables (Primary) with CLI Flags (Fallback)

## Status

Proposed

## Date

2026-09-01

## Context

The application needs a configuration mechanism to accept user settings including:
- ADFS credentials: `USERNAME`, `PASSWORD`, `TOTP_SECRET`
- Date range: `START_DATE`, `END_DATE`
- Timezone: `TZ` (default `Asia/Singapore`)
- Cron schedule: `FETCH_CRON` (default `0 1 * * *`)
- Server port: `SERVER_PORT` (default `8080`)
- ICS storage path: `ICS_STORAGE_PATH` (default `./timetable.ics`)
- Browser mode: `BROWSER_MODE` (default `auto`)
- Browser executable: `BROWSER_EXECUTABLE` (optional, for explicit browser path)
- Browser control URL: `BROWSER_CONTROL_URL` (for remote mode)
- Browser headless: `BROWSER_HEADLESS` (default `true`)
- Per-user database settings for in-app settings storage

The configuration must support both environment variables (primary) and CLI flags (fallback), with the same underlying config options available through both mechanisms.

## Decision Drivers

- **Security-first**: Sensitive values (credentials) should not appear in process listings or logs.
- **Convenience**: CLI flags allow quick overrides without modifying environment.
- **Operational clarity**: Environment variables are the canonical source for production deployments.
- **Idempotent behavior**: Same config from either source produces identical behavior.
- **Go 1.26 compatibility**: Libraries must support current Go version.

## Considered Options

### Option A: `caarlos0/env` + `spf13/pflag` (Selected)

Use `github.com/caarlos0/env/v10` for environment variable parsing into a Go struct, and `github.com/spf13/pflag` for CLI flag parsing with fallback to env var values.

### Option B: Pure `os.Getenv` + `flag` (stdlib only)

Use only Go stdlib: `os.Getenv` for env vars and `flag` package for CLI flags.

### Option C: Pure CLI flags with defaults from env

Use `spf13/viper` or similar to unify env vars and CLI flags into a single configuration layer with hierarchical defaults.

### Option D: YAML/JSON config file + env override

Read a config file (YAML/JSON) as base, override with env vars, then with CLI flags.

## Decision

We will choose **Option A: `caarlos0/env` + `spf13/pflag`**.

## Rationale

### Why Option A over Option B

- **Type safety**: `caarlos0/env` supports struct tags for type conversion (e.g., parsing durations, booleans, custom types), reducing boilerplate and error-prone manual parsing.
- **Cleaner code**: A single config struct with env tags is more maintainable than scattered `os.Getenv` calls with manual type conversions.
- **CLI flag integration**: `spf13/pflag` provides POSIX-style flags (`--flag`, `-f`), shell completion support, and flag grouping — far superior to stdlib `flag`.
- **Fallback semantics**: `pflag` supports `Lookup().Changed` to detect whether a flag was explicitly set, enabling clean fallback: CLI flag → env var → hardcoded default.

### Why Option A over Option C (Viper)

- **Minimal dependencies**: Viper adds significant transitive dependencies. The requirement to reduce third-party libraries favors the lighter `caarlos0/env` + `spf13/pflag` combination.
- **Simpler mental model**: Two focused libraries vs. Viper's complex hierarchy (config file, env, flags, defaults, remote sources). The requirement is simpler: env primary, CLI fallback.
- **Explicit over implicit**: Viper's automatic env binding can lead to subtle bugs (e.g., case sensitivity, nested key mapping). Explicit tag-based parsing is more transparent.

### Why Option A over Option D (Config file)

- **Security**: Storing credentials in a config file on disk is a security risk. The requirement specifies credentials via env vars only.
- **Operational simplicity**: Containerized/deployed environments typically inject config via env vars. A config file adds deployment complexity.
- **Per-user settings in database**: The requirement states per-user settings are stored in the database, not in a config file.

### Why Option A over Status Quo (no solution)

This is a greenfield project. Option A establishes a clean, secure, and maintainable configuration pattern.

## Implementation Details

### Config Struct

```go
type BrowserMode string

const (
    BrowserAuto   BrowserMode = "auto"
    BrowserSystem BrowserMode = "system"
    BrowserRod    BrowserMode = "rod"
    BrowserRemote BrowserMode = "remote"
)

type Config struct {
    Username        string        `env:"USERNAME" envDefault:""`
    Password        string        `env:"PASSWORD" envDefault:""`
    TOTPSecret      string        `env:"TOTP_SECRET" envDefault:""`
    StartDate       time.Time     `env:"START_DATE" envDefault:""`
    EndDate         time.Time     `env:"END_DATE" envDefault:""`
    TZ              string        `env:"TZ" envDefault:"Asia/Singapore"`
    FetchCron       string        `env:"FETCH_CRON" envDefault:"0 1 * * *"`
    ServerPort      int           `env:"SERVER_PORT" envDefault:"8080"`
    ICSStoragePath  string        `env:"ICS_STORAGE_PATH" envDefault:"./timetable.ics"`
    
    // Browser configuration (see ADR-0005)
    BrowserMode      BrowserMode `env:"BROWSER_MODE" envDefault:"auto"`
    BrowserExecutable string     `env:"BROWSER_EXECUTABLE" envDefault:""`
    BrowserControlURL string     `env:"BROWSER_CONTROL_URL" envDefault:""`
    BrowserHeadless   bool       `env:"BROWSER_HEADLESS" envDefault:"true"`
}
```

### Browser Mode Resolution

- `auto` (default): Try system browser → fall back to Rod-managed Chromium
- `system`: Require installed Chrome/Edge/Chromium
- `rod`: Use Rod-managed Chromium (auto-download)
- `remote`: Connect to externally managed browser via `BROWSER_CONTROL_URL`

### Security Note

Security-sensitive Chromium arguments (`--no-sandbox`, `--disable-setuid-sandbox`, etc.) are **implementation-owned** by the browser package (ADR-0005), not exposed as configuration knobs. Application configuration exposes only:

```
BROWSER_MODE
BROWSER_EXECUTABLE
BROWSER_CONTROL_URL
BROWSER_HEADLESS
```

### Flag-then-Env Fallback Pattern

```go
fs := pflag.NewFlagSet("app", pflag.ContinueOnError)
username := fs.String("username", "", "ADFS username")
password := fs.String("password", "", "ADFS password")
// ... other flags

fs.Parse(os.Args[1:])

// Apply CLI flag values if explicitly set, otherwise use env
if fs.Lookup("username").Changed {
    cfg.Username = *username
} else {
    // caarlos0/env handles the rest
}
```

Actually, the cleaner pattern is: parse env into config struct first, then override with CLI flags if changed.

### Per-User Settings

Per-user settings (non-sensitive, application-specific) are stored in a local SQLite database (using `github.com/mattn/go-sqlite3` or `modernc.org/sqlite` — pure Go alternative). These are separate from the config struct and managed independently.

## Consequences

### Positive
- Credentials never appear in process listings (`ps aux`) when using env vars.
- CLI flags provide quick overrides for development and debugging.
- Clean, type-safe config parsing with minimal code.
- Each config option has a single source of truth (env) with an explicit override path (CLI).

### Negative
- Two libraries add to the dependency count (though both are minimal and well-maintained).
- Users must understand the precedence: CLI flag > env var > default.
- `caarlos0/env` does not support complex nested structs with env tags — flat config only.

### Neutral / Operational
- Development: `APP_USERNAME=user APP_PASSWORD=pass ./app --server-port 9090`
- Production: Docker/Kubernetes env var injection is the primary mechanism.
- The `TZ` default ensures correct behavior even if the host system timezone differs.

## Alternatives Considered

### Option B: Pure stdlib

**Summary**: `os.Getenv` + `flag` from Go stdlib.

**Benefits**: Zero third-party dependencies for config.

**Costs**: Manual type conversion boilerplate, no flag grouping or completion, less ergonomic API.

**Reason rejected**: The boilerplate and reduced ergonomics outweigh the benefit of avoiding two small, well-maintained libraries.

### Option C: Viper

**Summary**: `github.com/spf13/viper` for unified config with hierarchical defaults.

**Benefits**: Single library, supports config files, env, flags, and remote sources.

**Costs**: Large dependency tree (~20 transitive deps), complex behavior that can obscure bugs.

**Reason rejected**: Overkill for the requirement (env primary, CLI fallback). The minimal-dependency principle favors lighter alternatives.

### Option D: Config File

**Summary**: YAML/JSON config file with env override.

**Benefits**: Persistent configuration, easy to version control (for non-sensitive settings).

**Costs**: Security risk for credentials, adds deployment complexity, not aligned with the requirement.

**Reason rejected**: Credentials must be env-only. Per-user settings go to database. A config file serves neither purpose.

## Follow-Ups

- Related ADRs: ADR-0001 (ADFS auth), ADR-0003 (ICS caching), ADR-0004 (cron scheduling).
- Implementation tracking lives outside this ADR.

## References

- [caarlos0/env](https://github.com/caarlos0/env) — Environment variable parsing for Go structs.
- [spf13/pflag](https://github.com/spf13/pflag) — POSIX-style command line flags compatible with Go's `flag` package.
- [go-sqlite3](https://github.com/mattn/go-sqlite3) — SQLite3 driver for Go (if CGO is acceptable).
- [modernc.org/sqlite](https://github.com/mattn/go-sqlite3) — Pure Go SQLite implementation (no CGO).
- [ADR-0005](0005-browser-abstraction-layer-with-rod.md) — Browser abstraction layer (browser configuration)
