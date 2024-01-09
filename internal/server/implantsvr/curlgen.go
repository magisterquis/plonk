package implantsvr

/*
 * handlers.go
 * HTTP Handlers
 * By J. Stuart McMurray
 * Created 20231208
 * Last Modified 20231208
 */

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"text/template"

	"github.com/magisterquis/plonk/internal/def"
	"github.com/magisterquis/plonk/lib/plog"
	"golang.org/x/net/idna"
)

var (
	// rawTemplate is the stock implant generation template.
	//
	//go:embed curlgen.tmpl
	RawTemplate string

	// tmpl is the parsed stock template.
	ptmpl = template.Must(template.New("curlgen").Parse(RawTemplate))
)

// TemplateParams holds the parameters we pass to the template.
type TemplateParams struct {
	RandN string /* Random base36 number, for ImplantID */
	URL   string /* C2 URL for /{t,o}/ImplantID */
}

// handleCurlGen handles a request for a curl-based implant generation
func (s *Server) handleCurlGen(w http.ResponseWriter, r *http.Request) {
	sl := s.requestLogger(r)

	/* Work out the template to use. */
	var tmpl *template.Template
	fn := filepath.Join(s.Dir, def.TemplateFile)
	if b, err := os.ReadFile(fn); errors.Is(err, os.ErrNotExist) {
		/* No new template. */
		tmpl = ptmpl
	} else if nil == err {
		sl = sl.With(def.LKFilename, fn)
		/* User gave us one. */
		tmpl, err = template.New("loaded").Parse(string(b))
		if nil != err {
			plog.ErrorError(
				sl, def.LMCurlGen, fmt.Errorf(
					"parsing: %w",
					err,
				),
			)
			return
		}
	} else {
		plog.ErrorError(
			sl, def.LMCurlGen, fmt.Errorf(
				"reading template: %w",
				err,
			),
			def.LKFilename, fn,
		)
		return
	}

	/* Generate template parameters. */
	c2, err := c2URL(r)
	if nil != err {
		plog.ErrorError(sl, def.LMCurlGen, fmt.Errorf(
			"determining URL: %w",
			err,
		))
		return
	}
	params := TemplateParams{
		RandN: strconv.FormatUint(rand.Uint64(), 36),
		URL:   c2,
	}
	sl = sl.With(def.LKParameters, params)

	/* Execute the template and send it back. */
	var b bytes.Buffer
	if err := tmpl.Execute(&b, params); nil != err {
		plog.ErrorError(sl, def.LMCurlGen, fmt.Errorf(
			"executing template: %w",
			err,
		))
	}
	b.WriteTo(w)

	sl.Info(def.LMCurlGen)
}

// c2URL tries to get a C2 URl from r.  We try a query/form parameter, a
// c2: header, the Host: header, and the SNI, in that order.
func c2URL(r *http.Request) (string, error) {
	/* Parse the query and form and try to get it from there. */
	if err := r.ParseForm(); nil != err {
		return "", fmt.Errorf("parsing request: %w", err)
	}
	if p := r.Form.Get(def.C2URLParam); "" != p {
		return p, nil
	}

	/* Try to get it as a header. */
	if p := r.Header.Get(def.C2URLParam); "" != p {
		return p, nil
	}

	/* Failing that, try the Host: header. */
	if p, err := idna.ToASCII(r.Host); nil != err {
		return "", fmt.Errorf("punycoding %s: %w", r.Host, err)
	} else if "" != p {
		return p, nil
	}

	/* No Host: header.  Probably HTTP/1.0.  Try the SNI. */
	if p := r.TLS.ServerName; "" != p {
		return p, nil
	}

	/* Out of ideas at this point. */
	return "", errors.New("out of ideas")
}
