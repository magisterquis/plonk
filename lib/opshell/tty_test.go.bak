package opshell

/*
 * tty_test.go
 * Tests for tty.go
 * By J. Stuart McMurray
 * Created 20231207
 * Last Modified 20231207
 */

import (
	"bytes"
	"io"
	"testing"
)

func TestShellInPTYMode(t *testing.T) {
	for _, c := range []struct {
		mode PTYConfig
		want bool
	}{
		{PTYDefault, false},
		{PTYForce, true},
		{PTYDisable, false},
	} {
		c := c /* :( */
		t.Run(c.mode.String(), func(t *testing.T) {
			t.Parallel()
			pr, pw := io.Pipe()
			defer pw.Close()
			var b bytes.Buffer
			s, err := Config[int]{
				Reader:      pr,
				Writer:      &b,
				ErrorWriter: &b,
				PTYMode:     c.mode,
			}.New()
			if nil != err {
				t.Fatalf("Error creating shell: %s", err)
			}
			if got := s.InPTYMode(); got != c.want {
				t.Fatalf(
					"InPTYMode incorrect:\n"+
						"mode: %s\n"+
						" got: %t\n"+
						"want: %t",
					c.mode,
					got,
					c.want,
				)
			}
		})
	}
}
