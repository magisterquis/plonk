package operatorsvr

/*
 * handle.go
 * Tests for handle.go
 * By J. Stuart McMurray
 * Created 20231205
 * Last Modified 20231207
 */

import (
	"context"
	"fmt"
	"net"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/magisterquis/plonk/internal/def"
	"github.com/magisterquis/plonk/lib/estream"
	"golang.org/x/exp/maps"
	"golang.org/x/sync/semaphore"
)

func TestHandleConn_FastDisconnects(t *testing.T) {
	s, _, c := newTestServer(t)
	nConn := 100
	maxConns := int64(50)
	a := s.l.Addr().String()
	connSem := semaphore.NewWeighted(maxConns)

	/* Make and disconnect a bunch of connections, quickly. */
	ech := make(chan error, 1)
	go func() { ech <- fmt.Errorf("server died: %w", s.Wait()) }()
	for i := 1; i < nConn+1; i++ {
		go func(n int) {
			connSem.Acquire(context.Background(), 1)
			defer connSem.Release(1)
			nc, err := net.Dial("unix", a)
			if nil != err {
				ech <- fmt.Errorf(
					"connection %d/%d failed: %w",
					n,
					nConn,
					err,
				)
				return
			}
			nes := estream.New(nc)
			name := fmt.Sprintf("nc-%d", n+1)
			donech := make(chan struct{}, 1)
			var doneOk atomic.Bool
			estream.AddHandler(
				nes,
				def.LMOpConnected,
				func(en string, data def.EDLMOpConnected) {
					if name != data.OpName {
						return
					}
					doneOk.Store(true)
					close(donech)
				},
			)
			go func() {
				err := nes.Run()
				if nil != err && !doneOk.Load() {
					ech <- fmt.Errorf(
						"handling events (%d): %w",
						n,
						err,
					)
				}
			}()
			if err := nes.Send(
				def.ENName,
				def.EDName(name),
			); nil != err {
				ech <- fmt.Errorf(
					"sending name (%d): %w",
					n,
					err,
				)
			}

			<-donech
			nc.Close()
		}(i)
	}

	/* Watch for connected/disconnected messages. */
	gch := make(chan string, nConn)
	es := estream.New(c)
	h := func(n string, m struct{ CNum int }) {
		gch <- fmt.Sprintf("%s: % 5d", n, m.CNum)
	}
	estream.AddHandler(es, def.LMOpConnected, h)
	estream.AddHandler(es, def.LMOpDisconnected, h)
	go func() {
		ech <- fmt.Errorf("estream: %w", es.Run())
	}()

	/* Logs we expect to get. */
	logs := make(map[string]struct{})
	for i := 2; i < nConn+2; i++ {
		for _, op := range []string{"", "dis"} {
			logs[fmt.Sprintf(
				"Operator %sconnected: % 5d",
				op,
				i,
			)] = struct{}{}
		}
	}

	/* Wait for enough connected/disconnected messages or an error. */
	var (
		gotNMsg  = 0
		wantNMsg = 2 * nConn
		msgWait  = 2 * time.Second
		deadline = time.Now().Add(time.Second)
	)
	for gotNMsg < wantNMsg {
		select {
		case <-time.After(time.Until(deadline)):
			left := maps.Keys(logs)
			slices.Sort(left)
			t.Fatalf(
				"Got %d/%d logs in %s, remaining:\n%s",
				gotNMsg,
				wantNMsg,
				msgWait,
				strings.Join(left, "\n"),
			)
		case err := <-ech:
			t.Fatalf("Error: %s", err)
		case msg := <-gch:
			if _, ok := logs[msg]; !ok {
				t.Fatalf("Missing: %s", msg)
			}
			delete(logs, msg)
			gotNMsg++
		}
	}
}
