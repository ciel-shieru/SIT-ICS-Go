# ADR-0026: Exclude SIT Campus Address from Wakey-SITizen JSON Location

## Status

Accepted

## Date

2026-11-27

## Context

ADR-0024 appends the public SIT campus address (the user-specified literal constant `SITCampusAddress` in `internal/calendar/campus_address.go`) to `Event.Location` at fetch time: `runFetchWithStatus` (`internal/app/fetch.go`) runs the `calendar.WithSITCampusAddress(source, location)` pass over in-campus events (non-BrightSpace source and `IsConcreteVenue`) immediately before `cache.UpdateWithSourceReconciliation`. The purpose is that the ICS `LOCATION` property carries the full address so calendar clients offer click-to-maps navigation to the campus. Because the wakey-sitizen JSON endpoint (ADR-0021) and the `GET /status` rendered views (ADR-0023) map the same in-memory `Event` model, ADR-0024 documented and accepted a side effect: the wakey-sitizen JSON `location` field (`newWakeySitizenHandler`, `internal/server/handlers.go:193`) and the `/status` wakey representation (`renderWakeyStatusJSON`, `internal/server/status.go:173`) also contain the address — "accepted and not special-cased".

New user requirement (verbatim): "do not include the SIT campus address into the wakey-sitizen JSON endpoint, since the app is meant for SIT students and does not do anything with the address. the point of adding it to the VEVENT was for users to click on the address in the calendar clients and get a shortcut to a maps app navigation to the campus."

This ADR partially supersedes ADR-0024 — specifically its "Side Effects on /wakey-sitizen/timetable.json and /status" section (the "accepted and not special-cased" statement) and its corresponding negative-consequence bullet. ADR-0024 is not edited (ADRs are immutable); its ICS decision remains in force. The decision must determine how the JSON views return the base venue, which views are affected, what (if anything) changes in the calendar package, and what the change does to ETags, `schema_version`, and the deterministic UID formula (ADR-0003).

## Decision Drivers

- **Explicit user requirement**: the wakey-sitizen JSON `location` field must not contain the campus address; the ICS `LOCATION` property must keep it, because click-to-maps navigation in calendar clients is the only purpose of the address.
- **ICS unchanged**: all 7 ICS outputs must remain byte-identical; no renderer, cache, fetch-pipeline, or UID-formula change (a second round of UID churn is not acceptable when the first round was already accepted in ADR-0024).
- **Single home for the inverse**: the inverse of ADR-0024's fetch-time append belongs in package `calendar` next to `SITCampusAddress`, not inlined into each server view.
- **No schema change**: the JSON shape (fields, types, order) must remain unchanged, so `wakeySchemaVersion` (ADR-0021) stays 1.
- **Result-identical `event_type`**: the `event_type` classification (`strings.EqualFold` against "Online") must yield identical results after stripping.

## Considered Options

### Option A: View-Time Strip via Shared calendar.Helper

Add an exported `calendar.BaseVenue` helper in `internal/calendar/campus_address.go` that removes one trailing `SITCampusAddress`, and apply it at exactly the two JSON mapping sites that currently emit `Location: e.Location`: `newWakeySitizenHandler` (`internal/server/handlers.go:193`) and `renderWakeyStatusJSON` (`internal/server/status.go:173`).

### Option B: Store the Raw Location, Append at ICS Render Time

Revert the storage model: keep raw venues in the cache and append the address inside the ICS render path. Rejected: ADR-0024 already rejected render-time append for the reason that the disk round-trip (`parseICS`, `internal/calendar/persistence.go`) reads rendered `LOCATION` back verbatim into `Event.Location`, so after a restart every render would double-append on top of already-addressed locations. Reverting would additionally change the hash input of `EventID` (UIDs currently hash the addressed location), causing a second UID churn for every in-campus event, and would rewrite ADR-0024's accepted storage decision instead of narrowly following it.

### Option C: Inline Strip Duplicated in Both Server Views

Apply `strings.TrimSuffix(e.Location, calendar.SITCampusAddress)` directly in each of the two handlers, with no calendar-package helper. Rejected: scatters the inverse of the fetch-time append across the server package, away from the calendar package that owns the constant and the append; the inverse would have no single testable home and could drift from `WithSITCampusAddress` semantics.

### Option D: Status Quo

Keep the address in the JSON views (ADR-0024's accepted side effect). Rejected: does not meet the explicit new user requirement.

## Decision

We will choose **Option A: View-Time Strip via Shared calendar.Helper**.

## Rationale

### New calendar package API

`internal/calendar/campus_address.go` (next to the constant) gains:

```go
// BaseVenue returns location with one trailing occurrence of
// SITCampusAddress removed, unchanged otherwise. It is the inverse of
// WithSITCampusAddress for in-campus events (ADR-0026).
func BaseVenue(location string) string {
	return strings.TrimSuffix(location, SITCampusAddress)
}
```

One trailing occurrence is removed — exactly the inverse of the append, which is idempotent (the `HasSuffix` guard in `WithSITCampusAddress` ensures an addressed location never carries the constant twice).

### Wiring

Exactly two mapping sites change, both currently `Location: e.Location`:

- `newWakeySitizenHandler` (`internal/server/handlers.go:193`) → `Location: calendar.BaseVenue(e.Location)` — serves `GET /wakey-sitizen/timetable.json` (ADR-0021).
- `renderWakeyStatusJSON` (`internal/server/status.go:173`) → `Location: calendar.BaseVenue(e.Location)` — renders the wakey representation served by `GET /status` (ADR-0023).

### What Is Unchanged

- **ICS rendering and the fetch-time append**: `WithSITCampusAddress` still runs in the `runFetchWithStatus` loop before `cache.UpdateWithSourceReconciliation`; all 7 ICS outputs remain byte-identical (LOCATION property, `EscapeText`/`FoldText` output, and the `/status` SHA-256 hashes).
- **Persistence round-trip**: `parseICS` still reads `LOCATION` back verbatim into `Event.Location`; the stored model is unchanged.
- **EventID/UID formula (ADR-0003)**: `BaseVenue` is not part of the hash input; no UID churn results from this change.
- **`event_type` computation**: stays `strings.EqualFold(e.Location, "Online")` on the stored location; online events never carry the address (`IsConcreteVenue` is false for "online", so `WithSITCampusAddress` never appends), so the classification is result-identical on stripped values.
- **`wakeySchemaVersion` stays 1** (`handlers.go:94`): the JSON shape — fields, types, order — is unchanged; only field values become shorter. ADR-0021 bumps only on breaking changes; ADR-0024's accepted side effect set the same precedent.
- **`WakeyEvent` struct and JSON field set** (`handlers.go:105`) are untouched.

### One-Time ETag Invalidation

After deployment the wakey JSON body changes for every addressed cached event, so the ETag of `/wakey-sitizen/timetable.json` (`calendar.ComputeETag` over the body) and the `/status` wakey ETag (`etags` map, `status.go:57`) invalidate once and clients refetch once. ICS ETags and the `/status` SHA-256 hashes are stable (outputs byte-identical). UIDs are unaffected (`BaseVenue` is not in the EventID path).

### Known Asymmetry (Documented, Not a Defect)

A hypothetical venue whose text itself ends with the full `SITCampusAddress` constant is left unaddressed at fetch time (the `HasSuffix` idempotency guard in `WithSITCampusAddress` leaves it unchanged) yet would be stripped in the JSON views — the JSON would show the truncated venue while the ICS keeps the full text. This is effectively unreachable in practice: venues are short room/building codes.

### Data Privacy

The only real data in the repo remains the single user-authorized public institutional address constant (ADR-0024); all test fixtures stay synthetic or reference the constant only.

## Consequences

### Positive

- The companion app JSON is clean: the `location` field carries the base venue only (e.g. "W1-05-07"); the ICS `LOCATION` keeps the full address for click-to-maps in calendar clients.
- The inverse of the fetch-time append has a single home in package `calendar`, unit-testable next to the constant and the append.
- No schema bump, no UID churn, no renderer/cache/fetch changes; all 7 ICS outputs byte-identical.

### Negative

- One-time wakey ETag invalidation: after deployment the `/wakey-sitizen/timetable.json` ETag and the `/status` wakey ETag change once and clients refetch once.
- Known asymmetry: a venue ending in the address constant is left unaddressed at fetch time but stripped in the JSON views (effectively unreachable, documented above).

### Neutral / Operational

- Outputs affected: the wakey JSON bodies only (endpoint response and `/status` representation). The 7 ICS files, their ETags, and the `/status` SHA-256 hashes are unchanged.
- `event_type` is result-identical: online events never carry the address, so the stored-location `strings.EqualFold` classification is unchanged by the strip.
- Tests planned (per the implementation plan): a `calendar` package `TestBaseVenue` table test plus an inverse-of-append invariant test (`BaseVenue` of an appended concrete-venue location returns the trimmed base venue; unchanged for non-venue locations); a server package extension of `TestNewWakeySitizenHandler_Basic` (seeded location carries the address; JSON returns the base venue; escaped-form leak assertion on the response body, because `json.Marshal` encodes real newlines as the two-character `\n` sequence); a new `TestNewWakeySitizenHandler_BaseVenueOnlyInJSON` covering addressed, plain, and online venue classes plus `event_type`; and a new `/status` test proving the rendered wakey representation exposes base venues, via ETag-equality with `renderWakeyStatusJSON` output.
- Verification gate: `go vet` + `go test -count=1 -race` (repo toolchain via mise: `mise exec go -- go ...`; set `GOCACHE=/tmp/sit-ics-gocache` if the default build cache is sandbox-blocked). Known pre-existing failure: `internal/totp` `TestGenerateAtOffset`/`-1_period` (VULN-2026-001, documented, NOT fixed here).

## Follow-Ups

- No required follow-ups.
- ADR-0024 is not edited (ADRs are immutable); this ADR records the partial supersession.

## Related ADRs

- **ADR-0003**: ICS Generation and Caching Strategy (deterministic `EventID`/UID formula deliberately unchanged — `BaseVenue` is not in the UID path)
- **ADR-0021**: Wakey-SITizen JSON Timetable Endpoint (`location` field becomes the base venue; `schema_version` remains 1; one-time ETag invalidation)
- **ADR-0023**: Server Status and Manual Refresh Endpoints (the `/status` wakey representation is stripped identically; ICS SHA-256 hashes stable)
- **ADR-0024**: Append SIT Campus Address to In-Campus Event Locations (partially superseded — its "Side Effects on /wakey-sitizen/timetable.json and /status" section, i.e. the "accepted and not special-cased" statement, and its corresponding negative-consequence bullet are superseded by this ADR; the ICS decision remains in force and ADR-0024 stays Accepted in the index and is not edited)
