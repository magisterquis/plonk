package main

/*
 * tail.go
 * Tail the logfile
 * By J. Stuart McMurray
 * Created 20230423
 * Last Modified 20230429
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
	"os/signal"
	"strconv"
	"sync/atomic"
	"syscall"

	"golang.org/x/sys/unix"
)

// NextImplantID is a placeholder implantID used to select the next implant
// which calls back.
const NextImplantID = "-next-"

// LineReader wraps a bufio.Reader and reads lines of unlimited length.
type LineReader struct {
	reader *bufio.Reader
	buf    bytes.Buffer
}

// Tail represents a tail process.
type Tail struct {
	killSent atomic.Bool /* True if we killed it ourselves. */
	cmd      *exec.Cmd
}

// ReadLine reads a line from l.  ReadLine is not safe to be called from
// multiple goroutines simultaneously.  The returned slice refers to storage
// internal to l; it should not be modified or read simultaneously with calls
// to ReadLine..
func (l *LineReader) ReadLine() ([]byte, error) {
	/* Get a full line.  These get big. */
	l.buf.Reset()

	/* Try to get lines until we hit a newline. */
	for {
		line, prefix, err := l.reader.ReadLine()
		if nil != err {
			return nil, err
		}
		l.buf.Write(line)
		if !prefix {
			break
		}
	}
	return l.buf.Bytes(), nil
}

// TailLogfile starts tail -f tailing the logfile.  It returns a reader which
// receives logfile lines as well as a handle to the tail child process.  The
// last n lines (as with tail -n +n) are sent on the returned reader.
func TailLogfile(logfile string, n int) (*LineReader, *Tail, error) {
	/* Tail the logfile.  It'd be nice if there were a native Go way to do
	this. */
	tail := exec.Command("tail", "-f", "-n", strconv.Itoa(n), logfile)
	tailo, err := tail.StdoutPipe()
	tail.Stderr = os.Stderr /* For just in case. */
	if nil != err {
		return nil, nil, fmt.Errorf("getting tail's stdout: %w", err)
	}
	if err := tail.Start(); nil != err {
		return nil, nil, fmt.Errorf("starting tail: %w", err)
	}

	return &LineReader{reader: bufio.NewReader(tailo)},
		&Tail{cmd: tail},
		nil
}

// killed returns true if the err indicates the process was killed.
func killed(err error) bool {
	/* If this isn't an exiting process, not much to do. */
	var ee *exec.ExitError
	if !errors.As(err, &ee) {
		return false
	}
	/* We should be able to get the underlying details. */
	ws, ok := ee.Sys().(syscall.WaitStatus)
	if !ok {
		return false
	}
	/* We can finally check if we were signalled. */
	if !ws.Signaled() {
		return false
	}
	return syscall.SIGKILL == ws.Signal()
}

// GetNextCallbackID gets the next callback ID from the logfile.  It catches
// and ignores SIGHUP, to make 'pkill plonk' work without killing users of
// -next-.  Before returning, it unregisteres its SIGHUP-catcher.
func GetNextCallbackID(logfile string) (string, error) {
	/* Catch and ignore SIGHUP.  No need to read from the channel, as the
	send will be non-blocking anyway. */
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, unix.SIGHUP)
	defer func() {
		signal.Stop(ch)
		close(ch)
	}()

	/* Watch the logfile for the next callback. */
	reader, tail, err := TailLogfile(logfile, 0)
	if nil != err {
		return "", fmt.Errorf("starting tail: %w", err)
	}
	defer func() { /* Wait for tail to exit. */
		if err := tail.cmd.Wait(); nil == err {
			/* Should only ever be killed. */
			log.Fatalf(
				"[%s] Tail exited unexpectedly",
				MessageTypeError,
			)
		} else if killed(err) && tail.killSent.Load() {
			/* This is normal. */
			return
		} else {
			/* This isn't. */
			log.Fatalf(
				"[%s] Tail exited with error: %s",
				MessageTypeError,
				err,
			)
		}
	}()
	defer func() {
		tail.killSent.Store(true)
		if err := tail.cmd.Process.Kill(); nil != err {
			log.Fatalf(
				"[%s] Killing tail: %s",
				MessageTypeError,
				err,
			)
		}
	}()

	/* Watch the logfile for the next callback line. */
	for {
		/* Get the next line. */
		line, err := reader.ReadLine()
		if nil != err {
			log.Fatalf(
				"[%s] Tailing logfile: %s",
				MessageTypeError,
				err,
			)
		}
		/* If we have an ID, we're done. */
		id, ok := getIDFromCallbackLine(line)
		if ok {
			return id, nil
		}
	}
}

// UpdateWithNextIfNeeded updates s to point to the next implant ID, if s is
// NextImplantID.  If it's not, UpdateWithNextIfNeeded is a no-op.  On error,
// the program is terminated.
func UpdateWithNextIfNeeded(s *string, logfile string) {
	/* If we don't need the next implantID, life's easy. */
	if NextImplantID != *s {
		return
	}

	/* Get the next implantID. */
	id, err := GetNextCallbackID(logfile)
	if nil != err {
		log.Fatalf(
			"[%s] Getting implantID from next callback: %s",
			MessageTypeError,
			err,
		)
	}

	*s = id
}

// getIDFromCallbackLine returns an ID if the line was a sufficient well-formed
// CALLBACK log line.  If the line was well-formed but the ID was empty,
// ("", true) is returned.
func getIDFromCallbackLine(l []byte) (id string, ok bool) {
	/* Extract the important bits. */
	ms := LineRE.FindSubmatch(l)
	if 4 != len(ms) {
		return "", false
	}
	/* Make sure it's a callback. */
	if MessageTypeCallback != string(ms[2]) {
		return "", false
	}

	/* Get the JSON bit. */
	parts := bytes.SplitN(ms[3], []byte{' '}, 5)
	if 5 != len(parts) {
		return "", false
	}

	/* Unmarshal to get the ID. */
	var idj struct{ ID string }
	if err := json.Unmarshal(parts[4], &idj); nil != err {
		return "", false
	}

	return idj.ID, true
}
