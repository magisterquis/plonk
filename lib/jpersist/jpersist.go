// Package jpersist - Persist as JSON to disk
package jpersist

/*
 * jpersist.go
 * Persist as JSON to disk
 * By J. Stuart McMurray
 * Created 20231007
 * Last Modified 20231008
 */

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/magisterquis/plonk/lib/waiter"
)

// DefaultFilePermissions is used when creating files.
const DefaultFilePermissions = os.FileMode(0640)

// Config configures a manager.
type Config struct {
	// File is the name of the file written when
	File string

	// FilePermissions is used when creating a new file.  If unset,
	// DefaultFilePermissions will be used.
	FilePermissions os.FileMode

	// WriteDelay adds a delay between a call to Manager.Unlock and
	// writing to file to allow for other modifications.  This prevents a
	// burst of modifications from causing a burst of disk activity.
	WriteDelay time.Duration

	// OnError is called in its own goroutine when Unlock or UnlockAndWrite
	// return an error, if not nil.  The passed in error will be an
	// UnlockError.
	OnError func(err error)
}

// Manager handles locking and persistence for a variable of type T, which will
// never be nil.
type Manager[T any] struct {
	C *T /* Originally C for Config, but also C for Ctate. */

	l          sync.RWMutex
	conf       Config
	f          *os.File
	lastHash   [sha256.Size]byte
	writeTimer *time.Timer

	writeWaiter waiter.Waiter[struct{}]
}

// NewManager returns a new Manager which wraps a variable of type T, ready for
// use.  The config is optional.  If the config is non-nil and config.File
// names a non-empty file, the file's contents will be unmarshalled into the
// returned Manager's C. The returned Manager will keep a copy of config, if
// not nil.
func NewManager[T any](config *Config) (*Manager[T], error) {
	/* Roll a new manager. */
	if nil == config {
		config = new(Config)
	}
	mgr := &Manager[T]{
		C:    new(T),
		conf: *config,
	}

	/* Set config defaults. */
	if 0 == mgr.conf.FilePermissions {
		mgr.conf.FilePermissions = DefaultFilePermissions
	}

	/* If we're not dealing with a file, no need to do anything else. */
	if "" == mgr.conf.File {
		return mgr, nil
	}

	/* Make sure we can marshal this type. */
	if _, _, err := marshal(mgr.C); nil != err {
		mgr.closeAndNil()
		return nil, CannotMarshalError(err)
	}

	/* Reload is more or less load the first time. */
	if err := mgr.Reload(); nil != err &&
		!("" == mgr.conf.File && errors.Is(err, ErrNoFile)) {
		mgr.closeAndNil()
		return nil, fmt.Errorf("initial load: %w", err)
	}

	/* Re-write the data.  This ensures that the file is writeable, that
	the initial hash is up-to-date, and (re-)indents the data. */
	if err := mgr.Write(); nil != err {
		mgr.closeAndNil()
		return nil, fmt.Errorf("initial write: %w", err)
	}

	return mgr, nil
}
