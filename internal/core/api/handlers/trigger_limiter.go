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

// ceilDuration rounds d up to the nearest multiple of unit, so a denial
// never reports a retry time that has already passed.
func ceilDuration(d, unit time.Duration) time.Duration {
	if rem := d % unit; rem != 0 {
		return d - rem + unit
	}
	return d
}

// Check reports whether a trigger for key is permitted, pruning stale keys.
// It does not record the trigger; call Record after the trigger actually
// fires so a failed dispatch does not start a cooldown. When denied it
// returns how long remains before the next trigger is permitted.
func (l *TriggerLimiter) Check(key string) (bool, time.Duration) {
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
	return true, 0
}

// Record marks key as triggered at the current time.
func (l *TriggerLimiter) Record(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.last[key] = l.now()
}
