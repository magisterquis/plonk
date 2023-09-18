// Package clgen - Generate a self-backgrounding, shell-powered cURL loop
package clgen

/*
 * clgen.go
 * Generate a shell-powered cURL loop
 * By J. Stuart McMurray
 * Created 20230726
 * Last Modified 20230911
 */

import (
	"fmt"

	"github.com/magisterquis/plonk/internal/lib"
)

// Compile-time-settable variables.
var (
	/* TemplateFile is the path to the template for generating cURL
	commands. */
	TemplateFile = "clgen.tmpl"

	/* C2URLParam is the parameter used to explicitly set the C2 URL in
	the returned script.  It may be a URL query parameter, a POST
	parameter, or a header name. */
	C2URLParam = "c2url"

	/* IntervalParam is the parameter used to set the callback interval.
	It is used in the same way as C2URLParam. */
	IntervalParam = "cbint"

	/* IDParam is the parameter used to set the ImplantID to a static
	value. It is used in the same way as C2URLParam. */
	IDParam = "id"

	/* CallbackInterval is the default callback interval for the script,
	as parsed by time.ParseDuration.  This will be rounded domwn to the
	nearest whole second. */
	CallbackInterval = "5s"
)

// MessageTypeCLGen is how we tag failures in cURL loop things.
const MessageTypeCLGen lib.MessageType = "CURLLOOP"

// templateParams holds the parameters we pass to the template.
type templateParams struct {
	RandN    string /* Random base36 number, for ImplantID */
	URL      string /* C2 URL for /{t,o}/ImplantID */
	Interval int    /* Callback interval, in seconds */
	ID       string /* Static ImplantID */
}

// Init works out where the template file should be and loads the template from
// the template file if it exists or from the embedded template if not to make
// sure it parses.
func Init() error {
	/* Work out where we'll store the template. */
	TemplateFile = lib.AbsPath(TemplateFile)

	/* Load the initial template. */
	if _, err := loadTemplate(); nil != err {
		return fmt.Errorf("parsing initial template: %w", err)
	}

	return nil
}
