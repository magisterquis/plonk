package subcom

/*
 * list.go
 * List registered subcommands
 * By J. Stuart McMurray
 * Created 20231020
 * Last Modified 20231128
 */

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"text/tabwriter"

	"golang.org/x/exp/maps"
)

// Specs returns the Specs added to c.  The caller owns the returned slice.
func (c *Cdr[T]) Specs() map[string]Spec[T] {
	c.l.RLock()
	defer c.l.RUnlock()
	return maps.Clone(c.specs)
}

// Table returns a formatted table of commands and descriptions, suitable for
// use in Usage statements.  The table will not end in a newline.  If c has no
// commands, an empty string is returned.
func (c *Cdr[T]) Table() string {
	c.l.RLock()
	defer c.l.RUnlock()

	/* Get the registered commands. */
	names := maps.Keys(c.specs)
	sort.Strings(names)

	/* Format nicely. */
	var buf bytes.Buffer
	tw := tabwriter.NewWriter(&buf, 0, 8, 1, ' ', 0)
	for _, name := range names {
		fmt.Fprintf(tw, "%s", name)
		/* Add the description if we have one. */
		if d := c.specs[name].Description; "" != d {
			fmt.Fprintf(tw, "\t-\t%s", d)
		}
		fmt.Fprintf(tw, "\n")
	}
	tw.Flush()

	/* Don't return with a newline. */
	return strings.TrimRight(buf.String(), "\r\n")
}
