package client

/*
 * commands.go
 * Command handlers
 * By J. Stuart McMurray
 * Created 20231206
 * Last Modified 20240120
 */

import (
	"errors"
	"fmt"
	"strings"

	"github.com/magisterquis/plonk/internal/def"
	"github.com/magisterquis/plonk/lib/opshell"
	"github.com/magisterquis/plonk/lib/subcom"
)

// logsCmd requests to watch logs.
const logsCmd = ",l"

// shell is the type of our shell, used in command handlers.
type shell = *opshell.Shell[*Client]

// quitHandler gently quits.
func quitHandler(s shell, name, args []string) error { return opshell.ErrQuit }

// setiHandler sets the Implant ID.  If it's called with no ID, it will send a
// request for a list of implants.
func setiHandler(s shell, name, args []string) error {
	/* If we have no ID, just ask for a list of implants. */
	if 0 == len(args) {
		/* Send a request for a list. */
		if err := s.V().es.Send(def.ENListSeen, nil); nil != err {
			return fmt.Errorf("sending event: %s", err)
		}
		return nil
	}

	/* Save the implant ID. */
	id := strings.Join(args, "")
	if "" == id {
		s.ErrorLogf("Need an ID, please")
		return nil
	}
	s.V().id.Store(&id)

	/* Tell the user. */
	s.SetPrompt(id + opshell.DefaultPrompt)
	s.Logf("Interacting with %s", id)
	s.Logf("Use %s to return to watching Plonk's logs", logsCmd)
	return nil
}

// logsHandler goes to streaming logs.  It also prints the previously-missed
// log entries.
func logsHandler(s shell, name, args []string) error {
	/* If we're already watching logs, the user probably goofed. */
	if nil == s.V().id.Load() {
		s.Logf("Already watching Plonk's logs")
		return nil
	}

	/* Note we're no longer watching an implant. */
	s.V().id.Store(nil)
	s.SetPrompt(def.LogsPrompt)

	/* Give the user the missed logs, if any. */
	lls := s.V().lr.MessagesAndClear()
	msg := "\t" + strings.Replace(
		strings.Join(lls, "\n"),
		"\n",
		"\n\t",
		-1,
	)
	switch {
	case 0 == len(lls): /* Easy day. */
	case len(lls) == s.V().lr.Cap(): /* Missed lotsa messages. */
		s.Logf(
			"Last %d missed log messages:\n%s",
			len(lls),
			msg,
		)
	default: /* Got all the messages we missed. */
		s.Logf(
			"Missed %d log messages:\n%s",
			len(lls),
			msg,
		)
	}

	/* Tell him we're watching logs. */
	s.Logf("Watching plonk's own logs")
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
	if nil == idp || "" == *idp {
		s.Logf("I've not heard of that one, sorry.  Need ,seti?")
		return nil
	}

	/* We do have an implant, then.  We'll send it along. */
	if err := s.V().es.Send(def.ENEnqueue, def.EDEnqueue{
		ID:   *idp,
		Task: line,
	}); nil != err {
		return fmt.Errorf("sending enqueue event: %w", err)
	}
	return nil
}
