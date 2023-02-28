// Plonk is a simple HTTP-based file/C2 server.
package main

/*
 * plonk.go
 * Simple HTTP-based file/C2 server
 * By J. Stuart McMurray
 * Created 20230223
 * Last Modified 20230228
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
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

// Default paths, compile-time settable.
var (
	defaultWorkDir = "plonk.d"
)

var (
	// Verbosef is a verbose logger.
	Verbosef = log.Printf
	// VerbOn will be set if the user passed -verbose
	VerbOn bool
	// workingDir is our working directory.
	workingDir string
	// flog is a logger which only writes to the logfile.  It should
	// not be used until logging is initialized.
	flog atomic.Pointer[log.Logger]
)

func init() {
	/* Try to prevent nil pointer dereferences.  In theory, another ini
	could still be a problem. */
	flog.Store(log.Default())
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
		&VerbOn,
		"verbose",
		VerbOn,
		"Log ALL the things",
	)
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

  C2 tasking is retrieved by a request to /%s/<ImplantID>.  The /<ImplantID>
  may be empty; Plonk treats this as an IDless implant.  Tasking is stored in a
  single JSON file (currently %s/%s), which may be updated by
  hand or with -task or -implant, as below.  Plonk doesn't do anything to
  process tasking; whatever it gets it sends directly to the implant.

  Output from implants is sent in an HTTP request body to /%s/<ImplantID>, or
  just /%s for an IDless implant.

  HTTP verbs for all requests are ignored.  HTTP requests for paths other than
  the above are served a single static file, by default %s/%s.

  All of the above is logged in %s/%s.  

  When Plonk gets a SIGHUP, it reopens the taskfile and forgets the self-signed
  certificates it's generated as well as its list of seen implants.  When Plonk
  gets a SIGUSR1, it writes the self-signed certificates it's generated to the
  local certificate directory.

  The first time Plonk is run, it is helpful to use -verbose.

Usage: %s -task implantID|- [task...]

  Adds a task for the given implant, or - for the IDless implant.  This
  invocation af Plonk must be run with the same idea of the tasking file
  (currently %s/%s) as the server process.

Usage: %s -implant implantID|-

  Interactive(ish) operation.  Given an implant ID (or - for the IDlessimplant)
  it queues as tasking non-blank, non #-prefixed lines it reads on standard
  input and displays relevant logfile lines on standard output.  Probably best
  used with rlwrap.  Like -task, this invocation of Plonk must be run with the
  same idea of the tasking file (currently %s/%s) as well as the
  logfile (currently %s/%s).

Options:
`,
			/* Server help */
			os.Args[0],
			*workDir,
			*workDir, Env.LocalCertDir,
			*workDir, Env.StaticFilesDir,
			Env.FilesPrefix,
			Env.TaskPrefix,
			*workDir, Env.TaskFile,
			Env.OutputPrefix, Env.OutputPrefix,
			*workDir, Env.DefaultFile,
			*workDir, Env.LogFile,

			/* -task help */
			os.Args[0],
			*workDir, Env.TaskFile,

			/* -implant help */
			os.Args[0],
			*workDir, Env.TaskFile,
			*workDir, Env.LogFile,
		)
		flag.PrintDefaults()
	}
	flag.Parse()

	/* If we're just printing the environment, life's easy. */
	if *printEnv {
		PrintEnv()
		return
	}

	/* Parse environment things. */
	httpTimeout, err := time.ParseDuration(Env.HTTPTimeout)
	if nil != err {
		log.Fatalf(
			"[%s] Parsing HTTP timeout (%s) %q: %s",
			MessageTypeError,
			EnvVarName(&Env.HTTPTimeout),
			Env.HTTPTimeout,
			err,
		)
	}
	outputMax, err := strconv.ParseInt(Env.OutputMax, 0, 64)
	if nil != err {
		log.Fatalf(
			"[%s] Parsing max output size (%s) %q: %s",
			MessageTypeError,
			EnvVarName(&Env.OutputMax),
			Env.OutputMax,
			err,
		)
	}
	if 0 >= outputMax {
		log.Fatalf(
			"[%s] Max output size (%s) must be greater than "+
				"zero, not %d",
			MessageTypeError,
			EnvVarName(&Env.OutputMax),
			outputMax,
		)
	}

	/* Make sure we'll have something to listen for. */
	if "" == *httpAddr && "" == *httpsAddr {
		log.Fatalf(
			"[%s] Need a listen addres (-http-addr/-https-addr)",
			MessageTypeError,
		)
	}

	/* Be in our working directory. */
	if err := os.MkdirAll(*workDir, 0770); nil != err {
		log.Fatalf(
			"[%s] Making working directory (-work-dir) %q: %s",
			MessageTypeError,
			*workDir,
			err,
		)
	}
	if err := os.Chdir(*workDir); nil != err {
		log.Fatalf(
			"[%s] Changing to working directory %q: %s",
			MessageTypeError,
			*workDir,
			err,
		)
	}
	workingDir, err = os.Getwd()
	if nil != err {
		log.Fatalf(
			"[%s] Getting working directory: %s",
			MessageTypeError,
			err,
		)
	}

	/* Work out logging. */
	logFile, err := os.OpenFile(
		Env.LogFile,
		os.O_RDWR|os.O_CREATE|os.O_APPEND, /* RDWR because -interact. */
		0660,
	)
	if nil != err {
		log.Fatalf(
			"[%s] Opening logfile (%s) %q: %s",
			MessageTypeError,
			EnvVarName(&Env.LogFile),
			Env.LogFile,
			err,
		)
	}
	defer logFile.Close()
	flog.Store(log.New(logFile, "", log.LstdFlags))
	log.SetOutput(io.MultiWriter(logFile, os.Stdout))
	if !VerbOn {
		Verbosef = func(string, ...any) {}
	}
	Verbosef("[%s] Working directory: %s", MessageTypeInfo, workingDir)
	Verbosef(
		"[%s] Logfile: %s",
		MessageTypeInfo,
		AbsPath(logFile.Name()),
	)

	/* Make sure domain whitelist entries are valid globs. */
	for _, wlD := range wlDomains {
		_, err := filepath.Match(wlD, "")
		if nil == err {
			continue
		}
		log.Fatalf(
			"[%s] Bad domain whitelist glob: %s",
			MessageTypeError,
			err,
		)
	}

	/* Make directories. */
	mkdir := func(p *string, which string) {
		if err := os.MkdirAll(*p, 0770); nil != err {
			log.Fatalf(
				"[%s] making %s files directory (%s) %q: %s",
				MessageTypeError,
				which,
				EnvVarName(p),
				*p,
				err,
			)
		}
	}
	mkdir(&Env.StaticFilesDir, "static files")
	Verbosef(
		"[%s] Static files directory: %s",
		MessageTypeInfo,
		AbsPath(Env.StaticFilesDir),
	)
	mkdir(&Env.LocalCertDir, "TLS certificates")
	Verbosef(
		"[%s] Non-Let's Encrypt TLS certificates directory: %s",
		MessageTypeInfo,
		AbsPath(Env.LocalCertDir),
	)
	mkdir(&Env.LECertDir, "Let's Encrypt cache")
	Verbosef(
		"[%s] Let's Encrypt certificates directory: %s",
		MessageTypeInfo,
		AbsPath(Env.LECertDir),
	)

	/* Set up the files. */
	if err := OpenTaskFile(Env.TaskFile); nil != err {
		log.Fatalf(
			"[%s] Opening taskfile (%s) %q: %s",
			MessageTypeError,
			EnvVarName(&Env.TaskFile),
			Env.TaskFile,
			err,
		)
	}
	Verbosef(
		"[%s] Taskfile: %s",
		MessageTypeInfo,
		AbsPath(Env.TaskFile),
	)
	if f, err := os.OpenFile(
		Env.DefaultFile,
		os.O_RDONLY|os.O_CREATE,
		0660,
	); nil != err {
		log.Fatalf(
			"[%s] opening default file (%s) %q: %s",
			MessageTypeError,
			EnvVarName(&Env.DefaultFile),
			Env.DefaultFile,
			err,
		)
	} else {
		f.Close()
	}

	/* If we're going interactive or just queuing a task, life's easy. */
	if "" != *interact { /* Interact with an implant. */
		if err := Interact(*interact, logFile.Name()); nil != err {
			log.Fatalf(
				"[%s] Interacting with %q: %s",
				MessageTypeError,
				*interact,
				err,
			)
		}
		return
	} else if "" != *queueTask { /* Queue a task */
		/* Make the task a single string. */
		t := strings.Join(flag.Args(), " ")
		/* ID - really means "". */
		id := *queueTask
		if "-" == id {
			id = ""
		}
		/* Do the deed. */
		if err := AddTask(id, t, false); nil != err {
			log.Fatalf(
				"[%s] Queuing task %q for %q: %s",
				MessageTypeError,
				t,
				id,
				err,
			)
		}
		return
	}

	/* Set up HTTP handlers.  This is a bit silly. */
	handle := func(p *string, which string, h http.Handler) {
		*p = "/" + strings.Trim(*p, "/")
		if "/" == *p {
			log.Fatalf(
				"[%s] HTTP path prefix for %s (%s) may not "+
					"be empty or /",
				MessageTypeError,
				which,
				EnvVarName(p),
			)

		}
		h = http.TimeoutHandler(h, httpTimeout, "")
		h = http.StripPrefix(*p, h)
		http.Handle(*p+"/", h)
		http.Handle(*p, h)
	}
	handle(
		&Env.FilesPrefix,
		"static files",
		LogHandler(http.FileServer(http.Dir(Env.StaticFilesDir))),
	)
	handle(&Env.TaskPrefix, "task", http.HandlerFunc(HandleTask))
	handle(&Env.OutputPrefix, "output", http.MaxBytesHandler(
		http.HandlerFunc(HandleOutput),
		outputMax,
	))
	http.Handle("/", LogHandler(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			http.ServeFile(w, r, Env.DefaultFile)
		},
	)))
	Verbosef(
		"[%s] Default file: %s",
		MessageTypeInfo,
		AbsPath(Env.DefaultFile),
	)

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
				err:   HTTPServer.Serve(httpL),
			}
		}()
		Verbosef(
			"[%s] HTTP address: %s",
			MessageTypeInfo,
			httpL.Addr(),
		)
	}
	if "" != *httpsAddr {
		httpsL, err := tls.Listen("tcp", *httpsAddr, MakeTLSConfig(
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
				err:   HTTPServer.Serve(httpsL),
			}
		}()
		Verbosef(
			"[%s] HTTPS address: %s",
			MessageTypeInfo,
			httpsL.Addr(),
		)
	}
	log.Printf("Ready")

	/* Wait for something to go wrong. */
	ferr := <-ech
	log.Fatalf(
		"[%s] Serving %s: %s",
		MessageTypeError,
		ferr.which,
		ferr.err,
	)
}

// AbsPath is like filepath.Abs, but uses workingDir as the working directory.
func AbsPath(path string) string {
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	return filepath.Join(workingDir, path)
}
