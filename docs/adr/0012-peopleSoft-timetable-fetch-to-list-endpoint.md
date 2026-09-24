# ADR-0012: Switch PeopleSoft Timetable Fetch to SSR_SSENRL_LIST Endpoint

## Status

Proposed

## Date

2026-09-05

## Context

ADR-0011 established browser-based timetable fetch using the PeopleSoft endpoint `SA_LEARNER_SERVICES.SSR_SSENRL_SCHD_W.GBL`. This endpoint uses a weekly grid layout — it requires navigating to the endpoint with a week-date parameter, submitting a POST form, and parsing a table where columns represent days of a single week. The application had to iterate week-by-week (default: 180-day range, ~26 iterations) to collect all scheduled classes.

Each weekly fetch required:
1. Navigating to the start page (`SSR_SSENRL_SCHD_W.GBL?ICAGTarget=start`)
2. Navigating to the timetable endpoint with week-date query params
3. Waiting for the AJAX response
4. Extracting the weekly grid HTML
5. Parsing a 7-column grid with rowspan-based cell tracking

The HTML structure used full day names (`Monday 21 Sep`) in the table header and embedded course code, section, type, time range, and location as newline-separated text within colored grid cells.

PeopleSoft has an alternate endpoint `SA_LEARNER_SERVICES.SSR_SSENRL_LIST.GBL` (My Class Schedule) that returns all scheduled classes in a single response without date parameters. This endpoint uses a list-based layout with per-meeting-date rows grouped by course, rather than a weekly grid.

## Decision Drivers

- **Single fetch for all classes**: Eliminate the week-by-week iteration loop; one navigation returns the entire semester schedule
- **Include class name**: The new endpoint returns the full class name (e.g., "Computer Organization and Architecture") alongside the course code, which should be included in ICS event descriptions
- **Simpler HTML structure**: Per-meeting-date rows are easier to parse than a weekly grid with rowspan-based cell tracking
- **Reduce browser navigations**: Each weekly fetch required 2 navigations (start page + timetable); the new endpoint requires only 1

## Considered Options

### Option A: Switch to SSR_SSENRL_LIST.GBL (No Parameters)

Navigate directly to `SA_LEARNER_SERVICES.SSR_SSENRL_LIST.GBL` with no URL params or form data. Parse the list-based HTML structure where courses are grouped by `DERIVED_REGFRM1_DESCR20$` divs, each containing a `CLASS_MTG_VW` table with per-meeting-date rows. Each row has columns: Class Nbr, Section, Component, Days & Times, Room, Instructor, Start/End Date.

### Option B: Keep SSR_SSENRL_SCHD_W.GBL, Optimize Weekly Iteration

Keep the existing weekly grid endpoint but optimize the iteration — e.g., fetch a larger date range per call, or reduce the number of weeks fetched. Continue parsing the weekly grid with rowspan tracking.

### Option C: Use Both Endpoints as Fallback

Use the new list endpoint as primary and fall back to the weekly grid endpoint if the list endpoint fails or returns no data. Maintain both parsers.

## Decision

We will choose **Option A: Switch to SSR_SSENRL_LIST.GBL (No Parameters)**.

## Rationale

### Why Option A over Option B

Option B retains the fundamental inefficiency of the weekly grid approach: ~26 browser navigations for a 180-day range, each requiring form submission and AJAX waiting. The weekly grid parsing is also more complex — it requires rowspan resolution, visited-cell tracking, and full-day-name parsing. Option A reduces fetches to a single navigation and uses a simpler row-based parser.

### Why Option A over Option C

Option C doubles the maintenance burden by keeping two parsers and two endpoints in the codebase. The list endpoint is the canonical "My Class Schedule" view provided by PeopleSoft — it is the intended interface for this data. There is no evidence that the weekly grid endpoint provides additional value that the list endpoint cannot.

### How the New Structure Differs

**Old (`SSR_SSENRL_SCHD_W.GBL`)**: Weekly grid layout
- Table with 7 day-columns (Mon–Sun) and time-row headers
- Each cell contains: `CourseCode - Section\nType\nTimeRange\nLocation`
- Header format: `Monday 21 Sep`, `Tuesday 22 Sep`, etc.
- Required rowspan tracking and visited-cell deduplication

**New (`SSR_SSENRL_LIST.GBL`)**: List layout with course groups
- Courses grouped by `div[id^="win0divDERIVED_REGFRM1_DESCR20$"]` containers
- Each group has a header row with `PAGROUPDIVIDER` class containing: `CourseCode - ClassName`
- Meeting table (`CLASS_MTG_VW`) with columns: Class Nbr, Section, Component, Days & Times, Room, Instructor, Start/End Date
- Each row is one meeting instance with its own start/end date and day/time
- Day format: abbreviated (`Mo`, `Tu`, `We`, `Th`, `Fr`, `Sa`, `Su`)

**Parsing changes**:
- `dayDateRe` (full day names + date) replaced with `dayAbbrevRe` (abbreviated day)
- Removed weekly grid traversal with rowspan tracking and visited-cell deduplication
- Added `findCourseHeader()` to extract course code and class name from the `PAGROUPDIVIDER` row within each course group div
- Added `computeEntryDay()` to calculate the actual meeting date from the row's start/end date + day abbreviation (e.g., meeting date `17/09/2026` (Thursday) + schedule day `Thu` → entry day `17/09/2026`; meeting date `17/09/2026` + schedule day `Mo` → entry day `21/09/2026`)
- `parseMeetingDate()` extracts only the start date from `Start/End Date` column (format: `DD/MM/YYYY - DD/MM/YYYY`)

**Fetch flow changes**:
- `fetchTimetable()` no longer accepts a `weekDate` parameter (passes empty string)
- Removed navigation to start page; navigates directly to list endpoint
- Added `page.WaitNavigation(proto.PageLifecycleEventNameNetworkAlmostIdle)` for AJAX stabilization
- Uses `page.HTML()` instead of `page.Element("body").Text()` for full HTML extraction
- `main.go` removed the week-by-week `for` loop; calls `FetchTimetable(ctx, "")` once

**Entry struct changes**:
- Added `ClassName string` field to `peoplesoft.Entry`
- ICS event description now includes `Class: {ClassName}` between `Course:` and `Section:`

## Consequences

### Positive
- **Single fetch**: One browser navigation replaces ~26 weekly navigations for a 180-day range
- **Simpler main.go**: Removed the week iteration loop, start date / end date range logic for fetching, and cumulative entry collection
- **Class name in events**: ICS descriptions now include the full class name (e.g., "Computer Organization and Architecture")
- **Simpler parser**: Row-based parsing replaces grid traversal with rowspan tracking and visited-cell deduplication
- **More reliable date computation**: Each meeting row carries its own start/end date, reducing ambiguity about which week a class falls in

### Negative
- **Tighter coupling to PeopleSoft DOM**: The parser relies on specific div IDs (`win0divDERIVED_REGFRM1_DESCR20$*`) and table classes (`CLASS_MTG_VW`) that PeopleSoft may change
- **Loss of weekly grid view**: If PeopleSoft changes the list endpoint but retains the weekly grid, there is no fallback parser in the codebase
- **Day abbreviation ambiguity**: The abbreviated day format (`Mo`, `Tu`, etc.) is less self-describing than full day names; if PeopleSoft changes abbreviations, the regex must be updated
- **`weekDate` parameter removed from `FetchTimetable`**: Callers no longer pass a week date; the function signature changed from `FetchTimetable(ctx, weekDate)` to `FetchTimetable(ctx, "")`

### Neutral / Operational
- The `peoplesoft.Entry` struct gained a `ClassName` field; existing serialized data will have an empty `ClassName` (preserved by ICS upsert semantics)
- The `ExtractYear()` function remains unchanged and derives the year from the `Start/End Date` column
- Debug logging (`BROWSER_DEBUG`) covers both the new navigation and HTML extraction steps

## Alternatives Considered

### Option B: Keep Weekly Grid, Optimize Iteration

**Summary**: Retain `SSR_SSENRL_SCHD_W.GBL` but reduce the number of weekly fetches by increasing the date range per call or caching more aggressively.

**Benefits**: Preserves the existing parser; the weekly grid is a well-understood structure.

**Costs**: Still requires multiple browser navigations; the weekly grid parser remains complex with rowspan and visited-cell tracking; no class name is available in this view.

**Reason rejected**: The fundamental inefficiency (multiple navigations) and parsing complexity remain. Option A eliminates both.

### Option C: Dual-Endpoint Fallback

**Summary**: Try the list endpoint first; fall back to the weekly grid if it fails. Maintain both parsers.

**Benefits**: Resilience against PeopleSoft endpoint changes; no single point of failure.

**Costs**: Doubles parser maintenance; adds fallback logic complexity; the weekly grid code path would be dead code in normal operation.

**Reason rejected**: The list endpoint is the canonical PeopleSoft "My Class Schedule" view. There is no evidence it is less stable than the weekly grid. The maintenance cost of two parsers outweighs the marginal resilience benefit.

## Follow-Ups

- Monitor PeopleSoft for any changes to the `SSR_SSENRL_LIST.GBL` page structure
- If PeopleSoft adds filtering/pagination to the list endpoint, the parser may need to handle multiple pages
- Related ADRs:
  - **ADR-0011**: Browser-based timetable fetch (superseded — endpoint and parser changed)
  - **ADR-0005**: Browser abstraction layer (`FetchTimetable` signature changed from `(ctx, weekDate)` to `(ctx, "")`)

## References

- [PeopleSoft SSR_SSENRL_LIST.GBL](https://in4sit.singaporetech.edu.sg/psc/CSSISSTD/EMPLOYEE/SA/c/SA_LEARNER_SERVICES.SSR_SSENRL_LIST.GBL) — My Class Schedule endpoint
- **ADR-0011**: Previous browser-based fetch using `SSR_SSENRL_SCHD_W.GBL` weekly grid endpoint
- **ADR-0005**: Browser abstraction layer (extended by `FetchTimetable` signature change)
