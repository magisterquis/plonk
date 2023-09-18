package clgen

/*
 * load.go
 * (Re)load the template
 * By J. Stuart McMurray
 * Created 20230726
 * Last Modified 20230911
 */

import (
	_ "embed"
	"errors"
	"fmt"
	"os"
	"text/template"
)

// embeddedTemplateName is the name we use for our template if we've used the
// embedded version.
const embeddedTemplateName = "emdedded_template"

// embeddedTemplate is the template to use when we don't have a file with one
// and need to make one.
//
//go:embed clgen.tmpl
var embeddedTemplate []byte

// Template returns the raw, pre-parsed template.  If the embedded template was
// used, the bool is true.
func Template() ([]byte, bool, error) {
	var embedded bool

	/* Get template contents, and ensure we've got them in a file. */
	tb, err := os.ReadFile(TemplateFile)
	if nil != err {
		/* Not having a file isn't a real error. */
		if errors.Is(err, os.ErrNotExist) {
			tb = embeddedTemplate
			embedded = true
		} else {
			return nil, false, fmt.Errorf(
				"reading from %s: %w",
				TemplateFile,
				err,
			)
		}
	}

	return tb, embedded, nil
}

// loadTemplate loads a template from TemplateFile, or from defaultTemplate if
// TemplateFile doesn't exist
func loadTemplate() (*template.Template, error) {
	/* Load the template. */
	tb, embedded, err := Template()
	if nil != err {
		return nil, fmt.Errorf("loading template: %w", err)
	}

	/* Work out its name. */
	name := embeddedTemplateName
	if !embedded {
		name = TemplateFile
	}

	/* Parse the template. */
	tmpl, err := template.New(name).Parse(string(tb))
	if nil != err {
		return nil, fmt.Errorf("parsing %s: %w", name, err)
	}

	return tmpl, nil
}
