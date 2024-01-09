package operatorsvr

/*
 * handle.go
 * Handle an operator conn
 * By J. Stuart McMurray
 * Created 20231129
 * Last Modified 20231207
 */

import (
	"errors"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/magisterquis/plonk/internal/def"
	"github.com/magisterquis/plonk/lib/estream"
	"github.com/magisterquis/plonk/lib/plog"
	"golang.org/x/sys/unix"
)

// acceptConns accepts op conns and sends them off for handling.
func (s *Server) acceptConns() {
	defer s.l.Close()
	for {
		/* Pop a new connection from the queue. */
		c, err := s.l.AcceptUnix()
		if nil == err { /* Got one. */
			s.connsWG.Add(1)
			go s.handleConn(c)
			continue
		}

		/* If someone closed the listener, we're done. */
		if errors.Is(err, net.ErrClosed) {
			return
		}

		/* If we've too many connections, wait and try again. */
		if errors.Is(err, unix.EMFILE) || errors.Is(err, unix.ENFILE) {
			plog.WarnError(
				s.SL,
				def.LMTemporaryAcceptError,
				err,
			)
			time.Sleep(def.AcceptWait)
			continue
		}

		/* A real error. */
		s.ew.AlwaysBroadcast(fmt.Errorf("accept: %w", err))
		return
	}
}

// handleConn handles a single operator conn.
func (s *Server) handleConn(c *net.UnixConn) {
	defer c.Close()
	defer s.connsWG.Done()

	/* Turn into an operator conn. */
	oc := &opConn{
		es: estream.New(c),
		sm: s.SM,
	}
	cnum := s.cNum.Add(1) /* Connection number. */
	oc.sl.Store(s.SL.With(def.LKConnNumber, cnum))
	defer oc.es.Close()

	/* Make sure we're still actually accepting conns and then register
	this one. */
	s.connsL.Lock()
	if nil == s.conns {
		s.connsL.Unlock()
		return
	}
	s.conns[oc] = struct{}{}
	defer func() {
		s.connsL.Lock()
		defer s.connsL.Unlock()
		delete(s.conns, oc)
	}()
	s.connsL.Unlock()

	/* Get the user's name and add it to logged messages. */
	var (
		ech = make(chan error, 3)
		nch = make(chan string)
	)
	estream.AddHandler(oc.es, "", func(en string, _ any) {
		/* Unexpected event. */
		ech <- fmt.Errorf("unexpected %q event", en)
	})
	estream.AddHandler(oc.es, def.ENName, func(en string, name def.EDName) {
		/* Got a name . */
		nch <- string(name)
	})
	go func() { /* Try to get the first (name) event. */
		if err := oc.es.RunOnce(); nil != err {
			ech <- err
		}
	}()
	select { /* Wait for a name or error. */
	case <-time.After(def.OpNameWait): /* Don't wait too long. */
		oc.name.Store(fmt.Sprintf("cnum-%d", cnum))
		plog.InfoError(oc.SL(), def.LMOpInitialNameError, errors.New(
			"timeout",
		), def.LKOpName, oc.name.Load())
	case err := <-ech: /* Something went wrong. */
		plog.InfoError(oc.SL(), def.LMOpInitialNameError, err)
		return
	case name := <-nch: /* Got a name. */
		oc.name.Store(name)
	}
	oc.sl.Store(oc.SL().With(def.LKOpName, &oc.name))

	/* Set up event handlers. */
	estream.AddHandler(oc.es, "", oc.defaultHandler)
	estream.AddHandler(oc.es, def.ENName, oc.nameHandler)
	estream.AddHandler(oc.es, def.ENEnqueue, oc.enqueueHandler)
	estream.AddHandler(oc.es, def.ENListSeen, oc.listSeenHandler)

	/* Send logs as well. */
	pr, pw := io.Pipe()
	defer pw.Close()
	defer pr.Close()
	s.FW.Add(pw, nil)
	go func() {
		err := oc.es.SendJSONSLogs(pr)
		go pw.CloseWithError(err)
		ech <- fmt.Errorf("sending slogs: %w", err)
	}()

	/* Handle events from the client. */
	go func() {
		err := oc.es.Run()
		go pr.CloseWithError(err)
		ech <- fmt.Errorf("handling events: %w", err)
	}()

	/* Wait for something to go wrong. */
	oc.SL().Info(def.LMOpConnected)
	if err := <-ech; nil == err ||
		errors.Is(err, io.EOF) ||
		errors.Is(err, io.ErrClosedPipe) ||
		errors.Is(err, net.ErrClosed) ||
		errors.Is(err, unix.EPIPE) {
		/* Not really an error, as such. */
		oc.SL().Info(def.LMOpDisconnected)
	} else {
		/* Shouldn't happen. */
		et := fmt.Sprintf("%T", err)
		if ue := errors.Unwrap(err); nil != ue {
			et += fmt.Sprintf(" (%T)", ue)
		}
		plog.ErrorError(
			oc.SL(), def.LMOpDisconnected, err,
			def.LKErrorType, et,
		)
	}
}
