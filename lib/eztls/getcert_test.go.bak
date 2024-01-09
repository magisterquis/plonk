package eztls

/*
 * getcert_test.go
 * Tests for getcert.go
 * By J. Stuart McMurray
 * Created 20231209
 * Last Modified 20231209
 */

import (
	"crypto/tls"
	"os"
	"testing"
)

func TestCertGetter(t *testing.T) {
	dir := t.TempDir()
	cg, err := Config{
		CacheDir:          dir,
		SelfSignedDomains: []string{"foo.com", "*.bar.com"},
	}.CertificateGetter()
	if nil != err {
		t.Fatalf("Error generating certificate-getter: %s", err)
	}

	for _, c := range []struct {
		name   string
		should bool
	}{
		{"foo.com", true},
		{"bar.com", false},
		{"tridge.bar.com", true},
		{"moose.com", false},
		{"", false},
	} {
		c := c /* :( */
		t.Run(c.name, func(t *testing.T) {
			cert, err := cg(&tls.ClientHelloInfo{
				ServerName: c.name,
			})
			if (nil == cert) == (nil == err) {
				t.Errorf(
					"Cert/err nil mismatch:\n"+
						"nil cert: %t\n"+
						"nil  err: %t",
					nil == cert,
					nil == err,
				)
			}
			if c.should != (err == nil) {
				t.Errorf(
					"Incorrect return:\n"+
						" got: %t\n"+
						"want: %t\n"+
						" err: %s",
					cert != nil,
					c.should,
					err,
				)
			}
		})
	}

	t.Run("cachedir", func(t *testing.T) {
		des, err := os.ReadDir(dir)
		if nil != err {
			t.Fatalf("Error reading cache directory: %s", err)
		}
		if n := len(des); 1 != n {
			t.Fatalf(
				"Expected 1 file in cache directory, got %d",
				n,
			)
		}
		if ssCacheKey != des[0].Name() {
			t.Errorf(
				"Incorrect cache file name:\n got: %s\nwant %s",
				des[0].Name(),
				ssCacheKey,
			)
		}
	})
}

func TestCertificateGetter_Wildcard(t *testing.T) {
	dir := t.TempDir()
	cg, err := Config{
		CacheDir:          dir,
		SelfSignedDomains: []string{"*"},
	}.CertificateGetter()
	if nil != err {
		t.Fatalf("Error generating certificate-getter: %s", err)
	}
	fc, err := cg(&tls.ClientHelloInfo{
		ServerName: "",
	})
	if nil != err {
		t.Fatalf("Error generating first cert: %s", err)
	}

	for _, c := range []string{
		"kittens.com",
		"foo",
		"",
		"1",
		"\xFF",
	} {
		c := c /* :( */
		t.Run(c, func(t *testing.T) {
			cert, err := cg(&tls.ClientHelloInfo{ServerName: c})
			if nil != err {
				t.Fatalf(
					"Cert not generated for %q: %s",
					c,
					err,
				)
			}
			if !cert.Leaf.Equal(fc.Leaf) {
				t.Fatalf("Cert changed for %q", c)
			}
		})
	}
}
