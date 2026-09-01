# ADR-0004: Cron Scheduling and Timezone Handling

## Status

Proposed

## Date

2026-09-01

## Context

The application must schedule periodic timetable fetches using a cron expression (default: `0 1 * * *` — daily at 1 AM). All dates and times involved in the application — including cron schedules, date ranges (`START_DATE`, `END_DATE`), ICS event timestamps, and timezone identifiers — must follow the `TZ` environment variable. If `TZ` is unspecified, the default is `Asia/Singapore`.

The application runs as a long-lived process (daemon/service) that:
1. Starts at a configured time.
2. Loads ICS data from disk into memory.
3. Serves HTTP requests from memory.
4. Runs the upstream fetch asynchronously on a cron schedule.
5. Gracefully handles timezone transitions (e.g., no DST changes in Singapore, but the system should still respect `TZ` correctly).

## Decision Drivers

- **Single timezone source**: All time operations use one timezone (`TZ` env var).
- **Predictable scheduling**: Cron jobs run at the intended wall-clock time regardless of host system timezone.
- **Go 1.26 compatibility**: Libraries must support the target Go version.
- **Minimal dependencies**: Prefer stdlib or well-maintained, focused libraries.
- **Idempotent scheduling**: Missed or delayed cron runs should not cause duplicate fetches.

## Considered Options

### Option A: `robfig/cron/v3` + `time.LoadLocation` (Selected)

Use `github.com/robfig/cron/v3` for cron expression parsing and scheduling, with Go's `time.LoadLocation` from the `TZ` env var for all timezone operations.

### Option B: Stdlib Timer-Based Scheduling

Use Go's `time.Ticker` or `time.AfterFunc` to implement a simple interval-based scheduler instead of cron expressions.

### Option C: System Cron + Standalone Binary

Rely on the host system's cron daemon to invoke the binary at scheduled intervals, with each run being a short-lived process (fetch → generate ICS → exit).

### Option D: `gobwas/gocron` or Alternative Cron Library

Use an alternative Go cron library with different features (e.g., context support, second-level precision).

## Decision

We will choose **Option A: `robfig/cron/v3` + `time.LoadLocation`**.

## Rationale

### Why Option A over Option B

- **Cron expression support**: The requirement specifies a cron syntax schedule (`0 1 * * *` default). Stdlib timers only support fixed intervals (e.g., every hour), not cron expressions (e.g., "every day at 1 AM", "every Monday at 3 AM").
- **Flexibility**: Cron expressions allow users to specify complex schedules (e.g., "every weekday at 2 AM", "first day of month at midnight") without custom logic.

### Why Option A over Option C (System Cron)

- **Self-contained**: The application manages its own schedule, reducing deployment dependencies (no system cron configuration needed).
- **Stateful**: Running as a long-lived process allows in-memory ICS caching (ADR-0003), which is more efficient than restarting for each fetch.
- **Async update pattern**: The requirement specifies loading ICS from disk on startup, serving from memory, and updating asynchronously — this pattern requires a long-lived process, not short-lived invocations.

### Why Option A over Option D

- **Maturity**: `robfig/cron/v3` is the most widely used cron library in Go, with a large user base and proven reliability.
- **Simplicity**: Alternative libraries add features (second-level precision, context support) that are not required. The default cron second precision is sufficient.
- **Dependency count**: All alternatives add similar dependency counts. `robfig/cron/v3` has zero transitive dependencies.

### Timezone Handling

All time operations use `time.LoadLocation(tz)` where `tz` is the `TZ` env var (default `"Asia/Singapore"`):

- **Cron schedule**: `cron.New(cron.WithLocation(loc))` ensures cron jobs fire at the intended wall-clock time in the specified timezone.
- **Date ranges**: `START_DATE` and `END_DATE` are parsed with the configured timezone.
- **ICS timestamps**: All `DTSTART` and `DTEND` values use `TZID=Asia/Singapore` (or whichever timezone is configured).
- **System time**: All `time.Now()` calls are converted to the configured timezone for display/logic purposes.

## Implementation Details

### Cron Initialization

```go
func newCronScheduler(tz string) (*cron.Cron, error) {
    loc, err := time.LoadLocation(tz)
    if err != nil {
        return nil, fmt.Errorf("invalid timezone %q: %w", tz, err)
    }
    return cron.New(cron.WithLocation(loc), cron.WithLogger(cron.StandardLogger{})), nil
}
```

### Schedule Registration

```go
spec := config.FetchCron // e.g., "0 1 * * *"
_, err := scheduler.AddFunc(spec, func() {
    fetchAndUpdateICS()
})
if err != nil {
    return fmt.Errorf("invalid cron spec %q: %w", spec, err)
}
scheduler.Start()
```

### Timezone Validation

On startup, validate the `TZ` value before proceeding:

```go
loc, err := time.LoadLocation(config.TZ)
if err != nil {
    return fmt.Errorf("invalid TZ %q: %w", config.TZ, err)
}
// Set as the default timezone for all operations
time.Local = loc
```

### Date Range Parsing

```go
parseDate := func(s string) (time.Time, error) {
    return time.ParseInLocation("2006-01-02", s, loc)
}
startDate, err := parseDate(config.StartDate)
endDate, err := parseDate(config.EndDate)
```

### ICS Timestamp Generation

```go
func toICSTime(t time.Time) string {
    return t.In(loc).Format("20060102T150405")
}
```

Output format: `20260907T090000` (UTC-like format with `TZID` parameter in the ICS header).

## Consequences

### Positive
- Cron jobs fire at the correct wall-clock time regardless of host timezone.
- All time operations are consistent — one timezone source (`TZ` env var).
- Flexible scheduling: users can specify any valid cron expression.
- Self-contained: no system cron dependency.
- In-memory caching (ADR-0003) is preserved across cron runs.

### Negative
- If the process is down during a scheduled run, the fetch is skipped (no catch-up). This is acceptable for a timetable app where stale data is better than duplicate/conflicting data.
- `robfig/cron/v3` does not support second-level precision by default (seconds are always `0`). This is sufficient for the "daily at 1 AM" use case.
- Changing `TZ` requires a restart (timezone is loaded once at startup).

### Neutral / Operational
- Singapore does not observe DST, so timezone transitions are not a concern for the default `Asia/Singapore`. However, the system correctly handles timezones that do observe DST.
- The cron logger (`cron.StandardLogger{}`) logs scheduled runs for operational visibility.
- The async fetch pattern means the HTTP server continues serving stale data if the cron job fails — this is acceptable and preferred over blocking responses.

## Alternatives Considered

### Option B: Stdlib Timer

**Summary**: Use `time.Ticker` for fixed-interval scheduling (e.g., every 24 hours).

**Benefits**: Zero dependencies, simple.

**Costs**: No cron expression support, fixed intervals don't align with wall-clock time (e.g., a 24-hour ticker started at 3 AM will run at 3 AM, 3 AM, 3 AM — not "daily at 1 AM").

**Reason rejected**: The requirement specifies cron syntax scheduling. Fixed intervals cannot express "daily at 1 AM" correctly.

### Option C: System Cron

**Summary**: Use host system's cron daemon to invoke the binary periodically.

**Benefits**: Simple, leverages existing infrastructure.

**Costs**: Requires system cron configuration, incompatible with the in-memory caching pattern (each run is a fresh process), no state preservation between runs.

**Reason rejected**: Incompatible with the long-lived process + in-memory cache architecture specified in the requirements.

### Option D: Alternative Cron Library

**Summary**: Use `gobwas/gocron`, `python-cron` port, or similar.

**Benefits**: Some libraries offer context support or second-level precision.

**Costs**: Less mature, smaller user base, no significant advantage over `robfig/cron/v3` for this use case.

**Reason rejected**: `robfig/cron/v3` is the de facto standard Go cron library with zero transitive deps and sufficient feature set.

## Follow-Ups

- Related ADRs: ADR-0001 (ADFS auth), ADR-0002 (config), ADR-0003 (ICS caching).
- Implementation tracking lives outside this ADR.

## References

- [robfig/cron/v3](https://github.com/robfig/cron) — Cron job scheduler for Go.
- [Cron Expression Format](https://en.wikipedia.org/wiki/Cron#Overview) — Standard cron syntax (5-field).
- [Go time.LoadLocation](https://pkg.go.dev/time#LoadLocation) — Timezone loading from IANA database.
