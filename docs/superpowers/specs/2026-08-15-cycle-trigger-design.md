# Cycle trigger — design doc

Date: 2026-08-15
Status: Approved (user)
Branch: main

## Subject

kbarr is a self-hosted anime library manager built as four services — core
(API, port 8282), indexer (8082), downloader (8083), metadata (8081). Each
service runs background poll loops that do the work: monitor polls, download
processing, availability checks, metadata refreshes, AniDB title syncs. Every
loop records its runs into the `cycle_status` table via `cycle.Recorder`.

The System page shows these cycles as rows with last run / next run / duration
and a status ring. Today the user can only wait for the next scheduled pass.
The feature: a **"Run now" button per cycle** that wakes that cycle's loop so
it executes immediately — with visible feedback that the run happened.

## Direction

A single secured core endpoint, `POST /api/cycles/{service}/{cycle}/trigger`,
that routes to the right loop. Core-resident cycles (`availability`,
`metadata_refresh`) are woken in-process via a channel the loop selects on.
Worker-service cycles are woken over HTTP: each service exposes `POST /trigger`
that sends a non-blocking signal into its own loop's select. The loop owns
execution, so a triggered pass is recorded in `cycle_status` exactly like a
scheduled pass — the System page's next poll shows the run truthfully, with
duration and a fresh next-run time. No work ever runs twice simultaneously.

## Routing table

| Service | Cycle | Mechanism |
|---|---|---|
| `core` | `availability` | in-process channel wake of `AvailabilityChecker.Poll` |
| `core` | `metadata_refresh` | in-process channel wake of `PollMetadataRefresh` |
| `indexer` | `monitor_poll`, `process_missing` | HTTP proxy → `INDEXER_HEALTH_ADDR` + `/trigger` |
| `downloader` | `downloader_poll` | HTTP proxy → `DOWNLOADER_HEALTH_ADDR` + `/trigger` |
| `metadata` | `anidb_sync` | HTTP proxy → `METADATA_ADDR` + `/trigger` |

Unknown service or cycle → 404. Upstream service unreachable → 502 with a
message the UI can display.

Notes:

- `process_missing` runs inside the monitor poll (`processMonitors`), so both
  indexer cycles share the same wake: the poll immediately processes pending
  and missing monitors.
- The downloader's existing `POST /trigger` handler currently fire-and-forgets
  `ProcessPending` + `UpdateDownloading` in a goroutine. It is converted to
  the same channel wake — this also upgrades the Downloads page trigger
  (which proxies to it) to recorded and race-free behavior.

## Backend changes

### Core

- `AvailabilityChecker.Poll` and `PollMetadataRefresh` each gain a
  `trigger chan struct{}`; a non-blocking `Trigger()` (send) is exposed and
  each loop's select gets a `case <-trigger:` branch next to its timer.
- `api.NewRouter(...)` accepts the two trigger funcs; `RegisterRoutes`
  registers the new route.
- New handler `TriggerCycle`: a `switch` on `{service, cycle}` that either
  calls the in-process trigger func or POSTs to the mapped service address.
  Status: 204 accepted; 404 unknown; 502 unreachable.

### Indexer (`internal/indexer/service/service.go`, `cmd/indexer/main.go`)

- `IndexerService` gains `trigger chan struct{}` and a non-blocking
  `Trigger()`.
- `PollAndQueue`: the sleep select (after `didWork == false`) adds
  `case <-s.trigger:` — the loop wakes, runs `processMonitors`, records the
  pass as usual.
- `cmd/indexer/main.go`: add `POST /trigger` → `svc.Trigger()`.

### Downloader (`internal/downloader/service/downloader.go`, `cmd/downloader/main.go`)

- Same channel pattern on `PollAndDownload`'s select.
- Convert existing `POST /trigger` handler from goroutine to `svc.Trigger()`.

### Metadata (`internal/metadata/service/runtime.go`, `cmd/metadata/main.go`)

- `StartTitlesSync` select gains `case <-trigger:`; `Trigger()` non-blocking
  send; `cmd/metadata/main.go` adds `POST /trigger`.

## Frontend — System page

- Desktop table: new **Run** column, appended **last** (after Duration) so
  existing test cell indexes (`cells[2]`) stay valid. Each row gets a compact
  `xs` **"Run now"** button.
- Mobile cards: same button in the card header.
- Button states: idle "Run now"; in flight → loader, disabled; success → brief
  check/confirmation; failure → red + error notification with the message from
  the 502 body.
- After a successful trigger, refetch the cycle data immediately (not waiting
  for the 15s poll) so the user sees the run start.
- Disabled while a request for that cycle is pending.

## Tests

- Go: handler test for `TriggerCycle` — unknown cycle 404; unreachable
  service 502; in-process trigger called for core cycles; HTTP POST reaches
  mock servers for indexer/downloader/metadata. Channel tests for each
  service's loop (trigger causes a pass to run).
- Web: SystemPage test — clicking "Run now" POSTs the right URL, shows
  loading, shows confirmation on success. Existing tests updated only if the
  new column shifts assertions.

## Constraints

- Loop owns execution — no fire-and-forget goroutines from the trigger
  endpoint (single-writer semantics per cycle).
- Triggered runs must be recorded by the loop's normal `rec.Start`/`rec.End`
  path, so the System page shows them.
- Keep `POST /api/availability/check` and `POST /api/downloads/trigger`
  working as before (the downloader one now semantically wakes the loop).
- Commit convention: plain sentence-case messages, no prefixes.
- Test suite: 62 web tests passing, `go test ./...` green, `tsc`/`eslint`
  clean before committing.

## Out of scope

- Bulk "Run all now" button.
- Per-cycle scheduling/interval editing from the System page.
- Live (push) status updates — the 15s poll refresh is sufficient.