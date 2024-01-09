package plog

/*
 * plog_test.go
 * Tests for plog.go
 * By J. Stuart McMurray
 * Created 20231010
 * Last Modified 20231010
 */

import (
	"fmt"
	"log/slog"
	"regexp"
	"testing"
)

func TestNewTestLogger(t *testing.T) {
	untimeRE := regexp.MustCompile(`{"time":"[^"]+",`)
	for _, c := range []struct {
		have string
		f    func(have string, lv *slog.LevelVar, l *slog.Logger)
		want string
	}{{
		have: "debug/debug",
		f: func(have string, lv *slog.LevelVar, l *slog.Logger) {
			l.Debug(have)
		},
		want: `"level":"DEBUG","msg":"debug/debug"}` + "\n",
	}, {
		have: "debug/info",
		f: func(have string, lv *slog.LevelVar, l *slog.Logger) {
			l.Info(have)
		},
		want: `"level":"INFO","msg":"debug/info"}` + "\n",
	}, {
		have: "info/debug",
		f: func(have string, lv *slog.LevelVar, l *slog.Logger) {
			lv.Set(slog.LevelInfo)
			l.Debug(have)
		},
		want: "",
	}, {
		have: "info/info",
		f: func(have string, lv *slog.LevelVar, l *slog.Logger) {
			lv.Set(slog.LevelInfo)
			l.Info(have)
		},
		want: `"level":"INFO","msg":"info/info"}` + "\n",
	}} {
		c := c /* :( */
		t.Run(c.have, func(t *testing.T) {
			lv, lb, l := NewTestLogger()
			c.f(c.have, lv, l)
			got := untimeRE.ReplaceAllString(lb.String(), "")
			if got != c.want {
				t.Fatalf(
					"Incorrect log\n"+
						"have: %s\n"+
						" got: %s\n"+
						"want: %s",
					c.have,
					got,
					c.want,
				)
			}
		})
	}
}

func TestRemoveTimestamp(t *testing.T) {
	for _, c := range []struct {
		have string
		want string
	}{{
		have: `{"time":"2023-11-18T00:01:30.51751336+01:00",` +
			`"level":"DEBUG","msg":"kittens"}` + "\n",
		want: `{"time":"","level":"DEBUG","msg":"kittens"}`,
	}, {
		have: `{"time":"2023-11-18T00:01:30.51751336+01:00",` +
			`"level":"DEBUG","msg":"kittens"}`,
		want: `{"time":"","level":"DEBUG","msg":"kittens"}`,
	}} {
		c := c /* :( */
		t.Run("", func(t *testing.T) {
			got := RemoveTimestamp(c.have)
			if c.want == got {
				return
			}
			t.Errorf(" got: %s", got)
		})
	}
}

func ExampleRemoveTimestamp() {
	msg := `{"time":"2023-10-19T22:36:37.247404361+02:00",` +
		`"level":"ERROR","msg":"kittens"}`
	fmt.Printf("%s\n", RemoveTimestamp(msg))

	// Output: {"time":"","level":"ERROR","msg":"kittens"}
}
