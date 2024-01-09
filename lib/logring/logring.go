// Package logring - Ring buffer for log messages
package logring

/*
 * logring.go
 * Ring buffer for log messages
 * By J. Stuart McMurray
 * Created 20231219
 * Last Modified 20231219
 */

import (
	"bytes"
	"log"
	"slices"
	"strings"
	"sync"
)

// Ring is a ring buffer.  Each call to Ring.Write stores a log message.  All
// of Ring's methods are safe for concurrent use.
type Ring struct {
	next    int
	bufLen  int
	nStored int
	buf     []string
	l       sync.Mutex
	lb      bytes.Buffer
	lw      *log.Logger
}

// New returns a new ring which stores the last n messages written to it.
func New(n int) *Ring {
	r := &Ring{
		bufLen: n,
		buf:    make([]string, n),
	}
	r.lw = log.New(&r.lb, "", log.LstdFlags)
	return r
}

// Printf is like log.Printf, but log lines are stored in r.
func (r *Ring) Printf(format string, v ...any) {
	r.l.Lock()
	defer r.l.Unlock()

	r.lw.Printf(format, v...)
	r.buf[r.next] = strings.Trim(r.lb.String(), "\n")
	r.next++
	r.next %= r.bufLen

	/* Update the number of messages we have. */
	if r.nStored < r.bufLen {
		r.nStored++
	}

	/* For next time. */
	r.lb.Reset()
}

// Messages returns the messages stored in the Ring.  The messages will not
// end in a newline.
func (r *Ring) Messages() []string {
	r.l.Lock()
	defer r.l.Unlock()
	return r.messages()
}

// messages returns the messages stored in the ring.  The messages will not
// end in a newline.  messages' caller must hold r.l.
func (r *Ring) messages() []string {
	ret := make([]string, 0, r.bufLen)
	for i := 0; i < r.bufLen; i++ {
		m := r.buf[(i+r.next)%r.bufLen]
		if "" == m {
			continue
		}
		ret = append(ret, m)
	}
	return slices.Clip(ret)
}

// MessagesAndClear is like Messages, but also clears the buffer.
func (r *Ring) MessagesAndClear() []string {
	r.l.Lock()
	defer r.l.Unlock()
	ret := r.messages()
	for i := range r.buf {
		r.buf[i] = ""
	}
	r.nStored = 0
	return ret
}

// Cap returns the capacity of the buffer.  This is the value passed to New.
func (r *Ring) Cap() int { return r.bufLen }

// Len returns the number of messages stored in the buffer.
func (r *Ring) Len() int { return r.nStored }
