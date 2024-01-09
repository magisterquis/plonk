package implantsvr

/*
 * infowriter.go
 * ResponseWriter which keeps track of some info
 * By J. Stuart McMurray
 * Created 20231208
 * Last Modified 20231208
 */

import (
	"bytes"
	"io"
	"net/http"
	"sync/atomic"
	"testing"
)

// infoWriter is an http.ResponseWriter which wraps another http.ResponseWriter
// and keeps track of how much data was sent to Write, and other such things.
// The atomic fields provide up-to-date information.
type infoWriter struct {
	Wrapped http.ResponseWriter

	Written    atomic.Uint64 /* Number of bytes passed to Write. */
	StatusCode atomic.Uint64 /* Status code passed to WriteHeader. */

	b atomic.Pointer[bytes.Buffer] /* For testing. */
}

// Header wraps iw.Wrapped.Header
func (iw *infoWriter) Header() http.Header { return iw.Wrapped.Header() }

// Write passes bytes to Write, and adds the number written to the iw.Written.
// If no status code has been set with iw.WriteHeader, iw.StatusCode will be
// set to http.StatusOK.
func (iw *infoWriter) Write(b []byte) (int, error) {
	var (
		n   int
		err error
	)
	/* If we're in a test, also save the body. */
	if testing.Testing() {
		iw.b.CompareAndSwap(nil, new(bytes.Buffer))
		n, err = io.MultiWriter(iw.Wrapped, iw.b.Load()).Write(b)
	} else {
		n, err = iw.Wrapped.Write(b)
	}
	/* Save how much we've written so far. */
	iw.Written.Add(uint64(n))

	/* Make sure we have a status code. */
	iw.StatusCode.CompareAndSwap(0, http.StatusOK)

	return n, err
}

// WriteHeader passes the statusCode to iw.Wrapped, and stores it in
// iw.StatusCode.
func (iw *infoWriter) WriteHeader(statusCode int) {
	iw.StatusCode.Store(uint64(statusCode))
	iw.Wrapped.WriteHeader(statusCode)
}
