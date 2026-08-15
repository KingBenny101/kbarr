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
