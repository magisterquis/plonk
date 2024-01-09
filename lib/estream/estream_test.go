package estream

/*
 * estream_test.go
 * JSON-based event stream
 * By J. Stuart McMurray
 * Created 20231122
 * Last Modified 20231205
 */

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"testing"
)

func streamPair() (*Stream, *Stream) {
	leftConn, rightConn := net.Pipe()
	return New(leftConn), New(rightConn)
}

func TestStreamSend(t *testing.T) {
	for _, c := range []struct {
		name string
		have any
		want string /* fmt.Sprintf */
	}{{
		name: "bool",
		have: true,
		want: "true",
	}, {
		name: "number",
		have: 100,
		want: "100",
	}, {
		name: "string",
		have: "kittens",
		want: `"kittens"`,
	}, {
		name: "struct",
		have: struct {
			S string
		}{"kittens"},
		want: `map[string]interface {}{"S":"kittens"}`,
	}} {
		c := c /* :( */
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			rs, ss := streamPair()
			gch := make(chan string, 1)
			AddHandler(rs, c.name, func(n string, v any) {
				if n != c.name {
					t.Errorf("Name mismatch, got %s", n)
				}
				gch <- fmt.Sprintf("%#v", v)
			})

			ner := 3
			ech := make(chan error, ner)
			go func() {
				ech <- ss.Send(c.name, c.have)
				ech <- ss.Close()
			}()
			go func() { ech <- rs.Run() }()

			if got := <-gch; got != c.want {
				t.Errorf(
					"Receive incorrect:\n got: %s\n want: %s",
					got,
					c.want,
				)
			}

			for i := 0; i < ner; i++ {
				err := <-ech
				if nil == err || errors.Is(err, io.EOF) {
					continue
				}
				t.Errorf("Error: %s", err)
			}
		})
	}
}

func TestStream_DefaultHandler(t *testing.T) {
	rs, ss := streamPair()
	nch := make(chan string, 1)
	gch := make(chan string, 1)
	AddHandler(rs, "", func(name string, rm json.RawMessage) {
		nch <- name
		gch <- string(rm)
	})

	name := "testevent"
	have := struct {
		N int
		S string
	}{1024, "kittens"}
	want := `{"N":1024,"S":"kittens"}`

	ech := make(chan error, 1)
	go func() { ech <- rs.Run() }()
	if err := ss.Send(name, have); nil != err {
		t.Fatalf("Send error: %s", err)
	}
	if err := ss.Close(); nil != err {
		t.Fatalf("Close error: %s", err)
	}
	if err := <-ech; !errors.Is(err, io.EOF) {
		t.Fatalf("Run error: %s", err)
	}
	if ngot := <-nch; name != ngot {
		t.Errorf("Name incorrect:\n got: %s\nwant: %s", ngot, name)
	}
	if got := <-gch; got != want {
		t.Errorf("Message incorrect:\n got: %s\n want: %s", got, want)
	}
}

func TestStream_DeleteHandler(t *testing.T) {
	name := "kittens"

	rs, ss := streamPair()
	dch := make(chan string, 2)
	nch := make(chan string, 2)
	AddHandler(rs, "", func(name string, v json.RawMessage) {
		dch <- name
	})
	AddHandler(rs, name, func(name string, v json.RawMessage) {
		nch <- name
	})
	go rs.Run()

	if err := ss.Send(name, nil); nil != err {
		t.Fatalf("First send: %s", err)
	}

	select {
	case got := <-dch:
		t.Fatalf("Default handler called with name %q", got)
	case got := <-nch:
		if got != name {
			t.Fatalf("Wrong name: %q", got)
		}
	}

	AddHandler[int](rs, name, nil)
	if err := ss.Send(name, nil); nil != err {
		t.Fatalf("Second send: %s", err)
	}

	select {
	case got := <-dch:
		if got != name {
			t.Fatalf("Default handler got wrong name: %q", got)
		}
	case got := <-nch:
		t.Fatalf("Deleted handler called with name %q", got)
	}
}

func TestAddHandler_Delete(t *testing.T) {
	name := "kittens"
	s := New(nil)
	if _, ok := s.hm.Load(name); ok {
		t.Fatalf("Handler set by default")
	}
	AddHandler(s, name, func(string, int) {})
	if h, ok := s.hm.Load(name); !ok {
		t.Fatalf("Handler was not actually set")
	} else if nil == h {
		t.Fatalf("Nil handler set")
	}
	AddHandler(s, name, Delete)
	if _, ok := s.hm.Load(name); ok {
		t.Fatalf("Handler not actually deleted")
	}
}

func Example() {
	/* Set up a pair of event handlers. */
	done := make(chan struct{})
	leftConn, rightConn := net.Pipe()
	leftStream := New(leftConn)
	AddHandler[int](leftStream, "number", func(name string, number int) {
		fmt.Printf("Left got a number: %d\n", number)
		if err := leftStream.Send("boolean", true); nil != err {
			fmt.Printf(
				"Error sending boolean to right side: %s",
				err,
			)
		}
	})
	rightStream := New(rightConn)
	AddHandler[bool](rightStream, "boolean", func(name string, tf bool) {
		fmt.Printf("Right got a boolean: %t\n", tf)
		close(done)
	})

	/* Start processing events. */
	go func() {
		if err := leftStream.Run(); nil != err &&
			!errors.Is(err, io.EOF) {
			fmt.Printf("Left run error: %s\n", err)
		}
	}()
	go func() {
		if err := rightStream.Run(); nil != err &&
			!errors.Is(err, io.EOF) {
			fmt.Printf("Right run error: %s\n", err)
		}
	}()

	/* Send an event which makes an event. */
	if err := rightStream.Send("number", 100); nil != err {
		fmt.Printf("Error sending number to left side: %s", err)
	}

	/* Wait for the events to finish processing. */
	<-done

	//output:
	// Left got a number: 100
	// Right got a boolean: true
}
