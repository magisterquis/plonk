package operatorsvr

/*
 * opconn_test.go
 * Tests for opconn.go
 * By J. Stuart McMurray
 * Created 20231129
 * Last Modified 20231209
 */

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"
	"testing"

	"github.com/magisterquis/plonk/internal/def"
	"github.com/magisterquis/plonk/lib/estream"
	"github.com/magisterquis/plonk/lib/plog"
)

func TestOpConnDefaultHandler(t *testing.T) {
	_, lb, c := newTestServer(t)

	jes := estream.New(c)
	if err := jes.Send("kittens", []int{1, 2, 3}); nil != err {
		t.Fatalf("Error sending message: %s", err)
	}

	waitForLine(lb)
	want := `{"time":"","level":"WARN","msg":"Unexpected message",` +
		`"message_type":"kittens","message":[1,2,3],` +
		`"opname":"` + testClientName + `","cnum":1}`
	if got := plog.RemoveTimestamp(lastLine(lb)); got != want {
		t.Fatalf("Incorrect log\n got: %s\nwant: %s", got, want)
	}
}

func TestOpConn_LogEvent(t *testing.T) {
	var (
		s, _, c = newTestServer(t)
		jes     = estream.New(c)
		ech     = make(chan error)
		mch     = make(chan [2]string)
		haves   = []struct {
			msg  string
			args []any
		}{
			{msg: "m1", args: []any{"k1", "v1"}},
			{msg: "m2 xx", args: []any{"k2", true}},
		}
		wantPre  = `{"time":"","level":"INFO","msg":`
		wantPost = `}`
		wants    = [][2]string{
			{"m1", `"m1","k1":"v1"`},
			{"m2 xx", `"m2 xx","k2":true`},
		}
	)
	defer c.Close()
	estream.AddHandler(jes, "", func(name string, rm json.RawMessage) {
		mch <- [2]string{
			name,
			plog.RemoveTimestamp(string(rm)),
		}
	})
	go func() {
		ech <- jes.Run()
	}()
	go func() {
		for _, have := range haves {
			s.SL.Info(have.msg, have.args...)
		}
	}()
	go func() { ech <- s.Wait() }()
	var gots [][2]string
	for range wants {
		select {
		case err := <-ech:
			t.Fatalf("Error: %s", err)
		case got := <-mch:
			gots = append(gots, got) /* Good. */
		}
	}

	cmp := func(a, b [2]string) int {
		if c := strings.Compare(a[0], b[0]); 0 != c {
			return c
		}
		return strings.Compare(a[1], b[1])
	}
	slices.SortFunc(gots, cmp)
	slices.SortFunc(wants, cmp)

	for i, got := range gots {
		want := wants[i]
		want[1] = wantPre + want[1] + wantPost
		if got == want {
			continue /* Good. */
		}
		t.Fatalf(
			"Message incorrect:\n"+
				" got: %s\n"+
				"want: %s",
			got,
			want,
		)
	}
}

func TestOpConnNameHandler(t *testing.T) {
	s, _, c := newTestServer(t)
	es := estream.New(c)
	have := "kittens"
	pr, pw := io.Pipe()
	s.FW.Add(pw, nil)
	var (
		gch = make(chan string, 1)
		ech = make(chan error)
	)
	go func() {
		defer pr.Close()
		l, more, err := bufio.NewReader(pr).ReadLine()
		if nil != err {
			ech <- err
		} else if more {
			ech <- errors.New("line too long")
		} else {
			gch <- plog.RemoveTimestamp(string(l))
		}
	}()
	if err := es.Send(def.ENName, def.EDName(have)); nil != err {
		t.Fatalf("Error sending new name: %s", err)
	}
	var got string
	select {
	case err := <-ech:
		t.Fatalf("Error getting log line: %s", err)
	case got = <-gch:
	}
	want := `{"time":"","level":"INFO","msg":"Operator name change",` +
		`"oldname":"test_client","opname":"kittens","cnum":1}`
	if got != want {
		t.Fatalf(
			"Name change log incorrect:\n"+
				"have: %s\n"+
				" got: %s\n"+
				"want: %s",
			have,
			got,
			want,
		)
	}
}

func TestOpConnQueueHandler(t *testing.T) {
	var (
		s, _, c  = newTestServer(t)
		es       = estream.New(c)
		ech      = make(chan error, 3)
		gotch    = make(chan def.EDLMTaskQueued)
		haveID   = "kittens"
		haveTask = "moose"
		want     = def.EDLMTaskQueued{
			ID:     haveID,
			Task:   haveTask,
			OpName: testClientName,
			QLen:   1,
		}
	)
	estream.AddHandler(es, def.ENEnqueue, func(_ string, data def.EDEnqueue) {
		ech <- fmt.Errorf(
			"enqueue: id:%s task:%s err:%s",
			data.ID,
			data.Task,
			data.Error,
		)
	})
	estream.AddHandler(es, "", func(n string, d json.RawMessage) {
		ech <- fmt.Errorf("unexpected %q event: %s", n, d)
	})
	estream.AddHandler(es, def.LMTaskQueued, func(
		_ string,
		got def.EDLMTaskQueued,
	) {
		gotch <- got
	})
	go func() {
		if err := es.Run(); nil != err {
			ech <- err
		}
		ech <- errors.New("server died")
	}()
	go func() {
		if err := es.Send(def.ENEnqueue, def.EDEnqueue{
			ID:   haveID,
			Task: haveTask,
		}); nil != err {
			ech <- err
		}
	}()

	select {
	case err := <-ech:
		t.Fatalf("Error: %s", err)
	case got := <-gotch:
		if got == want {
			break
		}
		t.Fatalf(
			"Incorrect task queued message:\n"+
				" got: %+v\n"+
				"want: %+v",
			got,
			want,
		)
	}

	/* Make sure the task was acutally queued properly. */
	if nil == s.SM.C.TaskQ {
		t.Fatalf("TaskQ is nil")
	}
	q := s.SM.C.TaskQ[haveID]
	switch l := len(q); l {
	case 0:
		t.Fatalf("TaskQ is empty")
	case 1: /* Good. */
		if got := q[0]; haveTask != got {
			t.Fatalf(
				"Incorrect task queued: want:%s got:%s",
				haveTask,
				got,
			)
		}
	default:
		t.Fatalf("TaskQ has wrong size: got:%d want:1", l)
	}
}

func TestOpConnListSeenHandler(t *testing.T) {
	var (
		s, _, c = newTestServer(t)
		es      = estream.New(c)
		nImp    = 5
		ech     = make(chan error, 2)
		gch     = make(chan def.EDSeen, 1)
		lch     = make(chan string, 1)
	)
	/* See some implants. */
	s.SM.Lock()
	for i := 0; i < nImp; i++ {
		s.SM.C.Saw(fmt.Sprintf("id-%d", i), fmt.Sprintf("from-%d", i))
	}
	origWant := s.SM.C.LastSeen
	s.SM.Unlock()

	/* Pass the want list through JSON and back, to remove the m field. */
	var want def.EDSeen
	jb, err := json.Marshal(origWant)
	if nil != err {
		t.Fatalf("Error JSONing want list: %s", err)
	}
	if err := json.Unmarshal(jb, &want); nil != err {
		t.Fatalf("Error unJSONing want list: %s", err)
	}

	/* Ask for the list. */
	estream.AddHandler(es, def.ENListSeen, func(_ string, data def.EDSeen) {
		gch <- data
	})
	estream.AddHandler(
		es,
		def.LMSentSeenList,
		func(_ string, rm json.RawMessage) { lch <- string(rm) },
	)
	go func() { ech <- es.Run() }()
	go func() {
		if err := es.Send(def.ENListSeen, nil); nil != err {
			ech <- err
		}
	}()
	var got def.EDSeen
	select {
	case err := <-ech:
		t.Fatalf("Error waiting on list: %s", err)
	case got = <-gch: /* Good. */
	}

	if got != want {
		t.Errorf(
			"Incorrect list received:\n got: %+v\nwant: %+v",
			got,
			want,
		)
	}

	var lm string
	select {
	case err := <-ech:
		t.Fatalf("Error waiting on log: %s", err)
	case lm = <-lch: /* Good. */
	}

	wantLog := `{"time":"","level":"DEBUG","msg":"Sent implant list",` +
		`"opname":"test_client","cnum":1}`
	if got := plog.RemoveTimestamp(lm); got != wantLog {
		t.Errorf("Log incorrect:\n got: %s\nwant: %s", got, wantLog)
	}
}
