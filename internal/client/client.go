// Package client - Interactive plonk client
package client

/*
 * client.go
 * Interactive plonk client
 * By J. Stuart McMurray
 * Created 20231130
 * Last Modified 20231208
 */

import (
	"errors"
	"io"
	"log"
	"net"
	"path/filepath"
	"sync/atomic"

	"github.com/magisterquis/mqd"
	"github.com/magisterquis/plonk/internal/def"
	"github.com/magisterquis/plonk/lib/estream"
	"github.com/magisterquis/plonk/lib/opshell"
	"github.com/magisterquis/plonk/lib/subcom"
)

// welcomeMessage welcomes the user to Plonk.
const welcomeMessage = ` ___________________________________
/        Welcome to Plonk!          \
\ Use ,help for a list of commands. /
 -----------------------------------
        \   ^__^
         \  (oo)\_______
            (__)\       )\/\
                ||----w |
                ||     ||`

// Client implements the server side of Plonk.  Before starting, its public
// fields should be populated.
type Client struct {
	Dir   string
	Debug bool
	Name  string /* Operator name. */

	/* I/O streams, which may be TTYs. */
	Stdin  io.Reader /* Default: os.Stdin. */
	Stdout io.Writer /* Default: os.Stdout. */
	Stderr io.Writer /* Default: os.Stderr. */

	shell *opshell.Shell[*Client]
	es    *estream.Stream
	id    atomic.Pointer[string] /* Current Implant ID. */
}

// Run runs the client.  It returns an exit code to return to the shell.
func (c *Client) Run() int {
	mqd.TODO("Handler for /c")
	mqd.TODO("Handler for /p")

	/* Upgrade to a friendlier shell. */
	var err error
	c.shell, err = opshell.Config[*Client]{
		Reader:      c.Stdin,
		Writer:      c.Stdout,
		ErrorWriter: c.Stderr,
		Prompt:      def.LogsPrompt,
	}.New()
	if nil != err {
		log.Printf("Error setting up shell: %s", err)
		return 1
	}
	defer c.shell.ResetTerm()
	c.shell.SetV(c)
	c.shell.SetSplitter(opshell.CutCommand)
	c.shell.SetCommandErrorHandler(commandErrorHandler)
	subcom.AddSpecs(c.shell.Cdr(), []subcom.Spec[shell]{{
		Name:        ",help",
		Description: "This help",
		Handler:     helpHandler,
	}, {
		Name:        ",quit",
		Description: "Gracefully quit",
		Handler:     quitHandler,
	}, {
		Name:        ",name",
		Description: "Change the name used for logging",
		Handler:     nameHandler,
	}, {
		Name:        ",task",
		Description: "Queue up a task for an implant",
		Handler:     enqueueHandler,
	}, {
		Name:        ",seti",
		Description: "Interact with an implant",
		Handler:     setiHandler,
	}, {
		Name:        logsCmd,
		Description: "Interact with no implant and just watch logs",
		Handler:     setiHandler,
	}, {
		Name:        ",list",
		Description: "List recently-seen implants",
		Handler:     listHandler,
	}})

	/* Connect to the server. */
	fn := filepath.Join(c.Dir, def.OpSock)
	sc, err := net.DialUnix("unix", nil, &net.UnixAddr{
		Name: fn,
		Net:  "unix",
	})
	if nil != err {
		c.shell.ErrorLogf(
			"Could not connect to the server at %s: %s",
			fn,
			err,
		)
		return 2
	}
	defer sc.Close()

	ech := make(chan error, 2)

	/* Set up to receive events from the server. */
	c.es = estream.New(sc)
	estream.AddHandler(c.es, "", c.handleUnknownEvent)
	estream.AddHandler(c.es, def.ENGoodbye, c.handleGoodbyeEvent)
	estream.AddHandler(c.es, def.ENListSeen, c.handleListSeenEvent)
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
	go func() {
		if err := c.es.Send(
			def.ENName,
			def.EDName(c.Name),
		); nil != err {
			ech <- err
		}
	}()

	/* Start handling events and commands. */
	go func() { ech <- c.es.Run() }()
	go func() { ech <- c.shell.HandleCommands() }()

	/* Wait for something to go wrong. */
	if err := <-ech; nil != err &&
		!errors.Is(err, opshell.ErrQuit) &&
		!errors.Is(err, io.EOF) {
		c.shell.SetPrompt("")
		c.shell.ErrorLogf("Fatal error: %s", err)
		return 3
	}
	if c.shell.InPTYMode() {
		c.shell.SetPrompt("")
		c.shell.ErrorLogf("Goodbye.")
	}
	return 0
}

// Debugf logs a message to the shell if c.Debug is set.
func (c *Client) Debugf(format string, args ...any) {
	if !c.Debug {
		return
	}
	c.shell.Logf(format, args...)
}

// setiOrNil returns true if id either matches the implant set with ,seti or
// we've got no implant at all.
func (c *Client) setiOrNil(id string) bool {
	idp := c.id.Load()
	if nil == idp {
		return true
	}
	return id == *idp
}
