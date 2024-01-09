package eztls

/*
 * errors.go
 * Error types
 * By J. Stuart McMurray
 * Created 20231223
 * Last Modified 20231223
 */

import "fmt"

// BadPatternError is returned by HostWhitelist to indicate a malformed or
// otherwise unusable pattern.
type BadPatternError struct {
	Pattern string
	Err     error
}

// Error implements the error interface.
func (err BadPatternError) Error() string {
	return fmt.Sprintf("bad pattern %q: %s", err.Pattern, err.Err)
}

// Unwrap returns err.Err.
func (err BadPatternError) Unwrap() error { return err.Err }

// NotWhitelistedError is returned by the autocert.HostPolicy returned by
// HostWhitelist if the requested host doesn't match a whitelisted pattern.
type NotWhitelistedError struct {
	Host string
}

// Error implements the error interface.
func (err NotWhitelistedError) Error() string {
	return fmt.Sprintf(
		"host %q not allowed by any whitelist pattern",
		err.Host,
	)
}
