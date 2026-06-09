package logger

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"time"
)

const bufSize = 200

type LogEntry struct {
	Time    string `json:"time"`
	Level   string `json:"level"`
	Message string `json:"msg"`
	Attrs   string `json:"attrs,omitempty"`
}

type ringBuffer struct {
	mu      sync.RWMutex
	entries []LogEntry
	head    int
	count   int
}

func newRingBuffer() *ringBuffer {
	return &ringBuffer{entries: make([]LogEntry, bufSize)}
}

func (b *ringBuffer) push(e LogEntry) {
	b.mu.Lock()
	b.entries[b.head] = e
	b.head = (b.head + 1) % bufSize
	if b.count < bufSize {
		b.count++
	}
	b.mu.Unlock()
}

func (b *ringBuffer) all() []LogEntry {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.count == 0 {
		return nil
	}
	out := make([]LogEntry, b.count)
	start := (b.head - b.count + bufSize) % bufSize
	for i := range b.count {
		out[i] = b.entries[(start+i)%bufSize]
	}
	return out
}

var global = newRingBuffer()

type bufHandler struct {
	inner slog.Handler
	buf   *ringBuffer
}

func (h *bufHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

func (h *bufHandler) Handle(ctx context.Context, r slog.Record) error {
	attrs := ""
	r.Attrs(func(a slog.Attr) bool {
		attrs += fmt.Sprintf(" %s=%v", a.Key, a.Value)
		return true
	})
	h.buf.push(LogEntry{
		Time:    r.Time.UTC().Format(time.RFC3339),
		Level:   r.Level.String(),
		Message: r.Message,
		Attrs:   attrs,
	})
	return h.inner.Handle(ctx, r)
}

func (h *bufHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &bufHandler{inner: h.inner.WithAttrs(attrs), buf: h.buf}
}

func (h *bufHandler) WithGroup(name string) slog.Handler {
	return &bufHandler{inner: h.inner.WithGroup(name), buf: h.buf}
}

func Init() {
	level := slog.LevelInfo
	if os.Getenv("KBARR_LOG_LEVEL") == "debug" {
		level = slog.LevelDebug
	}
	inner := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	slog.SetDefault(slog.New(&bufHandler{inner: inner, buf: global}))
}

func HandleLogs(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	entries := global.all()
	if entries == nil {
		entries = []LogEntry{}
	}
	_ = json.NewEncoder(w).Encode(entries)
}
