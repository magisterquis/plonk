package def

/*
 * events.go
 * Stream events types
 * By J. Stuart McMurray
 * Created 20231129
 * Last Modified 20231206
 */

import "time"

// Event names
const (
	ENGoodbye  = "goodbye"  /* Server's closing. */
	ENName     = "name"     /* Operator name. */
	ENEnqueue  = "enqueue"  /* Enqueue(d) task. */
	ENListSeen = "listseen" /* List seen implants. */
)

// EDGoodbye indicates the server is shutting down.
type EDGoodbye struct {
	Message string
}

// EDName sets the per-connection Operator name.
type EDName string

// EDSeen contains the last-seen implants.
type EDSeen [NSeen]struct {
	ID   string    /* Implant ID. */
	When time.Time /* When last seen. */
}

// EDEnqueue is a queued task.
type EDEnqueue struct {
	ID    string
	Task  string
	Error string
}

// EDLMTaskQueued is a log message indicating a queued task.
type EDLMTaskQueued struct {
	ID     string
	Task   string
	OpName string
	QLen   int
}

// EDLMOpConnected is a log message indicating a new operator has connected.
// It also works for disconnections.
type EDLMOpConnected struct {
	OpName string
	CNum   int
}

// EDLMTaskRequest is a log message sent after a request for tasking.
type EDLMTaskRequest struct {
	ID    string
	Task  string /* Empty means none sent. */
	QLen  int
	Error string
}

// EDLMOutputRequest is a log message sent after a request to send output.
type EDLMOutputRequest struct {
	ID     string
	Output string /* Empty means none sent. */
	Error  string
}

// EDLMNewImplant informs about a newly-seen implant.
type EDLMNewImplant struct {
	ID string
}

// EDLMFileRequest tells us when someone's asked for a file.
type EDLMFileRequest struct {
	StatusCode int    `json:"status_code"`
	RemoteAddr string `json:"remote_address"`
	Filename   string
	Size       int
}
