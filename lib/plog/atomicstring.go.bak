package plog

/*
 * atomicstring.go
 * Atomically-settable LogValuer string
 * By J. Stuart McMurray
 * Created 20230827
 * Last Modified 20231207
 */

import (
	"log/slog"
	"sync/atomic"
)

// AtomicString is like atomic.Value, but for a string.  It implements
// slog.LogValuer.
type AtomicString struct{ v atomic.Value }

// Load gets the stored string.  It returns the empty string if no string has
// been stored.
func (a *AtomicString) Load() string {
	switch v := a.v.Load(); v {
	case nil:
		return ""
	default:
		return v.(string)
	}
}

// Store stores a new value in a.
func (a *AtomicString) Store(s string) { a.v.Store(s) }

// Swap stores a new value in a.  The old value is returned, along with a bool
// bool indicating if there was actually an old value.
func (a *AtomicString) Swap(s string) (old string, hadOld bool) {
	switch v := a.v.Swap(s); v {
	case nil:
		return "", false
	default:
		return v.(string), true
	}
}

// LogValue returns the string in a suitable form for slogging.  This is handy
// for calling slog.Logger.With without needing to worry about someone changing
// the string in a.
func (a *AtomicString) LogValue() slog.Value {
	return slog.StringValue(a.Load())
}
