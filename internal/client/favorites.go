package client

/*
 * favorites.go
 * Handler for favorites command
 * By J. Stuart McMurray
 * Created 20250503
 * Last Modified 20250503
 */

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"slices"
	"strings"
	"text/template"

	"github.com/magisterquis/plonk/lib/opshell"
	"github.com/magisterquis/simpleshsplit"
)

// DefaultTemplateName is what we call the default template.  This is used
// to select a template when ,f has no arguments.
const DefaultTemplateName = "default"

// favoritesHandler handles the ,f command.  It takes the arguments as a
// filename plus a list of key=value pairs, looks for the file in the favorites
// txtar archive, and passes the key=value pairs to the file as a text
// template.  The first argument without a = is taken to be the template name.
// If no template name is found, DefaultTemplateName is used.
func favoritesHandler(s shell, name, args []string) error {
	/* The args []string above is misleading.  It's always one string. */
	switch len(args) {
	case 0: /* Not a big deal. */
	case 1: /* Not actually split, obviously. */
		var err error
		if args, err = simpleshsplit.SplitGoUnquote(
			args[0],
		); nil != err {
			s.V().ErrorLogf("Error splitting arguments: %s", err)
			return nil
		}
	default: /* Shouldn't happen. */
		s.V().ErrorLogf(
			"BUG: Got %d arguments, expected <=1: %q",
			len(args),
			args,
		)
		return nil
	}

	/* Get command flags. */
	var (
		fset   = flag.NewFlagSet("favorites", flag.ContinueOnError)
		fsBuf  = new(bytes.Buffer)
		dryRun = fset.Bool(
			"dry-run",
			false,
			"Just print the tasking which would be sent",
		)
		file = fset.String(
			"file",
			s.V().Favorites,
			"Favorites template `file`",
		)
	)
	fset.SetOutput(fsBuf)
	fset.Usage = func() {
		fmt.Fprintf(
			fsBuf,
			`Usage: %s [options] [name] [key=value...]

Executes a favorites template and sends the result to the selected implant.
key=value pairs will be passed to the template.
A template other than the default may be chosen by passing a name, which must
not contain an =.

Options:
`,
			name[len(name)-1],
		)
		fset.PrintDefaults()
	}
	if err := fset.Parse(args); errors.Is(err, flag.ErrHelp) {
		s.ErrorWrite(fsBuf.Bytes())
		return nil
	} else if nil != err {
		s.V().ErrorLogf("Error parsing flags: %s", err)
		return nil
	}

	/* Work out which template to use and what to send it. */
	tName, tArgs, err := parseFavoritesArgs(fset.Args())
	if nil != err {
		s.V().ErrorLogf("Error parsing non-flag arguments: %s", err)
		return nil
	}

	/* Template the template to get a task to task. */
	task, err := executeFavoritesTemplate(*file, tName, tArgs)
	if errors.Is(err, fs.ErrNotExist) {
		s.V().ErrorLogf("Looks like %q doesn't exist", s.V().Favorites)
		return nil
	} else if nil != err {
		s.V().ErrorLogf("Error executing template: %s", err)
		return nil
	}

	/* If asked, we can just print what we would have sent. */
	if *dryRun {
		var nl string
		if !strings.HasSuffix(task, "\n") {
			nl = "\n"
		}
		s.Printf(
			s.V().color(
				opshell.ColorMagenta,
				"Would have enqueued the following "+
					"%d bytes:\n\n"+
					"%s%s",
			),
			len(task),
			task,
			nl,
		)
		return nil
	}

	/* Queue up the tasking. */
	enqueueTask(s, task)

	return nil
}

// parseFavoritesArgs parses the arguments to ,f.  It expects zero or more
// key=value pairs and zero or one string without an =.  The =-less string
// is returned along with the key=value pairs parsed into a map.
func parseFavoritesArgs(args []string) (string, map[string]string, error) {
	/* Work out the template name. */
	name := DefaultTemplateName
	if idx := slices.IndexFunc(args, func(s string) bool {
		return !strings.Contains(s, "=")
	}); -1 != idx {
		name = args[idx]
		args = slices.Delete(args, idx, idx+1)
	}

	/* Work out the key/value pairs. */
	kvs := make(map[string]string, len(args))
	for _, arg := range args {
		k, v, found := strings.Cut(arg, "=")
		if !found {
			return "", nil, fmt.Errorf(
				"found an extra template name: %s",
				arg,
			)
		}
		kvs[k] = v
	}

	return name, kvs, nil
}

// executefavoritesTemplate parses templates from file, parses the args as
// key=value pairs, and executes the template with the given name.
func executeFavoritesTemplate(file, name string, data any) (string, error) {
	/* Grab the template. */
	raw, err := os.ReadFile(file)
	if nil != err {
		return "", fmt.Errorf("reading %s: %w", file, err)
	}
	if 0 == len(bytes.TrimSpace(raw)) {
		return "", errors.New("empty file")
	}
	tmpl, err := template.
		New(DefaultTemplateName).
		Option("missingkey=zero").
		Parse(string(raw))
	if nil != err {
		return "", fmt.Errorf("parsing templates: %w", err)
	}

	/* Put them together and send them back. */
	b := new(bytes.Buffer)
	if err := tmpl.ExecuteTemplate(b, name, data); nil != err {
		return "", fmt.Errorf("executing template %q: %s", name, err)
	}

	return b.String(), nil
}
