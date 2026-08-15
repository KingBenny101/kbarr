# Run Now Spam Prevention

Date: 2026-08-15

## Problem

The "Run now" button on the System page can be spammed: the `loading` state only lasts while the trigger HTTP request is in flight, after which the button is immediately clickable again. The server accepts unlimited triggers (`internal/core/api/handlers/cycle_trigger.go`), so a user (or a direct API caller) can fire the expensive cycle loops back-to-back (e.g., `process_missing` re-searches every missing monitor on each trigger).

## Goal

A user cannot spam the trigger: the button is disabled while the cycle is running and for 30 seconds after each trigger, and the API enforces the same 30-second minimum interval server-side so direct calls can't bypass the UI.

## Design

### Server: per-cycle trigger limiter

- New `triggerLimiter` in `internal/core/api/handlers/cycle_trigger.go` (or its own small file):
  - Mutex-guarded map keyed by `"<service>/<cycle>"` → last trigger time.
  - `Allow(key) bool` records the timestamp and returns true; returns false if the last trigger was within the configured interval.
  - Stale entries (older than the interval) are pruned on access so the map stays small.
  - Interval is a constructor parameter (`newTriggerLimiter(interval)`), so tests can use a short window.
- `TriggerCycle` handler gains a limiter argument (default `30 * time.Second` wired in `internal/core/api/routes.go`).
- A trigger rejected by the limiter returns `429 Too Many Requests` with a message: "already triggered recently, try again in Xs".
- No changes to what a trigger does; only frequency is bounded.

### Client: disable the button

In `web/src/pages/SystemPage.tsx`:

- Track per-cycle last-trigger timestamps in state: `Record<string, number>` (key `"<service>/<cycle>"`).
- In both `CycleTable` and `CycleCards`, the Run now button is disabled when:
  - the cycle is running (`v.running`), or
  - a request is in flight (existing `loading` state), or
  - the 30s cooldown since the last trigger has not elapsed.
- While cooling down, the button label shows `Wait {n}s` (counts down with the existing 1s clock tick) so the disabled state is self-explanatory.
- On success, record the trigger time; on failure, do not start a cooldown so a transient failure doesn't lock the button — with one exception: a **429** response means a trigger already happened recently, so the client starts the 30s cooldown from "now" on 429 as well.

### Error handling

- 429 responses already surface through the existing error-toast path (`body.error.message`); no extra client code needed beyond not recording a cooldown on failure.

### Testing

- Go: handler test — trigger, immediate re-trigger → 429; after the (short, test-configured) window → 200.
- Web: SystemPage test — button disabled with `Wait {n}s` after a successful trigger; disabled while the cycle is running; re-enabled after the cooldown elapses (fake timers).

## Out of scope

- Per-user rate limits, auth changes, queueing of triggers, or changes to other pages.