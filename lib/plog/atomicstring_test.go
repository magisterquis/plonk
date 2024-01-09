package plog

/*
 * atomicstring_test.go
 * Atomically-settable LogValuer string
 * By J. Stuart McMurray
 * Created 20230827
 * Last Modified 20231006
 */

import (
	"strings"
	"testing"
)

func TestAtomicString(t *testing.T) {
	as := new(AtomicString)
	if got := as.Load(); "" != got {
		t.Fatalf("Expected empty string after new, got %q", got)
	}
	have := "kittens"
	old, hadOld := as.Swap(have)
	if hadOld {
		t.Fatalf("Swap after new returned true")
	}
	if "" != old {
		t.Fatalf(
			"Expected empty old string frow Swap after new, got %q",
			old,
		)
	}

	if got := as.Load(); have != got {
		t.Fatalf(
			"Load after Swap incorrect: got:%q want:%q",
			got,
			have,
		)
	}

	have = "moose"
	as.Store(have)
	if got := as.Load(); have != got {
		t.Fatalf(
			"Load after Store incorrect: got:%q want:%q",
			got,
			have,
		)
	}
}

func TestAtomicStringLogValue(t *testing.T) {
	as := new(AtomicString)
	_, lb, sl := NewTestLogger()
	sl = sl.With("AS", as)
	sl.Info("m1")
	as.Store("s2")
	sl.Info("s2")
	as.Store("s3")
	sl.Info("s3")

	var gots []string
	for _, v := range strings.Split(lb.String(), "\n") {
		if "" == v {
			continue
		}
		gots = append(gots, RemoveTimestamp(v))
	}

	want := `{"time":"","level":"INFO","msg":"m1","AS":""}
{"time":"","level":"INFO","msg":"s2","AS":"s2"}
{"time":"","level":"INFO","msg":"s3","AS":"s3"}`
	if got := strings.Join(gots, "\n"); got != want {
		t.Fatalf("Incorrect LogValue:\n got:\n%s\nwant:\n%s", got, want)
	}
}
