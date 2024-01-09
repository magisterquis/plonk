package def

/*
 * log.go
 * Log messages and keys
 * By J. Stuart McMurray
 * Created 20231110
 * Last Modified 20231211
 */

// Log messages
const (
	LMCaughtSignal   = "Caught signal, exiting"
	LMCurlGen        = "Implant generation"
	LMExfil          = "Exfil"
	LMFileRequest    = "Static file requested"
	LMImplantServing = "Implant service started"
	LMNewImplant     = "New implant"
	LMOpConnected    = "Operator connected"
	LMOpDisconnected = "Operator disconnected"
	LMOpListening    = "Operator listener started"
	LMOpNameChange   = "Operator name change"
	LMOutputRequest  = "Output"
	LMSentSeenList   = "Sent implant list"
	LMServerReady    = "Server ready"
	LMTaskQueued     = "Task queued"
	LMTaskRequest    = "Task request"

	/* Errors */
	LMDefaultFileFailed    = "Opening default file failed"
	LMHTTPError            = "HTTP error"
	LMHTTPErrorFailed      = "HTTP error logger failed"
	LMOpInitialNameError   = "Error getting initial operator name"
	LMServerDied           = "Server died"
	LMStateWriteFailed     = "State write failed"
	LMTemporaryAcceptError = "Temporary accept error"
	LMUnexpectedMessage    = "Unexpected message"
)

// Log keys
const (
	LKAddress     = "address"
	LKConnNumber  = "cnum"
	LKDirname     = "dirname"
	LKErrorType   = "error_type"
	LKFilename    = "filename"
	LKHTTPAddr    = "http_addr"
	LKHTTPSAddr   = "https_addr"
	LKHash        = "hash"
	LKHost        = "host"
	LKID          = "id"
	LKLocation    = "location"
	LKMessage     = "message"
	LKMessageType = "message_type"
	LKMethod      = "method"
	LKOpName      = "opname"
	LKOpOldName   = "oldname"
	LKOutput      = "output"
	LKParameters  = "parameters"
	LKQLen        = "qlen"
	LKRemoteAddr  = "remote_address"
	LKReqPath     = "requested_path"
	LKSNI         = "sni"
	LKSignal      = "signal"
	LKSize        = "size"
	LKStatusCode  = "status_code"
	LKTask        = "task"
	LKURL         = "url"
)