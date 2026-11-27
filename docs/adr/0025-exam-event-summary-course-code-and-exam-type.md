# ADR-0025: Exam Event Summary: Course Code and Exam Type

## Status

Accepted

## Date

2026-11-27

## Context

The application builds PeopleSoft exam-timetable events into the in-memory event cache (ADR-0003) during the fetch: the exams block of runFetchWithStatus (internal/app/fetch.go, exam construction block at lines 160-184) parses the PeopleSoft exam page (ParseExamHTML, internal/peoplesoft/exam_parser.go) into the Exam struct (internal/peoplesoft/exam.go: Date, Time, CourseCode, CourseName, Location, SeatNumber, ExamType, with ExamType values like "Final Exam") and converts each exam to a calendar.Event with Source "peoplesoft-exams". Exam events are rendered to a dedicated ICS output (timetable-exams.ics via ICS_EXAMS_PATH, served at /timetable-exams.ics) and also appear in the main combined timetable (timetable.ics); after a complete successful exams fetch, "peoplesoft-exams" is registered as an authoritative source so reconciliation (ADR-0003) can delete absent upcoming exam events. The exams fetch and its ICS output were added recently and have no dedicated ADR; the nearest records are ADR-0003 (cache/UID/reconciliation), ADR-0014 (explicit ICS endpoint pattern), and ADR-0023 (server status and output plumbing). The VEVENT title is the SUMMARY property, rendered by the shared writer from Event.Summary (internal/calendar/render.go, line 95).

The exam summary is currently built as:

```go
Summary: fmt.Sprintf("%s — %s (%s)", exam.CourseCode, exam.CourseName, exam.ExamType)
```

i.e. CODE — Full Module Name (Exam Type).

User requirement: "For exam events, change the VEVENT title to only include [Course] - [Type], and do not include the full name of the module in the title. retain all info in the description."

The decision must determine the new SUMMARY format, which other event fields are affected, and what the change does to the deterministic UID formula (ADR-0003).

## Decision Drivers

- **Exact user-specified format**: the new SUMMARY is exactly [Course] - [Type] — the module code, a single ASCII hyphen with surrounding spaces, and the exam type, e.g. FAK2501 - Final Exam. The full module name must not appear in the title.
- **Retain all info in the description**: the user explicitly requires that all information, including the full module name, remain in DESCRIPTION.
- **"VEVENT title" means SUMMARY only**: Event.Title (the full module name) is a separate field that the renderer emits as the X-PeopleSoft-Title metadata property (render.go, non-brightspace branch), not as the title; it must keep carrying the module name.
- **No UID formula change**: EventID (internal/calendar/render.go) hashes Summary; the formula itself must not change (changing it would break idempotency for all events — ADR-0003).
- **Round-trip stability**: on startup, LoadFromFile/parseICS (internal/calendar/persistence.go) reads the rendered SUMMARY back verbatim into Event.Summary; a subsequent fetch must build a byte-identical summary so UIDs (which hash Summary) remain stable and reconciliation (ADR-0003) stays idempotent — the format must be built at event construction, not at render time (the same reasoning ADR-0024 used for LOCATION).
- **Scope**: only exam events (Source "peoplesoft-exams") change; the regular PeopleSoft timetable event summaries (CODE - Section (Type), fetch.go line 106) and BrightSpace event titles are untouched.

## Considered Options

### Option A: App-Layer Format Change (One Line in fetch.go)

Change only the exam Summary format string in the exams construction block of runFetchWithStatus to fmt.Sprintf("%s - %s", exam.CourseCode, exam.ExamType). No parser, renderer, cache, or UID-formula changes.

### Option B: Keep the Module Name, ASCII Hyphen Only

fmt.Sprintf("%s - %s (%s)", exam.CourseCode, exam.CourseName, exam.ExamType) — fixes the em-dash rendering defect but keeps the module name in the title. Rejected: does not meet the explicit user requirement to drop the module name from the title.

### Option C: Render-Time Title Rewrite

Rewrite the SUMMARY of "peoplesoft-exams" events inside Render. Rejected: the disk round-trip reads rendered SUMMARY back verbatim into Event.Summary, so after a restart the in-memory model rebuilt by the next fetch (old format) would diverge from the persisted one (new format); UIDs computed from the in-memory Summary in mergeEvents would churn on every cycle, and the JSON and /status views (which read Event.Summary) would diverge from the ICS.

### Option D: Change the EventID Formula to Ignore the Summary

Recompute UIDs so title changes do not change UIDs. Rejected: changing the UID formula breaks idempotency for all events (ADR-0003) — every event, not just exam events, would get new UIDs, and the existing "any content change changes the UID" semantics (which reconciliation and subscribers rely on) would no longer hold.

## Decision

We will choose **Option A: App-Layer Format Change (One Line in fetch.go)**.

## Rationale

### The Change

Exactly one production line changes — the exam Summary in internal/app/fetch.go:

```go
Summary: fmt.Sprintf("%s - %s", exam.CourseCode, exam.ExamType),
```

For an exam with CourseCode "FAK2501" and ExamType "Final Exam", the rendered title becomes SUMMARY: FAK2501 - Final Exam. The separator is a plain ASCII hyphen with a space on each side.

### What "VEVENT title" Means

SUMMARY is the VEVENT title (render.go renders SUMMARY from Event.Summary at line 95). Event.Title stays exam.CourseName; because exam events carry Source "peoplesoft-exams" (not brightspace-), the renderer emits it as the X-PeopleSoft-Title metadata property (render.go, non-brightspace branch, lines 114-123), so the full module name continues to be carried on the event.

### Description Unchanged

Event.Description stays fmt.Sprintf("Course: %s\nModule: %s\nSeat: %s\nType: %s", exam.CourseCode, exam.CourseName, exam.SeatNumber, exam.ExamType) — all information, including the full module name, is retained in the description as required.

### Side Effect (Fix): the Em Dash Was Silently Stripped

The old format used an em dash (U+2014, bytes e2 80 94) between the code and the module name. The renderer's EscapeText (render.go, lines 154-177) strips non-ASCII characters, so today's rendered SUMMARY is actually "FAK2501  Fake Exam Module (Final Exam)" — with a doubled space where the em dash was dropped. The new ASCII "-" renders as intended.

### One-Time UID Churn (Accepted)

EventID (render.go, lines 29-72) hashes Summary among the other fields; for events with a non-empty Source (exam events) the hash input is Source-Summary-Location-DTStart(date)-DTStart(time)-DTEnd(time). UIDs are computed over incoming events in mergeEvents (internal/calendar/cache.go, line 460). Because Summary changes, every "peoplesoft-exams" event gets a new UID on the first fetch after deployment. Reconciliation (ADR-0003) then replaces upcoming exam events in subscribers' calendars and retains past ones, which keep the old title until they naturally expire. This is consistent with the existing semantics where any content change already changes the UID. The EventID formula is deliberately not changed (changing it would break idempotency of all events — ADR-0003).

### No Parser Changes

internal/peoplesoft/exam_parser.go already yields ExamType and CourseName as separate Exam fields; the format change is purely in the app-layer event construction, and the parser, browser, and auth layers are untouched.

### Tests

- Two existing seed fixtures in internal/app/fetch_test.go carry the old module-name-in-summary format: "FAK2501 - Fake Exam Module (Final Exam)" (line 634, TestExamsFetchFailureRetainsExamEvents) and "FAK2501 - Stale Exam Module (Final Exam)" (line 699, TestExamsFetchSuccessReconcilesExamEvents). These are cache-seed fixtures, not assertions on fetched output, so the tests pass either way; they will be updated to the new format ("FAK2501 - Final Exam") for fixture fidelity.
- TestExamsFetchSuccessReconcilesExamEvents (internal/app/fetch_test.go, lines 664-730) will be extended to assert that the fetched exam event has Summary exactly "FAK2502 - Final Exam", Title unchanged ("Synthetic Statistics Module"), and Description still containing all four fields, including "Module: Synthetic Statistics Module".

### Side Effects on /wakey-sitizen/timetable.json and /status

The shared event model means the JSON summary field for exam events (exam events are included in that endpoint — only brightspace-* sources are excluded, ADR-0021) and the /status rendered views (ADR-0023) will show the new format. The user scoped the request to the VEVENT SUMMARY property, but this consequence is accepted and not special-cased. The JSON shape is unchanged, so no schema_version bump is required.

### Scope

Only events with Source "peoplesoft-exams" change. Regular PeopleSoft timetable events keep their CODE - Section (Type) summaries (fetch.go, timetable construction block, line 106), and BrightSpace event titles/summaries are untouched. The exams output plumbing — timetable-exams.ics via ICS_EXAMS_PATH (internal/config/config.go, line 22) served at /timetable-exams.ics (internal/server/server.go, line 59) and written by saveAllOutputs → cache.SaveOutputs — is unchanged.

## Consequences

### Positive

- Exam VEVENT titles are compact and uniform: CODE - Exam Type, legible in calendar clients even when the module name is long.
- All information, including the full module name, remains available: in DESCRIPTION and in the X-PeopleSoft-Title property.
- The old em-dash doubled-space rendering defect is fixed as a side effect, with no renderer changes.
- A one-line change with no new dependencies or configuration; the parser, renderer, cache, and server code are untouched.
- Deterministic and round-trip stable: identical upstream exam data yields identical Summary and EventID on every fetch, including across restarts.

### Negative

- One-time UID churn on deployment: every exam event gets a new UID on the first fetch after deploy; upcoming exam events are replaced in subscribers' calendars while past exam events retain the old title until they expire.
- The /wakey-sitizen/timetable.json summary field for exam events and the /status views now show the shortened format even though the request was scoped to the ICS property; accepted and not special-cased.

### Neutral / Operational

- Outputs affected: timetable.ics, timetable-exams.ics, and the in-campus exam events in timetable-campus.ics (the campus projection includes non-BrightSpace, non-online events), plus the /status rendered views. All xsite-*.ics outputs are unchanged.
- In a degraded cycle after deployment in which the exams fetch fails (fetch.go retains the existing exam events and does not register "peoplesoft-exams" as authoritative), retained exam events simply keep the old title until the next successful exams fetch.
- The exams fetch and its ICS output (timetable-exams.ics, Source "peoplesoft-exams") were added recently and have no dedicated ADR; this ADR documents only the summary-format decision and cites ADR-0003 and ADR-0014 for the cache and endpoint plumbing.

## Follow-Ups

- No required follow-ups.
- If a dedicated ADR for the exams fetch/output is written later, it should reference this ADR for the SUMMARY format.

## Related ADRs

- **ADR-0003**: ICS Generation and Caching Strategy (deterministic EventID/UID formula and reconciliation semantics — the formula is deliberately unchanged; one-time UID churn accepted)
- **ADR-0014**: Xsite ICS HTTP Endpoints (explicit-endpoint pattern of the ICS outputs, including timetable-exams.ics)
- **ADR-0021**: Wakey-SITizen JSON Timetable Endpoint (accepted side effect on the summary field for exam events)
- **ADR-0023**: Server Status and Manual Refresh Endpoints (/status rendered views show the new format)
- **ADR-0024**: Append SIT Campus Address to In-Campus Event Locations (companion change to exam VEVENTs — LOCATION content; same round-trip/UID reasoning)
