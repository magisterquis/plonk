// Package opshell - Operator's interactive shell
package opshell

/*
 * opshell.go
 * Operator's interactive shell
 * By J. Stuart McMurray
 * Created 20231112
 * Last Modified 20240119
 */

import (
	"io"
	"log"
	"sync"
	"sync/atomic"

	"github.com/magisterquis/plonk/lib/subcom"
	"golang.org/x/term"
)

// Shell represents the interface between an operator and the rest of the
// program.  See the documentation for Shell.V for more information about T.
// Use Config.New to create a new shell.
type Shell[T any] struct {
	/* Output streams. */
	stdout    io.Writer
	stderr    io.Writer
	logger    *log.Logger
	errlogger *log.Logger
	writeL    sync.Mutex

	/* Termish functions.  For a description, see the similarly-named
	functions in golang.org/x/term.Terminal. */
	readLine    func() (string, error)
	setPrompt   func(string)
	setSize     func(int, int) error
	escapeCodes func() *term.EscapeCodes

	isPTY bool /* True if we're in PTY mode. */

	/* Reset the terminal state. */
	resetL sync.Mutex
	reset  func() error

	/* Command-handling. */
	errorHandler atomic.Pointer[func(*Shell[T], string, error) error]
	splitter     atomic.Pointer[func(string) ([]string, error)]
	cdr          *subcom.Cdr[*Shell[T]]
	v            atomic.Pointer[T]
}

// ReadLine reads a line from s.
func (s *Shell[T]) ReadLine() (string, error) { return s.readLine() }

// Escape returns the shell's escape sequences.  See term.EscapeCodes for more
// information.
func (s *Shell[T]) Escape() *term.EscapeCodes { return s.escapeCodes() }

// V returns a value of type T which may be used to store persistent state
// during the shell's lifetime (e.g. between handled commands).  When the Shell
// is first created V will return a zero value of type T.  SetV may be used to
// change the value V returns.
func (s *Shell[T]) V() T {
	/* If we already have something stored, return it. */
	if p := s.v.Load(); nil != p {
		return *p
	}
	/* Failing that, return the zero value. */
	var v T
	s.v.CompareAndSwap(nil, &v)
	return v
}

// SetV changes the value returned by s.V.  Under the hood, it stores a pointer
// to v, not v itself.
func (s *Shell[T]) SetV(v T) { s.v.Store(&v) }
