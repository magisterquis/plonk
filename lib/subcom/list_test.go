package subcom

/*
 * list_test.go
 * Tests for list.go
 * By J. Stuart McMurray
 * Created 20231020
 * Last Modified 20231128
 */

import (
	"testing"
)

func TestCommands(t *testing.T) {
	h := func(_ int, _, _ []string) error { return nil }
	specs := []Spec[int]{
		{Name: "h1", Description: "d1", Handler: h},
		{Name: "h2", Description: "d2", Handler: h},
		{Name: "h3", Description: "d3", Handler: h},
	}
	cdr := New(specs)

	cds := cdr.Specs()
	for _, spec := range specs {
		d, ok := cds[spec.Name]
		if !ok {
			t.Errorf("Command %q not found", spec.Name)
			continue
		}
		if d.Description != spec.Description {
			t.Errorf(
				"Incorrect description for command %q\n"+
					" got: %s\n"+
					"want: %s",
				spec.Name,
				d.Description,
				spec.Description,
			)
		}
		delete(cds, spec.Name)
	}

	for c, d := range cds {
		t.Errorf(
			"Got extraneous command %s (description %s)",
			c,
			d.Description,
		)
	}
}

func TestTable(t *testing.T) {
	h := func(_ int, _, _ []string) error { return nil }
	specs := []Spec[int]{
		{Name: "h1", Description: "d1", Handler: h},
		{Name: "h2a", Handler: h},
		{Name: "h2xxx", Description: "d2", Handler: h},
		{Name: "h3", Description: "d3", Handler: h},
		{Name: "h4", Description: "", Handler: h},
	}
	want := `h1 - d1
h2a
h2xxx - d2
h3    - d3
h4`
	cdr := New(specs)
	got := cdr.Table()
	if got != want {
		t.Errorf("Incorrect table:\n"+
			" got:\n%s\n"+
			"want:\n%s",
			got,
			want,
		)
	}
}
