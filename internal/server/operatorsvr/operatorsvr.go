// Package operatorsvr - Listen for and handle operator connections
package operatorsvr

/*
 * operatorsvr.go
 * Listen for and handle operator connections
 * By J. Stuart McMurray
 * Created 20231129
 * Last Modified 20231129
 */

import (
	"fmt"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"

	"github.com/magisterquis/plonk/internal/def"
	"github.com/magisterquis/plonk/internal/server/state"
	"github.com/magisterquis/plonk/lib/flexiwriter"
	"github.com/magisterquis/plonk/lib/jpersist"
	"github.com/magisterquis/plonk/lib/waiter"
)

// Server listens for and handles operators.  Populate its public fields and
// call Start to start it.
type Server struct {
	Dir string
	SL  *slog.Logger
	FW  *flexiwriter.Writer
	SM  *jpersist.Manager[state.State]

	cNum    atomic.Uint64
	l       *net.UnixListener
	ew      waiter.Waiter[error]
	conns   map[*opConn]struct{} /* Nil to add no more. */
	connsL  sync.Mutex
	connsWG sync.WaitGroup
}

// Start starts the server servig.
func (s *Server) Start() error {
	/* Map for active conns. */
	s.conns = make(map[*opConn]struct{})

	/* Work out our socket path, and make sure it doesn't exist already. */
	fn := filepath.Join(s.Dir, def.OpSock)
	if err := os.RemoveAll(fn); nil != err {
		return fmt.Errorf(
			"removing existing operator socket %s: %w",
			fn,
			err,
		)
	}

	/* Start the listener itself. */
	var err error
	s.l, err = net.ListenUnix("unix", &net.UnixAddr{Name: fn, Net: "unix"})
	if nil != err {
		return fmt.Errorf("failed to listen on %s: %w", fn, err)
	}
	s.SL.Debug(def.LMOpListening, def.LKAddress, s.l.Addr().String())

	/* Acept operator connections. */
	go s.acceptConns()
	return nil
}

// Stop stops the server.  It returns nil if it stopped nicely.  If a message
// is provided, it will be sent to connected clients.
func (s *Server) Stop(message string) error {
	/* Stop the listening socket. */
	if err := s.l.Close(); nil != err {
		s.ew.AlwaysBroadcast(fmt.Errorf("stopping listener: %w", err))
	}

	/* Close all connections. */
	s.connsL.Lock()
	for c := range s.conns {
		go c.Goodbye(message)
	}
	s.conns = nil /* Don't allow any more. */
	s.connsL.Unlock()

	/* Wait for the connections to finish. */
	s.connsWG.Wait()

	/* Set a default nil error if we haven't found one already. */
	s.ew.AlwaysBroadcast(nil)

	return s.Wait()
}

// Wait waits for the server to stop.  It returns nil if it stopped nicely.
func (s *Server) Wait() error { return s.ew.Wait() }
