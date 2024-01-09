// Package flexiwriter - Like io.MultiWriter, but with removable writers
package flexiwriter

/*
 * flexiwriter.go
 * Like io.MultiWriter, but with removable writers
 * By J. Stuart McMurray
 * Created 20231006
 * Last Modified 20231208
 */

import (
	"io"
	"sync"
)

// Writer is like io.MultiWriter, but with dynamic addition and removal
// of writers.  Writer's are safe to be called concurrently from multiple
// goroutines.
type Writer struct {
	l  sync.Mutex
	ws map[io.Writer]func(error)
}

// New returns a new Writer, ready for use.  Any writers passed to new will
// be added to the returned Writer as if pased to Add with a nil onremove.
func New(ws ...io.Writer) *Writer {
	fw := &Writer{ws: make(map[io.Writer]func(error))}
	for _, w := range ws {
		fw.Add(w, nil)
	}
	return fw
}

// WriteString is a convenience wrapper around fw.Write which writes s.
func (fw *Writer) WriteString(s string) { fw.Write([]byte(s)) }

// Write satisfies io.Writer.  If any of fw's writers returns an error, its
// onremove function is called and it is removed.  The size of p and nil are
// returned.  Write blocks until all of the underlying writes have finished.
func (fw *Writer) Write(p []byte) (n int, err error) {
	fw.l.Lock()
	defer fw.l.Unlock()

	/* Write to ALL the writers. */
	var (
		wg   sync.WaitGroup
		torm sync.Map
	)
	for w, or := range fw.ws {
		wg.Add(1)
		go func(w io.Writer, or func(error)) {
			defer wg.Done()
			/* Try to write to this writer. */
			_, err := w.Write(p)
			if nil == err { /* All worked. */
				return
			}

			/* Fail.  Let someone know and note that the writer
			should be removed. */
			if nil != or {
				go or(err)
			}
			torm.Store(w, true)
		}(w, or)
	}

	/* Wait for the write to finish. */
	wg.Wait()

	/* If we have anything to remove, remove it. */
	torm.Range(func(v, _ any) bool {
		delete(fw.ws, v.(io.Writer))
		return true
	})

	return len(p), nil
}

// Add adds a new writer to fw.  On removal, onremove wil be called in a
// separate goroutine if not nil and passed any error which caused the removal.
// If onremove was called as a result of calling fw.Remove, onremove will be
// passed nil.  If w has already been added to fw, w's existing onremove will
// set to the onremove passed to Add.  Adding a nil io.Writer is a no-op.
func (fw *Writer) Add(w io.Writer, onremove func(err error)) {
	/* Don't bother trying to add a nil writer. */
	if nil == w {
		return
	}

	fw.l.Lock()
	defer fw.l.Unlock()
	fw.ws[w] = onremove
}

// Remove removes a writer from fw.  If the writer was added with a non-nil
// onremove function, it will be called and passed nil.  If w isn't in fw,
// as can happen if the writer is removed by fw.Write, Remove is a no-op.
// Remove returns true if w was in fw.
func (fw *Writer) Remove(w io.Writer) bool {
	fw.l.Lock()
	defer fw.l.Unlock()

	/* Get the onremove function for this writer. */
	onremove, ok := fw.ws[w]
	if !ok {
		return false
	}

	/* Remove the writer and tell someone it's removed. */
	delete(fw.ws, w)
	if nil != onremove {
		go onremove(nil)
	}

	return true
}
