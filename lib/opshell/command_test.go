package opshell

/*
 * command.go
 * Handle command execution
 * By J. Stuart McMurray
 * Created 20231112
 * Last Modified 20231206
 */

import (
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"
	"testing"
)

type sctt = *Shell[map[string]string]

func TestShellHandleCommands(t *testing.T) {
	_, ob, eb, s := newTestShell(t, map[string]string{})
	s.Cdr().Add(
		"c1",
		"d1",
		func(s sctt, name, args []string) error {
			s.V()["k1"] = "v1"
			s.Printf("H1: Name:%s Args:%s", name, args)
			return nil
		},
	)
	s.Cdr().Add(
		"c2",
		"d2",
		func(s sctt, name, args []string) error {
			s.V()["k2"] = "v2"
			s.Printf("H2: Name:%s Args:%s", name, args)
			return nil
		},
	)

	for _, c := range []struct {
		have string
		want string
		ctxk string
		ctxv string
	}{{
		have: "c1 a1 a2 a3",
		want: "H1: Name:[c1] Args:[a1 a2 a3]",
		ctxk: "k1",
		ctxv: "v1",
	}, {
		have: "c2 a1 a2 a3",
		want: "H2: Name:[c2] Args:[a1 a2 a3]",
		ctxk: "k2",
		ctxv: "v2",
	}} {
		c := c /* :C */
		if !t.Run(c.have, func(t *testing.T) {
			if err := s.HandleCommand(c.have); nil != err {
				t.Errorf("Error handling command: %s", err)
			}
			if 0 != eb.Len() {
				t.Errorf("Error output: %q", eb)
			}
			defer ob.Reset()
			if got := ob.String(); got != c.want {
				t.Errorf(
					"Output incorrect:\n"+
						" got: %s\n"+
						"want: %s",
					got,
					c.want,
				)
			}
			if v, ok := s.V()[c.ctxk]; !ok {
				t.Errorf("Context did not have %q set", c.ctxk)
			} else if v != c.ctxv {
				t.Errorf(
					"Context[%s] incorrect:\n"+
						" got: %q\n"+
						"want: %q",
					c.ctxk,
					v,
					c.ctxv,
				)
			}
		}) {
			break
		}
	}

}

func TestHandleCommand_SplitError(t *testing.T) {
	ctx := make(map[string]string)
	_, _, _, s := newTestShell(t, ctx)
	err := s.HandleCommand(`foo"bar`)
	var pe SplitError
	if !errors.As(err, &pe) {
		t.Fatalf("Expected ParseError, got %T (%s)", err, err)
	}
}

func TestHandleCommands_Error(t *testing.T) {
	commands := []string{"foo bar tridge", "abc\""}
	for _, c := range []struct {
		name  string
		eh    func(*Shell[int], string, error) error
		owant string
		ewant string
	}{{
		name:  "default",
		owant: "",
		ewant: `Error handling command "foo bar tridge": command not found
Error handling command "abc\"": split error: missing terminating "
`,
	}, {
		name: "custom handler",
		eh: func(s *Shell[int], l string, err error) error {
			s.Printf("l: %s\n", l)
			s.Errorf("e: %s\n", err)
			return nil
		},
		owant: "l: foo bar tridge\nl: abc\"\n",
		ewant: "e: command not found\n" +
			"e: split error: missing terminating \"\n",
	}} {
		c := c /* :( */
		t.Run(c.name, func(t *testing.T) {
			//t.Parallel()
			w, ob, eb, s := newTestShell(t, 1)
			s.SetCommandErrorHandler(c.eh)
			go func() {
				for _, c := range commands {
					fmt.Fprintf(w, "%s\n", c)
				}
				w.Close()
			}()
			if err := s.HandleCommands(); !errors.Is(err, io.EOF) {
				t.Errorf(
					"HandleCommands returned "+
						"incorrect error: %s",
					err,
				)
			}

			if got := ob.String(); got != c.owant {
				t.Errorf(
					"Output incorrect\n"+
						" got: %s\n"+
						"want: %s",
					got,
					c.owant,
				)
			}
			if got := eb.String(); got != c.ewant {
				t.Errorf(
					"Error output incorrect\n"+
						" got: %q\n"+
						"want: %q",
					got,
					c.ewant,
				)
			}
		})
	}
}

func TestHandleCommands_UnknownHandler(t *testing.T) {
	for _, c := range []struct {
		line  string
		owant string
		ewant string
		h     func(*Shell[int], string, error) error
	}{{
		line: "no unknown handler",
		ewant: `Error handling command "no unknown handler": ` +
			`command not found` + "\n",
	}, {
		line: "unknown handler set",
		h: func(s *Shell[int], line string, err error) error {
			s.Printf("Line: %q\n", line)
			return nil
		},
		owant: `Line: "unknown handler set"` + "\n",
	}} {
		c := c /* :( */
		t.Run(c.line, func(t *testing.T) {
			t.Parallel()
			w, ob, eb, s := newTestShell(t, 1)
			s.SetCommandErrorHandler(c.h)
			ech := make(chan error)
			go func() { ech <- s.HandleCommands() }()
			fmt.Fprintf(w, "%s\n", c.line)
			w.Close()
			if err := <-ech; !errors.Is(err, io.EOF) {
				t.Fatalf("Error handling commands: %s", err)
			}
			if got := ob.String(); got != c.owant {
				t.Errorf(
					"Incorrect output:\n got: %q\nwant: %q",
					got,
					c.owant,
				)
			}
			if got := eb.String(); got != c.ewant {
				t.Errorf(
					"Incorrect error:\n got: %q\nwant: %q",
					got,
					c.ewant,
				)
			}
		})
	}
}

func TestSetSplitter(t *testing.T) {
	cutSplitter := func(s string) ([]string, error) {
		b, a, found := strings.Cut(s, " ")
		ret := []string{b}
		if found {
			ret = append(ret, a)
		}
		return ret, nil
	}
	for _, c := range []struct {
		line string
		want []string
		f    func(string) ([]string, error)
		cmd  string
	}{{
		line: "foo bar tridge",
		want: []string{"foo", "bar", "tridge"},
		cmd:  "foo",
	}, {
		line: "foo `bar tridge`",
		want: []string{"foo", "bar tridge"},
		cmd:  "foo",
	}, {
		line: "foo `bar tridge`",
		want: []string{"foo", "`bar tridge`"},
		f:    cutSplitter,
		cmd:  "foo",
	}, {
		line: "foo bar tridge",
		want: []string{"foo", "bar tridge"},
		f:    cutSplitter,
		cmd:  "foo",
	}} {
		c := c /* :( */
		t.Run(c.line, func(t *testing.T) {
			_, _, _, s := newTestShell(t, 0)
			var got []string
			s.Cdr().Add(
				c.cmd,
				"",
				func(_ *Shell[int], name, args []string) error {
					got = append(name, args...)
					return nil
				},
			)
			s.SetSplitter(c.f)
			if err := s.HandleCommand(c.line); nil != err {
				t.Fatalf("Handle error: %s", err)
			}
			if !slices.Equal(got, c.want) {
				t.Fatalf(
					"Split failed\n got: %q\nwant: %q",
					got,
					c.want,
				)
			}
		})
	}
}

func TestCutCommand(t *testing.T) {
	for _, c := range []struct {
		have string
		want []string
	}{{
		have: "foo bar tridge",
		want: []string{"foo", "bar tridge"},
	}, {
		have: "   foo  \t\n  bar tridge  \n",
		want: []string{"foo", "bar tridge  \n"},
	}, {
		have: "foo",
		want: []string{"foo"},
	}, {
		have: "foo       ",
		want: []string{"foo"},
	}, {
		have: "",
		want: []string{},
	}, {
		have: "     \n   \t   ",
		want: []string{},
	}} {
		c := c /* :* */
		t.Run(c.have, func(t *testing.T) {
			got, err := CutCommand(c.have)
			if nil != err {
				t.Fatalf("Error: %s", err)
			}
			if slices.Equal(got, c.want) {
				return
			}
			t.Fatalf(
				"Split failed:\n"+
					"have: %q\n got: %q\nwant: %q",
				c.have,
				got,
				c.want,
			)
		})
	}
}

func ExampleCutCommand() {
	parts, err := CutCommand("foo bar tridge")
	if nil != err {
		panic(err)
	}

	for _, v := range parts {
		fmt.Println(v)
	}

	//output:
	// foo
	// bar tridge
}
