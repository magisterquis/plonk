// Package server - Main subsystem wrangler
package server

/*
 * server.go
 * Main subsystem wrangler
 * By J. Stuart McMurray
 * Created 20231110
 * Last Modified 20231214
 */

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"sync/atomic"

	"github.com/magisterquis/plonk/internal/def"
	"github.com/magisterquis/plonk/internal/server/implantsvr"
	"github.com/magisterquis/plonk/internal/server/operatorsvr"
	"github.com/magisterquis/plonk/internal/server/state"
	"github.com/magisterquis/plonk/lib/flexiwriter"
	"github.com/magisterquis/plonk/lib/jpersist"
	"github.com/magisterquis/plonk/lib/waiter"
)

// Server implements the server side of Plonk.  Before starting, its public
// fields should be populated.  Start should be used to start it and then
// Wait to wait for something bad to happen.  Once Start is called, Server's
// fields should be treated as read-only.
type Server struct {
	Dir   string
	Debug bool

	/* Listen addresses. */
	HTTPAddr  string
	HTTPSAddr string

	/* TLS Config.  If HTTPSAddr is set and there's no domains set, no
	Let's Encrypt provisioning will take place and a self-signed cert
	will be used for all HTTP requests. */
	LEDomainWhitelist []string
	SSDomainWhitelist []string
	LEStaging         bool
	LEEmail           string

	/* Other config. */
	ExfilMax uint64

	/* Logging. */
	sl *slog.Logger
	fw *flexiwriter.Writer
	lf *os.File /* For closing during testing. */

	/* Persistent state. */
	sm *jpersist.Manager[state.State]

	/* Servers. */
	os *operatorsvr.Server
	is *implantsvr.Server

	/* Only for testing. */
	TestLogOutput io.Writer

	/* Stopping. */
	stopped atomic.Bool
	ew      waiter.Waiter[error]
}

// Start starts the server.
func (s *Server) Start() error {
	/* Make sure our directory exists. */
	if err := os.MkdirAll(s.Dir, def.DirPerms); nil != err {
		return fmt.Errorf("making directory: %s", err)
	}

	/* Set up subsystems. */
	if err := s.initLogging(); nil != err {
		return fmt.Errorf("initializing logging: %w", err)
	}

	/* Set up persistent state. */
	if sm, err := state.New(s.Dir, s.sl, func(err error) {
		s.Stop(fmt.Errorf("persistent state: %w", err))
	}); nil != err {
		return fmt.Errorf("initializing persistent state: %w", err)
	} else {
		s.sm = sm
	}

	/* Handle implant requests. */
	s.is = &implantsvr.Server{
		Dir:               s.Dir,
		SL:                s.sl,
		SM:                s.sm,
		ExfilMax:          s.ExfilMax,
		HTTPAddr:          s.HTTPAddr,
		HTTPSAddr:         s.HTTPSAddr,
		LEDomainWhitelist: s.LEDomainWhitelist,
		SSDomainWhitelist: s.SSDomainWhitelist,
		LEStaging:         s.LEStaging,
		LEEmail:           s.LEEmail,
	}
	if err := s.is.Start(); nil != err {
		return s.Stop(fmt.Errorf("starting implant service: %w", err))
	}
	go func() {
		err := s.is.Wait()
		if nil != err {
			err = fmt.Errorf("implant service mysteriously died")
		}
		s.Stop(err)
	}()

	/* Handle operator connections.  We do this after implants to avoid
	blowing away the operator socket if we can't listen, as happens when
	someone starts two instances of the server. */
	s.os = &operatorsvr.Server{
		Dir: s.Dir,
		SL:  s.sl,
		FW:  s.fw,
		SM:  s.sm,
	}
	if err := s.os.Start(); nil != err {
		return s.Stop(fmt.Errorf("starting operator service: %w", err))
	}
	go func() {
		err := s.os.Wait()
		if nil == err {
			err = fmt.Errorf("operator service mysteriously died ")
		}
		s.Stop(err)
	}()

	s.sl.Info(def.LMServerReady, def.LKDirname, s.Dir)

	return nil
}

// Stop stops the server.  It returns the same value as wait.  If no other
// error is to be returned, defErr is returned by both Wait and Stop.
func (s *Server) Stop(defErr error) error {
	/* Don't double-stop. */
	if !s.stopped.CompareAndSwap(false, true) {
		return s.Wait()
	}

	/* Stop servers. */
	if nil != s.os {
		/* Work out what to tell operators. */
		var msg string
		if nil != defErr {
			msg = "Error: " + defErr.Error()
		}
		/* Tell them goodbye. */
		if err := s.os.Stop(msg); nil != err {
			s.ew.AlwaysBroadcast(fmt.Errorf(
				"stopping operator service: %w",
				err,
			))
		}
	}
	if nil != s.is {
		if err := s.is.Stop(defErr); nil != err {
			s.ew.AlwaysBroadcast(fmt.Errorf(
				"stopping implant service: %w",
				err,
			))
		}
	}

	/* Flush state. */
	if err := s.sm.Write(); nil != err {
		s.ew.AlwaysBroadcast(fmt.Errorf("flushing state: %w", err))
	}

	/* If we're this far, everything's stopped and nothing's gone wrong. */
	s.ew.AlwaysBroadcast(defErr)
	return s.Wait()
}

// Wait waits for a fatal error or nil on clean shutdown with Stop.
func (s *Server) Wait() error { return s.ew.Wait() }

// CloseLogfile closes the logfile.  This is used during testing.
func (s *Server) CloseLogfile() error { return s.lf.Close() }

// SL returns s's logger.
func (s *Server) SL() *slog.Logger { return s.sl }