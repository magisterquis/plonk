package implantsvr

/*
 * errors.go
 * Error types and handlers
 * By J. Stuart McMurray
 * Created 20231207
 * Last Modified 20231207
 */

import (
	"bufio"
	"errors"
	"io"
	"log"
	"log/slog"

	"github.com/magisterquis/plonk/internal/def"
	"github.com/magisterquis/plonk/lib/plog"
)

// httpErrorLogger returns a log.Logger which logs messages to sl via
// plog.ErrorError and message def.LMHTTPError
func httpErrorLogger(sl *slog.Logger) *log.Logger {
	/* Logger which logs to a pipe. */
	pr, pw := io.Pipe()

	/* Read logged messages from the pipe, slog them. */
	go func() {
		scanner := bufio.NewScanner(pr)
		for scanner.Scan() {
			plog.ErrorError(
				sl,
				def.LMHTTPError,
				errors.New(scanner.Text()),
			)
		}
		if err := scanner.Err(); nil != err {
			plog.ErrorError(
				sl,
				def.LMHTTPErrorFailed,
				err,
			)
		}
	}()

	return log.New(pw, "", 0)
}
