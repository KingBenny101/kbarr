# Run Now Spam Prevention Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Prevent spamming the "Run now" button on the System page by disabling it while a cycle is running and for 30s after each trigger, with the same 30s minimum interval enforced server-side (429).

**Architecture:** A mutex-guarded `TriggerLimiter` in the core API rejects triggers within the interval with 429; the client tracks per-cycle last-trigger timestamps and disables the button (label switches to "Wait {n}s") while running or cooling down.

**Tech Stack:** Go (huma v2, slog), React + Mantine (SystemPage), Vitest + testing-library, `go test`.

**Spec:** `docs/superpowers/specs/2026-08-15-run-now-spam-prevention-design.md`

## Global Constraints

- Commit messages: plain sentence-case, no prefixes (repo convention).
- All code must pass: `go test ./...` (repo root), `npx vitest run`, `npx tsc --noEmit`, `npm run lint` in `web/` (lint has 1 pre-existing warning in `web/src/pages/LibraryPage.tsx` — leave untouched).
- 30-second cooldown constant on both sides: server `30 * time.Second`, client `30_000` ms.
- Do NOT reformat files unrelated to this change (repo has pre-existing gofmt drift).
- Client tests must NOT use fake timers: the suite mixes `usePolling` intervals with testing-library `waitFor`/`findBy`, which is flaky under faked timers. Cooldown-expiry behavior is covered by the Go limiter tests with an injected clock; client tests assert deterministic immediate state only.

---

### Task 1: Server-side trigger limiter (Go, TDD)

**Files:**
- Create: `internal/core/api/handlers/trigger_limiter.go`
- Create: `internal/core/api/handlers/trigger_limiter_test.go`
- Modify: `internal/core/api/handlers/cycle_trigger.go`
- Modify: `internal/core/api/handlers/cycle_trigger_test.go`
- Modify: `internal/core/api/routes.go`

**Interfaces:**
- Consumes: nothing.
- Produces: `NewTriggerLimiter(interval time.Duration) *TriggerLimiter` with method `Allow(key string) (bool, time.Duration)`; `TriggerCycle` gains a trailing `limiter *TriggerLimiter` parameter; `routes.go` wires `handlers.NewTriggerLimiter(30 * time.Second)`.

- [ ] **Step 1: Write the failing limiter tests**

Create `internal/core/api/handlers/trigger_limiter_test.go`:

```go
package handlers

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2"
)

func TestTriggerLimiterDeniesWithinInterval(t *testing.T) {
	now := time.Unix(1_000, 0)
	l := &TriggerLimiter{interval: 30 * time.Second, now: func() time.Time { return now }, last: map[string]time.Time{}}

	if ok, _ := l.Allow("core/availability"); !ok {
		t.Fatal("first trigger should be allowed")
	}
	if ok, remaining := l.Allow("core/availability"); ok {
		t.Fatal("second trigger within the interval should be denied")
	} else if remaining != 30*time.Second {
		t.Fatalf("remaining = %v, want 30s", remaining)
	}

	now = now.Add(10 * time.Second)
	if ok, remaining := l.Allow("core/availability"); ok {
		t.Fatal("still within the interval")
	} else if remaining != 20*time.Second {
		t.Fatalf("remaining = %v, want 20s", remaining)
	}

	now = now.Add(21 * time.Second)
	if ok, _ := l.Allow("core/availability"); !ok {
		t.Fatal("after the interval the trigger should be allowed")
	}
}

func TestTriggerLimiterTracksKeysSeparately(t *testing.T) {
	now := time.Unix(1_000, 0)
	l := &TriggerLimiter{interval: 30 * time.Second, now: func() time.Time { return now }, last: map[string]time.Time{}}

	if ok, _ := l.Allow("core/availability"); !ok {
		t.Fatal("first trigger should be allowed")
	}
	if ok, _ := l.Allow("indexer/process_missing"); !ok {
		t.Fatal("a different key should not be limited")
	}
}

func TestTriggerCycle429(t *testing.T) {
	called := 0
	h := TriggerCycle(map[string]func(){"availability": func() { called++ }},
		"http://localhost:1", "http://localhost:1", "http://localhost:1",
		NewTriggerLimiter(time.Hour))

	if _, err := h(context.Background(), &TriggerCycleInput{Service: "core", Cycle: "availability"}); err != nil {
		t.Fatal(err)
	}
	_, err := h(context.Background(), &TriggerCycleInput{Service: "core", Cycle: "availability"})
	if err == nil {
		t.Fatal("want 429 for immediate re-trigger")
	}
	if em, ok := err.(*huma.ErrorModel); !ok || em.Status != http.StatusTooManyRequests {
		t.Fatalf("want 429, got %v", err)
	}
	if called != 1 {
		t.Fatalf("core trigger ran %d times, want 1", called)
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/core/api/handlers/ -run 'TestTriggerLimiter|TestTriggerCycle429'`
Expected: FAIL — `TriggerLimiter` undefined.

- [ ] **Step 3: Implement the limiter**

Create `internal/core/api/handlers/trigger_limiter.go`:

```go
package handlers

import (
	"sync"
	"time"
)

// TriggerLimiter bounds how often a cycle can be triggered, keyed by
// "<service>/<cycle>". Denied triggers carry the remaining cooldown.
type TriggerLimiter struct {
	mu       sync.Mutex
	interval time.Duration
	now      func() time.Time
	last     map[string]time.Time
}

// NewTriggerLimiter returns a limiter allowing one trigger per key per interval.
func NewTriggerLimiter(interval time.Duration) *TriggerLimiter {
	return &TriggerLimiter{interval: interval, now: time.Now, last: make(map[string]time.Time)}
}

// Allow records a trigger for key when the interval since the previous
// trigger has elapsed. It returns whether the trigger is allowed and, when
// denied, how long remains before the next one is permitted.
func (l *TriggerLimiter) Allow(key string) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	for k, t := range l.last {
		if now.Sub(t) >= l.interval {
			delete(l.last, k)
		}
	}
	if prev, ok := l.last[key]; ok {
		remaining := l.interval - now.Sub(prev)
		if remaining > 0 {
			return false, remaining
		}
	}
	l.last[key] = now
	return true, 0
}
```

- [ ] **Step 4: Wire the limiter into the trigger handler**

In `internal/core/api/handlers/cycle_trigger.go`, change the signature and add the 429 rejection. Replace the first lines of the returned closure:

```go
func TriggerCycle(coreTriggers map[string]func(), indexerAddr, downloaderAddr, metadataAddr string, limiter *TriggerLimiter) func(context.Context, *TriggerCycleInput) (*struct{}, error) {
	return func(ctx context.Context, in *TriggerCycleInput) (*struct{}, error) {
		key := in.Service + "/" + in.Cycle
		if ok, remaining := limiter.Allow(key); !ok {
			slog.Info("Cycle trigger rejected", "service", in.Service, "cycle", in.Cycle, "retryIn", remaining.Round(time.Second))
			return nil, huma.Error429TooManyRequests(fmt.Sprintf("cycle already triggered recently, try again in %s", remaining.Round(time.Second)))
		}
		slog.Info("Cycle triggered", "service", in.Service, "cycle", in.Cycle)
		switch in.Service {
```

(`time` is already imported in this file.)

- [ ] **Step 5: Update the existing handler tests for the new signature**

In `internal/core/api/handlers/cycle_trigger_test.go`, every `TriggerCycle(...)` call gains a trailing `NewTriggerLimiter(time.Hour)` argument (a long interval so existing tests never trip it):
- line 14-15: `h := TriggerCycle(map[string]func(){"availability": func() { called = true }}, "http://localhost:1", "http://localhost:1", "http://localhost:1", NewTriggerLimiter(time.Hour))`
- line 27: `h := TriggerCycle(nil, "http://localhost:1", "http://localhost:1", "http://localhost:1", NewTriggerLimiter(time.Hour))`
- line 56: `h := TriggerCycle(nil, srv.URL, srv.URL, srv.URL, NewTriggerLimiter(time.Hour))`
- line 76: `h := TriggerCycle(nil, addr, addr, addr, NewTriggerLimiter(time.Hour))`

Add `"time"` to the imports.

- [ ] **Step 6: Wire the limiter in routes.go**

In `internal/core/api/routes.go`, add `"time"` to the imports and append the limiter argument inside `handlers.TriggerCycle(...)`:

```go
	handlers.TriggerCycle(
		coreTriggers,
		handlers.SvcAddr("INDEXER_HEALTH_ADDR", "http://localhost:8082"),
		handlers.SvcAddr("DOWNLOADER_HEALTH_ADDR", "http://localhost:8083"),
		handlers.SvcAddr("METADATA_ADDR", "http://localhost:8081"),
		handlers.NewTriggerLimiter(30*time.Second),
	))
```

- [ ] **Step 7: Run all Go tests**

Run: `go test ./...` (repo root)
Expected: all pass, including the 3 new tests and the 4 updated ones.

- [ ] **Step 8: Commit**

```bash
git add internal/core/api/handlers/trigger_limiter.go internal/core/api/handlers/trigger_limiter_test.go internal/core/api/handlers/cycle_trigger.go internal/core/api/handlers/cycle_trigger_test.go internal/core/api/routes.go
git commit -m "Reject trigger spam with a 30 second per-cycle limiter"
```

---

### Task 2: Client-side cooldown on the Run now button (web, TDD)

**Files:**
- Modify: `web/src/pages/SystemPage.tsx`
- Modify: `web/src/pages/SystemPage.test.tsx`

**Interfaces:**
- Consumes: `TriggerCycle` 429 behavior from Task 1 (client only needs `res.status === 429`).
- Produces: `TRIGGER_COOLDOWN_MS = 30_000` constant; `cooldownFor: (c: CycleStatus) => number` prop on `CycleTable` and `CycleCards`; per-cycle `lastTriggered` state in `SystemPage`.

- [ ] **Step 1: Write the failing component tests**

Append to `web/src/pages/SystemPage.test.tsx` (inside the `describe("SystemPage", ...)` block), adding `fireEvent` if not already imported (it is):

```tsx
    it("disables run now and shows the cooldown after triggering", async () => {
        renderPage()

        await screen.findByText("Availability check")

        const runNow = screen.getAllByRole("button", { name: "Run now" })
        expect(runNow[0]).toBeEnabled()

        fireEvent.click(runNow[0])

        const waiting = await screen.findByRole("button", { name: /wait \d+s/i })
        expect(waiting).toBeDisabled()
    })

    it("disables run now while the cycle is running", async () => {
        renderPage()

        await screen.findByText("Availability check")

        const runNow = screen.getAllByRole("button", { name: "Run now" })
        // CYCLES[1] (Metadata refresh) has state "running" in the fixture.
        expect(runNow[1]).toBeDisabled()
        expect(runNow[0]).toBeEnabled()
    })
```

Note: `showToast` uses `notifications.show`; the success toast after a trigger renders into Mantine's notifications store — if the click throws "Notifications provider is missing" in jsdom, wrap `renderPage`'s JSX with `<Notifications />` (import from `@mantine/notifications`) and keep the tests otherwise unchanged.

- [ ] **Step 2: Run the tests to verify they fail**

Run: `npx vitest run src/pages/SystemPage.test.tsx`
Expected: the two new tests FAIL (button never shows "Wait ..." / running button not disabled).

- [ ] **Step 3: Implement the cooldown**

In `web/src/pages/SystemPage.tsx`:

1. Add the constant next to `MISSING_RETRY_MIN` (line ~25):

```tsx
const TRIGGER_COOLDOWN_MS = 30_000
```

2. Add the `cooldownFor` prop to both `CycleTable` (line ~100) and `CycleCards` (line ~172) props and threads:

```tsx
function CycleTable({ cycles, offlineServices, now, runningKey, cooldownFor, onRun }: {
    cycles: CycleStatus[]
    offlineServices: Set<string>
    now: Date
    runningKey: string | null
    cooldownFor: (c: CycleStatus) => number
    onRun: (c: CycleStatus) => void
}) {
```

(identical signature change for `CycleCards`).

3. In `CycleTable`, replace the button cell (lines 147-156):

```tsx
                            <Table.Td ta="center">
                                <Button
                                    size="xs"
                                    variant="light"
                                    loading={runningKey === `${c.service}/${c.cycle}`}
                                    disabled={v.running || cooldownFor(c) > 0}
                                    onClick={() => onRun(c)}
                                >
                                    {cooldownFor(c) > 0 ? `Wait ${Math.ceil(cooldownFor(c) / 1000)}s` : "Run now"}
                                </Button>
                            </Table.Td>
```

4. In `CycleCards`, replace the button (lines 201-208):

```tsx
                                <Button
                                    size="compact-xs"
                                    variant="light"
                                    loading={runningKey === `${c.service}/${c.cycle}`}
                                    disabled={v.running || cooldownFor(c) > 0}
                                    onClick={() => onRun(c)}
                                >
                                    {cooldownFor(c) > 0 ? `Wait ${Math.ceil(cooldownFor(c) / 1000)}s` : "Run now"}
                                </Button>
```

5. In the `SystemPage` component: add state next to `runningKey` (line ~245):

```tsx
    const [lastTriggered, setLastTriggered] = useState<Record<string, number>>({})
```

6. Add the `cooldownFor` helper after `serverNow` (line ~259):

```tsx
    const cooldownFor = (c: CycleStatus) => {
        const t = lastTriggered[`${c.service}/${c.cycle}`]
        if (!t) return 0
        return Math.max(0, TRIGGER_COOLDOWN_MS - (serverNow.getTime() - t))
    }
```

7. Update `runCycle` (lines 290-311) to record the trigger time on success and on 429:

```tsx
    const runCycle = async (c: CycleStatus) => {
        const key = `${c.service}/${c.cycle}`
        setRunningKey(key)
        try {
            const res = await apiFetch(`${API_URL}/api/cycles/${c.service}/${c.cycle}/trigger`, { method: "POST" })
            if (!res.ok) {
                let msg = `Failed to trigger ${c.display_name}`
                try {
                    const body = await res.json()
                    if (body?.error?.message) msg = body.error.message
                } catch {
                    /* keep the fallback message */
                }
                showToast(msg, "error")
                if (res.status === 429) {
                    setLastTriggered((prev) => ({ ...prev, [key]: Date.now() }))
                }
                return
            }
            showToast(`${c.display_name} triggered`, "success")
            setLastTriggered((prev) => ({ ...prev, [key]: Date.now() }))
            void fetchCycles()
        } finally {
            setRunningKey(null)
        }
    }
```

8. Pass the new prop at both render sites (lines ~343 and ~345):

```tsx
                        <CycleTable cycles={allCycles} offlineServices={offlineServices} now={serverNow} runningKey={runningKey} cooldownFor={cooldownFor} onRun={runCycle} />
                        <CycleCards cycles={allCycles} offlineServices={offlineServices} now={serverNow} runningKey={runningKey} cooldownFor={cooldownFor} onRun={runCycle} />
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `npx vitest run src/pages/SystemPage.test.tsx`
Expected: 15 tests pass (13 existing + 2 new).

- [ ] **Step 5: Verify the full frontend checks**

Run: `npx vitest run && npx tsc --noEmit && npm run lint` (in `web/`)
Expected: 66+ tests pass, tsc clean, 0 lint errors (1 pre-existing warning).

- [ ] **Step 6: Manual smoke test**

With overmind running, open http://localhost:5173/system in a browser (login `KingBenny101` / `123456789`):
1. Click "Run now" on any idle cycle — button becomes disabled with "Wait 30s", then counts down and re-enables.
2. Click it again immediately after re-enable — the toast shows the server's 429 message, and the button still starts the cooldown.
3. While a cycle is running (e.g. right after triggering downloader poll), its button stays disabled.

- [ ] **Step 7: Commit**

```bash
git add web/src/pages/SystemPage.tsx web/src/pages/SystemPage.test.tsx
git commit -m "Disable the run now button during the trigger cooldown"
```

---

## Self-Review Notes

- Spec coverage: server 429 (Task 1), client disabled while running + 30s cooldown with "Wait {n}s" label (Task 2), 429 starts client cooldown (Task 2 step 3.7), tests both layers. ✓
- Placeholder scan: no TBD/TODO; all steps carry exact code/commands. ✓
- Type consistency: `NewTriggerLimiter(interval) *TriggerLimiter`, `Allow(key) (bool, time.Duration)`, `TriggerCycle(..., limiter *TriggerLimiter)`, `cooldownFor: (c: CycleStatus) => number` used consistently across tasks. ✓
- Deviation from spec (test mechanism): spec mentioned fake timers for client cooldown expiry; plan uses deterministic immediate-state assertions instead because fake timers are flaky with the suite's `usePolling` + `findBy` pattern. Server-side expiry is covered with an injected clock. The countdown math (`Math.max(0, 30000 - (now - t))`, `Math.ceil / 1000`) is trivially verifiable in review.