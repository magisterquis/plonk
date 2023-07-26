package clgen

/*
 * load.go
 * (Re)load the template
 * By J. Stuart McMurray
 * Created 20230726
 * Last Modified 20230726
 */

import (
	_ "embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"text/template"
)

// defaultTemplate is the template to use when we don't have a file with one
// and need to make one.
//
//go:embed clgen.tmpl
var defaultTemplate []byte

// loadTemplate loads a template from TemplateFile, from defaultTemplate if
// TemplateFile doesn't exist, which also will cause TemplateFile to be
// created.  loadTemplate must not be called until Init returns.
func loadTemplate() error {
	cTemplateL.Lock()
	defer cTemplateL.Unlock()

	/* Get template contents, and ensure we've got them in a file. */
	tb, err := os.ReadFile(TemplateFile)
	if nil != err {
		/* Not having a file isn't a real error. */
		if errors.Is(err, os.ErrNotExist) {
			tb = defaultTemplate
			if err := os.WriteFile(
				TemplateFile,
				tb,
				0660,
			); nil != err {
				return fmt.Errorf(
					"writing default template to %s: %s",
					TemplateFile,
					err,
				)
			}
		} else {
			return fmt.Errorf(
				"reading from %s: %w",
				TemplateFile,
				err,
			)
		}
	}

	/* Parse the template. */
	tmpl, err := template.New(
		filepath.Base(TemplateFile),
	).Parse(string(tb))
	if nil != err {
		return fmt.Errorf(
			"parsing template from %s: %w",
			TemplateFile,
			err,
		)
	}
	cTemplate = tmpl

	return nil
}
