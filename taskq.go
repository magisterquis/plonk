package main

/*
 * taskq.go
 * Read and write the task queue.
 * By J. Stuart McMurray
 * Created 20230223
 * Last Modified 20230225
 */

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"sync"

	"golang.org/x/sys/unix"
)

// TaskQueue holds the per-implant task queue.  There's probably a more
// efficient way to do this.
type TaskQ map[string][]string

// Task file.  Handlers share it, albeit with lots of disk I/O.
var (
	taskF *os.File
	taskC = sync.NewCond(new(sync.Mutex))
)

// OpenTaskFile opens the taskfile and keeps it available via ReadQ/WriteQ.  It
// then waits for SIGHUPs, upon receipt of which it reopens the taskfile.  If
// it can't open the taskfile on SIGHUP, it'll log a warning and wait for
// another SIGHUP.
func OpenTaskFile(taskfile string) error {
	if err := reopenTaskFile(taskfile); nil != err {
		return err
	}

	/* Reopen on SIGHUP. */
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, unix.SIGHUP)

	go func() {
		for range ch {
			/* Got a SIGHUP, close and reopen file. */
			if err := reopenTaskFile(taskfile); nil != err {
				log.Printf(
					"[%s] Re-opening taskfile "+
						"%q on SIGHUP: %s",
					MessageTypeError,
					taskfile,
					err,
				)
				continue
			}
			log.Printf(
				"[%s] Re-opened taskfile %s",
				MessageTypeSIGHUP,
				taskfile,
			)
		}
	}()

	return nil
}

// reopenTaskFile closes the taskfile if it's open and re-opens it.  It then
// wakes up any thread waiting on taskC.
func reopenTaskFile(taskfile string) error {
	taskC.L.Lock()
	defer taskC.L.Unlock()

	/* Close the old file eventually. */
	ctf := taskF
	defer ctf.Close()
	taskF = nil

	/* Try to open the new one. */
	var err error
	taskF, err = os.OpenFile(taskfile, os.O_RDWR|os.O_CREATE, 0660)
	if nil != err {
		return err
	}

	/* Wake up waiters. */
	taskC.Broadcast()

	return nil
}

// ReadQ reads TaskF into a TaskQ.  A corresponding call to WriteF must be
// called after a successful call to ReadQ or other calls to ReadQ will block.
func ReadQ() (TaskQ, error) {
	/* Make sure we have an taskfile.  If we don't it should be opened
	fairly soon. */
	taskC.L.Lock()
	for nil == taskF {
		taskC.Wait()
	}

	/* Also lock it away from other programs. */
	fd := int(taskF.Fd())
	if err := unix.Flock(fd, unix.LOCK_EX); nil != err {
		writeAndUnlock(true, false, nil)
		return nil, fmt.Errorf("locking: %w", err)
	}

	/* Get the taskings from the taskfile. */
	if err := rewind(); nil != err {
		writeAndUnlock(true, true, nil)
		return nil, err
	}
	var taskQ TaskQ
	if err := json.NewDecoder(taskF).Decode(&taskQ); nil != err {
		if errors.Is(err, io.EOF) { /* Empty file. */
			return TaskQ{}, nil
		}
		writeAndUnlock(true, true, nil)
		return nil, fmt.Errorf("decoding: %w", err)
	}

	return taskQ, nil
}

// WriteQ writes q to taskF if q is not nil.  It also unlocks taskC.L and
// taskF's flock.  WriteQ should not be called without a corresponding
// successful call to ReadQ.
func WriteQ(q TaskQ) error {
	return writeAndUnlock(true, true, q)
}

// writeAndUnlock writes q to taskF if q is not nil, and releases taskC's lock
// and taskF's flock, as instructed.
func writeAndUnlock(unMutex, unFlock bool, q TaskQ) error {
	var rerr error

	/* If we've something to write, clean up empty queues and update the
	file. */
	if nil != q {
		/* Remove empty queues. */
		for id, idq := range q {
			if 0 == len(idq) {
				delete(q, id)
			}
		}
		/* Write to the file .*/
		if err := rewind(); nil != err {
			rerr = errors.Join(rerr, err)
		} else if err := taskF.Truncate(0); nil != err {
			rerr = errors.Join(
				rerr,
				fmt.Errorf("truncating: %w", err),
			)
		} else {
			enc := json.NewEncoder(taskF)
			enc.SetIndent("", "\t")
			if err := enc.Encode(q); nil != err {
				rerr = errors.Join(rerr, fmt.Errorf(
					"writing: %w",
					err,
				))
			} else if err := taskF.Sync(); nil != err {
				rerr = errors.Join(rerr, fmt.Errorf(
					"flushing to disk: %w",
					err,
				))
			}
		}
	}

	/* Unlock locks. */
	if unFlock {
		if err := unix.Flock(
			int(taskF.Fd()),
			unix.LOCK_UN,
		); nil != err {
			rerr = errors.Join(fmt.Errorf(
				"releasing flock: %w",
				err,
			))
		}
	}
	if unMutex {
		taskC.L.Unlock()
		/* Glory be, no wonky error-handling. */
	}

	return rerr
}

// rewind resets taskF's file pointer to the beginning of the file.
func rewind() error {
	if _, err := taskF.Seek(0, os.SEEK_SET); nil != err {
		return fmt.Errorf(
			"seeking to beginning: %w",
			err,
		)
	}
	return nil
}
