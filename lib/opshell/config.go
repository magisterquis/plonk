package opshell

/*
 * config.go
 * Config to create a Shell
 * By J. Stuart McMurray
 * Created 20231128
 * Last Modified 20240119
 */

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"reflect"

	"github.com/magisterquis/plonk/lib/subcom"
	"golang.org/x/term"
)

// DefaultPrompt is the default prompt used if none is specified in a Config.
const DefaultPrompt = "> "

// PTYConfig is used in Config to indicate whether or not to enable PTY mode.
// See the documentation for Config.PTY for more details.
type PTYConfig int

const (
	PTYDefault PTYConfig = iota /* Check if Config.Reader is a Terminal. */
	PTYForce                    /* Always enable PTY mode */
	PTYDisable                  /* Never enable PTY mode */
)

// Config is the configuration needed to create a Shell.  See the documentation
// for Shell for more information.  The shell referred to in Config's fields'
// documentation is a shell created using Config.New.
type Config[T any] struct {
	// Reader is from where the shell draws its input.  If unset, os.Stdin
	// will be used.
	Reader io.Reader
	// Writer is to where the shell sends its output.  If unset, os.Stdout
	// will be used.
	Writer io.Writer
	// ErrWriter is where the shell sends error output via Shell.Errorf and
	// friends when not in PTY mode.
	ErrorWriter io.Writer

	// PTYMode determines whether or not shells created with this config
	// use PTY mode, i.e. golang.org/x/term.Terminals under the hood.  If
	// unset or set to PTYDefault, PTY mode is enabled if Reader is
	// a terminal, as determined by golang.org/x/term.IsTerminal.  When in
	// PTY mode, ErrorWriter is ignored and all output goes to Writer.
	PTYMode PTYConfig

	// Prompt sets the initial prompt.  If unset, DefaultPrompt is used.
	Prompt string
}

// New creates a new shell according to c.
func (c Config[T]) New() (*Shell[T], error) {
	/* Set config defaults. */
	setConfigDefaults(&c)

	/* Shell to return, with default functions. */
	s := &Shell[T]{cdr: subcom.New[*Shell[T]](nil)}
	s.SetSplitter(nil)
	s.SetCommandErrorHandler(nil)

	/* Try to get the input as a terminal.  We'll use this to determine if
	we need PTY mode, as well as putting it in raw mode. */
	var (
		fd     int = -1
		isTerm bool
	)
	if fder, ok := c.Reader.(interface{ Fd() uintptr }); ok {
		fd = int(fder.Fd())
		isTerm = term.IsTerminal(fd)
	}

	/* work out whether we should be in PTY mode. */
	var bePTY bool
	switch c.PTYMode {
	case PTYDefault:
		bePTY = isTerm
	case PTYForce:
		bePTY = true
	case PTYDisable:
		bePTY = false
	default:
		return nil, fmt.Errorf("unknown PTYMode %s", c.PTYMode)
	}

	/* If we're not in PTY mode, life's easy. */
	if !bePTY {
		if err := newNotPTY(c, s); nil != err {
			return nil, err
		}
		return s, nil
	}

	/* Use a terminal for our underlying I/O. */
	t := term.NewTerminal(
		readWriter{r: c.Reader, w: c.Writer},
		c.Prompt,
	)
	s.stdout = t
	s.stderr = t
	s.logger = log.New(t, "", log.LstdFlags)
	s.errlogger = s.logger
	s.readLine = t.ReadLine
	s.setPrompt = t.SetPrompt
	s.setSize = t.SetSize
	s.escapeCodes = func() *term.EscapeCodes { return t.Escape }
	s.isPTY = true

	/* If we have a terminal, try to put it in raw mode. */
	if isTerm {
		/* We're in a terminal.  Be in raw mode. */
		st, err := term.MakeRaw(fd)
		if nil != err {
			return nil, fmt.Errorf(
				"putting fd %d in raw mode: %w",
				fd,
				err,
			)
		}

		/* Save an unrawing function for later. */
		s.resetL.Lock()
		s.reset = func() error { return term.Restore(fd, st) }
		s.resetL.Unlock()

		/* Start handling terminal resizing as well. */
		if err := s.resizeTTY(fd); nil != err {
			return nil, fmt.Errorf(
				"setting initial terminal size on fd %d: %w",
				fd,
				err,
			)
		}
		go s.handleSIGWINCH(fd)
	}

	return s, nil
}

// newNotPTY returns a new Shell which isn't using a PTY.
func newNotPTY[T any](conf Config[T], s *Shell[T]) error {
	/* Output. */
	s.stdout = conf.Writer
	s.stderr = conf.ErrorWriter
	s.logger = log.New(conf.Writer, "", log.LstdFlags)
	s.errlogger = log.New(conf.ErrorWriter, "", log.LstdFlags)

	/* Input. */
	reader := bufio.NewReader(conf.Reader)
	s.readLine = func() (string, error) {
		var s []byte
		for {
			/* Get whatever chunk of line we can. */
			line, prefix, err := reader.ReadLine()
			s = append(s, line...)
			if nil != err {
				return string(line), err
			}
			/* If we've finished the line, we're done. */
			if !prefix {
				return string(s), nil
			}
		}
	}

	/* No-ops for PTY things. */
	s.setPrompt = func(string) {}
	s.setSize = func(int, int) error { return nil }
	ec := new(term.EscapeCodes)
	s.escapeCodes = func() *term.EscapeCodes { return ec }

	return nil
}

// setDefault sets a default in the config.  *f is set to def if f points to
// the zero value for its type.
func setDefault[T any](f *T, def T) {
	if reflect.ValueOf(f).Elem().IsZero() {
		*f = def
	}
}

// setConfigDefaults sets defaults for a Config.
func setConfigDefaults[T any](conf *Config[T]) {
	setDefault(&conf.Reader, io.Reader(os.Stdin))
	setDefault(&conf.Writer, io.Writer(os.Stdout))
	setDefault(&conf.ErrorWriter, io.Writer(os.Stderr))
	setDefault(&conf.Prompt, DefaultPrompt)
}

//go:generate stringer -type PTYConfig -trimprefix PTY
