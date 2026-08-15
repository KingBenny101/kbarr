package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2"
)

func TestTriggerCycleCore(t *testing.T) {
	called := false
	h := TriggerCycle(map[string]func(){"availability": func() { called = true }},
		"http://localhost:1", "http://localhost:1", "http://localhost:1", NewTriggerLimiter(time.Hour))

	_, err := h(context.Background(), &TriggerCycleInput{Service: "core", Cycle: "availability"})
	if err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("core trigger func was not called")
	}
}

func TestTriggerCycleUnknown(t *testing.T) {
	h := TriggerCycle(nil, "http://localhost:1", "http://localhost:1", "http://localhost:1", NewTriggerLimiter(time.Hour))

	cases := []TriggerCycleInput{
		{Service: "core", Cycle: "bogus"},
		{Service: "bogus", Cycle: "availability"},
		{Service: "indexer", Cycle: "bogus"},
	}
	for _, in := range cases {
		_, err := h(context.Background(), &in)
		if err == nil {
			t.Fatalf("want error for %+v", in)
		}
		if em, ok := err.(*huma.ErrorModel); !ok || em.Status != http.StatusNotFound {
			t.Fatalf("want 404 for %+v, got %v", in, err)
		}
	}
}

func TestTriggerCycleProxy(t *testing.T) {
	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if r.Method != http.MethodPost || r.URL.Path != "/trigger" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	h := TriggerCycle(nil, srv.URL, srv.URL, srv.URL, NewTriggerLimiter(time.Hour))
	for _, in := range []TriggerCycleInput{
		{Service: "indexer", Cycle: "monitor_poll"},
		{Service: "downloader", Cycle: "downloader_poll"},
		{Service: "metadata", Cycle: "anidb_sync"},
	} {
		if _, err := h(context.Background(), &in); err != nil {
			t.Fatalf("proxy trigger %+v failed: %v", in, err)
		}
	}
	if hits != 3 {
		t.Fatalf("hits = %d, want 3", hits)
	}
}

func TestTriggerCycleUnreachable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	addr := srv.URL
	srv.Close()

	h := TriggerCycle(nil, addr, addr, addr, NewTriggerLimiter(time.Hour))
	_, err := h(context.Background(), &TriggerCycleInput{Service: "indexer", Cycle: "monitor_poll"})
	if err == nil {
		t.Fatal("want error for unreachable service")
	}
	if em, ok := err.(*huma.ErrorModel); !ok || em.Status != http.StatusBadGateway {
		t.Fatalf("want 502, got %v", err)
	}
}
