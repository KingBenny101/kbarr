package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2"
)

func TestTriggerLimiterDeniesWithinInterval(t *testing.T) {
	now := time.Unix(1_000, 0)
	l := &TriggerLimiter{interval: 30 * time.Second, now: func() time.Time { return now }, last: map[string]time.Time{}}

	if ok, _ := l.Check("core/availability"); !ok {
		t.Fatal("first trigger should be allowed")
	}
	l.Record("core/availability")
	if ok, remaining := l.Check("core/availability"); ok {
		t.Fatal("second trigger within the interval should be denied")
	} else if remaining != 30*time.Second {
		t.Fatalf("remaining = %v, want 30s", remaining)
	}

	now = now.Add(10 * time.Second)
	if ok, remaining := l.Check("core/availability"); ok {
		t.Fatal("still within the interval")
	} else if remaining != 20*time.Second {
		t.Fatalf("remaining = %v, want 20s", remaining)
	}

	now = now.Add(10 * time.Second)
	if ok, remaining := l.Check("core/availability"); ok {
		t.Fatal("still within the interval; a denied check must not extend the window")
	} else if remaining != 10*time.Second {
		t.Fatalf("remaining = %v, want 10s", remaining)
	}

	now = now.Add(11 * time.Second)
	if ok, _ := l.Check("core/availability"); !ok {
		t.Fatal("after the interval the trigger should be allowed")
	}
}

func TestTriggerLimiterRemainingCeilsUp(t *testing.T) {
	now := time.Unix(1_000, 0)
	l := &TriggerLimiter{interval: 30 * time.Second, now: func() time.Time { return now }, last: map[string]time.Time{}}

	if ok, _ := l.Check("core/availability"); !ok {
		t.Fatal("first trigger should be allowed")
	}
	l.Record("core/availability")

	now = now.Add(29*time.Second + 900*time.Millisecond)
	ok, remaining := l.Check("core/availability")
	if ok {
		t.Fatal("still within the interval")
	}
	if remaining != 100*time.Millisecond {
		t.Fatalf("remaining = %v, want 100ms", remaining)
	}
	if rounded := ceilDuration(remaining, time.Second); rounded != time.Second {
		t.Fatalf("ceil(remaining) = %v, want 1s so the 429 message never says 0s", rounded)
	}
}

func TestTriggerLimiterTracksKeysSeparately(t *testing.T) {
	now := time.Unix(1_000, 0)
	l := &TriggerLimiter{interval: 30 * time.Second, now: func() time.Time { return now }, last: map[string]time.Time{}}

	if ok, _ := l.Check("core/availability"); !ok {
		t.Fatal("first trigger should be allowed")
	}
	l.Record("core/availability")
	if ok, _ := l.Check("indexer/process_missing"); !ok {
		t.Fatal("a different key should not be limited")
	}
}

func TestTriggerCycleFailedDispatchDoesNotStartCooldown(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	addr := srv.URL
	srv.Close()

	h := TriggerCycle(nil, addr, addr, addr, NewTriggerLimiter(time.Hour))
	for i := 0; i < 2; i++ {
		_, err := h(context.Background(), &TriggerCycleInput{Service: "indexer", Cycle: "monitor_poll"})
		if err == nil {
			t.Fatal("want error for unreachable service")
		}
		if em, ok := err.(*huma.ErrorModel); !ok || em.Status != http.StatusBadGateway {
			t.Fatalf("want 502, got %v", err)
		}
	}
}

func TestTriggerCycleUnknownDoesNotStartCooldown(t *testing.T) {
	h := TriggerCycle(nil, "http://localhost:1", "http://localhost:1", "http://localhost:1", NewTriggerLimiter(time.Hour))
	for i := 0; i < 2; i++ {
		_, err := h(context.Background(), &TriggerCycleInput{Service: "indexer", Cycle: "bogus"})
		if err == nil {
			t.Fatal("want error for unknown cycle")
		}
		if em, ok := err.(*huma.ErrorModel); !ok || em.Status != http.StatusNotFound {
			t.Fatalf("want 404, got %v", err)
		}
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
