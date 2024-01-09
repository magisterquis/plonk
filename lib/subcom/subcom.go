// Package subcom is a somewhat easy subcommand runner.
package subcom

/*
 * subcom.go
 * Relatively easy subcommand execution.
 * By J. Stuart McMurray
 * Created 20230812
 * Last Modified 20231128
 */

import (
	"sync"
)

// Handler is what's called when a cammand is executed.
type Handler[T any] func(ctx T, name, args []string) error

// Spec is used to add a command to a handler.
type Spec[T any] struct {
	Name        string     /* Command's callable name. */
	Description string     /* One-line description. */
	Handler     Handler[T] /* Nil to delete a handler. */
}

// Cdr holds and calls (sub)commands.  Cdr's methods are safe for concurrent
// use by multiple goroutines.
type Cdr[T any] struct {
	l     sync.RWMutex
	specs map[string]Spec[T]
}

// New returns a new Subcommander, ready for use.  If any Specs are provided
// the new Cdr will contain them.  specs may be nil.
func New[T any](specs []Spec[T]) *Cdr[T] {
	cdr := &Cdr[T]{specs: make(map[string]Spec[T])}
	AddSpecs(cdr, specs)
	return cdr
}

// Add adds a single Handler to c.  If handler is nil and c has a Handler with
// the given name, the Handler will be deleted.
func (c *Cdr[T]) Add(name, description string, handler Handler[T]) {
	c.l.Lock()
	defer c.l.Unlock()

	/* If we're deleting, life's easy. */
	if nil == handler {
		delete(c.specs, name)
		return
	}

	/* Save this handler spec. */
	c.specs[name] = Spec[T]{
		Name:        name,
		Description: description,
		Handler:     handler,
	}
}

// AddSpecs adds the given commands handlers to s, possibly overwriting
// existing handlers.  specs may be nil.
func AddSpecs[T any](c *Cdr[T], specs []Spec[T]) {
	for _, spec := range specs {
		c.Add(spec.Name, spec.Description, spec.Handler)
	}
}
