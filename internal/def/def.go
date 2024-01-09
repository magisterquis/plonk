// Package def - Defined constants
package def

/*
 * def.go
 * Defined constants
 * By J. Stuart McMurray
 * Created 20231110
 * Last Modified 20231208
 */

import (
	"fmt"
	"time"
)

// Flag defaults.
var (
	DefaultDir       = "plonk.d" /* Just basename. */
	DefaultHTTPAddr  = ""
	DefaultHTTPSAddr = "0.0.0.0:443"
	DefaultName      = "" /* Operator name. */
)

// Files and directories, within working directory
var (
	LogFile        = "log.json"
	OpSock         = "op.sock" /* Operator comms Unix socket. */
	StateFile      = "state.json"
	ExfilDir       = "exfil"
	StaticFilesDir = "files"
	TemplateFile   = "implant.tmpl"
	//
)

// Request URL paths.
var (
	/* Implant paths.  A slash will be added to the end before these are
	passed to http.ServeMux.Handle. */
	TaskPath    = "/t"
	ExfilPath   = "/p"
	FilePath    = "/f"
	CurlGenPath = "/c" /* Implant generator. */
	OutputPath  = "/o"
	//
	//	/* Operator paths. */
	//	CheckPath   = "/check"
	//	EnqueuePath = "/enqueue"
	//	ListPath    = "/implantlist"
	//	LogsPath    = "/logs"
	//
)

// Other configurables
var (
	StateWriteDelay  = "5s"
	StateWriteDelayD time.Duration
)

// Other nonconfigurables
const (
	C2URLParam = "c2" /* cURLGen query C2 URL parameter. */
	DirPerms   = 0750 /* Default directory permissions. */
	//	DummyAddr  = "http://dummy/"   /* Dummy Operator server address. */
	FilePerms = 0640 /* Default file permissions. */
	// MaxExfil   = 100 * 1024 * 1024 /* Exfil stops at 100MB. */
	MaxExfilOpenTries = 100             /* Maximum number of exfil filenames to try. */
	MaxOutput         = 1 * 1024 * 1024 /* Output stops at 1MB. */
	NSeen             = 10              /* Max number of seen IDs. */
	// NameHeader = "X-Operator"      /* Operator name HTTP header. */
	AcceptWait    = time.Second / 4        /* Wait between failed accepts. */
	OpNameWait    = 10 * time.Second       /* Wait for operator's name. */
	HTTPIOTimeout = 10 * time.Second       /* HTTP read/write timeout. */
	LogsPrompt    = "(plonk)> "            /* opshell prompt for log-watching. */
	NamelessName  = "the nameless implant" /* Name we give to ID "" */
)

// init converts some defaults from strings
func init() {
	d, err := time.ParseDuration(StateWriteDelay)
	if nil != err {
		panic(fmt.Sprintf(
			"parsing StateWriteDelay (%s): %s",
			StateWriteDelay,
			err,
		))
	}
	StateWriteDelayD = d
}
