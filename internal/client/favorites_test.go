package client

/*
 * favorites_test.go
 * Tests for favorites.go
 * By J. Stuart McMurray
 * Created 20250503
 * Last Modified 20250503
 */

import (
	"maps"
	"os"
	"path/filepath"
	"testing"
)

func eq[T comparable](a, b T) bool { return a == b }

// check calls t.Errorf if equal(got, want) returns false.
func check[T any](
	t *testing.T,
	what string,
	have any,
	got T,
	want T,
	equal func(T, T) bool,
) {
	t.Helper()
	if equal(got, want) {
		return
	}
	t.Errorf(
		"Incorrect %s:\n"+
			"have: %v\n"+
			" got: %v\n"+
			"want: %v",
		what,
		have,
		got,
		want,
	)
}

func TestParseFavoritesArgs(t *testing.T) {
	for n, c := range map[string]struct {
		have []string
		name string
		args map[string]string
		werr string
	}{"empty": {
		have: []string{},
		name: DefaultTemplateName,
	}, "everything": {
		have: []string{"kittens", "foo=bar", "moose=zoomies"},
		name: "kittens",
		args: map[string]string{"foo": "bar", "moose": "zoomies"},
	}, "two_templates": {
		have: []string{"kittens", "foo=bar", "moose"},
		werr: "found an extra template name: moose",
	}, "no_args": {
		have: []string{"kittens"},
		name: "kittens",
	}, "no_name": {
		have: []string{"kittens=zoomies", "foo=moose"},
		name: DefaultTemplateName,
		args: map[string]string{"kittens": "zoomies", "foo": "moose"},
	}} {
		t.Run(n, func(t *testing.T) {
			name, args, err := parseFavoritesArgs(c.have)
			if "" == c.werr && nil != err {
				t.Fatalf("Unexpected error: %s", err)
			} else if "" != c.werr && nil == err {
				t.Fatalf(
					"Unexpected success (expected %s)",
					err,
				)
			}
			check(t, "name", c.have, name, c.name, eq)
			check(t, "args", c.have, args, c.args, maps.Equal)
			if nil != err {
				check(
					t,
					"error",
					c.have,
					err.Error(),
					c.werr,
					eq,
				)
			}
		})
	}
}

// func executeFavoritesTemplate(file, name string, data any) (string, error) {
func TestExecuteFavoritesTemplate(t *testing.T) {
	for n, c := range map[string]struct {
		tmpl string
		name string
		data any
		want string
	}{"simple": {
		tmpl: "kittens",
		name: DefaultTemplateName,
		want: "kittens",
	}, "named_template": {
		tmpl: `kittens{{define "moose"}}worky{{end}}`,
		name: "moose",
		want: "worky",
	}, "with_args": {
		tmpl: `kittens{{define "moose"}}worky{{.foo}}{{end}}`,
		name: "moose",
		data: map[string]string{"foo": "bar"},
		want: "workybar",
	}} {
		t.Run(n, func(t *testing.T) {
			fn := filepath.Join(t.TempDir(), "tmpl")
			if err := os.WriteFile(
				fn,
				[]byte(c.tmpl),
				0600,
			); nil != err {
				t.Fatalf(
					"Error writing template to %s: %s",
					fn,
					err,
				)
			}
			if nil == c.data {
				c.data = make(map[string]string)
			}
			got, err := executeFavoritesTemplate(
				fn,
				c.name,
				c.data,
			)
			if nil != err {
				t.Fatalf("Error executing template: %s", err)
			}
			if got != c.want {
				t.Fatalf(
					"Incorrect template output:\n"+
						" got: %q\n"+
						"want: %q",
					got,
					c.want,
				)
			}
		})
	}
}
