# ADR-0014: Xsite ICS HTTP Endpoints

## Status

Accepted

## Date

2026-09-08

## Context

The SIT ICS Go project already served three ICS file paths both as disk upserts and HTTP endpoints via `internal/server/server.go`:

| HTTP Path | Disk File | Content |
|-----------|-----------|---------|
| `/timetable.ics` | `timetable.ics` | All events |
| `/timetable-online.ics` | `timetable-online.ics` | Online-only events (`IsOnline` filter) |
| `/timetable-campus.ics` | `timetable-campus.ics` | Non-online events (`IsNotOnlineAndNotBrightSpace` filter) |

With the BrightSpace D2L integration (ADR-0013), two new ICS file paths were added to disk output:

| Disk File | Content |
|-----------|---------|
| `xsite-events.ics` | BrightSpace calendar events only |
| `xsite-dropbox.ics` | BrightSpace dropbox due dates only |

These files were written by `SaveAllWithXsiteFiles()` in the cache layer. However, there were no corresponding HTTP endpoints for external consumers (e.g., other systems, mobile apps, calendar clients) to fetch them directly. The existing server infrastructure already handled `/timetable.ics`, `/timetable-online.ics`, and `/timetable-campus.ics` as HTTP handlers, but the new xSite files had no network-accessible equivalents.

The key question was how to expose these two new filtered ICS outputs as HTTP endpoints while maintaining consistency with the existing endpoint patterns and minimizing code duplication.

## Decision Drivers

- **Consistency with existing patterns**: New endpoints should follow the same structure as the existing `/timetable*.ics` endpoints
- **Reuse existing filtering**: Use the same `GetFiltered()` method on `ICSCache` that powers the existing endpoints, not a separate code path
- **Source-based filtering**: BrightSpace events are distinguished by the `Source` field (`"brightspace-calendar"` or `"brightspace-dropbox"`), which is already tracked per ADR-0013
- **No authentication**: Endpoints are public, matching the existing `/timetable*.ics` endpoints — no auth middleware needed
- **Graceful empty handling**: Return `204 No Content` when no events match, rather than an empty ICS file or `404 Not Found`
- **Content-Disposition header**: Set attachment filename for browser download compatibility
- **No new configuration**: Endpoints are always available when the server runs — no env vars or CLI flags required

## Considered Options

### Option A: Direct HTTP Handlers with Source-Based Filtering

Add two `http.HandleFunc` handlers in `internal/server/server.go` that call `cache.GetFiltered()` with predicates checking `e.Source == "brightspace-calendar"` and `e.Source == "brightspace-dropbox"`. Each handler sets `Content-Type: text/calendar` and `Content-Disposition: attachment` headers, returning `204 No Content` when no events match.

### Option B: Generic Filtered Endpoint with Query Parameter

Create a single generic endpoint (e.g., `/filtered.ics?source=brightspace-calendar`) that accepts a query parameter to select the filter predicate. This would reduce code duplication but would require the client to know the valid source values and would deviate from the existing explicit endpoint pattern.

### Option C: File-Based Serving (Static Files)

Serve the xSite ICS files as static files from disk using `http.FileServer`, similar to how some ICS servers expose files. This would require the cache to write the files before each request and would introduce race conditions between writes and reads.

## Decision

We will choose **Option A: Direct HTTP Handlers with Source-Based Filtering**.

## Rationale

### Why Option A over Option B

The existing server already uses explicit endpoints for each filter variant (`/timetable.ics`, `/timetable-online.ics`, `/timetable-campus.ics`). A generic endpoint with query parameters would break this established pattern and require clients to know valid source values at request time. Explicit endpoints are self-documenting — the URL path itself communicates what content a consumer will receive. Additionally, the two handlers are simple enough (approximately 10 lines each) that code duplication is not a meaningful concern.

### Why Option A over Option C

Static file serving via `http.FileServer` would introduce race conditions between cache writes and file reads, potentially serving partial or stale content. The existing endpoints do not serve from disk at request time — they compute the ICS content on each request using the in-memory cache. Option A maintains this same model: the handler reads from the in-memory `ICSCache`, computes the filtered ICS content, and writes it directly to the response. This ensures consistency between file upserts and HTTP responses, and avoids the complexity of coordinating disk I/O with HTTP serving.

### How It Works

Two new handlers are registered in `internal/server/server.go`:

```go
http.HandleFunc("/xsite-events.ics", handler)
http.HandleFunc("/xsite-dropbox.ics", handler)
```

Each handler:
1. Calls `s.cache.GetFiltered(s.tz, sourcePredicate, s.refreshInterval)` with the appropriate source filter
2. If no events match, returns `http.StatusNoContent` (204) with no body
3. Otherwise, sets `Content-Type: text/calendar`, `Content-Disposition: attachment; filename="xsite-{events,dropbox}.ics"`, and writes the ICS data to the response

The source predicates match the same values used by `filterEventsBySource()` in `cache.go` and the source tracking established in ADR-0013:
- `/xsite-events.ics` → `e.Source == "brightspace-calendar"`
- `/xsite-dropbox.ics` → `e.Source == "brightspace-dropbox"`

The `GetFiltered()` method is the same one used by the existing `/timetable-online.ics` and `/timetable-campus.ics` endpoints, ensuring consistent filtering behavior across all filtered endpoints.

## Consequences

### Positive

- **Consistent API surface**: External consumers can fetch xSite ICS data via the same HTTP interface as timetable data, without needing to read files from disk
- **No code duplication**: Reuses `GetFiltered()` — the same method used by existing filtered endpoints
- **Self-documenting URLs**: The endpoint paths clearly indicate what content each provides, matching the established pattern
- **Graceful empty handling**: `204 No Content` is semantically correct for "no events match this filter" and avoids sending empty or partial ICS files
- **Browser compatibility**: `Content-Disposition: attachment` header ensures browsers prompt for download rather than attempting to display the ICS inline
- **No configuration overhead**: Endpoints are always available — no env vars, CLI flags, or feature flags to manage
- **In-memory consistency**: Content is computed from the same in-memory cache used by file upserts, ensuring HTTP responses match disk files

### Negative

- **Code duplication**: Two nearly identical handlers (differing only in source predicate and filename) — acceptable given their simplicity but still duplication
- **No filtering flexibility**: Each endpoint serves exactly one source value; adding new filtered views requires adding new handlers
- **204 handling**: Some ICS clients may not handle `204 No Content` gracefully — they might treat it as an error or fail to parse the empty response
- **No caching headers**: The endpoints do not set `Cache-Control` or `ETag` headers, so every request recomputes the filtered ICS content (same as existing endpoints)

### Neutral / Operational

- **Same refresh interval**: The `refreshInterval` parameter passed to `GetFiltered()` is the same `ICS_REFRESH_INTERVAL` env var used by existing endpoints
- **Timezone filtering**: Events are filtered by timezone consistently with existing endpoints, using `s.tz` from the server configuration
- **Unauthenticated access**: Like existing endpoints, these are public — any client can fetch them without credentials
- **No new dependencies**: No new packages or imports required; uses only existing `internal/ics` and `net/http` functionality
- **Shutdown behavior**: The endpoints are served by the existing `http.Server` — they are automatically stopped when the server shuts down

## Alternatives Considered

### Option B: Generic Filtered Endpoint with Query Parameter

**Summary**: A single endpoint `/filtered.ics?source=brightspace-calendar` that accepts a query parameter to select the filter predicate, reducing handler code to a single function.

**Benefits**: Less code duplication; easier to add new filter dimensions without adding more handlers; the endpoint URL is shorter.

**Costs**: Breaks consistency with existing explicit endpoint pattern; clients must know valid source values at request time; the URL is less self-documenting; query parameter validation and error handling adds complexity (e.g., what happens with an invalid `source` value?).

**Reason rejected**: The existing server uses explicit endpoints for each filter, and the two new handlers are simple enough that duplication is not a meaningful concern. Self-documenting URLs are more important than minor code reduction for this use case. A generic endpoint would also make it harder to add endpoint-specific headers or behavior in the future if needed.

### Option C: Static File Serving

**Summary**: Serve `xsite-events.ics` and `xsite-dropbox.ics` as static files from disk using `http.FileServer`, similar to how some ICS servers expose files.

**Benefits**: Minimal handler code; leverages Go's built-in static file serving; potentially better performance for large files (though ICS files are typically small).

**Costs**: Race conditions between cache writes and file reads; stale content served until the next file write; inconsistent with existing endpoint behavior (which computes content on each request); additional complexity to coordinate disk I/O with HTTP serving.

**Reason rejected**: The existing endpoints compute ICS content on each request from the in-memory cache, ensuring consistency between HTTP responses and disk files. Static file serving would break this model and introduce race conditions. The in-memory cache approach is more correct for a concurrent server.

### Option D: No HTTP Endpoints for xSite Files

**Summary**: Only write `xsite-events.ics` and `xsite-dropbox.ics` to disk via `SaveAllWithXsiteFiles()`, without adding HTTP endpoints. External consumers would need to read the files directly from disk.

**Benefits**: No server code changes; no new endpoints to maintain; simpler architecture.

**Costs**: External consumers (other systems, mobile apps, calendar clients) cannot fetch xSite data via HTTP; defeats the purpose of running an HTTP server that serves ICS content; inconsistent with the existing pattern of serving all ICS outputs as HTTP endpoints.

**Reason rejected**: The project's primary value proposition is serving ICS content over HTTP. Exposing xSite files only on disk would significantly reduce the utility of the application for external consumers and break the consistency of the HTTP API.

## Follow-Ups

- Monitor whether ICS clients handle `204 No Content` responses gracefully; if not, consider returning an empty but valid ICS file instead
- Track demand for additional filtered endpoints (e.g., by course, by event type) — if multiple new filters are requested, consider whether a generic filtered endpoint makes sense
- Consider adding `Cache-Control` headers to all ICS endpoints to reduce unnecessary recomputation for clients that cache aggressively
- Related ADRs:
  - **ADR-0013**: BrightSpace D2L Event Integration (introduces source tracking and xSite ICS files)
  - **ADR-0003**: ICS Generation and Caching Strategy (`GetFiltered()` method foundation)
  - **ADR-0005**: Browser Abstraction Layer (indirectly — the data fed into the cache originates from browser automation)

## References

- `internal/server/server.go` — HTTP endpoint handlers for `/xsite-events.ics` and `/xsite-dropbox.ics`
- `internal/calendar/cache.go` — `GetFiltered()` method used by all filtered endpoints
- `internal/calendar/cache.go` — `filterEventsBySource()` predicate function (same source values)
- `internal/calendar/event.go` — `Event.Source` field definition
- **ADR-0013**: BrightSpace D2L Event Integration (source tracking for `"brightspace-calendar"` and `"brightspace-dropbox"`)
- **ADR-0003**: ICS Generation and Caching Strategy (cache filtering foundation)
