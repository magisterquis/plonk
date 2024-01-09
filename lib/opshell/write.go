package opshell

/*
 * write.go
 * Write to the shell
 * By J. Stuart McMurray
 * Created 20231112
 * Last Modified 20231121
 */

import (
	"fmt"
)

// Printf writes to s in the manner of fmt.Printf.
func (s *Shell[T]) Printf(format string, args ...any) (int, error) {
	s.writeL.Lock()
	defer s.writeL.Unlock()
	return fmt.Fprintf(s.stdout, format, args...)
}

// Logf is like Printf, but prepends a timestamp.
func (s *Shell[T]) Logf(format string, args ...any) {
	s.writeL.Lock()
	defer s.writeL.Unlock()
	s.logger.Printf(format, args...)
}

// Write writes to s's Writer.
func (s *Shell[T]) Write(b []byte) (int, error) {
	s.writeL.Lock()
	defer s.writeL.Unlock()
	return s.stdout.Write(b)
}

// Errorf is identical to Printf, but writes to s's ErrorWriter if not in PTY
// mode.
func (s *Shell[T]) Errorf(format string, args ...any) (int, error) {
	s.writeL.Lock()
	defer s.writeL.Unlock()
	return fmt.Fprintf(s.stderr, format, args...)
}

// ErrorWrite is identical to PrintWrite, but writes to s's ErrorWriter if not
// in PTY mode.
func (s *Shell[T]) ErrorWrite(b []byte) (int, error) {
	s.writeL.Lock()
	defer s.writeL.Unlock()
	return s.stderr.Write(b)
}

// ErrorLogf is identical to Logf, but writes to s's ErrorWriter if not in PTY
// mode.
func (s *Shell[T]) ErrorLogf(format string, args ...any) {
	s.writeL.Lock()
	defer s.writeL.Unlock()
	s.errlogger.Printf(format, args...)
}
