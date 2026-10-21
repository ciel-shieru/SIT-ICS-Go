# ADR-0023: Server Status and Manual Refresh Endpoints

- **Status:** Proposed
- **Date:** 2026-10-21

## Context

The companion app needs to display the server's latest fetch-job time and integrity information, and allow the user to request a fresh fetch without waiting for the cron schedule. Status inspection must not read the generated ICS files from disk because the service already keeps the source events in memory and disk reads cause unnecessary storage activity.

## Decision

Expose two additional HTTP endpoints:

- `GET /status` returns JSON with `last_fetch_job_time` (UTC RFC 3339 timestamp or null before the first job), `job_status` (`running` or `finished`), `last_status_in4sit` and `last_status_xsite` (`running`, `successful`, `failed`, or `unknown` before any corresponding attempt), an `etags` map for each currently served ICS/JSON representation, and a `sha256` map for the seven generated ICS outputs. Source statuses update when corresponding fetches start/finish and retain their most recent result otherwise. Every representation is rendered from the in-memory event cache and current alert configuration; this handler does not read any output file from disk. Responses use `Cache-Control: no-store`.
- `POST /refresh` returns HTTP 202 with a JSON acknowledgement after scheduling an immediate fetch. Requests during a running job are coalesced into one follow-up fetch so the request is not silently lost and concurrent fetches do not overlap.

Both handlers reject other HTTP methods with HTTP 405 and an `Allow` header.

## Consequences

The status hashes describe the current in-memory render of each output, not arbitrary external modifications to the persisted files. As with the existing timetable endpoints, access to `/refresh` follows the configured server bind address and deployment network controls; deployments exposing the service beyond a trusted network should restrict access at the network boundary.
