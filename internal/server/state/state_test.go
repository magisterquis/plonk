package state

/*
 * state_test.go
 * Tests for state.go
 * By J. Stuart McMurray
 * Created 20231207
 * Last Modified 20231214
 */

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/magisterquis/plonk/internal/def"
	"github.com/magisterquis/plonk/internal/server/perms"
	"github.com/magisterquis/plonk/lib/jpersist"
	"github.com/magisterquis/plonk/lib/plog"
)

//go:embed empty_state
var emptyState string

func init() {
	perms.MustSetProcessPerms()
}

func newTestState(t *testing.T) (
	sm *jpersist.Manager[State],
	dir string,
	lb *bytes.Buffer,
) {
	_, lb, sl := plog.NewTestLogger()
	dir = t.TempDir()
	sm, err := New(dir, sl, func(err error) {
		t.Fatalf("State error: %s", err)
	})
	if nil != err {
		t.Fatalf("Error making state manager: %s", err)
	}
	return sm, dir, lb
}

func TestTestNew_Smoketest(t *testing.T) {
	newTestState(t)
}

func TestNew_Smoketest(t *testing.T) {
	newTestState(t)
}

func TestStateSaw(t *testing.T) {
	sm, _, _ := newTestState(t)
	sm.Lock()
	defer sm.Unlock()

	nid := func(n int) (string, string) {
		return fmt.Sprintf("id%d", n), fmt.Sprintf("from%d", n)
	}

	for i := 0; i < def.NSeen; i++ {
		sm.C.Saw(nid(i))
	}
	sm.C.Saw(nid(5))
	sm.C.Saw(nid(100))

	for i, n := range [def.NSeen]int{100, 5, 9, 8, 7, 6, 4, 3, 2, 1} {
		want, wantFrom := nid(n)
		if got := sm.C.LastSeen[i].ID; got != want {
			t.Fatalf(
				"Seen[%d].ID is %q, expected %q",
				i,
				got,
				want,
			)
		}
		if got := sm.C.LastSeen[i].From; got != wantFrom {
			t.Fatalf(
				"Seen[%d].From is %s, expected %q",
				i,
				got,
				wantFrom,
			)
		}
	}
}

func TestStateSaw_SortOrder(t *testing.T) {
	sm, _, _ := newTestState(t)
	sm.Lock()
	defer sm.Unlock()
	s0 := sm.C.LastSeen[0]
	sm.C.Saw("kittens", "schoedinger's box")
	if got := sm.C.LastSeen[0]; got == s0 {
		t.Errorf(
			"0th element of state not changed after one Saw " +
				"(sort order backwards?)",
		)
	}

}

func TestStateInit(t *testing.T) {
	s := new(State)
	s.init()
	if nil == s.TaskQ {
		t.Errorf("TaskQ nil")
	}

	v := reflect.ValueOf(s).Elem()
	for i := 0; i < v.NumField(); i++ {
		f := v.Field(i)
		switch f.Kind() {
		case reflect.Chan, reflect.Func, reflect.Interface,
			reflect.Map, reflect.Pointer, reflect.Slice:
			if f.IsNil() {
				t.Errorf("Field %d is nil", i)
			}
		}
	}
}

func TestNew(t *testing.T) {
	_, _, sl := plog.NewTestLogger()
	d := t.TempDir()
	sm, err := New(d, sl, func(err error) {
		t.Fatalf("State error: %s", err)
	})
	if nil != err {
		t.Fatalf("New returned error: %s", err)
	}
	if nil == sm.C.TaskQ {
		t.Fatalf("TaskQ is nil")
	}

	b, err := os.ReadFile(filepath.Join(d, def.StateFile))
	if nil != err {
		t.Fatalf("Error reading state: %s", err)
	}

	if got := string(b); got != strings.TrimSuffix(emptyState, "\n") {
		t.Fatalf(
			"State file contents incorrect:\n"+
				" got:\n%q\n"+
				"want:\n%q",
			got,
			emptyState,
		)
	}
}

func TestStateUnlock(t *testing.T) {
	sm, dir, _ := newTestState(t)
	haveID := "kittens"
	haveTask := "moose"
	sm.Lock()
	q := sm.C.TaskQ[haveID]
	q = append(q, haveTask)
	sm.C.TaskQ[haveID] = q
	if err := sm.UnlockAndWrite(); nil != err {
		t.Fatalf("Unlock error: %s", err)
	}

	var got struct {
		TaskQ map[string][]string
	}
	want := got
	want.TaskQ = map[string][]string{haveID: []string{haveTask}}

	fn := filepath.Join(dir, def.StateFile)
	f, err := os.Open(fn)
	if nil != err {
		t.Fatalf("Error opening state file %s: %s", fn, err)
	}
	defer f.Close()
	if err := json.NewDecoder(f).Decode(&got); nil != err {
		t.Fatalf("Error reading state file %s: %s", f.Name(), err)
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf(
			"State incorrect after saving:\n"+
				" got: %#v\n"+
				"want: %#v",
			got,
			want,
		)
	}
}

func TestNew_Perms(t *testing.T) {
	_, dir, _ := newTestState(t)
	fi, err := os.Stat(filepath.Join(dir, def.StateFile))
	if nil != err {
		t.Fatalf("Stat failed: %s", err)
	}
	if got := fi.Mode().Perm(); def.FilePerms != got {
		t.Errorf("Incorrect perms: got:%o want:%o", got, def.FilePerms)
	}
}
