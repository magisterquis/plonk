package jpersist

/*
 * errors.go
 * Error types
 * By J. Stuart McMurray
 * Created 20231007
 * Last Modified 20231007
 */

import (
	"errors"
	"runtime"
)

// ErrNoFile indicates a read or write was attempted without a file configured.
var ErrNoFile = errors.New("no file configured")

// CannotMarshalError is a decorator returned by NewManager to indicate that
// the Managered type cannot be marshalled to JSON.
type CannotMarshalError error

// UnlockError is returned from Unlock and UnlockAndWrite and passed to
// Config.OnError to indicate something went wrong while unlocking.
type UnlockError struct {
	// CallerFile and CallerLine point to the caller of the function which
	// would have returned this error.  If the file and line weren't
	// available, they will be their respective zero values.
	CallerFile string
	CallerLine int

	// Err is this error's underlying error.
	Err error
}

// NewUnlockError returns a new UnlockError wrapping err.  Skip is the number
// of stack frames to skip to find the "real" caller.
func newUnlockError(err error, skip int) UnlockError {
	ue := UnlockError{Err: err}
	if _, f, l, ok := runtime.Caller(skip + 1); ok {
		ue.CallerFile = f
		ue.CallerLine = l
	}
	return ue
}

// Error satisties the error interface.
func (err UnlockError) Error() string { return err.Err.Error() }

// Unwrap return err.Err.
func (err UnlockError) Unwrap() error { return err.Err }
