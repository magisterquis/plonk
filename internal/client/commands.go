package client

/*
 * commands.go
 * Command handlers
 * By J. Stuart McMurray
 * Created 20231206
 * Last Modified 20240123
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
	logsCmd = ",l"
	// lastPsID is a pseudo-ID which ,i's the last-seen ID.
	lastPsID = "-last"
	// nextPsID is a pseudo-ID which ,i's the next-seen ID.
	nextPsID = "-next"
)

// shell is the type of our shell, used in command handlers.
type shell = *opshell.Shell[*Client]

// quitHandler gently quits.
func quitHandler(s shell, name, args []string) error { return opshell.ErrQuit }

// requestImplantList requests an implant list be sent our way.
func requestImplantList(s shell) error {
	if err := s.V().es.Send(def.ENListSeen, nil); nil != err {
		return fmt.Errorf("sending event: %s", err)
	}
	return nil
}

// setiHandler sets the Implant ID.  If it's called with no ID, it will send a
// request for a list of implants.
func setiHandler(s shell, name, args []string) error {
	/* If we have no ID, just ask for a list of implants. */
	if 0 == len(args) {
		return requestImplantList(s)
	}

	/* Save the implant ID. */
	id := strings.Join(args, "")
	if "" == id {
		s.ErrorLogf("Need an ID, please")
		return nil
	}

	/* setID sets the implant ID. */
	setID := func(nid string) {
		/* Set the ID as the current implant ID. */
		s.V().id.Store(&nid)

		/* Tell the user. */
		s.SetPrompt(nid + opshell.DefaultPrompt)
		s.Logf("Use %s to return to watching Plonk's logs", logsCmd)
	}

	/* Pseudo-IDs get special handling. */
	switch id {
	case lastPsID: /* Use the last implant we've seen. */
		var f func(eds def.EDSeen)
		f = func(eds def.EDSeen) {
			/* If we haven't seen any implants, not much we can
			do. */
			if 0 == len(eds) || "" == eds[0].ID ||
				eds[0].When.IsZero() {
				s.ErrorLogf("Server hasn't seen any implants")
				return
			}
			s.Logf(
				"Interacting with most recent implant %s",
				eds[0].ID,
			)
			setID(eds[0].ID)
		}
		if err := requestImplantList(s); nil != err {
			return fmt.Errorf("requesting implant list: %w", err)
		}
		s.V().psidList.Store(&f)
		return nil
	case nextPsID: /* Use the next implant which the server sees. */
		var f func(ni def.EDLMNewImplant)
		f = func(ni def.EDLMNewImplant) {
			s.Logf("Interacting with new implant %s", ni.ID)
			setID(ni.ID)
		}
		s.V().psidNew.Store(&f)
		return nil
	}

	/* Prevent calling other ID-setting functions.  We could, in theory,
	lose a race, but the user will be told which ID is set so it won't be
	that bad.  This code may need to be made less racy eventually. */
	s.V().psidList.Store(nil)
	s.V().psidNew.Store(nil)

	/* Finally, set the implant ID itself. */
	s.Logf("Interacting with %s", id)
	setID(id)

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
		s.Logf("I've not heard of that one, sorry.  Need ,i?")
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
