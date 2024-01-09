// Package estream - JSON-based event stream
package estream

/*
 * estream.go
 * JSON-based event stream
 * By J. Stuart McMurray
 * Created 20231122
 * Last Modified 20231207
 */

import (
	"encoding/json"
	"fmt"
	"io"
	"sync"
)

// handlerFunc is the type of a function which reads a payload and calls
// a handler.
type handlerFunc func(name string) error

// Delete may be passed to AddHandler to request deletion of a handler.  It is
// equivalent to AddHandler[any](s, name, nil), but somewhat easier to read.
var Delete = (func(string, any))(nil)

// Stream is a bidirectional stream of events.
type Stream struct {
	wl   sync.Mutex
	rwc  io.ReadWriteCloser
	enc  *json.Encoder
	dec  *json.Decoder
	hm   sync.Map
	runL sync.Mutex
	dh   handlerFunc
}

// New creats a new stream.  After creating the stream, call AddHandler to add
// handlers and then call Run to process events.
func New(rwc io.ReadWriteCloser) *Stream {
	s := &Stream{
		rwc: rwc,
		enc: json.NewEncoder(rwc),
		dec: json.NewDecoder(rwc),
	}
	s.dh = handlerFuncFrom(s, "", defaultHandler)
	return s
}

// Run start event processing on the stream.  The stream will not read from its
// underlying io.ReadWriteCloser until Run or RunOnce is called, which may
// cause blocking if care is not taken.
func (s *Stream) Run() error {
	/* Read and process events. */
	for {
		if err := s.RunOnce(); nil != err {
			return err
		}
	}
}

// RunOnce processes a single event on the stream.
func (s *Stream) RunOnce() error {
	/* Make sure we don't interleave reads. */
	s.runL.Lock()
	defer s.runL.Unlock()

	/* Get an event name. */
	var name string
	if err := s.dec.Decode(&name); nil != err {
		return fmt.Errorf("reading event name: %w", err)
	}
	/* Get the data and handle. */
	if err := s.handlerFunc(name)(name); nil != err {
		return fmt.Errorf(
			"starting %q event handler: %w",
			name,
			err,
		)
	}

	return nil
}

// Send sends an event with the given event name and data via the Stream's
// underlying io.ReadWriteCloser.
func (s *Stream) Send(name string, data any) error {
	s.wl.Lock()
	defer s.wl.Unlock()

	/* Send the event name. */
	if err := s.enc.Encode(name); nil != err {
		return fmt.Errorf("sending name: %w", err)
	}
	/* Send the event data. */
	if err := s.enc.Encode(data); nil != err {
		return fmt.Errorf("sending data: %w", err)
	}

	return nil
}

// Close closes s's underlying io.ReadWriteCloser.
func (s *Stream) Close() error {
	return s.rwc.Close()
}

// handler returns the correct handler for the given event name.  If none has
// been registered, the handler with the empty string name will be used, or
// failing that defaultHandler.
func (s *Stream) handlerFunc(name string) handlerFunc {
	/* Try to get a name-specific handler. */
	if h, ok := s.hm.Load(name); ok && nil != h {
		return h.(handlerFunc)
	}
	/* Failing that, try for the user-set default handler. */
	if h, ok := s.hm.Load(""); ok && nil != h {
		return h.(handlerFunc)
	}
	/* Failed.  Call the default handler. */
	return s.dh
}

// AddHandler adds a handler to a stream.  It is not a method on Stream due to
// restrictions on generic functions.  AddHandler may be called at any time,
// even during a call to s.Run.  If handler is nil, s's handler for the given
// name is deleted, if it has one.  To set a default handler, pass the empty
// string as the name.
func AddHandler[T any](s *Stream, name string, handler func(string, T)) {
	/* Delete if we're meant to. */
	if nil == handler {
		s.hm.Delete(name)
		return
	}

	/* Add a new handler for this event name. */
	s.hm.Store(name, handlerFuncFrom(s, name, handler))
}

// handlerFuncFrom returns a handlerFunc for the given name which calls f.
func handlerFuncFrom[T any](s *Stream, name string, f func(string, T)) handlerFunc {
	return func(n string) error {
		/* Get the event data. */
		data := new(T)
		if err := s.dec.Decode(data); nil != err {
			return fmt.Errorf("unmarshalling data: %w", err)
		}
		/* Call the handler. */
		go f(n, *data)
		return nil
	}
}

// defaultHandler is called when there is no registered handler for a message.
func defaultHandler(name string, v any) { /* No-op. */ }
