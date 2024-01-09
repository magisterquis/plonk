package estream

/*
 * logevents_test.go
 * Tests for logevents.go
 * By J. Stuart McMurray
 * Created 20231205
 * Last Modified 20231205
 */

import (
	"encoding/json"
	"io"
	"testing"

	"github.com/magisterquis/plonk/lib/plog"
)

func TestStreamSendJSONSLogs(t *testing.T) {
	var (
		rs, ss = streamPair()
		pr, pw = io.Pipe()
		_, sl  = plog.NewJSONLogger(pw)
		ech    = make(chan error)
		dch    = make(chan string)
		gch    = make(chan [2]string)
		wants  = [][2]string{
			{"m1", "kittens"},
		}
	)
	AddHandler(rs, "", func(name string, v json.RawMessage) {
		dch <- name
	})
	AddHandler(rs, "m1", func(name string, v struct{ S string }) {
		gch <- [2]string{name, v.S}
	})
	go func() { ech <- rs.Run() }()
	go func() { ech <- ss.SendJSONSLogs(pr) }()
	go func() {
		sl.Info("m1", "S", "kittens")
	}()
	for i, want := range wants {
		select {
		case d := <-dch:
			t.Fatalf("Default handler called for %q", d)
		case err := <-ech:
			t.Fatalf("Error: %s", err)
		case got := <-gch:
			if got == want {
				break
			}
			t.Errorf(
				"Send %d failed:\n got: %s\nwant: %s",
				i+1,
				got,
				want,
			)
		}
	}
}
