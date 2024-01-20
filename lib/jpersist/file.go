package jpersist

/*
 * file.go
 * File ops
 * By J. Stuart McMurray
 * Created 20231007
 * Last Modified 20231210
 */

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
)

// Reload closes and reopens m's file and reloads its contents into m.C.  If
// m wasn't configured with a file, Reload returns ErrNoFile.
func (m *Manager[T]) Reload() error {
	m.l.Lock()
	defer m.l.Unlock()

	/* If we don't have a configured file, someone goofed. */
	if "" == m.conf.File {
		return ErrNoFile
	}

	/* Read into a fresh wrapped value. */
	m.C = new(T)
	f, err := os.OpenFile(
		m.conf.File,
		os.O_RDONLY|os.O_CREATE,
		m.conf.FilePermissions,
	)
	if nil != err {
		return fmt.Errorf("opening %s: %s", m.conf.File, err)
	}
	defer f.Close()
	if err := json.NewDecoder(f).Decode(m.C); nil != err &&
		!errors.Is(err, io.EOF) {
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

	/* Open the file for writing. */
	f, err := os.OpenFile(
		m.conf.File,
		os.O_WRONLY|os.O_CREATE|os.O_TRUNC,
		m.conf.FilePermissions,
	)
	if nil != err {
		return fmt.Errorf("opening %s: %w", m.conf.File, err)
	}
	defer f.Close()

	/* We've changed.  Update the file and, if that worked the last
	hash. */
	if _, err := f.Write(b); nil != err {
		return fmt.Errorf("writing to file: %w", err)
	}
	m.lastHash = h

	return nil
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