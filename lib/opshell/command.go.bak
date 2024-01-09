package opshell

/*
 * command.go
 * Handle command execution
 * By J. Stuart McMurray
 * Created 20231112
 * Last Modified 20231206
 */

import (
	"errors"
	"fmt"
	"regexp"

	"github.com/magisterquis/plonk/lib/subcom"
	"github.com/magisterquis/simpleshsplit"
)

// cutCommandRE does the heavy lifting for CutCommand.
var cutCommandRE = regexp.MustCompile(`(?s)^\s*(\S+)(?:\s+(\S.*))?`)

// DefaultSplitter is the default function used for splitting commands into
// arguments.
var DefaultSplitter = simpleshsplit.SplitGoUnquote

// ErrQuit may returned by a command handler to cause Shell.HandleCommands to
// stop handling commands.  It is intended to be returned from handler for a
// quit command.
var ErrQuit = errors.New("quit requested")

// CutCommand is similar to strings.Cut, but returns the first non-whitespace
// portion and the remainder of the string with leading whitespace removed
// along with a nil error.  The returned slice will have a length of 0 if s
// was empty or whitespace, 1 if s didn't have two non-whitespace parts
// separated by whitespace, or 2 in any other case.
func CutCommand(s string) ([]string, error) {
	switch ms := cutCommandRE.FindStringSubmatch(s); len(ms) {
	case 0: /* Just spaces */
		return ms, nil
	case 3: /* Normal. */
		/* Don't bother with an empty string. */
		if "" == ms[2] {
			return ms[1:2], nil
		}
		/* Got both parts. */
		return ms[1:], nil
	default:
		return []string{}, nil
	}
}

// SplitError is a decorator indicating the string passed to HandleCommand
// failed to split into arguments; no command was executed.
type SplitError struct {
	Err error
}

// Error implements the error interface.
func (err SplitError) Error() string {
	return fmt.Sprintf("split error: %s", err.Err)
}

// Unwrap returns err.Err.
func (err SplitError) Unwrap() error { return err.Err }

// HandleCommands reads lines from s and uses the configured subcom.Cdr to
// handle commands.  If a command handler returns ErrQuit, command handling
// will stop and HandleCommands will return ErrQuit.
func (s *Shell[T]) HandleCommands() error {
	/* Get lines and handle them. */
	for {
		/* Get a line. */
		l, err := s.ReadLine()
		if nil != err {
			return fmt.Errorf("reading command: %w", err)
		}

		/* Don't care about blanks. */
		if "" == l {
			continue
		}

		/* Handle it. */
		err = s.HandleCommand(l)
		if nil == err { /* All is good. */
			continue
		}

		/* If quit was requested, we're done. */
		if errors.Is(err, ErrQuit) {
			return err
		}

		/* Something bad happened. */
		if eherr := (*s.errorHandler.Load())(s, l, err); nil != eherr {
			return eherr
		}
	}
}

// HandleCommand handles a single command sent to the shell.  If the command
// fails to split, a SplitError is returned.  This is useful for filtering
// lines returned by s.ReadLine before command processing.
func (s *Shell[T]) HandleCommand(cmd string) error {
	/* Parse into arguments. */
	var (
		args []string
		err  error
	)
	if p := s.splitter.Load(); nil != p {
		args, err = (*p)(cmd)
	} else {
		args, err = DefaultSplitter(cmd)
	}
	if nil != err {
		return SplitError{Err: err}
	}

	/* Send command off for execution. */
	return s.Cdr().Call(s, nil, args)
}

// Cdr returns s's Cdr.  This should be used to set command handlers before
// processing commands via s.HandleCommand or s.HandleCommands.
func (s *Shell[T]) Cdr() *subcom.Cdr[*Shell[T]] { return s.cdr }

// SetCommandErrorHandler sets a handler which s.HandleCommands will call when
// an error handling a command is encountered.  If f is nil,
// DefaultErrorHandler will be used.  If the handler returns an error, it will
// be returned by s.HandleCommands.
//
// In particular, two types of errors should be expected: SplitError, which
// indicates that a command was unable to be split into arguments, and
// subcom.ErrNotFound, which indicates that the underlying subcom.Cdr didn't
// have a handler for the requested command.
func (s *Shell[T]) SetCommandErrorHandler(h func(s *Shell[T], line string, err error) error) {
	if nil == h {
		h = DefaultErrorHandler
	}
	s.errorHandler.Store(&h)
}

// SetSplitter sets the function s.HandleCommand and s.HandleCommands use to
// split command lines into arguments.  If f is nil, DefaultSplitter will be
// used.
func (s *Shell[T]) SetSplitter(f func(string) ([]string, error)) {
	switch f {
	case nil:
		s.splitter.Store(&DefaultSplitter)
	default:
		s.splitter.Store(&f)
	}
}

// DefaultErrorHandler is the default handler used when nil or no function has
// been passed to SetSplitErrorHandler or SetUnknownCommandHandler.  It simply
// prints a message via s.Errorf.
func DefaultErrorHandler[T any](s *Shell[T], line string, err error) error {
	s.Errorf("Error handling command %q: %s\n", line, err)
	return nil
}
