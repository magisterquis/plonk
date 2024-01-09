package plog

/*
 * withhandler_test.go
 * Tests for withhandler.go
 * By J. Stuart McMurray
 * Created 20230827
 * Last Modified 20231006
 */

import (
	"bytes"
	"log/slog"
	"testing"
)

func TestWithHandler(t *testing.T) {
	var buf bytes.Buffer
	var lv slog.LevelVar
	l := slog.New(NewHandler(&lv, slog.NewTextHandler(
		&buf,
		&slog.HandlerOptions{
			ReplaceAttr: func(gs []string, a slog.Attr) slog.Attr {
				if 0 == len(gs) && slog.TimeKey == a.Key {
					return slog.Attr{}
				}
				return a
			},
		})))
	for _, c := range []struct {
		do   func()
		want string
	}{{
		do:   func() { l.Info("foo") },
		want: "level=INFO msg=foo\n",
	}, {
		do: func() {
			defer lv.Set(lv.Level())
			l.Debug("m1")
			lv.Set(slog.LevelDebug)
			l.Debug("m2")
		},
		want: "level=DEBUG msg=m2\n",
	}, {
		do: func() {
			l.Debug("foo")
		},
		want: "",
	}, {
		do: func() {
			var as AtomicString
			as.Store("v1")
			wh := l.With("k1", &as)
			wh.Info("m1")
			as.Store("v2")
			wh.Info("m2")
		},
		want: "level=INFO msg=m1 k1=v1\n" +
			"level=INFO msg=m2 k1=v2\n",
	}, {
		do: func() {
			l.
				With("k1", "v1").
				WithGroup("g1").
				With("k2", "v2").
				Info("m1", "k3", "v3")
		},
		want: "level=INFO msg=m1 g1.k3=v3 g1.k2=v2 k1=v1\n",
	}, {
		do: func() {
			var as AtomicString
			as.Store("v2")
			wh := l.
				With("k1", "v1").
				WithGroup("g1").
				With("k2", &as)
			wh.Info("m1", "k3", "v3")
			as.Store("v2a")
			wh.Info("m2", "k4", "v4")
		},
		want: "level=INFO msg=m1 g1.k3=v3 g1.k2=v2 k1=v1\n" +
			"level=INFO msg=m2 g1.k4=v4 g1.k2=v2a k1=v1\n",
	}} {
		c := c /* :( */
		t.Run("", func(t *testing.T) {
			buf.Reset()
			c.do()
			got := buf.String()
			if c.want != got {
				t.Fatalf(
					"Incorrect log output\n"+
						"want: %q\n"+
						" got: %q\n",
					c.want,
					got,
				)
			}
		})
	}
}
