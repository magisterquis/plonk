package implantsvr

/*
 * errors.go
 * Error types and handlers
 * By J. Stuart McMurray
 * Created 20231207
 * Last Modified 20231227
 */

import (
	"regexp"
)

// httpDebugLogREs holds a list of REs matching HTTP server log messages which
// should be logged at level DEBUG and not ERROR.
var httpDebugLogREs = []*regexp.Regexp{
	regexp.MustCompile(`^http: TLS handshake error from \S+: EOF$`),
}