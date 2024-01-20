// Package client - Interactive plonk client
package client

/*
 * client.go
 * Interactive plonk client
 * By J. Stuart McMurray
 * Created 20231130
 * Last Modified 20240120
 */

import (
	"errors"
	"fmt"
	"io"
	"net"
	"path/filepath"
	"sync/atomic"

	"github.com/magisterquis/plonk/internal/def"
	"github.com/magisterquis/plonk/lib/estream"
	"github.com/magisterquis/plonk/lib/logring"
	"github.com/magisterquis/plonk/lib/opshell"
	"github.com/magisterquis/plonk/lib/subcom"
	"github.com/magisterquis/plonk/lib/waiter"
)

// welcomeMessage welcomes the user to Plonk.
const welcomeMessage = ` ___________________________
/     Welcome to Plonk!     \
\ Try ,help to get started. /
 ---------------------------
        \   ^__^
         \  (oo)\_______
            (__)\       )\/\
                ||----w |
                ||     ||`

// Client implements the server side of Plonk.  Before starting, its public
// fields should be populated.
type Client struct {
	Dir      string
	Debug    bool
	Name     string /* Operator name. */
	Colorize bool   /* Output with colors. */

	/* I/O streams, which may be TTYs. */
	Stdin  io.Reader /* Default: os.Stdin. */
	Stdout io.Writer /* Default: os.Stdout. */
	Stderr io.Writer /* Default: os.Stderr. */

	shell *opshell.Shell[*Client]
	es    *estream.Stream
	lr    *logring.Ring
	id    atomic.Pointer[string]       /* Current Implant ID. */
	reset atomic.Pointer[func() error] /* Reset terminal. */
	sc    atomic.Pointer[net.UnixConn] /* Connection to Server. */
	ew    waiter.Waiter[error]
}

// Start starts the client.  Call Wait to wait for it to die, reset the
// terminal's saved state, and close the connection to the server.  There is no
// Stop method; close the input instead.
func (c *Client) Start() error {
	var (
		ech = make(chan error, 2)
		rt  func() error
	)

	/* Set up useful things. */
	c.lr = logring.New(def.NReplayLogs)

	/* Connect to the server. */
	fn := filepath.Join(c.Dir, def.OpSock)
	sc, err := net.DialUnix("unix", nil, &net.UnixAddr{
		Name: fn,
		Net:  "unix",
	})
	if nil != err {
		return fmt.Errorf(
			"connecting to the server at %s: %w",
			fn,
			err,
		)
	}
	c.sc.Store(sc)

	/* Upgrade to a friendlier shell. */
	c.shell, err = opshell.Config[*Client]{
		Reader:      c.Stdin,
		Writer:      c.Stdout,
		ErrorWriter: c.Stderr,
		Prompt:      def.LogsPrompt,
	}.New()
	if nil != err {
		err = fmt.Errorf("setting up shell: %w", err)
		goto fail
	}
	rt = c.shell.ResetTerm
	c.reset.Store(&rt)
	c.shell.SetV(c)
	c.shell.SetSplitter(opshell.CutCommand)
	c.shell.SetCommandErrorHandler(commandErrorHandler)
	subcom.AddSpecs(c.shell.Cdr(), []subcom.Spec[shell]{{
		Name:        ",help",
		ArgHelp:     "[topic]",
		Description: "This help; try \",help topics\"",
		Handler:     helpHandler,
	}, {
		Name:        ",quit",
		Description: "Gracefully quit",
		Handler:     quitHandler,
	}, {
		Name:        ",seti",
		ArgHelp:     "<implant ID>",
		Description: "Interact with an implant",
		Handler:     setiHandler,
	}, {
		Name: logsCmd,
		Description: "Interact with no implant and " +
			"just watch Plonk's logs",
		Handler: setiHandler,
	}, {
		Name:        ",list",
		Description: "List recently-seen implants",
		Handler:     listHandler,
	}})

	/* Set up to receive events from the server. */
	c.es = estream.New(sc)
	estream.AddHandler(c.es, "", c.handleUnknownEvent)
	estream.AddHandler(c.es, def.ENGoodbye, c.handleGoodbyeEvent)
	estream.AddHandler(c.es, def.ENListSeen, c.handleListSeenEvent)
	estream.AddHandler(c.es, def.LMCurlGen, c.handleImplantGenEvent)
	estream.AddHandler(c.es, def.LMExfil, c.handleExfilEvent)
	estream.AddHandler(c.es, def.LMFileRequest, c.handleFileRequestEvent)
	estream.AddHandler(c.es, def.LMNewImplant, c.handleNewImplantEvent)
	estream.AddHandler(c.es, def.LMOpConnected, c.handleOpConnectedEvent)
	estream.AddHandler(c.es, def.LMOpDisconnected, c.handleOpConnectedEvent)
	estream.AddHandler(c.es, def.LMOutputRequest, c.handleOutputRequestEvent)
	estream.AddHandler(c.es, def.LMTaskQueued, c.handleTaskQueuedEvent)
	estream.AddHandler(c.es, def.LMTaskRequest, c.handleTaskRequestEvent)

	/* Welcome the user. */
	if c.shell.InPTYMode() {
		c.shell.Printf("%s\n", welcomeMessage)
	}

	/* Send our name to the server. */
	if err = c.es.Send(def.ENName, def.EDName(c.Name)); nil != err {
		goto fail
	}

	/* Start handling events and commands. */
	go func() { ech <- c.es.Run() }()
	go func() { ech <- c.shell.HandleCommands() }()

	/* Wait for something to go wrong. */
	go func() {
		if err := <-ech; nil != err &&
			!errors.Is(err, opshell.ErrQuit) &&
			!errors.Is(err, io.EOF) {
			c.shell.SetPrompt("")
			c.shell.ErrorLogf("Fatal error: %s", err)
			c.ew.AlwaysBroadcast(err)
			return
		}
		/* Wait for all handlers to be finished.  This ensures we
		always get a bye from the server if one's sent. */
		c.es.Close()
		c.es.WaitForHandlers()
		if c.shell.InPTYMode() {
			c.shell.SetPrompt("")
			c.shell.ErrorLogf("Goodbye.")
		}
		c.ew.AlwaysBroadcast(nil)
	}()
	return nil

	/* This is a C-ish defer if we have an error. */
fail:
	if nil != c.shell {
		c.shell.ErrorLogf("Error: %s", err)
	}
	if nil != sc { /* Close the unix connection. */
		sc.Close()
	}
	if nil != rt { /* Reset the terminal. */
		rt()
	}
	return err
}

// Wait waits for c to end, closes the connection to the server, and resets
// the terminal, if applicable.  It then returns whatever error caused the
// client to die.
func (c *Client) Wait() error {
	/* Wait for an error.  We'll call c.ew.AlwaysBroadcast a few more
	times, so even if this would have been nil, it may change.  If there
	was a proper error, we'll still get it later. */
	c.ew.Wait()

	/* If we have a server connection, close it. */
	if sc := c.sc.Swap(nil); nil != sc {
		c.ew.AlwaysBroadcast(sc.Close())
	}

	/* Make sure to reset the terminal. */
	if f := c.reset.Load(); nil != f {
		c.ew.AlwaysBroadcast((*f)())
	}

	return c.ew.Wait()

}

// Debugf logs a message to the shell if c.Debug is set.
func (c *Client) Debugf(format string, args ...any) {
	if !c.Debug {
		return
	}
	c.shell.Logf(format, args...)
}

// color adds the color to s, as appropriate for c's shell, if c.Colorize is
// set.
func (c *Client) color(oc opshell.Color, s string) string {
	/* Don't bother if we're not coloring things. */
	if !c.Colorize {
		return s
	}
	return c.shell.Color(oc, s)

}

// noIDLogf logs with c.shell.Logf if we're not watching any implant, or
// stores it in c.lr for the next time we get a ,logs.
func (c *Client) noIDLogf(format string, v ...any) {
	if nil == c.id.Load() {
		c.shell.Logf(format, v...)
	} else {
		c.lr.Printf(format, v...)
	}
}

// idLogger is a super-kooky way to choose whether to log to c.shell or c.lr
// based on id.  It is like noIDLogf but chooses based on whether or not
// the user has called ,seti for the given ID.  We used a type like this to
// be able to pass an additional value to what would normally be something
// like c.idLogf(id, f, args...) without an id parameter so as to enable
// go vet checks.  Bleh.
type idLogger struct {
	c  *Client
	id string
}

// logf logs to il.c.shell if il.c's id is il.id.  Otherwise it logs to
// il.c.lr.
func (il idLogger) logf(format string, v ...any) {
	/* If we have this ID selected, send the message to the client. */
	if idp := il.c.id.Load(); nil == idp || il.id == *idp {
		il.c.shell.Logf(format, v...)
	} else { /* Save the message for later. */
		il.c.lr.Printf(format, v...)
	}
}
