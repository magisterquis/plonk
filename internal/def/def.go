// Package def - Defined constants
package def

/*
 * def.go
 * Defined constants
 * By J. Stuart McMurray
 * Created 20231110
 * Last Modified 20240119
 */

import (
	"fmt"
	"time"
)

// Files and directories, within working directory
var (
	DefaultFile    = "index.html" /* Default file served by HTTP. */
	ExfilDir       = "exfil"
	LogFile        = "log.json"
	OpSock         = "op.sock" /* Operator comms Unix socket. */
	StateFile      = "state.json"
	StaticFilesDir = "files"
	TemplateFile   = "implant.tmpl"
	DirEnvVar      = "PLONK_DIRECTORY"
	ColorEnvVar    = "PLONK_COLORIZE"
)

// Request URL paths.
var (
	CurlGenPath = "/c" /* Implant generator. */
	ExfilPath   = "/p"
	FilePath    = "/f"
	OutputPath  = "/o"
	TaskPath    = "/t"
)

// Other configurables
var (
	StateWriteDelayD time.Duration
)

// Other nonconfigurables
const (
	AcceptWait        = time.Second / 4  /* Wait between failed accepts. */
	C2URLParam        = "c2"             /* cURLGen query C2 URL parameter. */
	DirPerms          = 0770             /* Default directory permissions. */
	FilePerms         = 0660             /* Default file permissions. */
	HTTPIOTimeout     = 30 * time.Second /* HTTP read/write timeout. */
	LogsPrompt        = "(Plonk)> "      /* opshell prompt for log-watching. */
	MaxExfilOpenTries = 100              /* Maximum number of exfil filenames to try. */
	MaxOutput         = 1 * 1024 * 1024  /* Output stops at 1MB. */
	NSeen             = 10               /* Max number of seen IDs. */
	NReplayLogs       = 10               /* Number of logs to show with ,logs. */
	OpNameWait        = 10 * time.Second /* Wait for operator's name. */
	StateWriteDelay   = "5s"
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
