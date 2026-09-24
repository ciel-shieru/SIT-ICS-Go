# ADR-0017: xsite Rebranding of External-Facing BrightSpace Artifacts

## Status

Accepted

## Date

2026-09-20

## Context

The SIT ICS Go project serves BrightSpace D2L calendar events through multiple external-facing artifacts: environment variables, CLI flags, ICS file paths, and HTTP endpoints. These artifacts all used the `brightspace-` or `BRIGHTSPACE_` prefix, which was inconsistent with the project's established `xsite` naming convention for external distribution (already used in ADR-0014 for HTTP endpoints like `/xsite-events.ics` and `/xsite-dropbox.ics`).

The `BrightSpaceBaseURL` configuration field was a redundant abstraction — it always defaulted to `https://xsite.singaporetech.edu.sg` and was only used as a parameter to `FetchBrightSpace()` and `FetchBrightSpaceQuizzes()`. Since this URL is a fixed institutional endpoint, hardcoding it eliminates unnecessary configuration surface without reducing flexibility.

Additionally, users needed a single combined ICS file containing all BrightSpace/xsite events from all three sources (calendar, dropbox, quizzes). This required a new output path and HTTP endpoint.

The rebranding scope was intentionally limited to external-facing artifacts only. Internal code references (event source values like `"brightspace-calendar"`, package names like `brightspace`, filter functions) were preserved to maintain event idempotency and avoid breaking the UID generation formula.

## Decision

Rename all external-facing BrightSpace artifacts to use the `xsite` prefix:

### Configuration fields
- `BrightSpaceBaseURL` — removed entirely (hardcoded to `"https://xsite.singaporetech.edu.sg"`)
- `BrightSpaceAPIKey` — removed entirely
- `BrightSpaceAPISecret` — removed entirely
- `BrightSpaceEnabled` → `XsiteEnabled` (`env:"XSITE_ENABLED"`, default `"false"`)
- `BrightSpaceEventsPath` → `XsiteEventsPath` (`env:"XSITE_EVENTS_PATH"`, default `"./xsite-events.ics"`)
- `BrightSpaceDropboxPath` → `XsiteDropboxPath` (`env:"XSITE_DROPBOX_PATH"`, default `"./xsite-dropbox.ics"`)
- `BrightSpaceQuizzesPath` → renamed but env tag unchanged (`env:"BRIGHTSPACE_QUIZZES_PATH"`, default updated to `"./xsite-quizzes.ics"`)
- `XsitePath` — new field (`env:"XSITE_PATH"`, default `"./xsite.ics"`)
- `BrightSpaceCourseNameBlocklist` → `XsiteCourseNameBlocklist` (`env:"XSITE_COURSE_NAME_BLOCKLIST"`)
- `BrightSpaceCourseIDBlocklist` → `XsiteCourseIDBlocklist` (`env:"XSITE_COURSE_ID_BLOCKLIST"`)
- `BrightSpaceEventTitleBlocklist` → `XsiteEventTitleBlocklist` (`env:"XSITE_EVENT_TITLE_BLOCKLIST"`)
- `BrightSpaceEventLocationBlocklist` → `XsiteEventLocationBlocklist` (`env:"XSITE_EVENT_LOCATION_BLOCKLIST"`)
- `BrightSpaceQuizTitleBlocklist` → `XsiteQuizTitleBlocklist` (`env:"XSITE_QUIZ_TITLE_BLOCKLIST"`)
- `BrightSpaceEventsAlerts` → `XsiteEventsAlerts` (`env:"XSITE_EVENTS_ALERTS"`)
- `BrightSpaceDropboxAlerts` → `XsiteDropboxAlerts` (`env:"XSITE_DROPBOX_ALERTS"`)
- `BrightSpaceQuizzesAlerts` → `XsiteQuizzesAlerts` (`env:"XSITE_QUIZZES_ALERTS"`)

### CLI flags
- `brightspace-events-path` → `xsite-events-path`
- `brightspace-dropbox-path` → `xsite-dropbox-path`
- `brightspace-quizzes-path` → `xsite-quizzes-path` (new flag for the renamed field)
- `brightspace-base-url` — removed
- `brightspace-api-key` — removed
- `brightspace-api-secret` — removed
- `brightspace-course-name-blocklist` → `xsite-course-name-blocklist`
- `brightspace-course-id-blocklist` → `xsite-course-id-blocklist`
- `brightspace-event-title-blocklist` → `xsite-event-title-blocklist`
- `brightspace-event-location-blocklist` → `xsite-event-location-blocklist`
- `brightspace-events-alerts` → `xsite-events-alerts`
- `brightspace-dropbox-alerts` → `xsite-dropbox-alerts`
- `xsite-enabled` — flag already existed (config field renamed from `BrightSpaceEnabled`)
- `xsite-path` — new flag

### Cache Outputs struct
- `BSEvents` → `XsiteEvents`
- `BSDropbox` → `XsiteDropbox`
- `BSQuizzes` → `XsiteQuizzes`
- `Xsite` — new field for combo file

### HTTP endpoints
- `/brightspace-events.ics` → `/xsite-events.ics`
- `/brightspace-dropbox.ics` → `/xsite-dropbox.ics`
- `/quizzes.ics` → `/xsite-quizzes.ics`
- `/xsite.ics` — new endpoint (combo of all brightspace sources)

### What was NOT changed
- Internal package name (`brightspace`)
- Event source values (`"brightspace-calendar"`, `"brightspace-dropbox"`, `"brightspace-quizzes"`)
- Filter predicates in the cache layer (`filterEventsBySource(events, "brightspace-calendar")`)

## Consequences

### Positive

- **Naming consistency**: All external-facing artifacts now use the `xsite` prefix, aligning with ADR-0014's established naming for HTTP endpoints
- **Reduced configuration surface**: Removing `BrightSpaceBaseURL` eliminates a redundant env var that never changed from its default value
- **Combo endpoint**: Users can now fetch all BrightSpace/xsite events from a single endpoint (`/xsite.ics`) without combining multiple files
- **Clearer defaults**: File paths use the `xsite-` prefix, making it immediately clear which events each file contains

### Negative

- **Breaking change for env vars**: Users relying on `BRIGHTSPACE_EVENTS_PATH`, `BRIGHTSPACE_DROPBOX_PATH` environment variables will need to switch to `XSITE_EVENTS_PATH`, `XSITE_DROPBOX_PATH`
- **Breaking change for env vars**: Users relying on `BRIGHTSPACE_ENABLED` will need to switch to `XSITE_ENABLED`
- **Breaking change for env vars**: Users relying on `BRIGHTSPACE_COURSE_NAME_BLOCKLIST`, `BRIGHTSPACE_COURSE_ID_BLOCKLIST`, `BRIGHTSPACE_EVENT_TITLE_BLOCKLIST`, `BRIGHTSPACE_EVENT_LOCATION_BLOCKLIST` will need to switch to `XSITE_COURSE_NAME_BLOCKLIST`, `XSITE_COURSE_ID_BLOCKLIST`, `XSITE_EVENT_TITLE_BLOCKLIST`, `XSITE_EVENT_LOCATION_BLOCKLIST`
- **Breaking change for env vars**: Users relying on `BRIGHTSPACE_EVENTS_ALERTS`, `BRIGHTSPACE_DROPBOX_ALERTS`, `BRIGHTSPACE_QUIZZES_ALERTS` will need to switch to `XSITE_EVENTS_ALERTS`, `XSITE_DROPBOX_ALERTS`, `XSITE_QUIZZES_ALERTS`
- **Breaking change for env vars**: `BRIGHTSPACE_API_KEY` and `BRIGHTSPACE_API_SECRET` are no longer accepted (API key auth removed)
- **Breaking change for CLI flags**: Users relying on `--brightspace-events-path`, `--brightspace-dropbox-path` CLI flags will need to switch to `--xsite-events-path`, `--xsite-dropbox-path`
- **Breaking change for CLI flags**: Users relying on `--brightspace-course-name-blocklist`, `--brightspace-course-id-blocklist`, `--brightspace-event-title-blocklist`, `--brightspace-event-location-blocklist` will need to switch to `--xsite-course-name-blocklist`, `--xsite-course-id-blocklist`, `--xsite-event-title-blocklist`, `--xsite-event-location-blocklist`
- **Breaking change for CLI flags**: Users relying on `--brightspace-api-key`, `--brightspace-api-secret` will need to remove these flags (API key auth removed)
- **Breaking change for HTTP endpoints**: Existing `/brightspace-events.ics`, `/brightspace-dropbox.ics`, and `/quizzes.ics` endpoints are replaced with `/xsite-events.ics`, `/xsite-dropbox.ics`, and `/xsite-quizzes.ics`
- **Quizzes env var unchanged**: `BrightSpaceQuizzesPath` retains its `BRIGHTSPACE_QUIZZES_PATH` env tag to minimize breaking changes for the quizzes path specifically (the default file name changed from `brightspace-quizzes.ics` to `xsite-quizzes.ics`)

### Neutral / Operational

- **Internal compatibility preserved**: Event source values, package names, and filter predicates remain unchanged, ensuring UID generation and event matching continue to work correctly
- **Backward compatible internal behavior**: The `filterEventsBySource()` function still uses `"brightspace-calendar"`, `"brightspace-dropbox"`, and `"brightspace-quizzes"` as source values — only the external file naming changed

## Alternatives Considered

### Option A: Full rename (internal and external)
Rename everything including package names, source values, and internal references.

**Rejected**: Would break UID generation (which includes source values), break event matching in filters, and require changes across many more files. The internal `brightspace` naming is implementation detail; only external-facing artifacts need to follow the `xsite` convention.

### Option B: No changes
Keep the existing naming as-is.

**Rejected**: Inconsistent with ADR-0014's `xsite` naming for HTTP endpoints. The redundancy between `BRIGHTSPACE_BASE_URL` and the hardcoded URL is unnecessary configuration surface.

## Follow-Ups

- Update user documentation and deployment guides to reflect new env var names and CLI flags
- Consider deprecation warnings for old env var names in a future version (e.g., accept `BRIGHTSPACE_EVENTS_PATH` as alias that maps to `XSITE_EVENTS_PATH`)
- Monitor usage of the new `/xsite.ics` combo endpoint

## References

- **ADR-0013**: BrightSpace D2L Event Integration (introduces source tracking)
- **ADR-0014**: Xsite ICS HTTP Endpoints (establishes `xsite` naming convention)
- `internal/config/config.go` — Configuration struct with renamed fields
- `internal/config/flags.go` — CLI flag definitions
- `internal/calendar/cache.go` — Outputs struct and SaveOutputs method
- `internal/server/server.go` — HTTP endpoint registration
- `internal/server/handlers.go` — Handler functions
