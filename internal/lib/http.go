package lib

/*
 * http.go
 * HTTP utility functions
 * By J. Stuart McMurray
 * Created 20230223
 * Last Modified 20230726
 */

import (
	"log"
	"net/http"
	"strings"
)

// httpLogger logs HTTP messages using Verbosef.  Each write is a single
// log entry.
type httpLogger struct{}

// Write implements io.Writer.  Each call results in a single log entry.
func (h httpLogger) Write(b []byte) (int, error) {
	Verbosef("[%s] %s", MessageTypeHTTP, b)
	return len(b), nil
}

// HTTPServer is an http server configured for better logging.
var HTTPServer = http.Server{ErrorLog: log.New(httpLogger{}, "", 0)}

// ImplantID gets the Implant ID from the request, which is the path less
// the leading slash.  It may return an empty string.
func ImplantID(r *http.Request) string {
	return strings.TrimPrefix(r.URL.Path, "/")
}

// LogHandler wraps h and logs every request, if -verbose is on
func LogHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if VerbOn {
			RLog(r, MessageTypeFile, "")
		}
		h.ServeHTTP(w, r)
	})
}
