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
