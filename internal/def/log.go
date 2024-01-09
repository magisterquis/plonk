package def

/*
 * log.go
 * Log messages and keys
 * By J. Stuart McMurray
 * Created 20231110
 * Last Modified 20231208
 */

// Log messages
const (
	LMExfil = "Exfil"
	// LMHTTPRequest      = "HTTP request"
	// LMImplant          = "Generated implant"
	LMImplantServing = "Implant service started"
	LMOutputRequest  = "Output"
	LMTaskRequest    = "Task request"
	LMFileRequest    = "Static file requested"
	LMOpListening    = "Operator listener started"
	LMOpConnected    = "Operator connected"
	LMOpDisconnected = "Operator disconnected"
	LMServerReady    = "Server ready"
	LMCaughtSignal   = "Caught signal, exiting"
	// LMOpAnotherName    = "Operator sent another name"
	// LMGotOpName        = "Operator sent name"
	LMOpNameChange = "Operator name change"
	LMTaskQueued   = "Task queued"
	LMSentSeenList = "Sent implant list"
	LMNewImplant   = "New implant"
	//
	// /* Operator HTTP requests. */
	// LMWatchingLogs = "Watching log stream"
	//
	// /* Errors */
	// LMExfilMkdirFailed = "Exfil directory creation failed"
	// LMExfilOpenFailed      = "Exfil file open failed"
	LMHTTPError       = "HTTP error"
	LMHTTPErrorFailed = "HTTP error logger failed"
	// LMLogWriteFailed       = "Log write failed"
	// LMNoC2URL              = "Could not guess C2 URL"
	// LMOperatorHTTPError    = "Operator HTTP error"
	// LMServerError          = "Server error"
	LMStateWriteFailed     = "State write failed"
	LMCurlGen              = "Implant generation"
	LMTemporaryAcceptError = "Temporary accept error"
	LMUnexpectedMessage    = "Unexpected message"
	LMServerDied           = "Server died"
	LMOpInitialNameError   = "Error getting initial operator name"
)

// Log keys
const (
	LKRemoteAddr = "remote_address"
	LKAddress    = "address"
	LKDirname    = "dirname"
	// LKDomains    = "TLS_domains"
	// LKError      = plog.LKeyError
	LKFilename   = "filename"
	LKID         = "id"
	LKTask       = "task"
	LKLocation   = "location"
	LKStatusCode = "status_code"
	// LKOpName     = "operator"
	LKConnNumber  = "cnum"
	LKOpName      = "opname"
	LKOpOldName   = "oldname"
	LKHTTPAddr    = "http_addr"
	LKHTTPSAddr   = "https_addr"
	LKParameters  = "parameters"
	LKMessageType = "message_type"
	LKMessage     = "message"
	LKSignal      = "signal"
	LKReqPath     = "requested_path"
	LKURL         = "url"
	LKMethod      = "method"
	LKSNI         = "sni"
	LKHost        = "host"
	LKQLen        = "qlen"
	LKOutput      = "output"
	LKErrorType   = "error_type"
	LKSize        = "size"
	LKHash        = "hash"

// LKState      = "state"
// LKNewName    = "new_name"
)
