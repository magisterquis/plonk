package main

/*
 * interact.go
 * Interact with an implant
 * By J. Stuart McMurray
 * Created 20230224
 * Last Modified 20230228
 */

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

var (
	/* lineRE grabs the bits of lines which are important to interactive
	ops. */
	lineRE = regexp.MustCompile(`^` +
		`(\d{4}/\d\d/\d\d \d\d:\d\d:\d\d) ` + /* Time and date. */
		`\[(` +
		strings.Join([]string{
			MessageTypeCallback,
			MessageTypeOutput,
			MessageTypeTaskQ,
			MessageTypeError,
		}, "|") +
		`)\] ` + /* Message type. */
		`(.*)`, /* Message body. */
	)

	/* elogf logs just to stderr. */
	elogf = log.New(os.Stderr, "", log.LstdFlags).Printf

	/* errNoPayload is returned from getJSONFromRLog if the message didn't
	actually have a payload. */
	errNoPayload = errors.New("no payload")

	/* errWrongID is returned by unmarshalRLog if the log message is for
	a different ID than the one specified. */
	errWrongID = errors.New("wrong ID")

	/* errReady is a pseudoerror indicating the output-watcher's ready. */
	errReady = errors.New("ready")

	/* taskRE gets the interesting parts of a task message, after the
	ID-specific prefix is removed. */
	taskQRE = regexp.MustCompile(`\(qlen (\d+)\): (.*)`)
)

// Interact interacts with the named implant.  - will be translated to "".
// Non-empty, non #-leading lines read on stdin will be queued as tasking.
// Output will be sent to stdout.  This won't work very well if there's no
// logfile.
func Interact(id, logfile string) error {
	/* - is really "" */
	if "-" == id {
		id = ""
	}

	/* Start both input and output going, and wait for something to die. */
	ech := make(chan error)
	start := make(chan struct{})
	go interactiveTasking(id, ech)
	go watchOutput(id, logfile, ech, start)

	/* Wait for the output-watcher to be ready and welcome the user. */
	if err := <-ech; !errors.Is(err, errReady) {
		return err
	}
	if "" == id {
		log.Printf(
			"Welcome.  Going interactive with the IDless implant.",
		)
	} else {
		log.Printf("Welcome.  Going interactive with %s.", id)
	}
	close(start)

	return <-ech
}

// interactiveTasking reads tasks from stdin and queues them for id.
func interactiveTasking(id string, ech chan<- error) {
	/* Get tasking from stdin. */
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		/* Grb a line of tasking. */
		l := strings.TrimSpace(scanner.Text())
		if "" == l || strings.HasPrefix(l, "#") {
			continue
		}
		/* Queue it. */
		if err := AddTask(id, l, true); nil != err {
			ech <- fmt.Errorf("queuing task %q: %w", l, err)
			return
		}
	}
	if err := scanner.Err(); nil != err {
		ech <- fmt.Errorf("reading tasking: %w", err)
	}

	/* Just a normal EOF. */
	ech <- nil
}

// watchOutput watches the logfile for output and sends it to stdout.
func watchOutput(id, logfile string, ech chan<- error, start <-chan struct{}) {
	/* Tail the logfile.  It'd be nice if there were a native Go way to do
	this. */
	tail := exec.Command("tail", "-f", logfile)
	tailo, err := tail.StdoutPipe()
	tail.Stderr = os.Stderr /* For just in case. */
	if nil != err {
		ech <- fmt.Errorf("getting tail's stdout: %w", err)
		return
	}
	if err := tail.Start(); nil != err {
		ech <- fmt.Errorf("starting tail: %w", err)
		return
	}

	/* Let our caller know we're ready and wait for him to welcome the
	user. */
	ech <- errReady
	<-start

	/* Watch for output and callback lines. */
	var (
		/* Prefix for task queue messages. */
		taskQPrefix = []byte(fmt.Sprintf(
			"%s %q",
			TaskMessagePrefix,
			id,
		))
		reader = bufio.NewReader(tailo)
		buf    bytes.Buffer
	)
	for {
		/* Get a full line.  These get big. */
		buf.Reset()
		for {
			l, prefix, err := reader.ReadLine()
			if nil != err {
				ech <- fmt.Errorf("tailing logfile: %w", err)
				return
			}
			buf.Write(l)
			if !prefix {
				break
			}

		}

		/* Print the output nicely. */
		watchOutputLine(id, taskQPrefix, buf.Bytes())
	}
}

// watchOutputLine processes a single log line.
func watchOutputLine(id string, taskQPrefix, line []byte) {
	/* Get the important bits. */
	ms := lineRE.FindSubmatch(line)
	if 4 != len(ms) {
		return
	}

	/* Format each message nicely. */
	var (
		msg string
		ok  bool
	)
	switch mt := string(ms[2]); mt {
	case MessageTypeCallback:
		ok, msg = parseCallbackLog(id, ms[3])
	case MessageTypeOutput:
		ok, msg = parseOutputLog(id, ms[3])
	case MessageTypeTaskQ:
		/* Don't care if it's not us. */
		if !bytes.HasPrefix(ms[3], taskQPrefix) {
			return
		}
		ok = true
		msg = parseTaskQLog(bytes.TrimPrefix(ms[3], taskQPrefix))
	case MessageTypeError:
		/* Already sufficiently nicely formatted. */
		fmt.Printf("%s\n", line)
		return
	default:
		elogf(
			"[%s] Unexpected message type %q in %q",
			MessageTypeError,
			mt,
			line,
		)
		return
	}

	/* If the message wasn't us, we're done. */
	if !ok {
		return
	}

	/* If we're not printing all the things and there's nothing to
	print, we're also done. */
	if "" == msg && !VerbOn {
		return
	}

	/* Add back the timestamp and type. */
	fmt.Printf("%s [%s] %s\n", ms[1], ms[2], strings.TrimRight(msg, "\n"))
}

// unmarshalRLog unmarshals the message from an RLog'd message.  Only the part
// of the log line after the [MessageType] should be passed in msg.  If, after
// unmarshaling, v is as struct with an ID field which has a value other than
// id, unmarshalRLog returns errWrongID.
func unmarshalRLog(id string, v any, msg []byte) error {
	/* Get the JSON bit. */
	parts := bytes.SplitN(msg, []byte{' '}, 5)
	if 5 != len(parts) {
		return errNoPayload
	}

	/* Do the unmarshaling. */
	if err := json.Unmarshal(parts[4], v); nil != err {
		return fmt.Errorf("un-JSONing: %w", err)
	}

	/* Check and see if we have the right ID. */
	rv := reflect.Indirect(reflect.ValueOf(v))
	if reflect.Struct != rv.Kind() {
		return nil
	}
	f := rv.FieldByName("ID")
	if !f.IsValid() {
		return nil
	}
	if id != f.String() {
		return errWrongID
	}

	return nil
}

// parseCallbackLog turns a callback log message into something human-friendly.
// It returns false if the log isn't for the right id.
func parseCallbackLog(id string, payload []byte) (ok bool, msg string) {
	/* Un-JSON. */
	var tl TaskLog
	if err := unmarshalRLog(id, &tl, payload); nil != err {
		if errors.Is(err, errWrongID) {
			return false, ""
		}
		return true, fmt.Sprintf("error parsing %q: %s", payload, err)
	}

	/* Turn this into something human-readable. */
	switch {
	case "" == tl.Task && "" == tl.Err:
		return true, ""
	case "" == tl.Task && "" != tl.Err:
		return true, fmt.Sprintf("Error sending tasking: %s", tl.Err)
	case "" != tl.Task && "" == tl.Err:
		return true, fmt.Sprintf("Sent task %q", tl.Task)
	case "" != tl.Task && "" != tl.Err:
		return true, fmt.Sprintf(
			"Sent task %q but encountered error: %s",
			tl.Task,
			tl.Err,
		)
	default:
		/* Unpossible */
		return true, "BUG: failed to work out tasking message"
	}
}

// parseOutputLog turns an output log message into something human-friendly.
// It returns false if the log isn't for the right id.
func parseOutputLog(id string, payload []byte) (ok bool, msg string) {
	/* Un-JSON. */
	var ol OutputLog
	if err := unmarshalRLog(id, &ol, payload); nil != err {
		if errors.Is(err, errWrongID) {
			return false, ""
		}
		return true, fmt.Sprintf("Error parsing %q: %s", payload, err)
	}

	/* Turn this into something human-readable. */
	var sb strings.Builder
	if "" != ol.Err {
		fmt.Fprintf(&sb, "Error: %s\n", ol.Err)
	}
	if "" != ol.Output {
		sb.WriteString("\n")
		sb.WriteString(ol.Output)
	}

	return true, sb.String()
}

// parseTaskQLog turns a task queue log message into something human-friendly.
func parseTaskQLog(payload []byte) string {
	/* Extract the important bits. */
	ms := taskQRE.FindSubmatch(payload)
	if 3 != len(ms) {
		return fmt.Sprintf("Malformed payload %q", payload)
	}

	/* Unquote the tasking, for user niceity. */
	t, err := strconv.Unquote(string(ms[2]))
	if nil != err {
		return fmt.Sprintf(
			"Error unquoting tasking %q: %s",
			ms[2],
			err,
		)
	}

	return fmt.Sprintf("Added task (queue length %s):\n%s", ms[1], t)
}
