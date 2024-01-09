package implantsvr

/*
 * errors_test.go
 * Error types and handlers
 * By J. Stuart McMurray
 * Created 20231207
 * Last Modified 20231207
 */

import (
	"bufio"
	"io"
	"testing"

	"github.com/magisterquis/plonk/lib/plog"
)

func TestHTTPErrorLogger(t *testing.T) {
	pr, pw := io.Pipe()
	_, sl := plog.NewJSONLogger(pw)
	hel := httpErrorLogger(sl)

	ms := []string{
		"m1",
		"m2",
		"m3 m4",
		"",
	}

	go func() {
		for _, m := range ms {
			hel.Print(m)
		}
	}()

	scanner := bufio.NewScanner(pr)
	for i, have := range ms {
		if !scanner.Scan() {
			break
		}
		got := plog.RemoveTimestamp(scanner.Text())
		want := `{"time":"","level":"ERROR","msg":"HTTP error",` +
			`"error":"` + have + `"}`
		if got != want {
			t.Errorf(
				"Log line %d/%d incorrect:\n"+
					"have: %s\n"+
					" got: %s\n"+
					"want: %s",
				i+1, len(ms),
				have,
				got,
				want,
			)
		}

	}
	if err := scanner.Err(); nil != err {
		t.Fatalf("Error reading from JSON slogger: %s", err)
	}
}
