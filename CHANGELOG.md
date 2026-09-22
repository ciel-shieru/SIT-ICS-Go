# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- **Smart Merge**: Merge BrightSpace Zoom events into PeopleSoft online classes for consolidated view
- **XSite Blocklist Wildcards**: Added wildcard (`*`) support to xsite blocklist patterns
- **Request Logging**: Added JSON request logging middleware with trusted proxy support
- **Smart Merge Module Validation**: Added module code format validation for better matching
- **CourseCode Population**: Populate CourseCode for BrightSpace events to improve matching
- **Substring Matching**: Use substring matching for module code comparison in smart merge
- **XSite Default Enabled**: XSite features now default to enabled (opt-out model)
- **Raw HTML Preservation**: Preserve raw HTML for BrightSpace descriptions with Zoom links

### Changed
- **Context Propagation**: Propagate context cancellation to gracefully abort fetch on SIGINT
- **XSite Rebranding**: Renamed BrightSpace configuration to xsite prefix across codebase
- **API Response Wrapping**: Wrap D2L API responses for quizzes, calendar events, and dropbox folders
- **Array Unmarshal**: Revert calendar events and dropbox folders to bare array unmarshal
- **Config Rename**: Renamed ICS_CAMPUS_ALERTS to TIMETABLE_CAMPUS_ALERTS
- **ICSCampusAlerts Rename**: Renamed ICSCampusAlerts to TimetableCampusAlerts and flag to timetable-campus-alerts
- **Server Bind Address**: Bind to 127.0.0.1 by default, 0.0.0.0 for container builds
- **CLI Help**: Show environment variable names in CLI --help descriptions
- **Data Privacy**: Added data privacy section to AGENTS.md

### Fixed
- **Smart Merge Matching**: Improve matching logic for BrightSpace events
- **Brightness End Time**: Fallback to end time when start time is empty
- **XSite Blocklist**: Added wildcard support to xsite blocklist patterns
- **D2L API Responses**: Wrap D2L API responses for quizzes, calendar events, and dropbox folders
- **Domain Validation**: Added comprehensive domain validation tests for HIGH-2 security fix

### Removed
- **API Key/Secret**: Removed BrightSpace API key and secret configuration

## [0.0.15] - 2026-09-22

### Added
- ETag, If-None-Match, Last-Modified, If-Modified-Since, and 304 Not Modified support for HTTP caching

### Changed
- XSite features now default to enabled (opt-out model instead of opt-in)

### Fixed
- Smart merge matching logic for Better matching

## [0.0.14] - 2026-09-22

### Added
- Wildcard (`*`) support to xsite blocklist patterns

### Changed
- Context propagation to gracefully abort fetch on SIGINT

## [0.0.13] - 2026-09-22

### Added
- JSON request logging middleware with trusted proxy support

## [0.0.12] - 2026-09-22

### Added
- BrightSpace quizzes endpoint and ICS output

### Changed
- Bind server to 127.0.0.1 by default, 0.0.0.0 for container builds

## [0.0.11] - 2026-09-22

### Added
- BrightSpace fallback to end time when start time is empty

## [0.0.10] - 2026-09-22

### Fixed
- BrightSpace revert calendar events and dropbox folders to bare array unmarshal

### Changed
- Wrap D2L API responses for quizzes, calendar events, and dropbox folders

## [0.0.9] - 2026-09-22

### Added
- BrightSpace D2L event integration

### Changed
- Rebrand BrightSpace external artifacts to xsite prefix
- Remove BrightSpace API key/secret configuration

## [0.0.8] - 2026-09-22

### Added
- Domain validation tests for HIGH-2 security fix

## [0.0.7] - 2026-09-22

### Added
- CLI help shows environment variable names

### Changed
- Rename ICSCampusAlerts to TimetableCampusAlerts
- Rename ICS_CAMPUS_ALERTS to TIMETABLE_CAMPUS_ALERTS

## [0.0.6] - 2026-09-22

### Added
- Data privacy section to AGENTS.md

## [0.0.5] - 2026-09-22

### Added
- Smart merge: Merge BrightSpace Zoom events into PeopleSoft online classes

## [0.0.4] - 2026-09-22

### Added
- Server endpoints for xsite events, dropbox, quizzes, and combined xsite

## [0.0.3] - 2026-09-22

### Added
- BrightSpace calendar events integration

## [0.0.2] - 2026-09-22

### Added
- PeopleSoft timetable parsing via browser

## [0.0.1] - 2026-09-22

### Added
- Initial release with ADFS authentication, timetable fetching, and ICS generation
