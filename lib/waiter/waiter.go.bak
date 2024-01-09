// Package waiter - Wait for a broadcast
package waiter

/*
 * waiter.go
 * Wait for a broadcast
 * By J. Stuart McMurray
 * Created 20231008
 * Last Modified 20231010
 */

import (
	"sync"
)

// Waiter makes it easy to wait for a broadcast.  This is a bit like a
// sync.Cond, but easier to use.  A waiter must not be copied after first use.
type Waiter[T any] struct {
	l       sync.Mutex
	always  *T         /* Message to send to new waiters. */
	waiters []chan<- T /* Waiters waiting for a message. */
}

// WaitChan returns a channel which will receive a value when Broadcast is
// called.  The channel is buffered; a call to Broadcast without a receive on
// the channel will not block or cause leakage.
func (w *Waiter[T]) WaitChan() <-chan T {
	w.l.Lock()
	defer w.l.Unlock()

	/* Waiter's channel. */
	ch := make(chan T, 1)

	/* If we have an always message, send it.  No need to save the
	channel. */
	if nil != w.always {
		ch <- *w.always
		close(ch)
		return ch
	}

	/* Save the waiter for when we get a message. */
	w.waiters = append(w.waiters, ch)
	return ch
}

// Wait waits until Broadcast is called and returns the value passed to
// Broadcast.
func (w *Waiter[T]) Wait() T {
	return <-w.WaitChan()
}

// Broadcast broadcasts e to all current waiters and closes their underlying
// channels.  If AlwaysBroadcast has been called, Broadcast is a no-op.
func (w *Waiter[T]) Broadcast(e T) {
	w.l.Lock()
	defer w.l.Unlock()

	/* If AlwaysBroadcast has been called, not much else to do. */
	if nil != w.always {
		return
	}

	w.broadcast(e)
}

// AlwaysBroadcast broadcasts e to all current waiters and saves it such that
// every new waiter also receives e.  This is useful for fatal errors.  Calls
// to AlwaysBroadcast after the first call to AlwaysBroadcast will not update
// the value returned to waiters.
func (w *Waiter[T]) AlwaysBroadcast(e T) {
	w.l.Lock()
	defer w.l.Unlock()

	/* If we've been called before, nothing to do. */
	if nil != w.always {
		return
	}

	/* Set the always value and broadcast it. */
	w.always = &e

	w.broadcast(e)
}

// broadcast broadcasts e and closes and removes the waiter channels.
// broadcast's caller must hold w.l.
func (w *Waiter[T]) broadcast(e T) {
	for _, ch := range w.waiters {
		ch <- e
		close(ch)
	}
	w.waiters = nil
}
