Configuration
=============
The bulk of Plonk's server configuration is via (hopefully self-explanatory)
command-line flags.  Please run Plonk with `-h` for a complete list.

Other configurables come in two forms: compile-time-settable defaults and a
couple of files in Plonk's directory.

Compile-time-settable Defaults
------------------------------
The following can be set at compile-time using `-ldflags "-X main.DefaultFoo=bar"`
to change Plonk's defaults.

Name                   | Default       | Description
-----------------------|---------------|------------
`main.DefaultDir`      |`plonk.d`      | [Plonk directory](../README.md#directory) basename; may also be an absolute path
`main.DefaultHTTPAddr` | _none_        | HTTP listen address
`main.DefaultHTTPSAddr`| `0.0.0.0:443` | HTTPS listen address
`main.DefaultName`     | _none_        | Operator name
`main.DefaultMaxExfil` | `100M`        | Maximum [exfil](./protocol.md#exfil-p) size

Files
-----
Two files in Plonk's directory also control how Plonk does what it does.  Both
are re-read every time they're used; no need to restart or SIGHUP the server.

### `index.html`
The contents of this file are returned for every HTTP request for which Plonk
doesn't have a [protocol handler](./protocol.md).  If this file doesn't exist, a
200 with an empty body is returned.  This file can be used to spoof another
very simple server, influence domain categorization, leave a note to nosy blue
teams, and so on.  `index.html` need not be HTML; the contents are served
as-is.

### `implant.tmpl`
Plonk tries to use this file to respond to requests to `/c` for [implant
generation](./protocol.md#implant-generation-c).  Please see the documentation
for more details.
