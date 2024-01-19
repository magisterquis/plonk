package opshell

/*
 * opshell_test.go
 * Operator's interactive shell
 * By J. Stuart McMurray
 * Created 20231112
 * Last Modified 20240119
 */

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"os"
	"reflect"
	"testing"

	"golang.org/x/term"
)

func testSetDefaultHelper[T comparable](
	name string,
	t *testing.T,
	have T,
	def T,
	want T,
) {
	t.Run(name, func(t *testing.T) {
		got := have
		setDefault(&got, def)
		if got != want {
			t.Errorf("got:%v want:%v", got, want)
		}
	})
}

func TestSetDefault(t *testing.T) {
	testSetDefaultHelper("string/set", t, "kittens", "moose", "kittens")
	testSetDefaultHelper("string/def", t, "", "kittens", "kittens")
	testSetDefaultHelper("stdin/def", t, os.Stdin, os.Stdout, os.Stdin)
	testSetDefaultHelper("stdin/def", t, nil, os.Stdout, os.Stdout)
}

func TestConfigDefaults(t *testing.T) {
	var conf Config[struct{}]
	setConfigDefaults(&conf)
	if os.Stdin != conf.Reader {
		t.Errorf("Reader set failed: got:%v", conf.Reader)
	}
	if os.Stdout != conf.Writer {
		t.Errorf("Writer set failed: got:%v", conf.Writer)
	}
	if os.Stderr != conf.ErrorWriter {
		t.Errorf("ErrorWriter set failed: got:%v", conf.ErrorWriter)
	}
	if DefaultPrompt != conf.Prompt {
		t.Errorf("Prompt set failed: got:%v", conf.Prompt)
	}
}

func newTestShell[T any](t *testing.T, v T) (
	w *io.PipeWriter,
	ob *bytes.Buffer,
	eb *bytes.Buffer,
	s *Shell[T],
) {
	pr, pw := io.Pipe()
	w = pw
	ob = new(bytes.Buffer)
	eb = new(bytes.Buffer)
	var err error
	s, err = Config[T]{
		Reader:      pr,
		Writer:      ob,
		ErrorWriter: eb,
	}.New()
	s.SetV(v)
	if nil != err {
		t.Fatalf("Error generating shell: %s", err)
	}
	return
}

func TestShellReadLine(t *testing.T) {
	l1 := "moose"
	l2 := "kittens"
	w, ob, eb, s := newTestShell(t, 1)
	go fmt.Fprintf(w, "%s\n%s\n", l1, l2)
	for i, want := range []string{l1, l2} {
		if got, err := s.ReadLine(); nil != err {
			t.Fatalf("Error reading line %d: %s", i+1, err)
		} else if got != want {
			t.Fatalf(
				"Line %d inconnect: got:%q want:%q",
				i+1,
				got,
				want,
			)
		}
	}
	if 0 != ob.Len() {
		t.Errorf("Output buffer not empty: %q", ob.String())
	}
	if 0 != eb.Len() {
		t.Errorf("Error buffer not empty: %q", eb.String())
	}
}

type testCtx struct {
	n int
}

func TestShellV(t *testing.T) {
	_, _, _, s := newTestShell(t, &testCtx{})
	want := testCtx{}
	if got := *s.V(); got != want {
		t.Errorf(
			"Incorrect initial context\n got: %#v\nwant: %#v",
			got,
			want,
		)
	}
	have := 10
	want = testCtx{n: 10}
	s.V().n = have
	if got := *s.V(); got != want {
		t.Errorf(
			"Incorrect context after set n\n"+
				"have: %#v\n"+
				" got: %#v\n"+
				"want: %#v",
			have,
			got,
			want,
		)
	}

	want = testCtx{n: 20}
	s.SetV(&want)
	if got := *s.V(); got != want {
		t.Errorf(
			"Incorrect context after SetV:\n"+
				" got: %#v\n"+
				"want: %#v\n",
			got,
			want,
		)
	}
}

func TestEscapeCodes(t *testing.T) {
	c1, c2 := net.Pipe()
	defer c1.Close()
	defer c2.Close()
	for _, c := range []struct {
		name string
		conf Config[int]
		ec   *term.EscapeCodes
	}{{
		name: "with_TTY",
		conf: Config[int]{
			Reader:      c1,
			Writer:      c1,
			ErrorWriter: c1,
			PTYMode:     PTYForce,
		},
		ec: term.NewTerminal(c1, "").Escape,
	}, {
		name: "no_TTY",
		conf: Config[int]{
			Reader:      c1,
			Writer:      c1,
			ErrorWriter: c1,
			PTYMode:     PTYDisable,
		},
		ec: new(term.EscapeCodes),
	}} {
		c := c /* :C */
		t.Run(c.name, func(t *testing.T) {
			s, err := c.conf.New()
			if nil != err {
				t.Fatalf("Error creating shell: %s", err)
			}
			if !reflect.DeepEqual(s.Escape(), c.ec) {
				t.Errorf("Escape codes not equal")
			}
		})
	}
}
