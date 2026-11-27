# ADR-0024: Append SIT Campus Address to In-Campus Event Locations

## Status

Accepted

## Date

2026-11-27

## Context

The application renders RFC 5545 ICS files with a custom in-tree writer (`internal/calendar/render.go`, no third-party ICS library) from three upstream sources: the PeopleSoft class timetable, the PeopleSoft exam timetable, and BrightSpace (xsite) events (ADR-0013). The resulting in-memory event cache (`internal/calendar/cache.go`) is served by the ICS endpoints (`timetable.ics`, `timetable-online.ics`, `timetable-campus.ics`, `timetable-exams.ics`, and the four `xsite-*.ics` files, ADR-0014), the `GET /wakey-sitizen/timetable.json` endpoint (ADR-0021), and the `GET /status` rendered views (ADR-0023). On startup the cache is rehydrated from the persisted ICS via `LoadFromFile`/`parseICS` (`internal/calendar/persistence.go`), which reads `LOCATION` back verbatim into `Event.Location`.

User requirement: for all in-campus events, append the following literal string to the VEVENT `LOCATION` property:

```
\nSingapore Institute of Technology \n1 Punggol Coast Rd, Singapore 828608
```

That is: a leading newline, `Singapore Institute of Technology ` (note the trailing space), a newline, and `1 Punggol Coast Rd, Singapore 828608`. For example, a location `FAK BLDG Hall A` becomes `FAK BLDG Hall A\nSingapore Institute of Technology \n1 Punggol Coast Rd, Singapore 828608`. The existing `EscapeText` (escapes `\n` and `\,`) and `FoldText` (75-octet folding) in `render.go` already handle this content, so no renderer changes are needed.

The decision must determine where in the fetch pipeline the address is attached, which events qualify, and what the attachment does to the deterministic UID formula (ADR-0003).

## Decision Drivers

- **Exact user-specified string**: the literal value, including the leading newline, the trailing space after "Technology", and the comma in the street address, is specified by the user and must be preserved byte-for-byte.
- **Single source of truth**: the in-memory `Event` model must remain canonical for both the ICS renderers and the JSON/status views; a render-time decoration would fork the stored model from what is rendered.
- **Round-trip stability**: after a restart, `LoadFromFile`/`parseICS` reads `LOCATION` back verbatim into `Event.Location`; a subsequent fetch must produce byte-identical locations so UIDs (which hash `Location`) remain stable and reconciliation (ADR-0003) stays idempotent.
- **Preserve existing matching semantics**: smart merge (`internal/smartmerge`, `MatchesLocationConditions`) treats PS locations "Online"/"TBD"/"To Be Advised"/"TBA" as potential Zoom online meetings, and the xsite blocklist (`RemoveWhere` in `internal/app/fetch.go`) matches location substrings. Neither may see the appended address.
- **No UID formula change**: `EventID` (`internal/calendar/render.go`) hashes `Location`; the formula itself must not change (changing it would break idempotency for all events — ADR-0003).
- **No renderer or dependency changes**: the address must be expressible through the existing `EscapeText`/`FoldText` pipeline (the comma needs `\,` escaping; the multiline content needs `\n` escaping and folding).
- **Data privacy**: `AGENTS.md` forbids organizational location data in the repo, with an exception only for public domain names. The user explicitly requested this public institutional campus address, which authorizes it in source and test fixtures.

## Considered Options

### Option A: Construction-Time Append (Final Normalization Pass)

Add `internal/calendar/campus_address.go` holding the address constant, the venue predicate, and the append function, and apply it once per fetch over the whole `icsEvents` slice in `runFetchWithStatus` (`internal/app/fetch.go`), in the existing loop that sets `DTStamp`, immediately before `cache.UpdateWithSourceReconciliation`.

### Option B: Render-Time Append

Append the address inside `Render` (or per-projection render options) for non-BrightSpace, non-online events. Rejected: the disk round-trip reads rendered `LOCATION` back into `Event.Location` verbatim, so after a restart every render would double-append on top of already-addressed locations, growing the value further on every subsequent cycle. The stored model would also lack the address, so the JSON and /status views would diverge from the ICS unless each view special-cased the append.

### Option C: Append at Each Event Producer

Modify the PeopleSoft entry-to-Event conversion and the exam conversion to append the address where events are built. Rejected: scatters the predicate across producers, misses future non-BrightSpace sources, and still requires the same shared predicate plus the same round-trip handling. A single final pass covers every source at one choke point.

## Decision

We will choose **Option A: Construction-Time Append (Final Normalization Pass)**.

## Rationale

### New calendar package API

`internal/calendar/campus_address.go` (package `calendar`) contains:

- `const SITCampusAddress = "\nSingapore Institute of Technology \n1 Punggol Coast Rd, Singapore 828608"` — the exact user-specified literal.
- An unexported `sitVenuePlaceholders` set: `{"tbd", "tba", "to be advised"}`.
- An exported `IsConcreteVenue(location string) bool`: true when the trimmed location is non-empty, not "online" (case-insensitive, trimmed), and not a placeholder.
- An exported `WithSITCampusAddress(source, location string) string`: returns `location` unchanged when `source` starts with `"brightspace-"` or the venue is not concrete; otherwise returns `TrimSpace(location) + SITCampusAddress`, with a defensive `HasSuffix` idempotency guard that leaves values already ending in `SITCampusAddress` unchanged.
- Optionally, `projection.go`'s `isBrightSpaceEvent` delegates to a shared `isBrightSpaceSource` helper so the `"brightspace-"` prefix has one home.

### Wiring

In `runFetchWithStatus` (`internal/app/fetch.go`), the existing loop that sets `DTStamp` — which runs just before `cache.UpdateWithSourceReconciliation`, i.e. after smart merge and after the BrightSpace event append — also sets:

```go
icsEvents[i].Location = calendar.WithSITCampusAddress(icsEvents[i].Source, icsEvents[i].Location)
```

### Why Construction Time, Not Render Time

The in-memory event model stays the single source of truth for all views. Round-trip stability: a fetch builds the addressed string → `Render` escapes/folds it → after a restart, `parseICS` (`internal/calendar/persistence.go`) unfolds/unescapes `LOCATION` back byte-identically into `Event.Location` → the next fetch builds the identical string from the identical upstream venue → the identical `EventID` → stable upserts (ADR-0003). A render-time append would double-append after the first disk round-trip because the parser reads `LOCATION` back verbatim.

### Which Events Qualify

Events with a non-BrightSpace source and a concrete venue:

- Regular PeopleSoft timetable events have an empty `Source` by design (to preserve their UID format); exam events carry `Source` `"peoplesoft-exams"`.
- Excluded: "Online" in any case/trimmed form (the predicate trims before comparing), the placeholders `tbd`/`tba`/`to be advised` (case-insensitive after trim), and empty locations (`Render` omits `LOCATION` entirely for empty values).

Smart merge (`internal/smartmerge.MatchesLocationConditions`) matches exactly that PS location set ("online", "tbd", "to be advised", "tba") plus BrightSpace locations containing "Zoom Online Meeting" as potential Zoom online meetings; an address on those locations would be wrong for merged Zoom events.

### Deliberate Distinction from the Campus Projection

The campus ICS projection (`IsCampus` in `internal/calendar/projection.go`, used for `timetable-campus.ics`) includes placeholder-venue events. The address predicate is intentionally **narrower**: TBD/TBA events appear in `timetable-campus.ics` without the address.

### Placement in the Fetch Pipeline

The pass runs after `smartmerge.MergeEvents` (which only mutates `Description`, never `Location`) and after the blocklist `RemoveWhere`, so merge and blocklist substring semantics are unchanged. It runs before UID computation — `mergeEvents` in `cache.go` computes `EventID` for each incoming event — so UIDs hash the final addressed location.

### One-Time UID Churn (Accepted)

Because `EventID` hashes `Location`, every in-campus event gets a new UID on the first fetch after deployment. Reconciliation (ADR-0003) then replaces upcoming in-campus events in subscribers' calendars and retains past events, which keep their pre-address location until they naturally expire. This is consistent with the existing semantics where any room/location change from PeopleSoft already changes the UID. The `EventID` formula itself is deliberately not changed (changing it would break idempotency of all events — ADR-0003).

### Side Effects on /wakey-sitizen/timetable.json and /status

The shared event model means the JSON `location` field for in-campus events now includes the address, and the `/status` rendered views show it too. The user scoped the request to the VEVENT `LOCATION` property, but this consequence is accepted and not special-cased. No `schema_version` bump is required (the JSON shape is unchanged). `event_type` classification is unaffected: it compares the location against "Online" via `strings.EqualFold` (in `internal/server/handlers.go`), and online events never get the address.

### Edge Cases

- `" Online "` → appears in the campus ICS (`IsOnline` does not trim) but gets no address (the predicate trims).
- "Online (Zoom)"-style non-exact strings **do** get the address (PeopleSoft emits exactly "Online" in practice).
- Venue whitespace is trimmed as part of normalization (`TrimSpace(location)`).
- The address contains a comma, which the existing `EscapeText` escapes as `\,` and `FoldText` folds to 75-octet lines.

### Data Privacy

`AGENTS.md` forbids organizational location data in the repo, with an exception only for public domain names. This public institutional campus address is in scope because the user explicitly requested it. All test fixtures reference the `SITCampusAddress` constant only, never a second real address.

## Consequences

### Positive

- In-campus VEVENTs carry the full campus address with no renderer changes; `EscapeText`/`FoldText` already handle the escaping and folding.
- Deterministic and round-trip stable: identical upstream data yields identical `Location` and `EventID` on every fetch, including across restarts.
- The address, predicate, and exclusion list live in one place in `internal/calendar` and are unit-testable without a browser; the optional shared `isBrightSpaceSource` helper centralizes the `"brightspace-"` prefix check.

### Negative

- One-time UID churn on deployment: every in-campus event gets a new UID on the first fetch after deploy; upcoming in-campus events are replaced in subscribers' calendars while past events retain their pre-address location until they expire.
- The `/wakey-sitizen/timetable.json` `location` field for in-campus events now includes the address even though the request was scoped to the ICS property; accepted and not special-cased.
- A venue string that coincides with an excluded value (e.g., a room literally named "TBD") is treated as a placeholder and silently gets no address — a false negative only, and the room name still appears.

### Neutral / Operational

- Outputs affected: `timetable.ics`, `timetable-campus.ics`, `timetable-exams.ics` (in-campus events), and the `/status` rendered views. `timetable-online.ics` and all `xsite-*.ics` outputs are unchanged (online events and `"brightspace-"` sources never receive the address).
- In a degraded fetch cycle (PeopleSoft failure, ADR-0020) the address pass still runs over whatever events exist in `icsEvents` (BrightSpace and exam events), which is correct: retained PeopleSoft events already carry the address from the last successful fetch thanks to round-trip stability, and pre-address retained events from a first-cycle-after-deploy degraded fetch simply stay unaddressed until the next successful fetch.
- Tests planned: a new `internal/calendar/campus_address_test.go` (predicate table, append table, and a `Render`→`parseICS` round-trip asserting identical `Location` and `EventID`); a new app test asserting `runFetch` appends the address to concrete PeopleSoft and exam venues and leaves Online/TBD/empty/BrightSpace locations byte-identical. No existing calendar/server test expectations change, because those tests construct events directly and do not traverse `fetch.go`.
- The user-authorized public institutional address becomes source and test-fixture content under the `AGENTS.md` data-privacy rule; fixtures reference only the `SITCampusAddress` constant.

## Follow-Ups

- No required follow-ups.
- Optionally, the `AGENTS.md` fetch-flow documentation gains a step note for the address pass (implementation subagent will do that).

## Related ADRs

- **ADR-0003**: ICS Generation and Caching Strategy (deterministic `EventID`/UID formula and reconciliation semantics — the formula is deliberately unchanged)
- **ADR-0013**: BrightSpace D2L Event Integration (`brightspace-` `Source` prefix excluded from the address)
- **ADR-0014**: Xsite ICS HTTP Endpoints (all `xsite-*.ics` outputs remain unchanged)
- **ADR-0020**: Resilient Fetch — BrightSpace Still Runs When PeopleSoft Timetable Fetch Fails (address pass in degraded cycles)
- **ADR-0021**: Wakey-SITizen JSON Timetable Endpoint (accepted side effect on the `location` field)
- **ADR-0023**: Server Status and Manual Refresh Endpoints (`/status` rendered views include the address)
