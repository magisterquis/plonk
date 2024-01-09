// Program fserv - Simple HTTPS fileserver
package main

/*
 * fserv.go
 * Simple HTTPS fileserver
 * By J. Stuart McMurray
 * Created 20231223
 * Last Modified 20231223
 */

import (
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"

	"github.com/gorilla/handlers"
	"github.com/magisterquis/plonk/lib/eztls"
)

func main() {
	/* Command-line flags. */
	var (
		lAddr = flag.String(
			"listen",
			"0.0.0.0:443",
			"Listen `address`",
		)
		leStaging = flag.Bool(
			"staging",
			false,
			"With -domain, use Let's Encrypt's staging environment",
		)
		dir = flag.String(
			"dir",
			".",
			"Static files `directory`",
		)
		ssDomains []string
		leDomains []string
	)
	flag.Func(
		"letsencrypt",
		"Optional Let's Encrypt `domain` (may be repeated)",
		func(s string) error {
			leDomains = append(leDomains, s)
			return nil
		},
	)
	flag.Func(
		"selfsigned",
		"Optional self-signed certificate domain `pattern` "+
			"(may be repeated)",
		func(s string) error {
			ssDomains = append(ssDomains, s)
			return nil
		},
	)
	flag.Usage = func() {
		fmt.Fprintf(
			os.Stderr,
			`Usage: %s [options]

Simple HTTPS fileserver.  With -letsencrypt, uses Let's Encrypt to fetch a TLS
certificate for the domain.  If no domains or patterns are given, a self-signed
certificate is used for all queries.

Use of Let's Encrypt with this program implies acceptance of Let's Encrypt's
Terms of Service.

Options:
`,
			os.Args[0],
		)
		flag.PrintDefaults()
	}
	flag.Parse()

	/* If we have no domains at all, self-sign ALL the domains. */
	if 0 == len(leDomains) && 0 == len(ssDomains) {
		ssDomains = append(ssDomains, "*")
		log.Printf("Using a self-signed certificate for all queries")
	}

	/* TLS listener, eventually. */
	var (
		l   net.Listener
		err error
	)

	/*
	 * For testing other listener setups, insert code here.
	 */

	/* If we're not testing a listener setup, start the default way. */
	if nil == l && nil == err {
		l, err = eztls.Config{
			Staging: *leStaging,
			TLSConfig: &tls.Config{
				NextProtos: eztls.HTTPSNextProtos,
			},
			Domains:           leDomains,
			SelfSignedDomains: ssDomains,
		}.Listen("tcp", *lAddr)
	}
	if nil != err {
		log.Fatalf("Error listening for connections: %s", err)
	}
	log.Printf("Listening on %s", l.Addr())

	/* Handle HTTP queries. */
	log.Fatalf("HTTP Server died: %s", (&http.Server{
		Handler: handlers.CombinedLoggingHandler(
			os.Stdout,
			http.FileServer(http.Dir(*dir)),
		),
	}).Serve(l))
}
