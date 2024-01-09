// Package pbuf - Parallel-safe bytes.Buffer replacement
package pbuf

/*
 * pbuf.go
 * Parallel-safe bytes.Buffer replacement
 * By J. Stuart McMurray
 * Created 20231215
 * Last Modified 20231215
 */

import (
	"bytes"
	"io"
	"sync"
)

// Buffer is a wrapper around bytes.Buffer with methods safe for parallel
// use.  A zero buffer is ready for use, but has a lock inside and should be
// passed around as a pointer.
type Buffer struct {
	buf bytes.Buffer
	l   sync.Mutex
}

func (b *Buffer) Available() int {
	b.l.Lock()
	defer b.l.Unlock()
	return b.buf.Available()
}

func (b *Buffer) AvailableBuffer() []byte {
	b.l.Lock()
	defer b.l.Unlock()
	return b.buf.AvailableBuffer()
}

func (b *Buffer) Bytes() []byte {
	b.l.Lock()
	defer b.l.Unlock()
	return b.buf.Bytes()
}

func (b *Buffer) Cap() int {
	b.l.Lock()
	defer b.l.Unlock()
	return b.buf.Cap()
}

func (b *Buffer) Grow(n int) {
	b.l.Lock()
	defer b.l.Unlock()
	b.buf.Grow(n)
}

func (b *Buffer) Len() int {
	b.l.Lock()
	defer b.l.Unlock()
	return b.buf.Len()
}

func (b *Buffer) Next(n int) []byte {
	b.l.Lock()
	defer b.l.Unlock()
	return b.buf.Next(n)
}

func (b *Buffer) Read(p []byte) (n int, err error) {
	b.l.Lock()
	defer b.l.Unlock()
	return b.buf.Read(p)
}

func (b *Buffer) ReadByte() (byte, error) {
	b.l.Lock()
	defer b.l.Unlock()
	return b.buf.ReadByte()
}

func (b *Buffer) ReadBytes(delim byte) (line []byte, err error) {
	b.l.Lock()
	defer b.l.Unlock()
	return b.buf.ReadBytes(delim)
}

func (b *Buffer) ReadFrom(r io.Reader) (n int64, err error) {
	b.l.Lock()
	defer b.l.Unlock()
	return b.buf.ReadFrom(r)
}

func (b *Buffer) ReadRune() (r rune, size int, err error) {
	b.l.Lock()
	defer b.l.Unlock()
	return b.buf.ReadRune()
}

func (b *Buffer) ReadString(delim byte) (line string, err error) {
	b.l.Lock()
	defer b.l.Unlock()
	return b.buf.ReadString(delim)
}

func (b *Buffer) Reset() {
	b.l.Lock()
	defer b.l.Unlock()
	b.buf.Reset()
}

func (b *Buffer) String() string {
	b.l.Lock()
	defer b.l.Unlock()
	return b.buf.String()
}

func (b *Buffer) Truncate(n int) {
	b.l.Lock()
	defer b.l.Unlock()
	b.buf.Truncate(n)
}

func (b *Buffer) UnreadByte() error {
	b.l.Lock()
	defer b.l.Unlock()
	return b.buf.UnreadByte()
}

func (b *Buffer) UnreadRune() error {
	b.l.Lock()
	defer b.l.Unlock()
	return b.buf.UnreadRune()
}

func (b *Buffer) Write(p []byte) (n int, err error) {
	b.l.Lock()
	defer b.l.Unlock()
	return b.buf.Write(p)
}

func (b *Buffer) WriteByte(c byte) error {
	b.l.Lock()
	defer b.l.Unlock()
	return b.buf.WriteByte(c)
}

func (b *Buffer) WriteRune(r rune) (n int, err error) {
	b.l.Lock()
	defer b.l.Unlock()
	return b.buf.WriteRune(r)
}

func (b *Buffer) WriteString(s string) (n int, err error) {
	b.l.Lock()
	defer b.l.Unlock()
	return b.buf.WriteString(s)
}

func (b *Buffer) WriteTo(w io.Writer) (n int64, err error) {
	b.l.Lock()
	defer b.l.Unlock()
	return b.buf.WriteTo(w)
}
