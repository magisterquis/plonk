package opshell

/*
 * colors.go
 * Color text
 * By J. Stuart McMurray
 * Created 20231130
 * Last Modified 20240119
 */

import (
	"net"
	"reflect"
	"strings"
	"testing"

	"golang.org/x/term"
)

func TestShellColor(t *testing.T) {
	have := "kittens"
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
			v := reflect.ValueOf(c.ec).Elem()
			rt := v.Type()
			for i := 0; i < rt.NumField(); i++ {
				fn := rt.Field(i).Name
				if "Reset" == rt.Field(i).Name {
					continue
				}
				cn := Color(strings.ToLower(fn))
				got := s.Color(cn, have)
				want := string(v.Field(i).Bytes()) +
					have +
					string(c.ec.Reset)
				if got != want {
					t.Errorf(
						"Incorrect color wrapping:\n"+
							"color: %s\n"+
							" have: %q\n"+
							"  got: %q\n"+
							" want: %q\n",
						cn,
						have,
						got,
						want,
					)
				}
			}

			fc := Color("fake_color")
			if got := s.Color(fc, have); got != have {
				t.Errorf(
					"Fake color changed string:\n"+
						"color: %s\n"+
						"  got: %q\n"+
						" want: %q",
					fc,
					got,
					have,
				)
			}
		})
	}
}
