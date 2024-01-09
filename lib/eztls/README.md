EZTLS
=====
Small wrapper around [acme/autocert](golang.org/x/crypto/acme/autocert) which
makes it somewhat easier to start a TLS server with a certificate provisioned
with [Let's Encrypt](https://letsencrypt.org) in simple configurations.


Examples
--------
In the simplest case, it's not much different than
[`net.Listen`](https://pkg.go.dev/net#Listen) plus a domain and whether or not
to use Let's Encrypt's
[staging environment](https://letsencrypt.org/docs/staging-environment/):
```go
l, err := eztls.Listen("tcp", "0.0.0.0:443", "example.com", true)
```

ListenConfig allows for a bit more control but less fuss than an
[`autocert.Manager`](https://pkg.go.dev/golang.org/x/crypto/acme/autocert#Manager):
```go
/* Use ALL the options. */
l, err := Config{
	Staging: true,
	TLSConfig: &tls.Config{
		NextProtos: append(
			slices.Clone(HTTPSNextProtos),
			"sneakiness",
		),
	},
	CacheDir: "/opt/certs/staging",
	Domains: []string{
		"example.com",
		"*.example.com",
		"example-*.de", /* example-1.de, example-2.de, etc. */
	},
	SelfSignedDomains: []string{
		"*.internal",
		"*.testnet",
	},
	Email: "admin@example.com",
}.Listen("tcp", "0.0.0.0:443")
if nil != err {
	log.Fatalf("Listen error: %s", err)
}

/* Accept and handle TLS clients. */
c, err := l.Accept()
go handle(c)
```

Most fields are optional:
```go
	l, err := ListenConfig("tcp", "0.0.0.0:443", Config{
		Domains: []string{"example.com", "*.example.com"},
	})
```

Self-Signed Certificates
------------------------
Self-signed certificate generation is possible with
`GenerateSelfSignedCertificate`, `SelfSignedGetter`, and
`Config.SelfSignedDomains`.  All of the above generate a certificate with a
singe DNS SAN: `*`,  This is intended for simple test cases not worth fiddling
about with a domain.
