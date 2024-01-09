package plog

/*
 * errorloglogger_test.go
 * Tests for errorloglogger.go
 * By J. Stuart McMurray
 * Created 20231018
 * Last Modified 20231117
 */

import "testing"

func TestErrorLogLogger(t *testing.T) {
	msg := "logged error message"
	_, lb, l := NewTestLogger()
	ell := ErrorLogLogger(msg, l)
	have := "kittens"
	ell.Printf("%s", have)
	want := `{"time":"","level":"ERROR","msg":"logged error message",` +
		`"error":"kittens"}`
	got := RemoveTimestamp(lb.String())
	if got != want {
		t.Fatalf(
			"Incorrect log:\n"+
				"have: %q\n"+
				" got: %q\n"+
				"want: %q",
			have,
			got,
			want,
		)
	}
}
