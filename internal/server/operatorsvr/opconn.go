package operatorsvr

/*
 * opconn.go
 * Operator connection
 * By J. Stuart McMurray
 * Created 20231129
 * Last Modified 20231208
 */

import (
	"encoding/json"
	"log/slog"
	"sync/atomic"

	"github.com/magisterquis/plonk/internal/def"
	"github.com/magisterquis/plonk/internal/server/state"
	"github.com/magisterquis/plonk/lib/estream"
	"github.com/magisterquis/plonk/lib/jpersist"
	"github.com/magisterquis/plonk/lib/plog"
	"github.com/magisterquis/plonk/lib/waiter"
)

// opConn represents an operator's connection.
type opConn struct {
	sl   atomic.Pointer[slog.Logger]
	es   *estream.Stream
	name plog.AtomicString
	nw   waiter.Waiter[string] /* True when we have a name. */
	sm   *jpersist.Manager[state.State]
}

// Goodbye sends a goodbye message to the operator and ends the connection.
// The message is sent to the operator.
func (oc *opConn) Goodbye(msg string) {
	oc.es.Send(def.ENGoodbye, def.EDGoodbye{Message: msg})
	oc.es.Close()
}

// defaultHandler handles messages of unknown type.
func (oc *opConn) defaultHandler(mtype string, data json.RawMessage) {
	oc.SL().Warn(
		def.LMUnexpectedMessage,
		def.LKMessageType, mtype,
		def.LKMessage, data,
	)
}

// nameHandler handles setting the operator's name.
func (oc *opConn) nameHandler(mtype string, data def.EDName) {
	name := string(data)
	old, hadOld := oc.name.Swap(name)
	if !hadOld { /* Set name for new conn. */
		oc.nw.AlwaysBroadcast(name)
	} else { /* Name change. */
		oc.SL().Info(
			def.LMOpNameChange,
			def.LKOpOldName, old,
		)
	}
}

// enqueueHandler handles queuing a task.
func (oc *opConn) enqueueHandler(mtype string, data def.EDEnqueue) {

	/* Make sure we have the bits we need. */
	if "" == data.ID {
		data.Error = "ID missing"
		oc.es.Send(mtype, data)
		return
	} else if "" == data.Task {
		data.Error = "Empty task"
		oc.es.Send(mtype, data)
		return
	}

	/* Get and update the task queue. */
	oc.sm.Lock()
	oc.sm.C.TaskQ[data.ID] = append(oc.sm.C.TaskQ[data.ID], data.Task)
	qlen := len(oc.sm.C.TaskQ[data.ID])
	oc.sm.UnlockAndWrite()

	oc.SL().Info(
		def.LMTaskQueued,
		def.LKID, data.ID,
		def.LKTask, data.Task,
		def.LKQLen, qlen,
	)
}

// listSeenhandler lists the implants we've seen.
func (oc *opConn) listSeenHandler(mtype string, _ any) {
	/* Get a copy of the list. */
	oc.sm.RLock()
	list := oc.sm.C.LastSeen
	oc.sm.RUnlock()

	/* Send it back. */
	if nil == oc.es.Send(def.ENListSeen, list) {
		oc.SL().Debug(def.LMSentSeenList)
	}
}

// SL returns the logger for the conn.
func (oc *opConn) SL() *slog.Logger { return oc.sl.Load() }
