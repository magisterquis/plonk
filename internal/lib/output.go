package lib

/*
 * output.go
 * Get task output
 * By J. Stuart McMurray
 * Created 20230223
 * Last Modified 20230726
 */

import (
	"errors"
	"io"
	"net/http"
)

// OutputLog is used to marshal command output to unambiguous JSON.
type OutputLog struct {
	ID     string
	Output string `json:",omitempty"`
	Err    string `json:",omitempty"`
}

// HandleOutput handles output from an implant
func HandleOutput(w http.ResponseWriter, r *http.Request) {
	l := OutputLog{ID: ImplantID(r)}
	mt := MessageTypeOutput

	/* Get the output itself.  It'll be truncated if it's too long; this
	is not an error. */
	b, err := io.ReadAll(r.Body)
	var mbe *http.MaxBytesError
	if errors.As(err, &mbe) {
		err = nil
	}
	if nil != err {
		l.Err = err.Error()
	}
	if 0 != len(b) { /* Even on error, might have got something? */
		l.Output = string(b)
	}

	/* Don't bother logging empty output. */
	RLogInteresting(l.ID, r, mt, l)
}
