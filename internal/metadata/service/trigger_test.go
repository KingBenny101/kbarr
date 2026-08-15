package service

import (
	"testing"
	"time"
)

func TestTriggerNonBlocking(t *testing.T) {
	svc := New(nil)

	// First trigger fills the buffered channel.
	svc.Trigger()
	select {
	case <-svc.trigger:
	default:
		t.Fatal("trigger not delivered")
	}

	// Channel is now empty again; a second trigger must not block.
	done := make(chan struct{})
	go func() {
		svc.Trigger()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Trigger blocked on an empty channel")
	}

	// A trigger while the channel is full must be dropped, not block.
	svc.Trigger()
	svc.Trigger()
	select {
	case <-svc.trigger:
	default:
		t.Fatal("expected a pending trigger")
	}
}