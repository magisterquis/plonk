package main

/*
 * tls.go
 * TLS config and cert-handling
 * By J. Stuart McMurray
 * Created 20230223
 * Last Modified 20230228
 */

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"log"
	"math/big"
	"net"
	"net/netip"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/acme"
	"golang.org/x/crypto/acme/autocert"
	"golang.org/x/exp/maps"
	"golang.org/x/sys/unix"
)

const (
	/* stagingURL is the URL to the Let's Encrypt staging directory
	server. */
	stagingURL = "https://acme-staging-v02.api.letsencrypt.org/directory"

	/* stagingCacheDir is the name of the directory in which we store
	certs for the staging server. */
	stagingCacheDir = "staging"

	/* ALPNs we support */
	httpALPN  = "http/1.1"
	http2ALPN = "h2"

	/* CertFileSuffix is appended to a domain to name the certificate
	file. */
	certFileSuffix = ".crt"

	/* KeyFileSuffix is appended to a domain to name the key file. */
	keyFileSuffix = ".key"

	/* defaultSNI is used for certs for TLS connections with no SNI. */
	defaultSNI = "default"

	/* selfSignedLifetime is the duration for which self-signed certs are
	valid.  It works out to about a year. */
	selfSignedLifetime = 365 * 24 * time.Hour
)

var (
	// localCache caches certs we've read from disk.
	localCache       = make(map[string]*tls.Certificate)
	localCacheL      sync.Mutex
	selfSignedCache  = make(map[string]selfSignedCert)
	selfSignedCacheL sync.Mutex
)

// selfSignedCert holds a self-signed cert we've generated, plus the PEM form
// in case we want to write to a file.
type selfSignedCert struct {
	certPEM []byte
	keyPEM  []byte
	cert    *tls.Certificate
}

// TLSSignals empties the local cache when we get a SIGHUP.  This makes it
// possible to update certs manually without downtime.  On SIGUSR1, write the
// cached self-signed certs to disk.
func TLSSignals() {
	/* Cert-forgetting. */
	hupch := make(chan os.Signal, 1)
	signal.Notify(hupch, unix.SIGHUP)
	go func() {
		for range hupch {
			localCacheL.Lock()
			selfSignedCacheL.Lock()
			n := len(localCache) + len(selfSignedCache)
			maps.Clear(localCache)
			maps.Clear(selfSignedCache)
			localCacheL.Unlock()
			selfSignedCacheL.Unlock()
			if 0 != n {
				log.Printf(
					"[%s] Forgot %d certificates "+
						"cached in memory",
					MessageTypeSIGHUP,
					n,
				)
			} else {
				log.Printf(
					"[%s] No certificates "+
						"cached in memory",
					MessageTypeSIGHUP,
				)
			}
		}
	}()

	/* Cert-writing. */
	usr1ch := make(chan os.Signal, 1)
	signal.Notify(usr1ch, unix.SIGUSR1)
	go func() {
		for range usr1ch {
			go saveSelfSignedCerts()
		}
	}()

}

// MakeTLSConfig makes a TLS config which tries to get certificates from
// the cache directory, failing that from Let's Encrypt for the domains in
// leDomains, and failing that generates a self-signed cert.
func MakeTLSConfig(leDomains, wlDomains []string, leEmail string, leStaging bool) *tls.Config {
	/* Roll a config for Let's Encrypt. */
	mgr := autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist(leDomains...),
		Email:      leEmail,
	}
	/* Staging changes a few things. */
	leCacheDir := Env.LECertDir
	if leStaging {
		leCacheDir = filepath.Join(leCacheDir, stagingCacheDir)
		mgr.Client = &acme.Client{
			DirectoryURL: stagingURL,
		}
		mgr.Client = &acme.Client{DirectoryURL: stagingURL}
	}
	leCacheDir = AbsPath(leCacheDir)

	/* Now that we're sure of the cache directory, set the cache. */
	mgr.Cache = autocert.DirCache(leCacheDir)

	/* O(1) lookups. */
	leds := make(map[string]struct{})
	for _, d := range leDomains {
		leds[d] = struct{}{}
	}

	// getCertificate gets the certificate for the given domain name.  It
	// first tries Let's Encrypt, if configured, then user-supplied certs,
	// then finally generates its own.
	getCertificate := func(
		chi *tls.ClientHelloInfo,
	) (*tls.Certificate, error) {
		/* If we're using Let's Encrypt for this one, life's easy. */
		if _, ok := leds[chi.ServerName]; ok {
			cert, err := mgr.GetCertificate(chi)
			if nil != err {
				return nil, fmt.Errorf(
					"getting Let's Encrypt "+
						"certificate: %w",
					err,
				)
			}
			return cert, nil
		}

		/* For our own certs, try to get a domain name or at least an
		IP address, or maybe just "default". */
		var d string
		if d = strings.Trim(chi.ServerName, "."); "" != d {
			/* Use SNI */
		} else if aper, ok := chi.Conn.LocalAddr().(interface {
			AddrPort() netip.AddrPort
		}); ok {
			/* Use IP address. */
			d = strings.Trim(
				aper.AddrPort().Addr().Unmap().String(),
				".",
			)
		} else if d = strings.Trim(
			chi.Conn.LocalAddr().String(),
			".",
		); "" != d {
			/* Use the local address, whatever it is. */
		} else {
			/* We tried. */
			d = defaultSNI
		}

		/* If we can get the cert from disk, do so. */
		cert, err := getLocalCert(d)
		if nil != err {
			return nil, fmt.Errorf("getting local cert: %w", err)
		} else if nil != cert {
			return cert, nil
		}

		/* Don't have files, either.  Roll our own. */
		cert, err = getSelfSigned(d, wlDomains)
		if nil != err {
			return nil, fmt.Errorf(
				"getting self-signed cert: %w",
				err,
			)
		}
		return cert, nil
	}

	return &tls.Config{
		GetCertificate: getCertificate,
		NextProtos:     []string{http2ALPN, httpALPN, acme.ALPNProto},
	}
}

// certAndKeyFilenames returns filenames for a non-Let's Encrypt cert and key
// for the given domain.
func certAndKeyFilenames(d string) (certF, keyF string) {
	if "" == d {
		d = defaultSNI
		log.Printf(
			"[%s] BUG: Got no SNI",
			MessageTypeError,
		)
	}
	f := filepath.Join(Env.LocalCertDir, d)
	return AbsPath(f + certFileSuffix), AbsPath(f + keyFileSuffix)
}

// getLocalCert gets a cert either from the cache or tries to read it from
// disk.  If there's no cert but no other errors occurred, getLocalCert returns
// (nil, nil).
func getLocalCert(d string) (*tls.Certificate, error) {
	localCacheL.Lock()
	defer localCacheL.Unlock()

	/* If we have it cached, life's easy. */
	if cert, ok := localCache[d]; ok {
		return cert, nil
	}

	/* If we have it as a file, get it. */
	certF, keyF := certAndKeyFilenames(d)
	cert, err := tls.LoadX509KeyPair(certF, keyF)
	if nil == err { /* Happy path. */
		localCache[d] = &cert
		return &cert, nil
	} else if errors.Is(err, os.ErrNotExist) {
		/* Just don't have it. */
		return nil, nil
	}

	/* Something actually went wrong. */
	return nil, fmt.Errorf(
		"getting certificate for %q from %q and %q: %w",
		d,
		certF, keyF,
		err,
	)
}

// getSelfSigned gets a self-signed cert for the domain d.  If there is no
// cached cert, it generates one.
func getSelfSigned(d string, wlDomains []string) (*tls.Certificate, error) {
	selfSignedCacheL.Lock()
	defer selfSignedCacheL.Unlock()

	/* If we already have a cert, life's easy. */
	c, ok := selfSignedCache[d]
	if ok {
		return c.cert, nil
	}

	/* If we're whitelisting, make sure this one's allowed. */
	if 0 != len(wlDomains) {
		var found bool
		for _, wlD := range wlDomains {
			if matched, err := filepath.Match(wlD, d); nil != err {
				log.Printf(
					"[%s] BUG: Uncaught invalid "+
						"domain glob %q: %s",
					MessageTypeError,
					wlD,
					err,
				)
			} else if matched {
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf(
				"domain not whitelisted: %s",
				d,
			)
		}
	}

	/* We don't, so generate one. */
	certP, keyP, err := generateCertAndKey(d)
	if nil != err {
		return nil, fmt.Errorf("generating cert and key: %w", err)
	}
	cert, err := tls.X509KeyPair(certP, keyP)
	if nil != err {
		return nil, fmt.Errorf("loading from PEM: %w", err)
	}
	selfSignedCache[d] = selfSignedCert{
		certPEM: certP,
		keyPEM:  keyP,
		cert:    &cert,
	}

	if "" == d {
		Verbosef(
			"[%s] Generated nameless self-signed certificate",
			MessageTypeTLS,
		)
	} else {
		Verbosef(
			"[%s] Generated self-signed certificate for %q",
			MessageTypeTLS,
			d,
		)
	}

	return &cert, nil
}

// generateCertAndKey generates a PEM-encoded cert and key for the give domain.
// Much of the below mooched from
// https://github.com/golang/go/blob/8e5f56a2e3a027e886d78f36675c275b9c845da0/src/crypto/tls/generate_cert.go
func generateCertAndKey(d string) (cert, key []byte, err error) {
	priv, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	if nil != err {
		return nil, nil, err
	}

	keyUsage := x509.KeyUsageDigitalSignature
	notBefore := time.Now()
	notAfter := notBefore.Add(selfSignedLifetime)
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, nil, fmt.Errorf(
			"generating serial number: %w",
			err,
		)
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject:      pkix.Name{CommonName: d},
		NotBefore:    notBefore,
		NotAfter:     notAfter,
		KeyUsage:     keyUsage,
		ExtKeyUsage: []x509.ExtKeyUsage{
			x509.ExtKeyUsageServerAuth,
		},
		BasicConstraintsValid: true,
	}
	if ip := net.ParseIP(d); ip != nil {
		template.IPAddresses = append(template.IPAddresses, ip)
	} else if "" != d {
		template.DNSNames = append(template.DNSNames, d)
	}

	derBytes, err := x509.CreateCertificate(
		rand.Reader,
		&template,
		&template,
		&priv.PublicKey,
		priv,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("creating certificate: %w", err)
	}

	privBytes, err := x509.MarshalPKCS8PrivateKey(priv)
	if nil != err {
		return nil, nil, fmt.Errorf("marshalling private key: %w", err)
	}
	certb := pem.EncodeToMemory(
		&pem.Block{Type: "CERTIFICATE", Bytes: derBytes},
	)
	keyb := pem.EncodeToMemory(
		&pem.Block{Type: "PRIVATE KEY", Bytes: privBytes},
	)

	return certb, keyb, nil
}

// saveSelfSignedCerts writes the cached self-signed certs to disk.
func saveSelfSignedCerts() {
	selfSignedCacheL.Lock()
	defer selfSignedCacheL.Unlock()

	/* Save ALL the certs. */
	for name, cert := range selfSignedCache {
		if err := saveSelfSignedCert(name, cert); nil != err {
			log.Printf(
				"[%s] Error saving self-signed cert "+
					"for %q: %s",
				MessageTypeError,
				name,
				err,
			)
		}
	}
}

// saveSelfSignedCert saves the name's certificate in the local certs
// directory.
func saveSelfSignedCert(name string, cert selfSignedCert) error {
	/* Work out where to save this thing. */
	certF, keyF := certAndKeyFilenames(name)

	/* Try to save it. */
	if err := os.WriteFile(certF, cert.certPEM, 0660); nil != err {
		return fmt.Errorf(
			"saving certificate to %q: %w",
			certF,
			err,
		)
	}
	if err := os.WriteFile(keyF, cert.keyPEM, 0600); nil != err {
		os.Remove(certF) /* Best effort. */
		return fmt.Errorf(
			"saving key to %q: %w",
			keyF,
			err,
		)
	}

	/* Tell the user what we did. */
	log.Printf(
		"[%s] Wrote keypair for %s to %s and %s",
		MessageTypeTLS,
		name,
		certF,
		keyF,
	)

	return nil
}
