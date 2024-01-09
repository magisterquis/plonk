package eztls

/*
 * selfsigned_test.go
 * Tests for selfsigned.go
 * By J. Stuart McMurray
 * Created 20231209
 * Last Modified 20231209
 */

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"
)

func TestSelfSignedGetter_Whitelist(t *testing.T) {
	domains := []string{"foo.com", "*.bar.com"}
	ssg, err := SelfSignedGetter(domains, 0, "")
	if nil != err {
		t.Fatalf("Error generating self-signed cert-getter: %s", err)
	}

	for _, c := range []struct {
		name   string
		should bool
	}{
		{"foo.com", true},
		{"tridge.bar.com", true},
		{"moose.com", false},
		{"", false},
	} {
		c := c /* :( */
		t.Run(c.name, func(t *testing.T) {
			cert, err := ssg(&tls.ClientHelloInfo{
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
}

func TestSelfSignedGetter_DirCache(t *testing.T) {
	d := t.TempDir()
	domains := []string{"*"}
	ssg, err := SelfSignedGetter(domains, 0, d)
	if nil != err {
		t.Fatalf("Error generating self-signed cert-getter: %s", err)
	}

	des, err := os.ReadDir(d)
	if nil != err {
		t.Fatalf("Error reading directory before usage: %s", err)
	}
	if 0 != len(des) {
		t.Fatalf("New directory not empty")
	}

	cert1, err := ssg(&tls.ClientHelloInfo{ServerName: "c1"})
	if nil != err {
		t.Fatalf("Error getting first cert: %s", err)
	}

	des, err = os.ReadDir(d)
	if nil != err {
		t.Fatalf(
			"Error reading directory after getting first cert: %s",
			err,
		)
	}
	switch len(des) {
	case 1: /* Good. */
	case 0:
		t.Errorf("Cert not written to directory")
	default:
		t.Errorf(
			"Expected 1 certificate written, got %d dents",
			len(des),
		)
	}
	if ssCacheKey != des[0].Name() {
		t.Errorf(
			"Incorrect cache file name:\n got: %s\nwant %s",
			des[0].Name(),
			ssCacheKey,
		)
	}

	cert2, err := ssg(&tls.ClientHelloInfo{ServerName: "c2"})
	if nil != err {
		t.Fatalf("Error getting second cert: %s", err)
	}

	if !cert1.Leaf.Equal(cert2.Leaf) {
		t.Fatalf("Certificates not equal")
	}

	cf1, err := os.ReadFile(filepath.Join(d, ssCacheKey))
	if nil != err {
		t.Fatalf("Error reading cached file: %s", err)
	}
	if 0 == len(cf1) {
		t.Fatalf("Cached file empty")
	}

	ssg2, err := SelfSignedGetter(domains, 0, d)
	if nil != err {
		t.Fatalf("Error generating second cert-generator: %s", err)
	}
	cert3, err := ssg2(&tls.ClientHelloInfo{ServerName: "c3"})
	if nil != err {
		t.Fatalf("Error getting third cert: %s", err)
	}
	if !cert3.Leaf.Equal(cert2.Leaf) {
		t.Errorf("Cert3 different than the first two")
	}
	cf2, err := os.ReadFile(filepath.Join(d, ssCacheKey))
	if nil != err {
		t.Fatalf(
			"Error reading cached file "+
				"after second cert-getter used:%s",
			err,
		)
	}
	if !bytes.Equal(cf1, cf2) {
		t.Fatalf("Cached file changed after second cert-getter used")
	}
}

func TestGenerateSelfSignedCertificate(t *testing.T) {
	for _, c := range []struct {
		subject     string
		dnsNames    []string
		ipAddresses []net.IP
		expiry      time.Time
	}{{
		subject: "kittens",
	}, {
		subject: "",
	}, {
		subject:  "kittens.com",
		dnsNames: []string{"kittens.com", "*.moose.com", ".", "*"},
		ipAddresses: []net.IP{
			net.IPv4(1, 2, 3, 4),
			net.IPv4(0, 0, 0, 0),
			net.ParseIP("::"),
			net.ParseIP("a::b"),
		},
		expiry: time.Now().Add(time.Minute),
	}} {
		c := c /* :C */
		t.Run(c.subject, func(t *testing.T) {
			g, err := generateSelfSignedCert(
				c.subject,
				c.dnsNames,
				c.ipAddresses,
				c.expiry,
			)
			if nil != err {
				t.Fatalf("Generation failed: %s", err)
			}

			if n := len(g.Certificate); 1 != n {
				t.Errorf("Expected 1 certificate, got %d", n)
			}

			if nil == g.Leaf {
				t.Fatalf("Leaf is nil")
			}

			want := c.subject
			if "" == want {
				want = SelfSignedSubject
			}
			want = "CN=" + want
			if got := g.Leaf.Subject.String(); want != got {
				t.Errorf(
					"Subject incorrect\n"+
						" got: %s\n"+
						"want: %s",
					got,
					want,
				)
			}

			if !slices.Equal(g.Leaf.DNSNames, c.dnsNames) {
				t.Errorf(
					"DNSNames incorrect:\n"+
						" got: %s\n"+
						"want: %s",
					g.Leaf.DNSNames,
					c.dnsNames,
				)
			}

			if !slices.EqualFunc(
				g.Leaf.IPAddresses,
				c.ipAddresses,
				func(a, b net.IP) bool { return a.Equal(b) },
			) {
				t.Errorf(
					"IPAddresses incorrect:\n"+
						" got: %s\n"+
						"want: %s",
					g.Leaf.IPAddresses,
					c.ipAddresses,
				)
			}

			gt := g.Leaf.NotAfter.UTC()
			wt := c.expiry.UTC().Truncate(time.Second)
			if !gt.Equal(wt) {
				t.Errorf(
					"Expiry incorrect:\n"+
						"got: %s\n"+
						"want: %s",
					gt,
					wt,
				)
			}
		})
	}
}

func ExampleSelfSignedGetter() {
	/* Self-signed certificate generator generation, handy for testing. */
	ssg, err := SelfSignedGetter([]string{"*"}, 0, "")
	if nil != err {
		log.Fatalf("Faild to generate certificate generator: %s", err)
	}

	/* Listen for and handshake with a TLS connection. */
	l, err := tls.Listen("tcp", "127.0.0.1:0", &tls.Config{
		GetCertificate: ssg,
	})
	if nil != err {
		log.Fatalf("Listen: %s", err)
	}
	go func() {
		c, err := l.Accept()
		if nil != err {
			log.Fatalf("Accept: %s", err)
		}
		if err := c.(*tls.Conn).Handshake(); nil != err {
			log.Fatalf("Handhake: %s", err)
		}
	}()
	defer l.Close()

	/* Connect to our fancy new server. */
	c, err := tls.Dial("tcp", l.Addr().String(), &tls.Config{
		ServerName:         "kittens.test",
		InsecureSkipVerify: true, /* Because self-signed. */
	})
	if nil != err {
		log.Fatalf("Dial: %s", err)
	}
	defer c.Close()

	/* What's the cert like? */
	svr := c.ConnectionState().PeerCertificates[0]
	fmt.Printf(
		"--Certificate info--\n"+
			" Subject: %s\n"+
			"DNS SANs: %s\n",
		svr.Subject,
		svr.DNSNames,
	)

	// Output:
	// --Certificate info--
	//  Subject: CN=eztls
	// DNS SANs: [*]
}
