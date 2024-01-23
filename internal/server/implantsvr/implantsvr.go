// Package implantsvr - Listen for and handle implant requests
package implantsvr

/*
 * implantsvr.go
 * Listen for and handle implant requests
 * By J. Stuart McMurray
 * Created 20231207
 * Last Modified 20240123
 */

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/magisterquis/plonk/internal/def"
	"github.com/magisterquis/plonk/internal/server/perms"
	"github.com/magisterquis/plonk/internal/server/state"
	"github.com/magisterquis/plonk/lib/eztls"
	"github.com/magisterquis/plonk/lib/jpersist"
	"github.com/magisterquis/plonk/lib/plog"
	"github.com/magisterquis/plonk/lib/waiter"
)

func init() {
	perms.MustSetProcessPerms()
}

// Server listens for and handles implant queries.  Populate its public fields
// and call Start to start it.  See the note on server.Config.LEDomainWhitelist
// for information about TLS certificate generation.
type Server struct {
	Dir               string
	SL                *slog.Logger
	HTTPAddr          string /* Listen address. */
	HTTPSAddr         string /* Listen address. */
	LEDomainWhitelist []string
	LEStaging         bool
	SSDomainWhitelist []string
	LEEmail           string
	SM                *jpersist.Manager[state.State]
	ExfilMax          uint64

	httpl  plog.AtomicString /* HTTP Listener, for testing. */
	httpsl plog.AtomicString /* Ditto, for HTTPS. */

	ew   waiter.Waiter[error]
	svr  *http.Server
	fh   http.Handler /* Static files handler. */
	cg   func(*tls.ClientHelloInfo) (*tls.Certificate, error)
	sdch <-chan struct{} /* Closed on shutdown. */

	noSeen bool     /* For testing. */
	seen   sync.Map /* Seen implants. */
}

// Start starts the server serving.
func (s *Server) Start() error {
	/* Make sure we have at least one listen address. */
	if "" == s.HTTPAddr && "" == s.HTTPSAddr {
		return fmt.Errorf("need a listen address")
	}

	/* Make sure the max exfil size makes sense. */
	if s.ExfilMax > math.MaxInt64 {
		return fmt.Errorf(
			"max exfil size (%d) too large, must be <= %d",
			s.ExfilMax,
			math.MaxInt64,
		)
	}

	/* Start listners. */
	var httpl, httpsl net.Listener
	var attrs = make([]any, 0, 4)
	if "" != s.HTTPSAddr {
		/* Make sure we don't not whitelist any domain. */
		if 0 == len(s.LEDomainWhitelist) &&
			0 == len(s.SSDomainWhitelist) {
			s.SSDomainWhitelist = []string{"*"}
		}
		/* Work out how to get TLS certificates. */
		var err error
		s.cg, err = eztls.Config{
			Staging:           s.LEStaging,
			Domains:           s.LEDomainWhitelist,
			SelfSignedDomains: s.SSDomainWhitelist,
			Email:             s.LEEmail,
		}.CertificateGetter()
		if nil != err {
			return fmt.Errorf(
				"getting TLS certificate-getter: %w",
				err,
			)
		}
		/* Start listening. */
		if httpsl, err = tls.Listen("tcp", s.HTTPSAddr, &tls.Config{
			GetCertificate: s.cg,
			NextProtos:     eztls.HTTPSNextProtos,
		}); nil != err {
			return fmt.Errorf(
				"listening for HTTPS on %s: %s",
				s.HTTPSAddr,
				err,
			)
		}
		s.httpsl.Store(httpsl.Addr().String())
		attrs = append(attrs, def.LKHTTPSAddr, httpsl.Addr().String())
	}
	if "" != s.HTTPAddr {
		var err error
		httpl, err = net.Listen("tcp", s.HTTPAddr)
		if nil != err {
			if nil != httpsl {
				httpsl.Close()
			}
			return fmt.Errorf(
				"listening for HTTP on %s: %w",
				s.HTTPAddr,
				err,
			)
		}
		s.httpl.Store(httpl.Addr().String())
		attrs = append(attrs, def.LKHTTPAddr, httpl.Addr().String())
	}

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
		ErrorLog: plog.ErrorLogLogger(
			def.LMHTTPError,
			s.SL,
			httpDebugLogREs...,
		),
	}

	/* Set up a channel to be closed when the server shuts down. */
	sdch := make(chan struct{})
	s.sdch = sdch
	s.svr.RegisterOnShutdown(sync.OnceFunc(func() { close(sdch) }))

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

// HTTPListenAddr returns the address on which the server's listening.  If it
// is not listening on HTTP, HTTPAddr returns "".  This is meant for testing,
// which uses passes a port of 0 in s.HTTPAddr.
func (s *Server) HTTPListenAddr() string { return s.httpl.Load() }

// HTTPSListenAddr is like HTTPListenAddr, for HTTPS.
func (s *Server) HTTPSListenAddr() string { return s.httpsl.Load() }
