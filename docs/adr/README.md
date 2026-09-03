# Architecture Decision Records

This directory contains Architecture Decision Records (ADRs) for the SIT ICS Go project.

| Number | Title | Status | Date |
|--------|-------|--------|------|
| [0001](0001-adfs-authentication-and-peoplesoft-timetable-retrieval-via-http-client.md) | ADFS Authentication and PeopleSoft Timetable Retrieval | Superseded | 2026-09-01 |
| [0002](0002-configuration-env-vars-and-cli-flags.md) | Configuration via Environment Variables (Primary) with CLI Flags (Fallback) | Proposed | 2026-09-01 |
| [0003](0003-ics-generation-and-caching-strategy.md) | ICS Generation and Caching Strategy | Proposed | 2026-09-01 |
| [0004](0004-cron-scheduling-and-timezone-handling.md) | Cron Scheduling and Timezone Handling | Proposed | 2026-09-01 |
| [0005](0005-browser-abstraction-layer-with-rod.md) | Browser Abstraction via AuthBrowser Interface with Rod Implementation | Proposed | 2026-09-02 |
| [0006](0006-authentication-flow-and-context-isolation.md) | Authentication Flow with Isolated Browser Contexts and Rod Waiting Primitives | Proposed | 2026-09-02 |
| [0007](0007-error-handling-and-timeout-hierarchy.md) | Application-Facing Error Model with Wrapped Rod Errors | Proposed | 2026-09-02 |
| [0008](0008-testing-strategy-with-three-layers.md) | Three-Layer Testing Strategy with Mock AuthBrowser | Proposed | 2026-09-02 |
| [0009](0009-complete-peoplesoft-authentication-via-samlresponse-submission.md) | Complete PeopleSoft Authentication via SAMLResponse Submission | Proposed | 2026-09-03 |
| [0010](0010-natural-browser-redirect-for-adfs-peoplesoft-saml-exchange.md) | Natural Browser Redirect for ADFS→PeopleSoft SAML Exchange | Accepted | 2026-09-03 |
| [0011](0011-browser-based-timetable-fetch-instead-of-http-client.md) | Browser-Based Timetable Fetch Instead of HTTP Client with Cookie Injection | Proposed | 2026-09-03 |

## Superseded ADRs

- **ADR-0001**: Superseded by ADR-0005 and ADR-0006. Original decision (HTTP client) invalidated by verified ADFS behavior requiring JavaScript execution.
