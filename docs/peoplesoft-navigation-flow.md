# PeopleSoft Navigation Flow

This documents the current timetable navigation after ADFS authentication. The browser remains open after authentication, and the browser implementations use `fetchTimetable()` in `internal/browser/shared.go`.

1. **Open the timetable endpoint.** `fetchTimetable()` validates the page and context, creates a context bounded by `BrowserConfig.NavigationTimeout`, and navigates to `https://in4sit.singaporetech.edu.sg/psc/CSSISSTD/EMPLOYEE/SA/c/SA_LEARNER_SERVICES.SSR_SSENRL_LIST.GBL`.
2. **Wait for the initial page.** It waits for Rod's `NetworkAlmostIdle` navigation waiter, then calls `WaitStable(5000)`. A stable-wait error is logged in debug mode and is not fatal.
3. **Check for a change-term control.** `handleTermSelection()` uses a five-second context. If `input[name='DERIVED_SSS_SCT_SSS_TERM_LINK']` is found, it attempts to click it. Click failures are debug-logged; the code then attempts to wait for `input.PSRADIOBUTTON`, logging rather than returning if that wait fails.
4. **Select the term.** The function next requires `input.PSRADIOBUTTON` to exist, obtains the matching radio buttons, and clicks the last one. Failure to find the selector or click the radio button returns an error.
5. **Continue to the timetable.** It finds and clicks `input[name='DERIVED_SSS_SCT_SSR_PB_GO']`, waits until at least one `table.PSGROUPBOXWBO` exists, and calls `WaitStable(5000)` again. A stable-wait error is debug-logged and is not fatal.
6. **Return the page.** `fetchTimetable()` reads the full page HTML with `page.HTML()` and returns it for parsing by `peoplesoft.ParseTimetableHTML()`.

**Current behavior note:** term selection is not fully optional. If `input.PSRADIOBUTTON` is absent, the required `Element()` check returns an error before the later empty-list check can skip selection. This document records current behavior; it does not imply that absence of the term-selection screen is currently handled successfully.
