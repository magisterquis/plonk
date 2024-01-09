package client

/*
 * events.go
 * Respond to events sent by the server.
 * By J. Stuart McMurray
 * Created 20231130
 * Last Modified 20231218
 */

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"path"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/magisterquis/plonk/internal/def"
)

// handleUnknownEvent is called when the server sends an event we're not
// expecting.  In debug mode, the event is logged.
func (c *Client) handleUnknownEvent(name string, data json.RawMessage) {
	/* Only bother if we're in debug mode. */
	if !c.Debug {
		return
	}

	/* Format the payload all nice and pretty. */
	var b bytes.Buffer
	if err := json.Indent(&b, data, "", "\t"); nil != err {
		c.shell.ErrorLogf("Error indenting %q message: %s", name, err)
		return
	}

	/* Log the message. */
	c.Debugf("Unhandled %q event:\n%s", name, b.String())
}

// handleGoodbyeEvent is called when the server lets us know it's shutting
// down.
func (c *Client) handleGoodbyeEvent(name string, data def.EDGoodbye) {
	/* If we don't actually have a message, we kinda fudge it, kindly. */
	if "" == data.Message {
		c.shell.ErrorLogf("Server said to say Goodbye.")
	} else {
		c.shell.ErrorLogf(
			"Server sent a valediction:\n\n%s\n\n",
			data.Message,
		)
	}
}

// handleOpConnectedEvent is called whenever the server says someone new's
// connected.
func (c *Client) handleOpConnectedEvent(name string, data def.EDLMOpConnected) {
	/* Work out what happened. */
	act := "connected or disconnected"
	switch name {
	case def.LMOpConnected:
		act = "Connected:"
	case def.LMOpDisconnected:
		act = "Disconnected:"
	}
	/* Log it. */
	c.shell.Logf(
		"[OPERATOR] %s %s (cnum:%d)",
		act,
		data.OpName,
		data.CNum,
	)
}

// handleTaskQueuedEvent is called whenever the server says someone's queued a
// task for an implant.
func (c *Client) handleTaskQueuedEvent(name string, data def.EDLMTaskQueued) {
	/* Log the queued task. */
	idLogger{c: c, id: data.ID}.logf(
		"[TASKQ] Task queued by %s for %s (qlen %d)\n%s",
		data.OpName,
		data.ID,
		data.QLen,
		data.Task,
	)
}

// handleTaskRequestEvent is called whenever the server says an implant called
// for tasking.
func (c *Client) handleTaskRequestEvent(name string, data def.EDLMTaskRequest) {
	/* Friendly name for logging. */
	idn := iName(data.ID)

	/* Log message buffer. */
	var sb strings.Builder
	sb.WriteString("[CALLBACK] ")
	defer func() {
		if 0 != sb.Len() {
			idLogger{c: c, id: data.ID}.logf("%s", sb.String())
		}
	}()

	/* If we have an error, we always care. */
	if "" != data.Error {
		sb.WriteString("Error sending task ")
		if t := data.Task; "" != t {
			fmt.Fprintf(&sb, "%q ", t)
		}
		fmt.Fprintf(&sb, "to %s: %s", idn, data.Error)
		return
	}

	/* If we sent a task, we always care as well. */
	if "" != data.Task {
		fmt.Fprintf(
			&sb,
			"Sent task to %s (qlen %d):\n%s",
			idn,
			data.QLen,
			data.Task,
		)
		return
	}

	/* We didn't send a task.  We only care if we're in debug mode. */
	if c.Debug {
		sb.WriteString(idn)
		return
	}

	/* Not in debug mode.  Clear the buffer to prevent a weird message. */
	sb.Reset()
}

// handleOutputRequestEvent is called when the server tells us an implant has
// sent back output.
func (c *Client) handleOutputRequestEvent(name string, data def.EDLMOutputRequest) {
	/* If we didn't get any output or error and we're not debugging,
	nothing to do. */
	if "" == data.Output && "" == data.Error && !c.Debug {
		return
	}

	/* Roll a log message. */
	var sb strings.Builder
	fmt.Fprintf(&sb, "[OUTPUT] From %s", iName(data.ID)) /* Implant name. */
	if "" != data.Error {                                /* Error, maybe. */
		fmt.Fprintf(&sb, " (error: %s)", data.Error)
	}
	switch o := data.Output; o { /* Output or placeholder. */
	case "": /* No output. */
		sb.WriteString(" (empty)")
	default: /* Output. */
		sb.WriteString("\n")
		sb.WriteString(o)
	}

	/* Send it back to the user. */
	idLogger{c: c, id: data.ID}.logf("%s", sb.String())
}

// handleListSeenEvent is called when we get an implant seen list from the
// server.
func (c *Client) handleListSeenEvent(name string, data def.EDSeen) {
	c.handleListSeenEventAt(name, data, time.Now())
}

// handleListSeenEventAt prints the data, but uses the specified time for
// calculating durations, for testing.
func (c *Client) handleListSeenEventAt(
	name string,
	data def.EDSeen,
	when time.Time,
) {
	/* We'll buffer the output to minimise interleaving. */
	var b bytes.Buffer

	/* Make a nice table. */
	tw := tabwriter.NewWriter(&b, 2, 8, 2, ' ', 0)
	fmt.Fprintf(tw, "ID\tFrom\tLast Seen\n")
	fmt.Fprintf(tw, "--\t----\t---------\n")
	for _, v := range data {
		/* Don't print empty slots. */
		if v.When.IsZero() {
			continue
		}
		s := when.Sub(v.When).Round(time.Millisecond)
		if 10*time.Second <= s {
			s = s.Round(time.Second)
		}

		fmt.Fprintf(
			tw,
			"%s\t%s\t%s (%s)\n",
			v.ID,
			v.From,
			v.When.Format(time.RFC3339),
			s,
		)
	}
	tw.Flush()

	/* Send it back to the user. */
	c.shell.Write(b.Bytes())
}

// handleNewImplantEvent is called when the server tells us it's seen a new
// Implant ID.
func (c *Client) handleNewImplantEvent(name string, data def.EDLMNewImplant) {
	idLogger{c: c, id: data.ID}.logf("[NEW] %s", iName(data.ID))
}

// handleFileRequestEvent is called when the server tells us someone's asked for
// a static file.
func (c *Client) handleFileRequestEvent(name string, data def.EDLMFileRequest) {
	/* Only log non-200's in debug mode. */
	if http.StatusOK != data.StatusCode && !c.Debug {
		return
	}

	/* What happened? */
	var sb strings.Builder
	fmt.Fprintf(&sb, "[FILE] %s %s", data.RemoteAddr, data.Filename)
	if 0 != data.Size {
		fmt.Fprintf(&sb, " %d bytes", data.Size)
	}
	if 200 != data.StatusCode {
		fmt.Fprintf(
			&sb,
			" (%d %s)",
			data.StatusCode,
			http.StatusText(data.StatusCode),
		)
	}

	c.noIDLogf("%s", sb.String())
}

// handleImplantGenEvent is called when the server tells us someone's
// generated an Implant script.
func (c *Client) handleImplantGenEvent(name string, data def.EDLMCurlGen) {
	c.noIDLogf(
		"[IMPLANTGEN] Told %s to call back to %s (random: %s)",
		data.RemoteAddr,
		data.Parameters.URL,
		data.Parameters.RandN,
	)
}

// handleExfilEvent is called when we get exfil.
func (c *Client) handleExfilEvent(name string, data def.EDLMExfil) {
	/* Work out the first element of the requested path, for
	,seti-friendliness. */
	var (
		id = "/"
		rp = data.RequestedPath
	)
	for "/" == id && "" != rp {
		id, rp = path.Split(rp)
		id = strings.Trim(id, "/")
	}
	if "" == id {
		id = rp
	}

	/* Tell the user what happened. */
	if "" != data.Error {
		idLogger{c: c, id: id}.logf(
			"[EXFIL] Error opening %s for exfil from %s: %s",
			data.Filename,
			data.RemoteAddr,
			data.Error,
		)
	} else {
		idLogger{c: c, id: id}.logf(
			"[EXFIL] Wrote %d bytes from %s to %s",
			data.Size,
			data.RemoteAddr,
			data.Filename,
		)
	}
}

// iName returns def.NamelessName if id is the empty string, or id itself if
// not.
func iName(id string) string {
	if "" == id {
		return def.NamelessName
	}
	return id
}