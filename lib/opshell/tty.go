package opshell

/*
 * tty.go
 * Handle TTY things
 * By J. Stuart McMurray
 * Created 20231112
 * Last Modified 20231207
 */

import (
	"fmt"
	"io"
	"os"
	"os/signal"

	"golang.org/x/sys/unix"
	"golang.org/x/term"
)

// readWriter combines a redaer and a writer.
type readWriter struct {
	r io.Reader
	w io.Writer
}

// Read reads from rw.r.
func (rw readWriter) Read(p []byte) (int, error) { return rw.r.Read(p) }

// Write writes to rw.w.
func (rw readWriter) Write(p []byte) (int, error) { return rw.w.Write(p) }

// SetSize sets s's size in PTY mode and is a no-op otherwise.  In PTY mode,
// s's size will be automatically changed upon receipt of SIGWINCH.
func (s *Shell[T]) SetSize(width, height int) error {
	return s.setSize(width, height)
}

// ResetTerm resets s's underlying Terminal to its original state if it was
// put into raw mode due to s being in PTY mode.  ResetTerm should be called
// to prevent stdio being in raw mode on program exit and is safe to call
// even if s is not in PTY mode.
func (s *Shell[T]) ResetTerm() error {
	s.resetL.Lock()
	defer s.resetL.Unlock()
	if nil != s.reset {
		return s.reset()
	}
	return nil
}

// handleSIGWINCH watches for SIGWINCH and handles changed terminal sizes on
// the given file descriptor.
func (s *Shell[T]) handleSIGWINCH(fd int) {
	/* Watch for the signal. */
	ch := make(chan os.Signal, 10)
	signal.Notify(ch, unix.SIGWINCH)

	/* Every time we're signalled, resize. */
	for range ch {
		if err := s.resizeTTY(fd); nil != err {
			s.ErrorLogf("Error resizing terminal: %s", err)
		}
	}
}

// resizeTTY resizes to the size from the TTY on file descriptor fd.
func (s *Shell[T]) resizeTTY(fd int) error {
	/* Get the current size. */
	w, h, err := term.GetSize(fd)
	if nil != err {
		return fmt.Errorf("getting size: %w", err)
	}
	/* Set our internal idea of the size. */
	if err := s.SetSize(w, h); nil != err {
		return fmt.Errorf("setting size: %w", err)
	}
	return nil
}

// SetPrompt sets the prompt s presents to the user if in PTY mode and is a
// no-op if not in PTY mode.
func (s *Shell[T]) SetPrompt(prompt string) { s.setPrompt(prompt) }

// InPTYMode indicates whether or not we're in PTY mode
func (s *Shell[T]) InPTYMode() bool { return s.isPTY }
