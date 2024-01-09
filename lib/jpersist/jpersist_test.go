package jpersist

/*
 * jpersist_test.go
 * Persist as JSON to disk
 * By J. Stuart McMurray
 * Created 20231007
 * Last Modified 20231007
 */

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

const testFileName = "managed.json"

type testStruct struct {
	S string
	N int
}

func tempFile(t *testing.T) *os.File {
	d := t.TempDir()
	fn := filepath.Join(d, testFileName)
	f, err := os.OpenFile(fn, os.O_RDWR|os.O_CREATE, 0700)
	if nil != err {
		t.Fatalf("Error opening temp file %q: %s", fn, err)
	}
	t.Cleanup(func() { f.Close() })
	return f
}

func TestNewManager_WithEmptyFile(t *testing.T) {
	f := tempFile(t)
	mgr, err := NewManager[testStruct](&Config{File: f.Name()})
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
	f := tempFile(t)
	have := testStruct{
		S: "one",
		N: 1,
	}
	b, _, err := marshal(have)
	if nil != err {
		t.Fatalf("Error JSONing test data: %s", err)
	}
	if _, err := f.Write(b); nil != err {
		t.Fatalf("Error writing test data: %s", err)
	}

	mgr, err := NewManager[testStruct](&Config{File: f.Name()})
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
