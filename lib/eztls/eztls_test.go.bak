package eztls

/*
 * eztls_test.go
 * Tests for eztls.go
 * By J. Stuart McMurray
 * Created 20231027
 * Last Modified 20231223
 */

import (
	"context"
	"crypto/tls"
	"errors"
	"log"
	"net"
	"path/filepath"
	"slices"
	"testing"
)

func TestListen_Smoketest(t *testing.T) {
	d := t.TempDir()
	sp := filepath.Join(d, "sock")
	l, err := Listen("unix", sp, "example.com", true)
	if nil != err {
		t.Fatalf("Error: %s", err)
	}
	l.Close()
}

func TestHostWhitelist(t *testing.T) {
	ctx := context.Background()

	for _, c := range []struct {
		patterns   []string
		allowed    []string
		notAllowed []string
		err        error
	}{{
		patterns:   []string{"xn--9caa.com"},
		allowed:    []string{"éÉ.com"}, /* Idea from acme/autocert. */
		notAllowed: []string{"füß.com"},
	}, {
		patterns:   []string{"example.com"},
		allowed:    []string{"example.com"},
		notAllowed: []string{"ex", "ample.com", ".com", "", ".", "*"},
	}, {
		patterns:   []string{"a.com", "b.com"},
		allowed:    []string{"a.com", "b.com"},
		notAllowed: []string{"c.com", "foo.a.com", "bar.b.com"},
	}, {
		patterns: []string{"*.a.com", "b.*"},
		allowed: []string{
			"foo.a.com",
			"bar.a.com",
			"tridge.a.com",
			"b.foo",
			"b.bar",
			"b.",
			".a.com",
		},
		notAllowed: []string{"a.com", "b", "c.com"},
	}, {
		patterns:   []string{"a.com", "b.com", "a.com"},
		allowed:    []string{"a.com", "b.com"},
		notAllowed: []string{"c.com"},
	}, {
		patterns:   []string{},
		allowed:    []string{},
		notAllowed: []string{"a.com"},
	}, {
		patterns:   nil,
		allowed:    []string{},
		notAllowed: []string{"a.com"},
	}, {
		patterns:   []string{"*"},
		allowed:    []string{"a", "kittens.com", ""},
		notAllowed: []string{"abc 123", "\x00"},
	}} {
		c := c /* :C */
		t.Run("", func(t *testing.T) {
			t.Parallel()
			wl, err := HostWhitelist(c.patterns)
			if nil != err {
				if errors.Is(err, c.err) {
					return
				}
				t.Fatalf("Error: %s", err)
			}
			for _, s := range c.allowed {
				err := wl(ctx, s)
				if nil == err {
					continue
				}
				t.Errorf(
					"Incorrectly rejected: %q (%s)",
					s,
					err,
				)
			}
			for _, s := range c.notAllowed {
				if err := wl(ctx, s); nil != err {
					continue
				}
				t.Errorf("Incorrectly allowed: %q", s)
			}
		})
	}
}

func ExampleListen() {
	/* Listen on port 443 on all interfaces for a TLS connection for
	example.com.  Equivalent to
		l, err := ListenConfig("tcp", "0.0.0.0:443", Config{
			Domains: []string{"example.com"},
		})
	*/
	l, err := Listen("tcp", "0.0.0.0:443", "example.com", true)
	if nil != err {
		log.Fatalf("Listen error: %s", err)
	}

	/* Accept and handle connections as usual. */
	handle(l)
}

func ExampleListenConfig() {
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

	/* Accept and handle connections as usual. */
	handle(l)
}

func ExampleListenConfig_minimal() {
	/* Equivalent to Listen("tcp", "0.0.0.0:443", "example.com", false) */
	l, err := Config{
		Domains: []string{"example.com"},
	}.Listen("tcp", "0.0.0.0:443")
	if nil != err {
		log.Fatalf("Listen error: %s", err)
	}

	/* Accept and handle connections as usual. */
	handle(l)
}

func handle(l net.Listener) { l.Close() }
