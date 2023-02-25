package main

/*
 * log.go
 * Handle logging
 * By J. Stuart McMurray
 * Created 20230225
 * Last Modified 20230225
 */

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"reflect"
	"sync"

	"golang.org/x/exp/maps"
	"golang.org/x/sys/unix"
)

// MessageType is used to tag logged messages
type MessageType string

const (
	MessageTypeCallback = "CALLBACK"
	MessageTypeError    = "ERROR"
	MessageTypeInfo     = "INFO"
	MessageTypeFile     = "FILE"
	MessageTypeHTTP     = "HTTP"
	MessageTypeOutput   = "OUTPUT"
	MessageTypeSIGHUP   = "SIGHUP"
	MessageTypeTLS      = "TLS"
	MessageTypeTaskQ    = "TASKQ"
	MessageTypeUnknown  = "UNKNOWN"
)

// DefaultLoge is the default name of our logfile.
const DefaultLog = "log"

var (
	/* seenIDs keeps track of which IDs we've seen, for better logging. */
	seenIDs  = make(map[string]struct{})
	seenIDsL sync.Mutex
)

// Delete all seenIDs on SIGHUP
func init() {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, unix.SIGHUP)
	go func() {
		for range ch {
			seenIDsL.Lock()
			n := len(seenIDs)
			maps.Clear(seenIDs)
			log.Printf(
				"[%s] Forgot %d first-seen IDs",
				MessageTypeSIGHUP,
				n,
			)
			seenIDsL.Unlock()
		}
	}()
}

// RLogInteresting legs like RLogJSON, but only if the ID hasn't been seen
// before, -verbose is given, or v is a struct and fields other than ID have
// been set in v.
func RLogInteresting(id string, r *http.Request, messageType string, v any) {
	/* See if we even want to log. */
	var interesting bool
	for _, f := range []func() bool{func() bool {
		return VerbOn /* -verbose */
	}, func() bool {
		seenIDsL.Lock()
		defer seenIDsL.Unlock()
		_, seen := seenIDs[id]
		if !seen {
			seenIDs[id] = struct{}{}
		}
		return !seen /* First time we've seen the ID. */
	}, func() bool {
		rv := reflect.ValueOf(v)
		t := rv.Type()
		/* Only care about structs. */
		if reflect.Struct != rv.Kind() {
			return false /* Not a struct. */
		}
		/* Look for non-ID fields being set. */
		for i := 0; i < rv.NumField(); i++ {
			/* ID will often be set. */
			if "ID" == t.Field(i).Name {
				continue
			}
			/* If any other field is set, it's interesting. */
			if !rv.Field(i).IsZero() {
				return true /* Non-ID field set. */
			}
		}
		return false /* No interesting fields set. */
	}} {
		if f() {
			interesting = true
			break
		}
	}

	/* If this isn't something we want to log, give up. */
	if !interesting {
		return
	}

	/* Do the logging. */
	RLogJSON(r, messageType, v)
}

// RLogf logs a message related to an HTTP request with fmt.Printf-style
// formatting.
func RLogf(r *http.Request, messageType string, format string, args ...any) {
	RLog(r, messageType, fmt.Sprintf(format, args...))
}

// RLogJSON JSONifies v and logs it.
func RLogJSON(r *http.Request, messageType string, v any) {
	j, err := json.Marshal(v)
	if nil != err {
		RLogf(r, MessageTypeError, "JSONing %#v: %s", v, err)
		return
	}

	/* Don't bother logging empty objects. */
	if "{}" == string(j) {
		j = nil
	}
	RLog(r, messageType, string(j))
}

// RLog logs a message related to an HTTP request.
func RLog(r *http.Request, messageType string, msg string) {
	log.Printf("%s", rLogMarshal(r, messageType, msg))
}

// rLogMarshal marshals a message into RLog format.
func rLogMarshal(r *http.Request, messageType string, msg string) string {
	/* Work out the SNI. */
	sni := "HTTP"
	if nil != r.TLS {
		if "" != r.TLS.ServerName {
			sni = r.TLS.ServerName
		} else {
			sni = "NoSNI"
		}
	}

	/* Roll the message itself. */
	lmsg := fmt.Sprintf(
		"[%s] %s %s %s %s",
		defString(messageType, MessageTypeUnknown),
		defString(r.RemoteAddr, "addr?"),
		sni,
		defString(r.Method, "verb?"),
		defString(r.RequestURI, "uri?"),
	)
	if 0 != len(msg) {
		lmsg += " " + msg
	}

	return lmsg
}

// defString returns s if it's not "", or def if it is.
func defString(s, def string) string {
	if "" != s {
		return s
	}
	return def
}
