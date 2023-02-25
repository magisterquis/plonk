// Package debug is used during development.  It shouldn't be in any release.
//
// All of the functions in this package log to stderr, regardless of how the
// log package's default logger is configured.
package debug

/*
 * listeners.go
 * Listen on the network
 * By J. Stuart McMurray
 * Created 20220829
 * Last Modified 20230120
 */

import (
	"fmt"
	"log"
	"net"
	"os"
	"runtime"
)

// selog logs to stderr with microsecond precision.
var selog = log.New(os.Stderr, "", log.LstdFlags|log.Lmicroseconds)

func init() {
	log.Printf("DEBUG PACKAGE LOADED")
}

// FinishThis prints a TODO: Finish this message with the current file and
// line, if they can be determined.
func FinishThis() {
	nLogf(1, "TODO: Finish this")
}

// Here prints "Here" with the current file and line, if they can be
// determined, meant for wolf fencing.
func Here() {
	nLogf(1, "Here")
}

// TODO prints a TODO: message with the current file and line, if they can be
// determined.
func TODO(f string, a ...any) {
	nLogf(1, "TODO: %s", fmt.Sprintf(f, a...))
}

// Logf logs a message with the current file and line, if they can be
// determined.
func Logf(f string, a ...any) {
	nLogf(1, f, a...)
}

// nLogf logs a message, but allows setting the depth of the stack frame from
// which to get the file and line.
func nLogf(n int, f string, a ...any) {
	m := fmt.Sprintf(f, a...)
	selog.Printf("[%s] %s", fileAndLineTag(n+1), m)
}

// fileAndLineTag returns a tag with the current file and line, if they can
// be determined, otherwise "Unknown location".  n will be passed to
// runtime.Caller and should be the number of functions between fileAndLineTag
// and the calling function of interest.
func fileAndLineTag(n int) string {
	_, fn, ln, ok := runtime.Caller(n + 1)
	if !ok {
		return "Unknown location"
	}
	return fmt.Sprintf("%s:%d", fn, ln)
}

// Listener listens for tcp connections on addr and calls f in its own
// goroutine for each accepted connection.  It return an error on any listen or
// accept errors, even temporary ones.  Log messages will be printed to stderr
// when the listen succeeds as well as for each accepted connection.
func Listener(addr string, f func(c net.Conn)) error {
	/* Start a listener going. */
	l, err := net.Listen("tcp", "127.0.0.1:3232")
	if nil != err {
		return fmt.Errorf("listen: %w", err)
	}
	nLogf(1, "Listening on %s", l.Addr())

	/* Handle each connection. */
	for {
		c, err := l.Accept()
		if nil != err {
			return fmt.Errorf("accept: %w", err)
		}
		nLogf(
			1,
			"Got connection %s<-%s",
			c.LocalAddr(),
			c.RemoteAddr(),
		)
		go f(c)
	}
}
