package client

/*
 * commands.go
 * Command handlers
 * By J. Stuart McMurray
 * Created 20231206
 * Last Modified 20231208
 */

import (
	"errors"
	"fmt"
	"strings"

	"github.com/magisterquis/plonk/internal/def"
	"github.com/magisterquis/plonk/lib/opshell"
	"github.com/magisterquis/plonk/lib/subcom"
)

const (
	// logsCmd requests to watch logs.
	logsCmd = ",logs"
)

// shell is the type of our shell, used in command handlers.
type shell = *opshell.Shell[*Client]

// helpHandler lists the available commands.
func helpHandler(s shell, name, args []string) error {
	s.Printf("Commands:\n\n%s\n", s.Cdr().Table())
	s.Printf(
		"\nTo get started, use ,list to see what implants have " +
			"called back and then ,seti\nto interact with one.\n",
	)
	return nil
}

// quitHandler gently quits.
func quitHandler(s shell, name, args []string) error { return opshell.ErrQuit }

// nameHandler asks the server to call us something else.
func nameHandler(s shell, name, args []string) error {
	/* Make sure we have a name. */
	if 0 == len(args) || "" == args[0] {
		s.ErrorLogf(
			"Plese also supply the name by which " +
				"you wish to be known.",
		)
		return nil
	}
	/* Send it to the server. */
	if err := s.V().es.Send(def.ENName, args[0]); nil != err {
		return fmt.Errorf("sending event: %w", err)
	}
	return nil
}

// enqueueHandler asks the server to queue up a task for an implant.
func enqueueHandler(s shell, name, args []string) error {
	/* Make sure we have an implant. */
	id := s.V().id.Load()
	if nil == id {
		s.ErrorLogf("Please set an implant ID first with ,seti")
		return nil
	}

	/* Make sure we have a task. */
	if 0 == len(args) || "" == args[0] {
		s.ErrorLogf("Need a task to send, please")
		return nil
	}

	/* Send a message to the server. */
	if err := s.V().es.Send(def.ENEnqueue, def.EDEnqueue{
		ID:   *id,
		Task: args[0],
	}); nil != err {
		return fmt.Errorf("sending event: %w", err)
	}
	return nil
}

// setiHandler sets the Implant ID.  If it's called as ,logs, it sets us back
// to just watching logs.
func setiHandler(s shell, name, args []string) error {
	/* If we're watching logs, go back to no implant ID. */
	if 0 != len(name) && logsCmd == name[len(name)-1] {
		s.V().id.Store(nil)
		s.SetPrompt(def.LogsPrompt)
		s.Logf("Watching server logs")
		return nil
	}

	/* Save the implant ID. */
	var id string
	if 0 != len(args) {
		id = args[0]
	}
	s.V().id.Store(&id)

	/* Tell the user. */
	s.SetPrompt(id + opshell.DefaultPrompt)
	switch id {
	case "":
		s.Logf("Interacting with the IDless implant")
	default:
		s.Logf("Interacting with implant %s", id)
	}
	return nil
}

// listHandler requests a list of recently-seen Implants.
func listHandler(s shell, name, args []string) error {
	/* Send a request for a list. */
	if err := s.V().es.Send(def.ENListSeen, nil); nil != err {
		return fmt.Errorf("sending event: %s", err)
	}
	return nil
}

// commandErrorHandler handles errors encountered during command processing.
func commandErrorHandler(s shell, line string, err error) error {
	var se opshell.SplitError
	/* Common errors. */
	if errors.Is(err, subcom.ErrNotFound) {
		return commandNotFoundHandler(s, line, err)
	} else if errors.As(err, &se) {
		s.ErrorLogf("BUG: Error splitting command: %s", err)
		return nil
	}

	s.ErrorLogf("Unexpected command error: %s", err)
	return nil
}

// commandNotFoundHandler is called when a command isn't known.  If we've got
// an implant teed up, we'll send it unless it starts with a comma.  Otherwise
// we'll just complain to the user.
func commandNotFoundHandler(s shell, line string, err error) error {
	idp := s.V().id.Load()
	/* If we don't have an implant teed up, probably a typo. */
	if nil == idp {
		s.Logf("I've not heard of that one, sorry.  Need ,seti?")
		return nil
	}

	/* If it didn't start with a comma, it's probably for the implant. */
	if !strings.HasPrefix(strings.TrimSpace(line), ",") {
		return enqueueHandler(s, nil, []string{line})

	}

	/* If it started with a comma, probably a typo. */
	idn := iName(*idp)
	s.Logf(
		"Looks like a typo.  "+
			"If it was meant for %s, please use ,task.",
		idn,
	)
	return nil
}
