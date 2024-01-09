// Package implantsvr - Listen for and handle implant requests
package implantsvr

/*
 * implantsvr.go
 * Listen for and handle implant requests
 * By J. Stuart McMurray
 * Created 20231207
 * Last Modified 20231208
 */

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/magisterquis/mqd"
	"github.com/magisterquis/plonk/internal/def"
	"github.com/magisterquis/plonk/internal/server/state"
	"github.com/magisterquis/plonk/lib/jpersist"
	"github.com/magisterquis/plonk/lib/waiter"
)

// Server listens for and handles implant queries.  Populate its public fields
// and call Start to start it.
type Server struct {
	Dir       string
	SL        *slog.Logger
	HTTPAddr  string /* Listen address. */
	HTTPSAddr string /* Listen address. */
	SM        *jpersist.Manager[state.State]
	NoExfil   bool

	httpAddr string /* Actual address, for testing. */

	ew  waiter.Waiter[error]
	svr *http.Server
	fh  http.Handler /* Static files handler. */

	noSeen bool     /* For testing. */
	seen   sync.Map /* Seen implants. */
}

// Start starts the server serving.
func (s *Server) Start() error {
	/* Make sure we have at least one listen address. */
	if "" == s.HTTPAddr && "" == s.HTTPSAddr {
		return fmt.Errorf("need a listen address")
	}

	/* Start listners. */
	var httpl, httpsl net.Listener
	var attrs = make([]any, 0, 4)
	if "" != s.HTTPAddr {
		var err error
		httpl, err = net.Listen("tcp", s.HTTPAddr)
		if nil != err {
			return fmt.Errorf(
				"listening on %s: %w",
				s.HTTPAddr,
				err,
			)
		}
		s.httpAddr = httpl.Addr().String()
		attrs = append(attrs, def.LKHTTPAddr, httpl.Addr().String())
	}
	mqd.TODO("Start HTTPS listener")

	/* Set up the static files handler, which will be called by
	handleStaticFiles. */
	sd := filepath.Join(s.Dir, def.StaticFilesDir)
	if err := os.MkdirAll(sd, def.DirPerms); nil != err {
		err := fmt.Errorf(
			"making static files directory %s: %w",
			sd,
			err,
		)
		return s.Stop(err)
	}
	s.fh = http.StripPrefix(def.FilePath, http.FileServer(http.Dir(sd)))

	/* Set up HTTP handler. */
	s.svr = &http.Server{
		Handler:           s.serveMux(),
		ReadHeaderTimeout: def.HTTPIOTimeout,
		WriteTimeout:      def.HTTPIOTimeout,
		IdleTimeout:       def.HTTPIOTimeout,
		ErrorLog:          httpErrorLogger(s.SL),
	}

	/* Start service on whatever listeners are listening. */
	for _, l := range []net.Listener{httpl, httpsl} {
		/* Don't bother if this isn't listening. */
		if nil == l {
			continue
		}
		/* Start the server going. */
		go func(ll net.Listener) {
			if err := s.svr.Serve(ll); nil != err &&
				!errors.Is(err, http.ErrServerClosed) {
				s.ew.AlwaysBroadcast(fmt.Errorf(
					"http service died on %s: %w",
					ll.Addr(),
					err,
				))
			}
		}(l)
	}
	s.SL.Debug(def.LMImplantServing, attrs...)

	return nil
}

// Stop stops the server and returns the first error encountered while doing
// so, or should all go well, defError.  After Stop is called, Wait will
// return a non-nil error.
func (s *Server) Stop(defError error) error {
	if nil != s.svr {
		if err := s.svr.Shutdown(context.Background()); nil != err {
			s.ew.AlwaysBroadcast(err)
			return err
		}
	}
	return nil
}

// Wait waits for the server to stop.  It returns nil if it stopped nicely.
func (s *Server) Wait() error { return s.ew.Wait() }
