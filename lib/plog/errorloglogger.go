package plog

/*
 * errorloglogger.go
 * log.Logger-generator for error logging.
 * By J. Stuart McMurray
 * Created 20231018
 * Last Modified 20231227
 */

import (
	"bytes"
	"log"
	"log/slog"
	"regexp"
	"strings"
)

// errorLogLogger is an io.Writer which writes messages to the underlying
// slog.Logger at level ERROR.  Whitespace is trimmed from logged messages.
type errorLogLogger struct {
	msg      string
	logger   *slog.Logger
	debugREs []*regexp.Regexp
}

// Write satisfies io.Writer.  It always returns len(p), nil
func (ell errorLogLogger) Write(p []byte) (n int, err error) {
	/* slog wuold get rid of the newline, but we also remove it for
	ease of regexing. */
	p = bytes.TrimRight(p, "\n")

	/* Work out how to log this thing. */
	lf := ell.logger.Error
	for _, dre := range ell.debugREs {
		if dre.Match(p) {
			lf = ell.logger.Debug
			break
		}
	}

	lf(ell.msg, LKeyError, strings.TrimSpace(string(p)))
	return len(p), nil
}

// ErrorLogLogger returns a log.Logger which logs at level ERROR to logger.
// Writes to the returned log.Logger are written to the underlying slog.Logger
// as an attribute pair with the Key LKeyError and message msg.  If any regular
// expressions are passed via debugREs, log messages written to the log.Logger
// which match any of the regular expressions will be logged at level DEBUG,
// not level ERROR.  This is useful on internet-facing servers for not logging
// EOFs caused by banner-grabbing.
func ErrorLogLogger(msg string, logger *slog.Logger, debugREs ...*regexp.Regexp) *log.Logger {
	return log.New(errorLogLogger{
		msg:      msg,
		logger:   logger,
		debugREs: debugREs,
	}, "", 0)
}