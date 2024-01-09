package subcom

/*
 * list_test.go
 * Tests for list.go
 * By J. Stuart McMurray
 * Created 20231020
 * Last Modified 20231216
 */

import (
	"strings"
	"testing"
)

func TestCommands(t *testing.T) {
	h := func(_ int, _, _ []string) error { return nil }
	specs := []Spec[int]{
		{Name: "h1", Description: "d1", Handler: h},
		{Name: "h2", Description: "d2", Handler: h},
		{Name: "h3", Description: "d3", Handler: h},
		{Name: "h4", ArgHelp: "a4", Description: "d4", Handler: h},
		{Name: "h5", ArgHelp: "a5", Description: "d5", Handler: h},
	}
	cdr := New(specs)
	cdr.Add("h6", "a6", "d6", h)
	specs = append(specs, Spec[int]{"h6", "a6", "d6", h})

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
		if d.ArgHelp != spec.ArgHelp {
			t.Errorf(
				"Incorrect argument help for command %q\n"+
					" got: %s\n"+
					"want: %s",
				spec.Name,
				d.ArgHelp,
				spec.ArgHelp,
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
		{Name: "h5", ArgHelp: "a5", Description: "d5", Handler: h},
		{Name: "h6", ArgHelp: "", Description: "d6", Handler: h},
		{Name: "h7", ArgHelp: "a7", Description: "", Handler: h},
		{Name: "h8yyyy", ArgHelp: "a8xxx", Description: "d8", Handler: h},
	}
	want := strings.Trim(`
h1           - d1
h2a            
h2xxx        - d2
h3           - d3
h4             
h5 a5        - d5
h6           - d6
h7 a7          
h8yyyy a8xxx - d8
h9 a9        - d9
`, "\n")
	cdr := New(specs)
	cdr.Add("h9", "a9", "d9", h)
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
