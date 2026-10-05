# ADR-0021: Wakey-SITizen JSON Timetable Endpoint

## Status

Accepted

## Date

2026-10-06

## Context

The upcoming Wakey-SITizen companion mobile app sets alarms automatically from the user's timetable. Consuming the existing ICS endpoints (`/timetable.ics` et al.) would require the app to implement a full RFC 5545 ICS parser in Kotlin, which is disproportionate for a client that only needs flat event data. The server already holds all events in the in-memory `ICSCache`, so exposing a filtered, pre-shaped JSON view is cheap.

Requirements:

- A single `GET /wakey-sitizen/timetable.json` endpoint returning the PeopleSoft timetable as JSON.
- BrightSpace/Xsite events (`Source` prefix `brightspace-`) are excluded — the app only needs class sessions for alarm scheduling.
- A `schema_version` field so the app can validate the contract before deserializing (with `kotlinx.serialization`), enabling safe server-side schema evolution.
- A `generated_at` field for client-side staleness detection.
- A fixed `Asia/Singapore` (`+08:00`) RFC 3339 timestamp format (SGT has no DST).

## Decision Drivers

- **No ICS parsing on the client**: JSON is natively supported by `kotlinx.serialization`; ICS is not.
- **Consistency with existing endpoints**: same in-memory cache source, same `Cache-Control`/`ETag` conditional-request pattern (ADR-0014, ADR-0018).
- **Deterministic output**: the JSON body must be byte-stable for identical cache state so `ETag`/`If-None-Match` conditional requests work; event ordering must therefore be deterministic.
- **Always 200 OK**: an empty timetable is a valid state (e.g., first run before the first fetch) and must serialize as `"events":[]`, not `204 No Content` or `null` — the client deserializes the body unconditionally.
- **Schema versioning**: `schema_version` starts at 1 and is bumped on any breaking change; the app rejects unknown versions instead of failing mid-deserialization.
- **No new configuration**: the endpoint is always available when the server runs, matching the existing endpoints.

## Considered Options

### Option A: Dedicated JSON Handler over ICSCache

Add `ICSCache.GetEvents()` (read-locked copy, deterministically sorted) and a purpose-built handler that filters out `brightspace-*` sources, maps each `Event` to a flat `WakeyEvent` JSON struct, and serializes with `encoding/json`.

### Option B: Serve ICS and Parse Client-Side

No server change; the app implements an ICS parser. Rejected: RFC 5545 parsing (folding, escaping, `TZID` parameters, `RECURRENCE-ID`) is a large, bug-prone client dependency for data the server already has in structured form.

### Option C: Generic JSON Endpoint with Query Parameters

A single `/api/events.json?filter=...` endpoint. Rejected: breaks the explicit-endpoint pattern (ADR-0014), forces the client to know server-internal filter names, and complicates schema versioning (one version number cannot cover multiple shapes).

## Decision

We will choose **Option A: Dedicated JSON Handler over ICSCache**.

## Rationale

### How It Works

1. `ICSCache.GetEvents()` (in `internal/calendar/cache.go`) returns a copy of all cached events under the read lock, sorted by `DTStart`, then `Location`, then `UID` — the same deterministic ordering used by the ICS renderers. Copying under the lock makes the result safe to modify by the caller.
2. `newWakeySitizenHandler` (in `internal/server/handlers.go`) skips events whose `Source` starts with `brightspace-`, maps each remaining event to a `WakeyEvent`, and formats `DTStart`/`DTEnd` as RFC 3339 in the configured `TZ` (e.g. `2026-09-07T09:00:00+08:00`).
3. `event_type` is `"online"` when `strings.EqualFold(Location, "Online")` (the same predicate as `calendar.IsOnline`), otherwise `"campus"`.
4. The response is `WakeyResponse{SchemaVersion: 1, GeneratedAt: cache.GetLastModified(), Timezone: tz, Events: [...]}`. `GeneratedAt` is the cache's last-modified time, i.e. when the server last fetched/updated the timetable.
5. The handler reuses the existing `ETag`/`If-None-Match`/`If-Modified-Since`/`Cache-Control` conditional-request pattern (including 304 handling) from the ICS handlers, and honors `SERVER_DISABLE_CACHING`.
6. The route is registered in `internal/server/server.go` alongside the seven ICS endpoints.

### Schema

```json
{
  "schema_version": 1,
  "generated_at": "2026-10-06T09:00:00+08:00",
  "timezone": "Asia/Singapore",
  "events": [
    {
      "course_code": "ALT2501",
      "title": "Lecture 1",
      "summary": "[ALT2501] Lecture 1",
      "dtstart": "2026-09-07T09:00:00+08:00",
      "dtend": "2026-09-07T11:00:00+08:00",
      "location": "W1-05-07",
      "event_type": "campus"
    }
  ]
}
```

`events` is always a JSON array (empty `[]` when there are no events), never `null`.

## Consequences

### Positive

- The app gets alarms-ready data with zero ICS parsing; `kotlinx.serialization` maps 1:1 onto the Go structs.
- `schema_version` + `generated_at` give the app a clean upgrade path and staleness detection.
- Conditional requests (ETag/304) reuse the existing pattern — no new caching code.
- Deterministic ordering keeps ETag validation stable across requests for identical cache state.
- No new dependencies, configuration, or authentication surface.

### Negative

- One more endpoint to maintain; the conditional-request block is duplicated (consistent with the existing ICS handlers, which also duplicate it).
- The `generated_at` value changes on every cache update even when event content is unchanged, so clients will refetch full bodies after each server-side update — acceptable given the small payload and the staleness semantics this field is intended to convey.
- BrightSpace events are not available to the app; if the app later needs them, a new field or endpoint (and a schema bump) is required.

### Neutral / Operational

- The endpoint is served by the existing `http.Server` and inherits its bind-address policy (loopback on desktop, all interfaces in container) and the JSON request logging middleware (ADR-0018).
- The same data is already exposed by the ICS endpoints; this endpoint adds no new class of sensitive information.

## Follow-Ups

- Bump `schema_version` and update the app contract whenever `WakeyResponse`/`WakeyEvent` change.
- If the app needs BrightSpace-derived alarms (e.g., quiz deadlines), decide between adding fields to this endpoint (schema bump) and a new dedicated endpoint.

## Related ADRs

- **ADR-0003**: ICS Generation and Caching Strategy (in-memory `ICSCache` foundation)
- **ADR-0014**: Xsite ICS HTTP Endpoints (explicit-endpoint pattern, source-based filtering)
- **ADR-0013**: BrightSpace D2L Event Integration (`Source` prefix `brightspace-*`)
- **ADR-0018**: JSON Request Logging and Trusted Proxy Support (logging middleware this endpoint passes through)
