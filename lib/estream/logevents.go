package estream

/*
 * logevents.go
 * Use logs as a source of events
 * By J. Stuart McMurray
 * Created 20231205
 * Last Modified 20231206
 */

import (
	"encoding/json"
	"fmt"
	"io"
)

// SendJSONSLogs reads JSON log objects as produced by slog.JSONHandler and
// sends them via s.Send using the msg field of the JSON objects as the event
// names.
func (s *Stream) SendJSONSLogs(r io.Reader) error {
	var (
		/* Small struct for extracting the name of each message. */
		msg struct {
			Name string `json:"msg"`
		}
		/* Each message, as bytes. */
		rm json.RawMessage
		/* Gives us JSON objects. */
		dec = json.NewDecoder(r)
	)
	for {
		/* Get the next log line */
		if err := dec.Decode(&rm); nil != err {
			return fmt.Errorf("reading next object: %w", err)
		}
		/* Get the event name. */
		if err := json.Unmarshal(rm, &msg); nil != err {
			return fmt.Errorf("unmarshalling name: %w", err)
		}
		/* Send the event out. */
		if err := s.Send(msg.Name, rm); nil != err {
			return fmt.Errorf("sending %q event: %w", msg.Name, err)
		}
	}
}
