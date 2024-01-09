package subcom

/*
 * call.go
 * Call a subcommand
 * By J. Stuart McMurray
 * Created 20231020
 * Last Modified 20231128
 */

import (
	"errors"
	"slices"
)

var (
	// ErrEmptyArgs is returned if an empty args slice was passed to
	// Cdr.Call.
	ErrEmptyArgs = errors.New("empty args")

	// ErrNotFound is returned by Cdr.Call if args[0] didn't have a
	// registered Handler.
	ErrNotFound = errors.New("command not found")
)

// Call calls a command registered with the Subcommander.  The command name is
// assumed to be args[0]; the name will be appended to parents and removed from
// he args passed to the CommandHandler.  If the comand wasn't found,
// ErrNotFound is returned.  Parents will not be modified by Call, at the cost
// of an allocation and copy.  A subslice of args will be passed to the
// CommandHandler; it will never be nil.
func (c *Cdr[T]) Call(ctx T, parents []string, args []string) error {
	/* Make sure we actually have a command name. */
	if 0 == len(args) {
		return ErrEmptyArgs
	}

	/* Get the command to call. */
	name := args[0]
	c.l.RLock()
	spec, ok := c.specs[name]
	c.l.RUnlock()
	if !ok {
		return ErrNotFound
	}

	/* Update the names and return a function which will call it. */
	cn := append(slices.Clone(parents), name)
	return spec.Handler(ctx, cn, args[1:])
}
