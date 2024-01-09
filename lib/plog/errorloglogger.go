package plog

/*
 * errorloglogger.go
 * log.Logger-generator for error logging.
 * By J. Stuart McMurray
 * Created 20231018
 * Last Modified 20231111
 */

import (
	"log"
	"log/slog"
	"strings"
)

// errorLogLogger is an io.Writer which writes messages to the underlying
// slog.Logger at level ERROR.  Whitespace is trimmed from logged messages.
type errorLogLogger struct {
	msg    string
	logger *slog.Logger
}

// Write satisfies io.Writer.  It always returns len(p), nil
func (ell errorLogLogger) Write(p []byte) (n int, err error) {
	ell.logger.Error(ell.msg, LKeyError, strings.TrimSpace(string(p)))
	return len(p), nil
}

// ErrorLogLogger returns a log.Logger which logs at level ERROR to logger.
// Writes to the returned log.Logger are written to the underlying slog.Logger
// as an attribute pair with the Key LKeyError and message msg.
func ErrorLogLogger(msg string, logger *slog.Logger) *log.Logger {
	return log.New(errorLogLogger{msg: msg, logger: logger}, "", 0)
}
