// Package eztls - Easy TLS listener with Let's Encrypt
//
// Use of this package implies acceptance of Let's Encrypts Terms of Service.
package eztls

/*
 * eztls.go
 * Easy TLS listener with Let's Encrypt
 * By J. Stuart McMurray
 * Created 20231027
 * Last Modified 20231223
 */

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"golang.org/x/crypto/acme"
	"golang.org/x/crypto/acme/autocert"
	"golang.org/x/net/idna"
)

// HTTPSNextProtos are the ALPNs to use with tls.Config.NextProtos to serve
// HTTPS.
var HTTPSNextProtos = []string{"h2", "http/1.1", acme.ALPNProto}

// StagingACMEDirectory is the staging ACME Directory URL used by this package.
const StagingACMEDirectory = "https://acme-staging-v02.api.letsencrypt.org/directory"

// CacheDirBase is the base part of the directory returned by DefaultCacheDir.
const CacheDirBase = "eztls-cache"

// Config is the configuration passed to ListenConfig.  It is roughly analogous
// to autocert.Manager but simplified.
type Config struct {
	// Staging, if true, causes the Let's Encrypt staging environment to
	// be used.
	Staging bool

	// TLSConfig, if non-nil, will be passed by ListenConfig to tls.Dial
	// after setting its GetCertificate field.  Supplying a custom TLS
	// config is useful for configuring ALPNs (NextProtos).  Don't forget
	// to add acme.ALPNProto.
	TLSConfig *tls.Config

	// CacheDir is the directory in which are cached previously-obtained
	// certificates and other state.  If no directory is given, the
	// directory returned by DefalutCacheDir will be used if Staging isn't
	// true and an in-memory cache will be used if Staging is true.
	// Multiple instances of this library as well as multiple
	// autcert.Managers may use the same cache directory.
	CacheDir string

	// Domains is the list of domain SNI patterns for which TLS certificates
	// will be obtained.  This will be passed directly to HostWhitelist.
	// Domains may be nil or empty to disable usage of Let's Encrypt.
	Domains []string

	// SelfSignedDomains is an optional list of domain SNI patterns for
	// which self-signed TLS certificates will be generated if one can't be
	// obtained from Let's Encrypt or isn't allowed by Domains.
	// SelfSignedDomains may be nil or empty to disable self-signed
	// certificate generation.  The self-signed certificate itself will
	// have a single * DNS SAN and SelfSignedNames may contain "*" to
	// allow all SNIs, even empty ones.
	SelfSignedDomains []string

	// Email specifies an optional contact email address.  Please see
	// autocert.Manager.Email for more information.
	Email string
}

// Listen creates a TLS listener using tls.Listen and autocert.Manager
// according to the config.  Please see Listen for more details.
func (c Config) Listen(network, laddr string) (net.Listener, error) {
	/* Work out how to get a TLS certificate. */
	if nil == c.TLSConfig {
		c.TLSConfig = new(tls.Config)
	}
	var err error
	if c.TLSConfig.GetCertificate, err = c.CertificateGetter(); nil != err {
		return nil, fmt.Errorf(
			"configuring certificate-getter: %w",
			err,
		)
	}

	/* Start listening. */
	l, err := tls.Listen(network, laddr, c.TLSConfig)
	if nil != err {
		return nil, fmt.Errorf("starting listener: %w", err)
	}

	return l, nil
}

// Listen creates a TLS listener accepting connections on the given network
// address.  It will obtain and refresh a TLS certificates for the given domain
// pattern (see HostWhitelist) using tls-alpn-01 automatically from Let's
// Encrypt.  It uses Let's Encrypt's production environment unless staging is
// true, in which case it uses Let's Encrypt's staging environment.
func Listen(network, laddr, domain string, staging bool) (net.Listener, error) {
	return Config{
		Staging: staging,
		Domains: []string{domain},
	}.Listen(network, laddr)
}

// HostWhitelist returns an autocert.HostPolicy where only the specified
// hostname patterns are allowed.  Unlike autocert.HostWhitelist, patterns are
// taken as globs as matched by filepath.Match.  HostWhitelist does not retain
// patterns.  Hosts checked against the whitelist will be converted to
// Punycode with idna.Lookup.ToASCII.  The patterns will all be lowercased.
func HostWhitelist(patterns []string) (autocert.HostPolicy, error) {
	/* Make sure all of the patterns are valid. */
	for _, p := range patterns {
		if _, err := filepath.Match(p, ""); nil != err {
			return nil, fmt.Errorf("bad pattern %q: %w", p, err)
		}
	}

	/* Function to check the whitelist. */
	ps := slices.Compact(slices.Clone(patterns))
	for i, p := range ps {
		ps[i] = strings.ToLower(p)
	}
	return func(_ context.Context, host string) error {
		/* Turn the host into ASCII. */
		h, err := idna.Lookup.ToASCII(host)
		if nil != err {
			return fmt.Errorf("punycoding: %w", err)
		}
		for _, p := range ps {
			/* Only error is a bad pattern, and we check for
			that above. */
			if m, _ := filepath.Match(p, h); m {
				return nil
			}
		}
		return NotWhitelistedError{host}
	}, nil
}

// DefaultCacheDir returns the default directory in which are cached
// previously-obtained certs and other state.
func DefaultCacheDir() (string, error) {
	cd, err := os.UserCacheDir()
	if nil != err {
		return "", fmt.Errorf(
			"determining user cache directory: %w",
			err,
		)
	}
	return filepath.Join(cd, CacheDirBase), nil
}
