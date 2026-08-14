# Search Trigger Button & System Page Clock Fixes

## Overview

Two independent features:
1. **Per-row "Search now" button** in the Monitored table - triggers an immediate indexer search for that monitor
2. **System page timing fixes** - fix misleading "next run says 7s ago" display and add "next missing search" clock

---

## Feature 1: Per-Row "Search Now" Button

### User Story
As a user monitoring episodes, I want to trigger an immediate search for a specific episode/season rather than waiting for the next scheduled indexer poll, so I can get results faster.

### Implementation

#### Backend: New API Endpoint
- **POST `/api/monitor/{id}/search`**
  - Sets the monitor's status to `'pending'` (triggers indexer to pick it up on next poll)
  - Returns 204 on success
  - Only works for `monitored = true` items (already being tracked)

#### Frontend: Monitor Table Changes
- **Desktop**: Add new column "Actions" with "Search now" button per row
- **Mobile**: Add "Search now" button in the monitor card below status pills
- **Button states**:
  - Idle: "Search" (enabled)
  - Loading: "Searching…" with spinner (disabled)
  - Error: "Failed" briefly, then back to "Search"
- **On click**: POST to new endpoint, show loading state, refresh monitor list after 2s

### Rationale
- Setting status to `'pending'` is the simplest trigger - the indexer's `processMonitors` already picks up `status = 'pending'` items
- No need for new indexer logic or inter-service communication
- Minimal backend change, maximal UX benefit

---

## Feature 2: System Page Timing Fixes

### Fix 1: "Next run says 7s ago" → "Due now" / "Overdue"

**Root Cause**: `formatRelative` in `web/src/lib/format.ts` treats any past timestamp as "X ago", including future-scheduled runs that have passed due to clock skew or short intervals.

**Fix**: Update `formatRelative`:
- If `ts` is in the future → "in X" (current behavior)
- If `ts` is in the past but within 30 seconds → "due now"
- If `ts` is in the past beyond 30 seconds → "overdue by X"

### Fix 2: Add "Next Missing Search" Clock

**Background**: The indexer re-searches missing items on a configurable `missingRetryInterval` (default 1440 min = 24h). The System page currently only shows the core availability cycle.

**Implementation**:
1. **Backend**: No API change needed - read the existing `missingRetryInterval` from config (or default 1440 min) and the most recent `last_finished_at` from the indexer's "process missing" cycle
2. **Frontend**: Add a new row/card in the System page table:
   - **Service**: "indexer"
   - **Cycle**: "missing retry"
   - **Display name**: "Missing search retry"
   - **Next run**: calculated as `last_finished_at + missingRetryInterval`
   - Use the same ring progress / time display as other cycles

**Note**: The indexer doesn't currently record its "process missing" cycle to the `cycle_status` table. We'll calculate the next run client-side from the indexer's `last_finished_at` (if available) or show "never" if not yet run.

### Rationale
- Users can see when the next missing-item re-search will happen
- Fixes the confusing "7s ago" display for imminent/overdue runs
- Consistent with existing cycle display patterns

---

## Architecture Summary

| Component | Changes |
|-----------|---------|
| `internal/core/api/handlers/monitor.go` | Add `TriggerMonitorSearch` handler |
| `internal/core/api/routes.go` | Register `POST /api/monitor/{id}/search` |
| `web/src/pages/MonitorPage.tsx` | Add search button column + mobile card button, handler, loading state |
| `web/src/lib/format.ts` | Update `formatRelative` for "due now" / "overdue" |
| `web/src/pages/SystemPage.tsx` | Add "Missing search retry" cycle row, use fixed `formatRelative` |
| `internal/core/api/routes.go` | No new routes for system page (uses existing `/api/cycles`) |

---

## Testing

- **Frontend**: Add tests for search button click, loading states, and `formatRelative` edge cases (past, near-future, far-future)
- **Backend**: Test trigger endpoint sets status to 'pending', handles not-found
- **Integration**: Verify indexer picks up 'pending' items on next poll

---

## Success Criteria

1. Clicking "Search now" on a monitored episode immediately triggers an indexer search (visible as status → "searching" within 1-2 polls)
2. System page never shows "X ago" for future runs; shows "due now" (≤30s past) or "overdue by X" (>30s past)
3. System page shows "Missing search retry" row with correct next run time based on `missingRetryInterval`

---

## Out of Scope

- Bulk "Search all" button (can be added later)
- Real-time WebSocket updates for search status (polling refresh is sufficient)
- Changing indexer cycle recording to `cycle_status` table (future improvement)