package jpersist

/*
 * lock.go
 * Lock and unlock (and write)
 * By J. Stuart McMurray
 * Created 20231007
 * Last Modified 20231008
 */

import (
	"time"
)

// RLock acquires a shared lock on m.  No changes should be made to m.C. Call
// RUnlock to release the lock.
func (m *Manager[T]) RLock() { m.l.RLock() }

// RUnlock releases a lock acquired by RLock.  Do not call RUnlock without
// calling RLock first.
func (m *Manager[T]) RUnlock() { m.l.RUnlock() }

// Lock acquires an exclusive lock on m.  This should be called before making
// changes to m.C.  Call Unlock or UnlockAndWrite to release the lock.
func (m *Manager[T]) Lock() { m.l.Lock() }

// Unlock releases a lock acquired by Lock().  If m.C has changed, it will be
// written to the file if one is configured, possibly after a configured
// delay.  If an error is returned, it will also be passed to the callback
// configured when m was created, if one was configured.  If the write is
// delayed, the error will only be passed to the callback, not returned. Do not
// call Unlock without calling Lock first.
func (m *Manager[T]) Unlock() error {
	/* If we're not delaying, life's easy. */
	if 0 == m.conf.WriteDelay {
		return m.unlockAndWrite(1)
	}

	/* We'll need to unlock eventually. */
	defer m.l.Unlock()

	/* If we're in a delay, no need to do anything more. */
	if nil != m.writeTimer {
		return nil
	}

	/* Not in the delay, so start one. */
	m.writeTimer = time.AfterFunc(m.conf.WriteDelay, func() {
		m.l.Lock()
		/* If someone already removed the timer, that someone also
		wrote.  Can happen if UnlockAndWrite's called and we get
		unlucky with timing. */
		if nil == m.writeTimer {
			return
		}
		/* Update the file and note we're no longer in a write
		delay. */
		m.unlockAndWrite(2)
		m.writeTimer = nil
	})
	return nil
}

// UnlockAndWrite is like Unlock, but does not wait before writing m.C to the
// configured file.  This is useful for reducing the risk of changes to m.C
// being lost should the program crash before writing.  Do not call
// UnlockAndWrite without calling Lock first.
func (m *Manager[T]) UnlockAndWrite() error {
	/* If we're in the middle of a write delay, no point in writing again
	later. */
	if nil != m.writeTimer {
		m.writeTimer.Stop()
		m.writeTimer = nil
	}

	/* Do the writing now. */
	return m.unlockAndWrite(1)
}

// unlockAndWrite is UnlockAndWrite, but assumes it'll be called from something
// inside this package.  Skip describes how many callers are to be skipped
// inside this package, for error reporting.
func (m *Manager[T]) unlockAndWrite(skip int) error {
	defer m.l.Unlock()

	/* Wake up anybody waiting after this call. */
	defer m.writeWaiter.Broadcast(struct{}{})

	/* Nothing really to do if we don't have a file. */
	if "" == m.conf.File {
		return nil
	}

	/* Try to write. */
	if err := m.write(); nil != err {
		/* Got an error.  Wrap it in some details and also send it to
		the callback, if we have one. */
		err = newUnlockError(err, skip)
		if nil != m.conf.OnError {
			go m.conf.OnError(err)
		}
		return err
	}

	return nil
}
