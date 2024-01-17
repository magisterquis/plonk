TLS
===
Unless explicitly started with `-https-address ""`, Plonk serves HTTPS requests
with either self-signed certificates or, with `-letsencrypt-domain`,
certificates from Let's Encrypt.  By default, a self-signed certificate is
presented to all HTTPS clients.  The following command-line flags can be used
to configure TLS:

Flag                           | Description
-------------------------------|-----------
`-letsencrypt-domain <domain>` | Provisions and uses a certificate for TLS requests for the given domain
`-letsencrypt-email <address>` | Optional email to use with Let's Encrypt
`-letsencrypt-staging`         | If set, uses Let's Encrypt's staging environment
`-selfsigned-domain <domain>`  | Whitelists a domain for serving a self-signed certificate

Both `-letsencrypt-domain` and `-selfsigned-domain` may be repeated and both
accept wildcards, e.g. `*.example.com`.

Let's Encrypt
-------------
Legitimate TLS certificates may be requested from Let's Encrypt by using
`-letsencrypt-domain`.  Certificates will be cached on disk, unles
`-letsencrypt-staging` is also given.

Use of `-letsencrypt-domain` constitutes acceptance of Let's Encrypt's Terms of
Service.

Self-Signed Certificates
------------------------
With no other options given, a self-signed certificate is generated, cached,
and used for every HTTPS request.  If `-selfsigned-domain` is used to whitelist
a domain, only those domains will be served.
