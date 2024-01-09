package eztls

/*
 * selfsigned.go
 * Make self-signed TLS certs.
 * By J. Stuart McMurray
 * Created 20231209
 * Last Modified 20231209
 */

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net"
	"sync"
	"time"

	"golang.org/x/crypto/acme/autocert"
)

// SelfSignedSubject is the subject name we use for self-signed certificates
// when no other subject has been provided.  This is settable at compile-time.
var SelfSignedSubject = "eztls"

// ssCacheKey is the key under which we store our selfsigned certificate.
const ssCacheKey = "eztls_selfsigned"

// DefaultSelfSignedCertLifespan is the amount of time self-signed certificates
// are valid, by default.  It is 10 years.
var DefaultSelfSignedCertLifespan = time.Until(time.Now().AddDate(10, 0, 0))

// SelfSignedGetter returns a function suitable for use in
// tls.Config.Certificate which returns a self-signed certificates for the
// domain patterns (see HostWhitelist) allowed by domains.  A single
// certificate with a wildcard DNS SAN is created for the given lifespan and
// cached in the given directory, or in memory if cacheDir is the empty string.
func SelfSignedGetter(domains []string, lifespan time.Duration, cacheDir string) (func(*tls.ClientHelloInfo) (*tls.Certificate, error), error) {
	/* Work out where we cache this thing. */
	var cache autocert.Cache
	if "" != cacheDir {
		var err error
		if cache, err = dirCache(cacheDir); nil != err {
			return nil, fmt.Errorf(
				"setting up directory cache in %s: %w",
				cacheDir,
				err,
			)
		}
	}

	/* Work out how long our certs will be alive. */
	if 0 == lifespan {
		lifespan = DefaultSelfSignedCertLifespan
	}

	/* Get the whitelist for domains. */
	wl, err := HostWhitelist(domains)
	if nil != err {
		return nil, fmt.Errorf("generating host whitelist: %w", err)
	}

	/* Return something to get the cert. */
	return (&ssGetter{
		policy:   wl,
		cache:    cache,
		lifespan: lifespan,
	}).getCertificate, nil
}

// ssGetter gets or makes self-signed certificates for *.  If cache is nil, the
// cert will only be cached in memory.  New certs will be created with the
// given lifespan.
type ssGetter struct {
	policy   autocert.HostPolicy
	cache    autocert.Cache
	cert     *tls.Certificate
	lifespan time.Duration
	l        sync.Mutex
}

// getCertificate is suitable for use in tls.Config.GetCertificate.
func (sg *ssGetter) getCertificate(chi *tls.ClientHelloInfo) (*tls.Certificate, error) {
	/* Make sure the certificate is allowed. */
	sni := chi.ServerName
	if err := sg.policy(context.Background(), sni); nil != err {
		if "" == sni {
			sni = "default certificates"
		}
		return nil, fmt.Errorf("%q disallowed by policy: %w", sni, err)
	}

	sg.l.Lock()
	defer sg.l.Unlock()

	/* If we don't have a certificate, try to get it from the cache. */
	if nil == sg.cert && nil != sg.cache {
		b, err := sg.cache.Get(context.Background(), ssCacheKey)
		switch {
		case nil == err: /* Worky. */
			if c, err := UnmarshalCertificate(
				b,
			); nil == err && nil != c {
				sg.cert = c
			}
		case errors.Is(err, autocert.ErrCacheMiss): /* It's ok :( */
		default: /* A real error. */
			return nil, fmt.Errorf("cache get: %w", err)
		}
	}

	/* If the cert's valid, use it. */
	if nil != sg.cert {
		now := time.Now()
		if !now.Before(sg.cert.Leaf.NotBefore) &&
			!now.After(sg.cert.Leaf.NotAfter) {
			return sg.cert, nil
		}
	}

	/* Nope, we'll need a new one. */
	var err error
	if sg.cert, err = GenerateSelfSignedCertificate(
		SelfSignedSubject,
		[]string{"*"},
		nil,
		sg.lifespan,
	); nil != err {
		return nil, fmt.Errorf("generating certificate: %w", err)
	}

	/* Also save it to the persistenter cache if we have one. */
	if nil != sg.cache {
		b, err := MarshalCertificate(sg.cert)
		if nil != err {
			return nil, fmt.Errorf(
				"marshalling certificate: %w",
				err,
			)
		}
		if err := sg.cache.Put(
			context.Background(),
			ssCacheKey,
			b,
		); nil != err {
			return nil, fmt.Errorf("cache put: %w", err)
		}
	}

	return sg.cert, nil
}

// GenerateSelfSignedCertificate generates a bare-bones self-signed certificate
// with the given subject, DNS and IP Address SANs, and lifespan.  The
// certificate's Leaf will be set.
func GenerateSelfSignedCertificate(subject string, dnsNames []string, ipAddresses []net.IP, lifespan time.Duration) (*tls.Certificate, error) {
	/* Make sure the cert will stay valid. */
	if 0 == lifespan {
		lifespan = DefaultSelfSignedCertLifespan
	}
	/* Generate it. */
	return generateSelfSignedCert(
		subject,
		dnsNames,
		ipAddresses,
		time.Now().Add(lifespan),
	)
}

// generateSelfSignedCert is like GenerateSelfSignedCert, but allows for an
// explicit expiry time, useful for testing.
func generateSelfSignedCert(subject string, dnsNames []string, ipAddresses []net.IP, notAfter time.Time) (*tls.Certificate, error) {
	/*
		Most of this inspired by
		https://github.com/golang/go/blob/46ea4ab5cb87e9e5d443029f5f1a4bba012804d3/src/crypto/tls/generate_cert.go#L7
	*/

	/* Make sure we have a subject. */
	if "" == subject {
		subject = SelfSignedSubject
	}

	/* Generate our private key. */
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if nil != err {
		return nil, fmt.Errorf("generating key: %w", err)
	}

	/* Gather all the important data for the cert. */
	keyUsage := x509.KeyUsageDigitalSignature
	notBefore := time.Now()
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, fmt.Errorf("generating serial number: %w", err)
	}
	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject:      pkix.Name{CommonName: subject},
		NotBefore:    notBefore,
		NotAfter:     notAfter,
		KeyUsage:     keyUsage,
		ExtKeyUsage: []x509.ExtKeyUsage{
			x509.ExtKeyUsageServerAuth,
		},
		BasicConstraintsValid: true,
		DNSNames:              dnsNames,
		IPAddresses:           ipAddresses,
	}

	/* Turn the certtificate into something the tls library can parse. */
	derBytes, err := x509.CreateCertificate(
		rand.Reader,
		&template,
		&template,
		&priv.PublicKey,
		priv,
	)
	if err != nil {
		return nil, fmt.Errorf("creating certificate: %w", err)
	}
	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: derBytes,
	})

	/* Key, as well. */
	privBytes, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return nil, fmt.Errorf("marshalling key: %w", err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: privBytes,
	})

	/* Finally, parse back into a tls.Certificate. */
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if nil != err {
		return nil, fmt.Errorf(
			"parsing PEM blocks into certificate: %w",
			err,
		)
	}

	/* Make sure Leaf is set. */
	leaf, err := x509.ParseCertificate(cert.Certificate[0])
	if nil != err {
		return nil, fmt.Errorf("parsing leaf: %w", err)
	}
	cert.Leaf = leaf

	return &cert, nil
}
