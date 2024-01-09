package server

/*
 * log.go
 * Logging subsystem
 * By J. Stuart McMurray
 * Created 20231129
 * Last Modified 20231214
 */

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/magisterquis/plonk/internal/def"
	"github.com/magisterquis/plonk/lib/flexiwriter"
	"github.com/magisterquis/plonk/lib/plog"
)

// initLogging initializes s's logging.
func (s *Server) initLogging() error {
	/* Open logfile. */
	fn := filepath.Join(s.Dir, def.LogFile)
	var err error
	if s.lf, err = os.OpenFile(
		fn,
		os.O_CREATE|os.O_WRONLY|os.O_APPEND,
		def.FilePerms,
	); nil != err {
		return fmt.Errorf("opening logfile %s: %w", fn, err)
	}

	/* Flexiwriter to send logs to multiple places. */
	s.fw = flexiwriter.New(s.lf)
	/* Also stdout if we're not testing. */
	if o := s.TestLogOutput; nil != o {
		s.fw.Add(o, nil)
	} else {
		s.fw.Add(os.Stdout, nil)
	}

	/* Logger to log to the flexiwriter. */
	var lv *slog.LevelVar
	lv, s.sl = plog.NewJSONLogger(s.fw)
	if s.Debug {
		lv.Set(slog.LevelDebug)
	}

	return nil
}
