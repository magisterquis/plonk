// Program plonk - Really simple HTTP-based C2 server
package main

/*
 * plonk.go
 * Really simple HTTP-based C2 server
 * By J. Stuart McMurray
 * Created 20231104
 * Last Modified 20231208
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

	"github.com/magisterquis/mqd"
	"github.com/magisterquis/plonk/internal/client"
	"github.com/magisterquis/plonk/internal/def"
	"github.com/magisterquis/plonk/internal/server"
	"github.com/magisterquis/plonk/internal/server/implantsvr"
	"github.com/magisterquis/plonk/lib/plog"
	"golang.org/x/sys/unix"
)

func main() {
	//var (
	//	sConf    = server.Config{LogOutput: os.Stdout}
	//	cConf    client.Config
	//	domainsL sync.Mutex
	//)
	/* Command-line flags. */
	mqd.TODO("Flag: -letsencrypt")
	mqd.TODO("Flag: -letsencrypt-email")
	mqd.TODO("Flag: -letsencrypt-staging")
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
			"http",
			def.DefaultHTTPAddr,
			"HTTP listen `address` (with -server)",
		)
		httpsAddr = flag.String(
			"https",
			def.DefaultHTTPSAddr,
			"HTTPS listen `address` (with -server)",
		)
		debug = flag.Bool(
			"debug",
			false,
			"Enable debug logging",
		)
		noExfil = flag.Bool(
			"disable-exfil",
			false,
			fmt.Sprintf(
				"Disable exfil (%s) requests (with -server)",
				def.ExfilPath,
			),
		)
	//		id = flag.String(
	//			"id",
	//			"",
	//			"Initial interactive implant `ID`",
	//		)
	)
	//flag.BoolVar(
	//	&sConf.LEStaging,
	//	"letsencrypt-staging",
	//	false,
	//	"Use Let's Encrypt's staging server (with -server)",
	//)
	//flag.StringVar(
	//	&sConf.HTTPSAddr,
	//	"https",
	//	def.DefaultHTTPSAddr,
	//	"HTTPS listen `address` (with -server)",
	//)
	//flag.Func(
	//	"letsencrypt",
	//	"Let's Encrypt-provisioned TLS `domain` (with -server, "+
	//		"may be repeated)",
	//	func(d string) error {
	//		domainsL.Lock()
	//		defer domainsL.Unlock()
	//		sConf.LEDomains = append(sConf.LEDomains, d)
	//		return nil
	//	},
	//)
	flag.Usage = func() {
		fmt.Fprintf(
			os.Stderr,
			`Usage: %s [options]

Really simple HTTP-based C2 server.

Options:
`,
			os.Args[0],
		)
		flag.PrintDefaults()
	}
	flag.Parse()

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
		os.Exit((&client.Client{
			Dir:   *dir,
			Debug: *debug,
			Name:  *opName,
		}).Run())
		panic("unpossible")
	}

	/* Start a server and wait for it to die. */
	svr := server.Server{
		Dir:       *dir,
		Debug:     *debug,
		HTTPAddr:  *httpAddr,
		HTTPSAddr: *httpsAddr,
		NoExfil:   *noExfil,
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
	/* If someone changed the default to an absolute path, use that. */
	if filepath.IsAbs(def.DefaultDir) {
		return def.DefaultDir
	}

	/* If we can get the home directory, try that. */
	if hd, err := os.UserHomeDir(); nil == err {
		return filepath.Join(hd, def.DefaultDir)
	}

	/* Failing that, we'll use the current directory. */
	return def.DefaultDir
}

// defaultName tries to get the current user's name.  It tries def.DefaultName,
// the current GECOS name, the current username, and the current user ID, in
// that order.
func defaultName() string {
	/* If we have a default set, life's easy. */
	if "" != def.DefaultName {
		return def.DefaultName
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
