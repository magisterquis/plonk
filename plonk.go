// Plonk is a simple HTTP-based file/C2 server.
package main

/*
 * plonk.go
 * Simple HTTP-based file/C2 server
 * By J. Stuart McMurray
 * Created 20230223
 * Last Modified 20230911
 */

import (
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/magisterquis/plonk/internal/lib"
	"github.com/magisterquis/plonk/internal/lib/clgen"
)

// Default paths, compile-time settable.
var (
	defaultWorkDir = "plonk.d"
)

func init() {
	/* Try to prevent nil pointer dereferences.  In theory, another ini
	could still be a problem. */
	lib.Flog.Store(log.Default())
}

func main() {
	var (
		leDomains []string
		wlDomains []string
	)
	var (
		httpAddr = flag.String(
			"http",
			"",
			"HTTP `address`",
		)
		httpsAddr = flag.String(
			"https",
			"0.0.0.0:443",
			"HTTPS `address`",
		)
		leStaging = flag.Bool(
			"letsencrypt-staging",
			false,
			"Use Let's Encrypt's staging server",
		)
		workDir = flag.String(
			"work-dir",
			defaultWorkDir,
			"Working `directory`",
		)
		queueTask = flag.String(
			"task",
			"",
			"Queue a task for the given implant `ID` "+
				"or - for an IDless implant",
		)
		interact = flag.String(
			"interact",
			"",
			"Interact with the given implant ID, "+
				"or - for an IDless implant",
		)
		leEmail = flag.String(
			"letsencrypt-email",
			"",
			"Optional email `address` to use for Let's Encrypt",
		)
		printEnv = flag.Bool(
			"print-env",
			false,
			"Print the configuration environment variables",
		)
		noExfil = flag.Bool(
			"no-exfil",
			false,
			"Do not handle exfil requests",
		)
		printCLGenTemplate = flag.Bool(
			"implant-template",
			false,
			fmt.Sprintf(
				"Print the %s template and exit",
				lib.Env.CLGenPrefix,
			),
		)
	)
	flag.Func(
		"letsencrypt",
		"Use Let's Encrypt to provision certificates for the given "+
			"`domain` (may be repeated)",
		func(d string) error {
			leDomains = append(leDomains, d)
			return nil
		},
	)
	flag.Func(
		"whitelist-self-signed",
		"Allow self-signed cert generation for the given "+
			"(possibly wildcarded) `domain` or IP address (may "+
			"be repeated, default *)",
		func(d string) error {
			wlDomains = append(wlDomains, d)
			return nil
		},
	)
	flag.BoolVar(
		&lib.VerbOn,
		"verbose",
		lib.VerbOn,
		"Log ALL the things",
	)
	os.Args[0] = "plonk"
	flag.Usage = func() {
		fmt.Fprintf(
			os.Stderr,
			`Usage: %s [options]

  HTTP(s)-based static file and rudimentary C2 server.

  Upon starting, Plonk will make a directory (-work-dir, currently %s),
  chdir into it, and make other supporting files and directories.  The names of
  these and several other things can be controlled with environment variables,
  listable with -print-env.

  TLS certificates may, in order of preference, be automatically provisioned
  using Let's Encrypt (-letsencrypt*), stored as pairs of
  %s/%s/domain.tld.{crt,key}, or failing that, generated as
  self-signed certificates.

  Do not use -letsencrypt unless you accept Let's Encrypt's Terms of Service.
  
  Files and directories under %s/%s/ will served when Plonk gets a
  request for a path under /%s/.

  A quick-n-dirty implant script can be retrieved from /%s.  By default, it
  will call back to the protocol, domain, and port from which it was requested.

  C2 tasking is retrieved by a request to /%s/<ImplantID>.  The /<ImplantID>
  may be empty; Plonk treats this as an IDless implant.  Tasking is stored in a
  single JSON file (currently %s/%s), which may be updated by
  hand or with -task or -implant, as below.  Plonk doesn't do anything to
  process tasking; whatever it gets it sends directly to the implant.

  Output from implants is sent in an HTTP request body to /%s/<ImplantID>, or
  just /%s for an IDless implant.

  Larger exfil is sent in an HTTP request body to /%s/<ImplantID>, or just /%s
  for an IDless implant.  The request bodies will be saved in files in
  %s/%s/.

  HTTP verbs for all requests are ignored.  HTTP requests for paths other than
  the above are served a single static file, by default %s/%s.

  All of the above is logged in %s/%s.  

  When Plonk gets a SIGHUP, it reopens the taskfile and forgets the self-signed
  certificates it's generated as well as its list of seen implants.  When Plonk
  gets a SIGUSR1, it writes the self-signed certificates it's generated to the
  local certificate directory.

  The first time Plonk is run, it is helpful to use -verbose.

Usage: %s -task implantID|-|%s [task...]

  Adds a task for the given implant, or - for the IDless implant.  This
  invocation af Plonk must be run with the same idea of the tasking file
  (currently %s/%s) as the server process.  The implantID may
  also be %s to automatically select the next implant which calls back.

Usage: %s -interact implantID|-|%s

  Interactive(ish) operation.  Given an implant ID (or - for the IDlessimplant)
  it queues as tasking non-blank, non #-prefixed lines it reads on standard
  input and displays relevant logfile lines on standard output.  Probably best
  used with rlwrap.  Like -task, this invocation of Plonk must be run with the
  same idea of the tasking file (currently %s/%s) as well as the
  logfile (currently %s/%s). The implantID may also be %s to
  automatically select the next implant which calls back.

Options:
`,
			/* Server help */
			os.Args[0],
			*workDir,
			*workDir, lib.Env.LocalCertDir,
			*workDir, lib.Env.StaticFilesDir,
			lib.Env.FilesPrefix,
			lib.Env.CLGenPrefix,
			lib.Env.TaskPrefix,
			*workDir, lib.Env.TaskFile,
			lib.Env.OutputPrefix, lib.Env.OutputPrefix,
			lib.Env.ExfilPrefix, lib.Env.ExfilPrefix,
			*workDir, lib.Env.ExfilDir,
			*workDir, lib.Env.DefaultFile,
			*workDir, lib.Env.LogFile,

			/* -task help */
			os.Args[0], lib.NextImplantID,
			*workDir, lib.Env.TaskFile, lib.NextImplantID,

			/* -implant help */
			os.Args[0], lib.NextImplantID,
			*workDir, lib.Env.TaskFile,
			*workDir, lib.Env.LogFile, lib.NextImplantID,
		)
		flag.PrintDefaults()
	}
	flag.Parse()

	/* If we're just printing the environment, life's easy. */
	if *printEnv {
		lib.PrintEnv()
		return
	}

	/* Parse environment things. */
	httpTimeout, err := time.ParseDuration(lib.Env.HTTPTimeout)
	if nil != err {
		log.Fatalf(
			"[%s] Parsing HTTP timeout (%s) %q: %s",
			lib.MessageTypeError,
			lib.EnvVarName(&lib.Env.HTTPTimeout),
			lib.Env.HTTPTimeout,
			err,
		)
	}

	/* Make sure we'll have something to listen for. */
	if "" == *httpAddr && "" == *httpsAddr {
		log.Fatalf(
			"[%s] Need a listen addres (-http-addr/-https-addr)",
			lib.MessageTypeError,
		)
	}

	/* Be in our working directory. */
	if err := os.MkdirAll(*workDir, 0770); nil != err {
		log.Fatalf(
			"[%s] Making working directory (-work-dir) %q: %s",
			lib.MessageTypeError,
			*workDir,
			err,
		)
	}
	if err := os.Chdir(*workDir); nil != err {
		log.Fatalf(
			"[%s] Changing to working directory %q: %s",
			lib.MessageTypeError,
			*workDir,
			err,
		)
	}
	lib.WorkingDir, err = os.Getwd()
	if nil != err {
		log.Fatalf(
			"[%s] Getting working directory: %s",
			lib.MessageTypeError,
			err,
		)
	}

	/* Work out logging. */
	logFile, err := os.OpenFile(
		lib.Env.LogFile,
		os.O_RDWR|os.O_CREATE|os.O_APPEND, /* RDWR because -interact. */
		0660,
	)
	if nil != err {
		log.Fatalf(
			"[%s] Opening logfile (%s) %q: %s",
			lib.MessageTypeError,
			lib.EnvVarName(&lib.Env.LogFile),
			lib.Env.LogFile,
			err,
		)
	}
	defer logFile.Close()
	lib.Flog.Store(log.New(logFile, "", log.LstdFlags))
	log.SetOutput(io.MultiWriter(logFile, os.Stdout))
	if !lib.VerbOn {
		lib.Verbosef = func(string, ...any) {}
	}
	lib.Verbosef(
		"[%s] Working directory: %s",
		lib.MessageTypeInfo,
		lib.WorkingDir,
	)
	lib.Verbosef(
		"[%s] Logfile: %s",
		lib.MessageTypeInfo,
		lib.AbsPath(logFile.Name()),
	)

	/* Make sure domain whitelist entries are valid globs. */
	for _, wlD := range wlDomains {
		_, err := filepath.Match(wlD, "")
		if nil == err {
			continue
		}
		log.Fatalf(
			"[%s] Bad domain whitelist glob: %s",
			lib.MessageTypeError,
			err,
		)
	}

	/* Make directories. */
	mkdir := func(p *string, which string) {
		if err := os.MkdirAll(*p, 0770); nil != err {
			log.Fatalf(
				"[%s] making %s files directory (%s) %q: %s",
				lib.MessageTypeError,
				which,
				lib.EnvVarName(p),
				*p,
				err,
			)
		}
	}
	mkdir(&lib.Env.StaticFilesDir, "static files")
	lib.Verbosef(
		"[%s] Static files directory: %s",
		lib.MessageTypeInfo,
		lib.AbsPath(lib.Env.StaticFilesDir),
	)
	mkdir(&lib.Env.LocalCertDir, "TLS certificates")
	lib.Verbosef(
		"[%s] Non-Let's Encrypt TLS certificates directory: %s",
		lib.MessageTypeInfo,
		lib.AbsPath(lib.Env.LocalCertDir),
	)
	mkdir(&lib.Env.LECertDir, "Let's Encrypt cache")
	lib.Verbosef(
		"[%s] Let's Encrypt certificates directory: %s",
		lib.MessageTypeInfo,
		lib.AbsPath(lib.Env.LECertDir),
	)
	if !*noExfil {
		mkdir(&lib.Env.ExfilDir, "Exfil")
		lib.Verbosef(
			"[%s] Exfil directory: %s",
			lib.MessageTypeInfo,
			lib.AbsPath(lib.Env.ExfilDir),
		)
	}

	/* Set up the files.  Naming is hard. */
	if err := lib.ReopenTaskFile(); nil != err {
		log.Fatalf(
			"[%s] Opening taskfile (%s) %q: %s",
			lib.MessageTypeError,
			lib.EnvVarName(&lib.Env.TaskFile),
			lib.Env.TaskFile,
			err,
		)
	}
	lib.Verbosef(
		"[%s] Taskfile: %s",
		lib.MessageTypeInfo,
		lib.AbsPath(lib.Env.TaskFile),
	)
	if f, err := os.OpenFile(
		lib.Env.DefaultFile,
		os.O_RDONLY|os.O_CREATE,
		0660,
	); nil != err {
		log.Fatalf(
			"[%s] opening default file (%s) %q: %s",
			lib.MessageTypeError,
			lib.EnvVarName(&lib.Env.DefaultFile),
			lib.Env.DefaultFile,
			err,
		)
	} else {
		f.Close()
	}

	/* If we're going interactive or just queuing a task, life's easy. */
	if "" != *interact { /* Interact with an implant. */
		lib.UpdateWithNextIfNeeded(interact, logFile.Name())
		if err := lib.Interact(*interact, logFile.Name()); nil != err {
			log.Fatalf(
				"[%s] Interacting with %q: %s",
				lib.MessageTypeError,
				*interact,
				err,
			)
		}
		return
	} else if "" != *queueTask { /* Queue a task */
		lib.UpdateWithNextIfNeeded(queueTask, logFile.Name())
		/* Make the task a single string. */
		t := strings.Join(flag.Args(), " ")
		/* ID - really means "". */
		id := *queueTask
		if "-" == id {
			id = ""
		}
		/* Do the deed. */
		if err := lib.AddTask(id, t, false); nil != err {
			log.Fatalf(
				"[%s] Queuing task %q for %q: %s",
				lib.MessageTypeError,
				t,
				id,
				err,
			)
		}
		return
	}

	/* Set up cURL loop generation. */
	if err := clgen.Init(); nil != err {
		log.Fatalf(
			"[%s] Setting up cURL loop generator: %s",
			lib.MessageTypeError,
			err,
		)
	}
	if *printCLGenTemplate {
		tb, _, err := clgen.Template()
		if nil != err {
			log.Fatalf("Error getting template: %s", err)
		}
		if _, err := os.Stdout.Write(tb); nil != err {
			log.Fatalf("Error writing template: %s", err)
		}
		return
	}
	lib.Verbosef(
		"[%s] cURL loop generator template (/%s): %s",
		lib.MessageTypeInfo,
		lib.Env.CLGenPrefix,
		lib.AbsPath(clgen.TemplateFile),
	)

	/* Set up HTTP handlers.  This is a bit silly. */
	handle := func(p *string, which string, h http.Handler, bareOk bool) {
		*p = "/" + strings.Trim(*p, "/")
		if "/" == *p {
			log.Fatalf(
				"[%s] HTTP path prefix for %s (%s) may not "+
					"be empty or /",
				lib.MessageTypeError,
				which,
				lib.EnvVarName(p),
			)

		}
		h = http.TimeoutHandler(h, httpTimeout, "")
		h = http.StripPrefix(*p, h)
		http.Handle(*p+"/", h)
		if bareOk {
			http.Handle(*p, h)
		}
	}
	handle(
		&lib.Env.FilesPrefix,
		"static files",
		lib.LogHandler(http.FileServer(
			http.Dir(lib.Env.StaticFilesDir),
		)),
		false,
	)
	handle(&lib.Env.TaskPrefix, "task", http.HandlerFunc(
		lib.HandleTask,
	), true)
	handle(&lib.Env.OutputPrefix, "output", http.MaxBytesHandler(
		http.HandlerFunc(lib.HandleOutput),
		lib.MustParseEnvInt(&lib.Env.OutputMax),
	), true)
	handle(&lib.Env.CLGenPrefix, "curl loop generation", http.HandlerFunc(
		clgen.Handler,
	), true)
	if !*noExfil {
		handle(&lib.Env.ExfilPrefix, "exfil", http.MaxBytesHandler(
			http.HandlerFunc(lib.HandleExfil),
			lib.MustParseEnvInt(&lib.Env.ExfilMax),
		), true)
	}
	http.Handle("/", lib.LogHandler(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			http.ServeFile(w, r, lib.Env.DefaultFile)
		},
	)))
	lib.Verbosef(
		"[%s] Default file: %s",
		lib.MessageTypeInfo,
		lib.AbsPath(lib.Env.DefaultFile),
	)

	/* Watch for Signal. */
	go lib.LogSignals()
	go lib.TaskQSignals()
	go lib.TLSSignals()

	/* Actually serve requests. */
	type serr struct {
		which string
		err   error
	}
	ech := make(chan serr)
	if "" != *httpAddr {
		httpL, err := net.Listen("tcp", *httpAddr)
		if nil != err {
			log.Fatalf(
				"Unable to listen for HTTP requests on %q: %s",
				*httpAddr,
				err,
			)
		}
		defer httpL.Close()
		go func() {
			ech <- serr{
				which: "HTTP",
				err:   lib.HTTPServer.Serve(httpL),
			}
		}()
		lib.Verbosef(
			"[%s] HTTP address: %s",
			lib.MessageTypeInfo,
			httpL.Addr(),
		)
	}
	if "" != *httpsAddr {
		httpsL, err := tls.Listen("tcp", *httpsAddr, lib.MakeTLSConfig(
			leDomains,
			wlDomains,
			*leEmail,
			*leStaging,
		))
		if nil != err {
			log.Fatalf(
				"Unable to listen for HTTPS requests on %q: %s",
				*httpAddr,
				err,
			)
		}
		defer httpsL.Close()
		go func() {
			ech <- serr{
				which: "HTTPS",
				err:   lib.HTTPServer.Serve(httpsL),
			}
		}()
		lib.Verbosef(
			"[%s] HTTPS address: %s",
			lib.MessageTypeInfo,
			httpsL.Addr(),
		)
	}
	log.Printf("Ready")

	/* Wait for something to go wrong. */
	ferr := <-ech
	log.Fatalf(
		"[%s] Serving %s: %s",
		lib.MessageTypeError,
		ferr.which,
		ferr.err,
	)
}
