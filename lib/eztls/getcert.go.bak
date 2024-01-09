package eztls

/*
 * getcert.go
 * Get a TLS certificate for a ClientHello
 * By J. Stuart McMurray
 * Created 20231209
 * Last Modified 20231223
 */

import (
	"crypto/tls"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/acme"
	"golang.org/x/crypto/acme/autocert"
)

// CertificateGetter returns a function suitable for use in
// tls.Config.GetCertificate.  Don't forget to add acme.ALPNProto to the
// NetxProtos slice in the tls.Config.
func (c Config) CertificateGetter() (func(*tls.ClientHelloInfo) (*tls.Certificate, error), error) {
	var cg certGetter

	/* Make sure we have at least one whitelisted domain. */
	if 0 == len(c.Domains) && 0 == len(c.SelfSignedDomains) {
		return nil, fmt.Errorf("no whitelisted domains")
	}

	/* Set some defaults in the config. */
	if "" == c.CacheDir && !c.Staging {
		cd, err := DefaultCacheDir()
		if nil != err {
			return nil, fmt.Errorf(
				"determining default cache directory: %w",
				err,
			)
		}
		c.CacheDir = cd
	}

	/* Get the self-signed cert-getter. */
	if 0 != len(c.SelfSignedDomains) {
		var err error
		if cg.ss, err = SelfSignedGetter(
			c.SelfSignedDomains,
			0,
			c.CacheDir,
		); nil != err {
			return nil, fmt.Errorf(
				"generating self-signed "+
					"certificate-getter: %w",
				err,
			)
		}
	}

	/* If we don't have any Let's Encrypt domains, we're good. */
	if 0 == len(c.Domains) {
		return cg.getCertificate, nil
	}

	/* Work out what hosts we're allowed to get. */
	policy, err := HostWhitelist(c.Domains)
	if nil != err {
		return nil, fmt.Errorf(
			"generating Let's Encrypt host policy: %w",
			err,
		)
	}

	/* Make sure our cache directory exists, if we're using one. */
	var cache autocert.Cache
	if "" != c.CacheDir {
		var err error
		if cache, err = dirCache(c.CacheDir); nil != err {
			return nil, fmt.Errorf(
				"gerenating directory cache in %s: %w",
				c.CacheDir,
				err,
			)
		}
	}

	/* Work out where to ask for a certificate. */
	var client *acme.Client
	if c.Staging {
		client = &acme.Client{DirectoryURL: StagingACMEDirectory}
	}

	/* Add Let's Encrypt to our certificate-getter. */
	cg.le = (&autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		Cache:      cache,
		HostPolicy: policy,
		Client:     client,
		Email:      c.Email,
	}).GetCertificate

	return cg.getCertificate, nil
}

// certGetter gets certificates, first from leGetter, and failing that from
// ssGetter.
type certGetter struct {
	le func(*tls.ClientHelloInfo) (*tls.Certificate, error)
	ss func(*tls.ClientHelloInfo) (*tls.Certificate, error)
}

// getCertificate gets a certificate, first from le, if non-nil, then from
// ss if le indicates that it's not allowed.
func (cg certGetter) getCertificate(chi *tls.ClientHelloInfo) (*tls.Certificate, error) {
	/* Try Let's encrypt first. */
	var (
		lerr error
		nwe  NotWhitelistedError
	)
	if nil != cg.le {
		c, err := cg.le(chi)
		switch {
		case nil == err: /* Got one :) */
			return c, nil
		case strings.HasSuffix(
			err.Error(),
			"not configured in HostWhitelist",
		) || errors.As(err, &nwe):
			/* Not in the whitelist, that's ok for now. */
			lerr = err
		default: /* Some other, significant error. */
			return nil, fmt.Errorf(
				"getting Let's Encrypt certificate %w",
				err,
			)
		}
	}

	/* Try a self-signed certificate. */
	if nil != cg.ss {
		c, err := cg.ss(chi)
		if nil != err {
			return nil, fmt.Errorf(
				"getting self-signed certificate: %w",
				err,
			)
		}
		return c, nil
	}

	/* LE failed and we're not self-signing. */
	return nil, lerr
}
