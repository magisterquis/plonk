package jpersist

/*
 * file_test.go
 * Tests for file.go
 * By J. Stuart McMurray
 * Created 20231007
 * Last Modified 20231007
 */

import (
	"encoding/hex"
	"os"
	"testing"
)

func TestManagerReload(t *testing.T) {
	f := tempFile(t)
	mgr, err := NewManager[testStruct](&Config{File: f.Name()})
	if nil != err {
		t.Fatalf("Error creating Manager: %s", err)
	}
	mgr.RLock()
	if 0 != mgr.C.N {
		t.Fatalf("Wrong mgr.C.N after creation: got:%d", mgr.C.N)
	}
	mgr.RUnlock()

	have := 10
	b, _, err := marshal(testStruct{N: have})
	if nil != err {
		t.Fatalf("Marshal error: %s", err)
	}
	if _, err := f.Write(b); nil != err {
		t.Fatalf("Write error: %s", err)
	}

	if err := mgr.Reload(); nil != err {
		t.Fatalf("Reload error: %s", err)
	}

	mgr.RLock()
	if have != mgr.C.N {
		t.Fatalf(
			"Wrong mgr.C.N after reload:\n got: %d\nwant: %d",
			mgr.C.N,
			have,
		)
	}
	mgr.RUnlock()
}

func TestManagerWrite(t *testing.T) {
	haveS := "kittens"
	haveN := 10
	want := `{
	"S": "kittens",
	"N": 10
}` /* A bit fragile. */

	f := tempFile(t)
	mgr, err := NewManager[testStruct](&Config{File: f.Name()})
	if nil != err {
		t.Fatalf("Error creating Manager: %s", err)
	}

	mgr.C.S = haveS
	mgr.C.N = haveN
	if err := mgr.Write(); nil != err {
		t.Fatalf("Write error: %s", err)
	}

	got, err := os.ReadFile(f.Name())
	if nil != err {
		t.Fatalf("Error reading after Write: %s", err)
	}
	if string(got) != want {
		t.Fatalf(
			"Write wrote unexpected data:\n"+
				"haveS: %q\n"+
				"haveN: %d\n"+
				"  got: %s\n"+
				" want: %s",
			haveS,
			haveN,
			got,
			want,
		)
	}
}

func TestMarshal(t *testing.T) {
	have := struct {
		Foo    string
		Bar    bool
		Tridge int `json:"moose"`
	}{
		Foo:    "kittens",
		Bar:    true,
		Tridge: 5,
	}
	want := `{
	"Foo": "kittens",
	"Bar": true,
	"moose": 5
}`
	wantHash := "ab25422977eba1a0d5f8781c2aebe187" +
		"6ec2a2385a5c2acca6c79f880c3d12bc"

	got, gotHash, err := marshal(have)
	if nil != err {
		t.Fatalf("Error: %s", err)
	}
	if string(got) != want {
		t.Fatalf(
			"Incorrect JSON\n"+
				"have: %#v\n"+
				" got: %q\n"+
				"want: %q",
			have,
			got,
			want,
		)
	}
	gotHex := hex.EncodeToString(gotHash[:])
	if wantHash != gotHex {
		t.Fatalf(
			"Incorrect hash\n"+
				"have: %q\n"+
				" got: %s\n"+
				"want: %s",
			got,
			gotHex,
			wantHash,
		)
	}
}
