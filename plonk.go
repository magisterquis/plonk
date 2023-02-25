// Plonk is a simple HTTP-based file/C2 server.
package main

/*
 * plonk.go
 * Simple HTTP-based file/C2 server
 * By J. Stuart McMurray
 * Created 20230223
 * Last Modified 20230225
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

	"github.com/magisterquis/plonk/internal/debug"
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
	var leDomains []string
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
	flag.BoolVar(
		&VerbOn,
		"verbose",
		VerbOn,
		"Log ALL the things",
	)
	flag.Usage = func() {
		fmt.Fprintf(
			os.Stderr,
			`Usage: %s                      [options]
       %s -task    implantID|- [task...]
       %s -implant implantID|-

Simple HTTP-based file/C2 server.

TODO: Usage example

TODO: Task docs

Do not use -letsencrypt unless you accept Let's Encrypt's Terms of Service.

Options:
`,
			os.Args[0],
			os.Args[0],
			os.Args[0],
		)
		flag.PrintDefaults()
	}
	flag.Parse()
	debug.TODO("Document better")

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
		absPath(logFile.Name()),
	)

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
		absPath(Env.StaticFilesDir),
	)
	mkdir(&Env.LocalCertDir, "TLS certificates")
	Verbosef(
		"[%s] Non-Let's Encrypt TLS certificates directory: %s",
		MessageTypeInfo,
		absPath(Env.LocalCertDir),
	)
	mkdir(&Env.LECertDir, "Let's Encrypt cache")
	Verbosef(
		"[%s] Let's Encrypt certificates directory: %s",
		MessageTypeInfo,
		absPath(Env.LECertDir),
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
		absPath(Env.TaskFile),
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
		absPath(Env.DefaultFile),
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
			Env.LocalCertDir,
			*leEmail,
			Env.LECertDir,
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

// absPath is like filepath.Abs, but uses workingDir as the working directory.
func absPath(path string) string {
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	return filepath.Join(workingDir, path)
}
