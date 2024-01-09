package jpersist

/*
 * file.go
 * File ops
 * By J. Stuart McMurray
 * Created 20231007
 * Last Modified 20231007
 */

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
)

// openFile opens m's file.  openFile's caller must have locked m.l for
// writing.
func (m *Manager[T]) openFile() error {
	/* Make sure we actually have a configured file. */
	if "" == m.conf.File {
		return ErrNoFile
	}

	/* Open the file, if we can. */
	f, err := os.OpenFile(
		m.conf.File,
		os.O_RDWR|os.O_CREATE,
		m.conf.FilePermissions,
	)
	if nil != err {
		return fmt.Errorf("opening %s: %w", m.conf.File, err)
	}

	/* Opened nicely, replace the old one. */
	if nil != m.f {
		m.f.Close()
	}
	m.f = f

	return nil
}

// Reload closes and reopens m's file and reloads its contents into m.C.  If
// m wasn't configured with a file, Reload returns ErrNoFile.
func (m *Manager[T]) Reload() error {
	m.l.Lock()
	defer m.l.Unlock()

	/* If we don't have a configured file, someone goofed. */
	if "" == m.conf.File {
		return ErrNoFile
	}

	/* (Re)Open the file. */
	if err := m.openFile(); nil != err {
		return fmt.Errorf("opening file: %w", err)
	}

	/* Read into a fresh wrapped value. */
	m.C = new(T)
	if err := json.NewDecoder(m.f).Decode(m.C); nil != err &&
		!errors.Is(err, io.EOF) {
		m.closeAndNil()
		return fmt.Errorf("unJSONing: %w", err)
	}

	return nil
}

// Write writes m.C as JSON to its configured file, if it's changed.  If
// m wasn't configured with a file, Write returns ErrNoFile.
func (m *Manager[T]) Write() error {
	m.l.Lock()
	defer m.l.Unlock()
	return m.write()
}

// Write is what write says it does, but requires its caller to have m locked
// for writing.
func (m *Manager[T]) write() error {
	/* If we don't have a configured file, someone goofed. */
	if "" == m.conf.File {
		return ErrNoFile
	}

	/* JSONify and see if we've changed. */
	b, h, err := marshal(m.C)
	if nil != err {
		return fmt.Errorf("JSONifying: %w", err)
	}
	if m.lastHash == h {
		return nil
	}

	/* Make sure we have an open file. */
	if nil == m.f {
		if err := m.openFile(); nil != err {
			return fmt.Errorf("opening file: %w", err)
		}
	}

	/* We've changed.  Update the file and, if that worked the last
	hash. */
	if _, err := m.f.Seek(0, io.SeekStart); nil != err {
		m.closeAndNil()
		return fmt.Errorf("seeking to beginning of file: %w", err)
	}
	if err := m.f.Truncate(0); nil != err {
		m.closeAndNil()
		return fmt.Errorf("truncating file: %w", err)
	}
	if _, err := m.f.Write(b); nil != err {
		m.closeAndNil()
		return fmt.Errorf("writing to file: %w", err)
	}
	m.lastHash = h

	return nil
}

// closeAndNil closes m.f and sets it to nil, to prevent leakage.
// closeAndNil's caller must hold m locked for writing.
func (m *Manager[T]) closeAndNil() {
	if nil == m.f {
		return
	}
	m.f.Close()
	m.f = nil
}

// marshal JSONs v and returns the JSON as well as its hash.
func marshal(v any) ([]byte, [sha256.Size]byte, error) {
	var h [sha256.Size]byte

	/* JSONify v. */
	b, err := json.MarshalIndent(v, "", "\t")
	if nil != err {
		return nil, h, err
	}

	/* Also get the hash. */
	h = sha256.Sum256(b)

	return b, h, nil
}
