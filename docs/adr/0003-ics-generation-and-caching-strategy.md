# ADR-0003: ICS Generation and Caching Strategy

## Status

Proposed

## Date

2026-09-01

## Context

The application must generate an iCalendar (`.ics`) file from the fetched PeopleSoft timetable data and serve it via a lightweight HTTP server. The ICS data needs to be:

1. Loaded from storage into memory on startup (to avoid repeated disk reads per request).
2. Served from memory for all HTTP requests.
3. Updated asynchronously after upstream fetch completes.
4. Persisted to disk as an offline copy.
5. Idempotent: repeated fetches must not create duplicate events.
6. Safe: when upstream data is missing historical events (e.g., fetching in 2027 for a 2026 semester), old events must not be deleted or purged — only created, replaced, or updated.

The ICS format must comply with RFC 5545, including proper timezone handling, event deduplication via `UID`, and correct escaping of special characters.

## Decision Drivers

- **Idempotent**: No duplicate events across fetches.
- **Safe upsert semantics**: Never delete or purge old events; only create, replace, or update.
- **Fast HTTP responses**: In-memory cache avoids disk I/O per request.
- **Offline copy**: Disk storage ensures availability if upstream fetch fails.
- **Minimal dependencies**: Avoid heavy ICS libraries when a minimal writer suffices.
- **RFC 5545 compliance**: Proper `UID`, `DTSTART`, `DTEND`, `TZID`, `DESCRIPTION`, `LOCATION`, etc.

## Considered Options

### Option A: In-Memory Cache + Disk Fallback with Upsert Semantics (Selected)

Maintain ICS data as a string in memory. On startup, load from disk. After upstream fetch, generate new ICS events, merge with existing events using upsert logic (create new, replace/update matching, preserve unmatched), write to both memory and disk. Serve all HTTP requests from memory.

### Option B: Full ICS Library (e.g., `github.com/arran4/gopcical` or `github.com/breml/ical-generator`)

Use a third-party library for full RFC 5545 compliance, including recurrence rules, timezone handling, and event management.

### Option C: Database-Backed ICS Storage

Store individual ICS events in a SQLite database, query and merge on each request, then write the combined ICS to disk and memory.

### Option D: Pure In-Memory (No Disk)

Keep ICS entirely in memory. No disk persistence.

## Decision

We will choose **Option A: In-Memory Cache + Disk Fallback with Upsert Semantics**.

## Rationale

### Why Option A over Option B

- **Minimal dependencies**: The requirement specifies reducing third-party libraries. Since each fetch produces a static snapshot (no RRULE recurrence), the ICS output is straightforward — one `VEVENT` per class meeting. A minimal in-app writer (~100 lines) suffices.
- **Full control**: Building the writer ensures correct handling of the specific event structure without library-specific quirks or version compatibility issues.
- **RFC 5545 edge cases**: Third-party ICS libraries sometimes have bugs or gaps in timezone handling, escaping, or property ordering. A minimal custom writer for a well-defined subset is easier to verify.

### Why Option A over Option C (Database-Backed)

- **Simplicity**: The event set is small (typically 10-30 events per week, ~200 events total for a trimester). A string-based in-memory representation is simpler and faster than database queries.
- **No additional dependency**: SQLite would add a database driver dependency. The config approach (ADR-0002) already uses SQLite for per-user settings; reusing it for ICS events adds unnecessary complexity.
- **Atomic writes**: Writing the full ICS string to disk is atomic enough for this use case. Database transactions would add complexity without meaningful benefit.

### Why Option A over Option D (Pure In-Memory)

- **Offline availability**: Disk storage ensures the ICS file is available if the app restarts or the upstream fetch fails.
- **User access**: Users may want to download or share the `.ics` file directly from disk.
- **Recovery**: On startup, loading from disk provides immediate data while the async fetch runs in the background.

### Why Option A over Status Quo (no solution)

This is a greenfield project. Option A establishes a robust, safe caching strategy that handles the specific requirement of preserving historical events.

## Implementation Details

### In-Memory Cache

```go
type ICSCache struct {
    mu    sync.RWMutex
    data  []byte // full ICS file as bytes
    dirty bool   // true if data was modified since last disk write
}
```

- `RWMutex` allows concurrent reads (HTTP requests) without blocking.
- Writes are serialized via exclusive lock.

### Startup Flow

1. Read ICS file from `ICS_STORAGE_PATH`. If file exists and is valid, parse events into memory. If file is missing or corrupted, start with empty event set.
2. Serve HTTP requests from the in-memory cache immediately.
3. Launch async goroutine to fetch timetable and update cache.

### Upsert Semantics (Critical)

When new data arrives from the upstream fetch:

1. **Parse existing events** from the in-memory ICS data into a map keyed by `UID`.
2. **Generate new events** from the fetched timetable data. Each event gets a deterministic `UID` based on: `{{course_code}}-{{section}}-{{day}}-{{start_time}}-{{date}}` (hashed via SHA-256 for uniqueness).
3. **Merge**:
   - **Create**: New events (UID not in existing map) are added.
   - **Replace/Update**: Events with matching UID are replaced entirely (not merged field-by-field). This handles schedule changes (time/location updates).
   - **Preserve**: Events in the existing map that are not in the new set are kept. This preserves historical events that PeopleSoft may not return (e.g., past weeks not included in the current fetch range).
4. **Regenerate** the full ICS string from the merged event set.
5. **Update** in-memory cache and write to disk (atomic write: write to temp file, then rename).

### ICS Writer

Minimal RFC 5545-compliant writer producing:

```
BEGIN:VCALENDAR
VERSION:2.0
PRODID:-//SIT Timetable//EN
CALSCALE:GREGORIAN
METHOD:PUBLISH
BEGIN:VEVENT
UID:<sha256_hash>
DTSTART;TZID=Asia/Singapore:20260907T090000
DTEND;TZID=Asia/Singapore:20260907T110000
SUMMARY:ABC 0002 - ALL (Lecture)
LOCATION:Online
DESCRIPTION:Course: ABC 0002\nSection: ALL\nType: Lecture
END:VEVENT
BEGIN:VEVENT
...
END:VEVENT
END:VCALENDAR
```

Key properties:
- `UID`: Deterministic SHA-256 hash for idempotent upsert.
- `DTSTART`/`DTEND`: With `TZID=Asia/Singapore` parameter.
- `SUMMARY`: Course code + section + type.
- `LOCATION`: From timetable data (e.g., "W1-05-07" or "Online").
- `DESCRIPTION`: Enriched with course details for reference.

### HTTP Serving

Lightweight HTTP server using Go stdlib `net/http`:

```go
http.HandleFunc("/timetable.ics", func(w http.ResponseWriter, r *http.Request) {
    cache.mu.RLock()
    defer cache.mu.RUnlock()
    w.Header().Set("Content-Type", "text/calendar")
    w.Header().Set("Content-Disposition", `attachment; filename="timetable.ics"`)
    w.Write(cache.data)
})
```

### Disk Write (Atomic)

```go
func (c *ICSCache) saveToDisk(path string) error {
    tmp := path + ".tmp"
    if err := os.WriteFile(tmp, c.data, 0600); err != nil {
        return err
    }
    return os.Rename(tmp, path)
}
```

File permissions are `0600` (owner-only read/write) to protect any sensitive data in descriptions.

## Consequences

### Positive
- Fast HTTP responses: all reads are in-memory, no disk I/O per request.
- Safe upsert: historical events are never deleted, even if upstream data is incomplete.
- Offline copy: disk file is always available for direct access or backup.
- Atomic disk writes: temp file + rename prevents corruption.
- Minimal dependencies: custom ICS writer (~100 lines) vs. heavy library.

### Negative
- Custom ICS writer must be maintained and tested for RFC 5545 compliance.
- In-memory cache size grows with the number of weeks fetched (typically <1 MB for a full trimester, so this is acceptable).
- Upsert logic requires deterministic UID generation — any change to the UID formula would break idempotency for existing events.

### Neutral / Operational
- Disk file permissions (`0600`) protect sensitive data but may need adjustment if multiple users share the system.
- The async fetch means there is a brief window after startup where the in-memory cache may contain stale data (mitigated by loading from disk first).
- HTTP server serves `Content-Type: text/calendar` with `Content-Disposition: attachment` to trigger download in browsers.

## Alternatives Considered

### Option B: Full ICS Library

**Summary**: Use `github.com/arran4/gopcical` or `github.com/breml/ical-generator` for RFC 5545 compliance.

**Benefits**: Full feature support (recurrence rules, timezone handling, validation).

**Costs**: Additional dependency, potential version compatibility issues, less control over output format.

**Reason rejected**: Overkill for static weekly snapshots with no recurrence. The minimal-dependency principle favors a custom writer.

### Option C: Database-Backed Storage

**Summary**: Store events in SQLite, query and merge on each request.

**Benefits**: Flexible querying, transactional updates, easier to add per-event metadata.

**Costs**: Additional dependency (SQLite driver), more complex code, slower for small event sets.

**Reason rejected**: The event set is small and the use case is simple (generate, merge, serve). Database adds complexity without meaningful benefit.

### Option D: Pure In-Memory

**Summary**: No disk persistence.

**Benefits**: Simpler code, no disk I/O.

**Costs**: No offline availability, no recovery after restart, no user-accessible file.

**Reason rejected**: The requirement explicitly specifies disk storage as an offline copy.

## Follow-Ups

- Related ADRs: ADR-0001 (ADFS auth), ADR-0002 (config), ADR-0004 (cron scheduling).
- Implementation tracking lives outside this ADR.

## References

- [RFC 5545](https://tools.ietf.org/html/rfc5545) — Internet Calendaring and Scheduling Core Object Specification (iCalendar)
- [RFC 5546](https://tools.ietf.org/html/rfc5546) — iCalendar Transport-Independent Interoperability Protocol (iTP)
- [gopcical](https://github.com/arran4/gopcical) — Go iCalendar library (if needed in future).
