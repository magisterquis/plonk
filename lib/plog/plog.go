// Package plog - Logging convenience functions
package plog

/*
 * plog.go
 * Logging convenience functions
 * By J. Stuart McMurray
 * Created 20230810
 * Last Modified 20231018
 */

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"os"
	"regexp"
	"strings"
)

// Canned attr keys.
const (
	LKeyError = "error"
)

var (
	/* DefaultLevelVar controls the level at which Logger logs. */
	DefaultLevelVar slog.LevelVar

	/* Default is our own default logger.  It may be used instead of
	slog.Default().  Its level is controlled by DefaultLevelVar. */
	Default = slog.New(NewHandler(&DefaultLevelVar, slog.Default().Handler()))
)

// removeTSRE
var removeTSRE = regexp.MustCompile(`^({"time":")[^"]+(")`)

// SetDebugLogging enables or disables DEBUG-level logs.
func SetDebugLogging(on bool) {
	if on {
		DefaultLevelVar.Set(slog.LevelDebug)
	} else {
		DefaultLevelVar.Set(slog.LevelInfo)
	}
}

// ErrorAttr turns the error into an slog.Attr with the key KeyError.
func ErrorAttr(err error) slog.Attr { return slog.Any(LKeyError, err) }

// logErrorLevel logs an error at the given level with the given, or Logger if
// nil.
func logErrorLevel(
	sl *slog.Logger,
	level slog.Level,
	msg string,
	err error,
	args ...any,
) {
	/* Work out where to log. */
	if nil == sl {
		sl = Default
	}

	/* Log it nicely. */
	sl.Log(
		context.Background(),
		level,
		msg,
		append(args, ErrorAttr(err))...,
	)
}

// InfoError is like ErrorError, with a level of info.
func InfoError(sl *slog.Logger, msg string, err error, args ...any) {
	logErrorLevel(sl, slog.LevelInfo, msg, err, args...)
}

// WarnError is like Error, with a level of warn.
func WarnError(sl *slog.Logger, msg string, err error, args ...any) {
	logErrorLevel(sl, slog.LevelWarn, msg, err, args...)
}

// ErrorError logs the error with sl.Error, or slog.Error if sl is nil.
func ErrorError(sl *slog.Logger, msg string, err error, args ...any) {
	logErrorLevel(sl, slog.LevelError, msg, err, args...)
}

// FatalError wraps ErrorError and then calls os.Exit(1).
func FatalError(sl *slog.Logger, msg string, err error, args ...any) {
	ErrorError(sl, msg, err, args...)
	os.Exit(1)
}

// NewTestLogger returns a new JSON slog.Logger wrapped in a Handler which logs
// to the returned bytes.Buffer.  The logger's Level is set to DEBUG.
func NewTestLogger() (*slog.LevelVar, *bytes.Buffer, *slog.Logger) {
	var lb bytes.Buffer
	lv, l := NewJSONLogger(&lb)
	lv.Set(slog.LevelDebug)
	return lv, &lb, l
}

// NewJSONLogger returns a new Logger which wraps a slog.NewJSONHandler which
// writes to w.  Only the LevelVar field is set in the slog.HandlerOptions
// passed to slog.NewJSONHandler.
func NewJSONLogger(w io.Writer) (*slog.LevelVar, *slog.Logger) {
	var lv slog.LevelVar
	return &lv, slog.New(NewHandler(&lv, slog.NewJSONHandler(
		w,
		&slog.HandlerOptions{Level: &lv},
	)))
}

// RemoveTimestamp removes a timestamp from a log message.  This is
// intended to be used in test functions.  The returned string will not end in
// a newline.
func RemoveTimestamp(s string) string {
	return strings.TrimRight(removeTSRE.ReplaceAllString(s, "$1$2"), "\n")
}
