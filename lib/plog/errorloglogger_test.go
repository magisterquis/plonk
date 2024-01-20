package plog

/*
 * errorloglogger_test.go
 * Tests for errorloglogger.go
 * By J. Stuart McMurray
 * Created 20231018
 * Last Modified 20231227
 */

import (
	"regexp"
	"testing"
)

func TestErrorLogLogger(t *testing.T) {
	for _, c := range []struct {
		msg  string
		err  string
		want string
		dREs []string
	}{{
		msg:  "m1",
		err:  "e1",
		want: `{"time":"","level":"ERROR","msg":"m1","error":"e1"}`,
	}, {
		msg:  "m1",
		err:  "e1\n",
		want: `{"time":"","level":"ERROR","msg":"m1","error":"e1"}`,
	}, {
		msg:  "m1",
		err:  "e1",
		want: `{"time":"","level":"DEBUG","msg":"m1","error":"e1"}`,
		dREs: []string{`e\d`},
	}, {
		msg:  "m1",
		err:  "e1",
		want: `{"time":"","level":"DEBUG","msg":"m1","error":"e1"}`,
		dREs: []string{`e\d`, `x`},
	}, {
		msg:  "m1",
		err:  "e1\n",
		want: `{"time":"","level":"DEBUG","msg":"m1","error":"e1"}`,
		dREs: []string{`e\d$`, `x`},
	}, {
		msg:  "m1",
		err:  "n1",
		want: `{"time":"","level":"ERROR","msg":"m1","error":"n1"}`,
		dREs: []string{`e\d`, `x`},
	}} {
		c := c /* :C */
		t.Run("", func(t *testing.T) {
			_, lb, l := NewTestLogger()
			dREs := make([]*regexp.Regexp, len(c.dREs))
			for i, v := range c.dREs {
				dREs[i] = regexp.MustCompile(v)
			}
			ErrorLogLogger(c.msg, l, dREs...).Printf("%s", c.err)
			got := RemoveTimestamp(lb.String())
			if got != c.want {
				t.Fatalf(
					"Incorrect log:\n"+
						" msg: %q\n"+
						" err: %q\n"+
						" REs: %s\n"+
						" got: %s\n"+
						"want: %s",
					c.msg,
					c.err,
					dREs,
					got,
					c.want,
				)
			}

		})
	}
}