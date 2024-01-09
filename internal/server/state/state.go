// Package state - Persistent state
package state

/*
 * state.go
 * Persistent state
 * By J. Stuart McMurray
 * Created 20231207
 * Last Modified 20231219
 */

import (
	"log/slog"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/magisterquis/plonk/internal/def"
	"github.com/magisterquis/plonk/lib/jpersist"
)

// State stores the state which persists between executions.  It should not be
// created directly; use New instead.
type State struct {
	/* Tasking holds the per-implant task queues. */
	TaskQ map[string][]string

	/* LastSeen holds the N last-seen implants. */
	LastSeen def.EDSeen
}

// New returns a new State wrapped in a jpersist.Manager which is persisted to
// the directory d.  Write errors will be logged via sl.
func New(dir string, sl *slog.Logger, onError func(error)) (*jpersist.Manager[State], error) {
	mgr, err := jpersist.NewManager[State](&jpersist.Config{
		File:            filepath.Join(dir, def.StateFile),
		FilePermissions: def.FilePerms,
		WriteDelay:      def.StateWriteDelayD,
		OnError:         onError,
	})
	if nil != err {
		return nil, err
	}
	mgr.C.init()
	mgr.Write()
	return mgr, nil
}

// NewTestState returns a new state suitable for testing.  If an error is
// encountered, NewTestState calls t.Fatalf.
func NewTestState(t *testing.T) *jpersist.Manager[State] {
	mgr, err := jpersist.NewManager[State](nil)
	if nil != err {
		t.Fatalf("Error creating state manager: %s", err)
	}
	mgr.C.init()
	return mgr
}

// Saw updates s.LastSeen to note the ImplantID was just seen and from where.
// It is the caller's responsibilty to ensure s's manager is properly locked.
func (s *State) Saw(id, from string) {
	ws := s.LastSeen[:]

	/* Work out where the ID is and add it if we don't have it. */
	idx := -1
	for i, v := range ws {
		if v.ID == id {
			idx = i
			break
		}
	}
	if -1 == idx {
		idx = len(ws) - 1
		ws[idx].ID = id
	}

	/* Update our time. */
	ws[idx].When = time.Now()
	ws[idx].From = from

	/* Sort for next time. */
	slices.SortStableFunc(ws, func(a, b def.ISeen) int {
		if res := b.When.Compare(a.When); 0 != res {
			return res
		} else if res := strings.Compare(b.ID, a.ID); 0 != res {
			return res
		} else {
			return strings.Compare(b.From, a.From)
		}
	})
}

// init makes sure s's maps and slices are non-nil.
func (s *State) init() {
	v := reflect.ValueOf(s).Elem()
	/* Set each nil field to something not nil. */
	for i := 0; i < v.NumField(); i++ {
		f := v.Field(i)

		/* Make something not nil to set if we need it. */
		var n reflect.Value
		switch f.Kind() {
		case reflect.Map:
			n = reflect.MakeMap(f.Type())
		case reflect.Slice:
			n = reflect.MakeSlice(f.Type(), 0, 0)
		default: /* Something we can't make. */
			continue
		}

		/* If we're not nil, don't care. */
		if !f.IsNil() {
			continue
		}

		/* Un-nil ourselves. */
		f.Set(n)
	}
}
