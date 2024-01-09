package jpersist

/*
 * jpersist_test.go
 * Persist as JSON to disk
 * By J. Stuart McMurray
 * Created 20231007
 * Last Modified 20231010
 */

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"golang.org/x/sys/unix"
)

const testFileName = "managed.json"

type testStruct struct {
	S string
	N int
}

func tempFileName(t *testing.T) string {
	return filepath.Join(t.TempDir(), testFileName)
}

func TestNewManager_NewFile(t *testing.T) {
	fn := filepath.Join(t.TempDir(), testFileName)
	mgr, err := NewManager[testStruct](&Config{File: fn})
	if nil != err {
		t.Fatalf("Error creating Manager: %s", err)
	}
	if nil == mgr.C {
		t.Fatalf("Manager has nil C")
	}
	if !reflect.ValueOf(*mgr.C).IsZero() {
		t.Fatalf("C is not Zero\ngot: %#v", *mgr.C)
	}
}

func TestNewManager_WithEmptyFile(t *testing.T) {
	mgr, err := NewManager[testStruct](&Config{File: tempFileName(t)})
	if nil != err {
		t.Fatalf("Error creating Manager: %s", err)
	}
	if nil == mgr.C {
		t.Fatalf("Manager has nil C")
	}
	if !reflect.ValueOf(*mgr.C).IsZero() {
		t.Fatalf("C is not Zero\ngot: %#v", *mgr.C)
	}
}

func TestNewManager_WithFile(t *testing.T) {
	fn := tempFileName(t)
	have := testStruct{
		S: "one",
		N: 1,
	}
	b, _, err := marshal(have)
	if nil != err {
		t.Fatalf("Error JSONing test data: %s", err)
	}
	if err := os.WriteFile(fn, b, 0600); nil != err {
		t.Fatalf("Error writing test data: %s", err)
	}

	mgr, err := NewManager[testStruct](&Config{File: fn})
	if nil != err {
		t.Fatalf("Error creating Manager: %s", err)
	}

	if nil == mgr.C {
		t.Fatalf("Manager has nil C")
	}
	if *mgr.C != have {
		t.Fatalf("Load failed\n got: %#v\nwant: %#v", mgr.C, have)
	}
}

func TestNewManager_NoFile(t *testing.T) {
	mgr, err := NewManager[testStruct](nil)
	if nil != err {
		t.Fatalf("Error creating manager: %s", err)
	}
	if nil == mgr.C {
		t.Fatalf("Manager has nil C")
	}
	if !reflect.ValueOf(*mgr.C).IsZero() {
		t.Fatalf("C is not Zero\ngot: %#v", *mgr.C)
	}
}

func TestManager_FDLeak(t *testing.T) {
	/* This one is a bit long. */
	if testing.Short() {
		t.Skipf("Short test requested")
	}
	/* Figure out how many files we can open. */
	var lim unix.Rlimit
	if err := unix.Getrlimit(unix.RLIMIT_NOFILE, &lim); nil != err {
		t.Fatalf("Error getting NOFILE rlimit: %s", err)
	}
	nTry := lim.Max + 10
	mgrs := make([]*Manager[int], nTry)
	d := t.TempDir()
	fn := filepath.Join(d, "test.json")
	for i := range mgrs {
		m, err := NewManager[int](&Config{File: fn})
		if nil != err {
			t.Fatalf("Error: %s", err)
		}
		if _, err := os.Stat(fn); nil != err {
			t.Fatalf(
				"Statefile %s stat error: %s",
				fn,
				err,
			)
		}
		mgrs[i] = m
	}
}
