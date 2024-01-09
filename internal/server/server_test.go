package server

/*
 * server_test.go
 * Tests for server.go
 * By J. Stuart McMurray
 * Created 20231129
 * Last Modified 20231219
 */

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/magisterquis/plonk/internal/def"
	"github.com/magisterquis/plonk/internal/server/perms"
	"github.com/magisterquis/plonk/lib/plog"
)

func init() {
	perms.MustSetProcessPerms()
}

func newTestServer(t *testing.T) (*Server, *bytes.Buffer) {
	var lb bytes.Buffer
	s := &Server{
		Dir:           filepath.Join(t.TempDir(), "plonk.d"),
		Debug:         true,
		HTTPAddr:      "127.0.0.1:0",
		ExfilMax:      1024,
		TestLogOutput: &lb,
	}
	if err := s.Start(); nil != err {
		t.Fatalf("Starting server: %s", err)
	}
	t.Cleanup(func() { s.Stop(errors.New("test finished")) })
	t.Cleanup(func() { s.lf.Close() })
	return s, &lb
}

func TestServer_Smoketest(t *testing.T) {
	newTestServer(t)
}

func TestServer_DirPerms(t *testing.T) {
	s, _ := newTestServer(t)
	if err := filepath.Walk(s.Dir, func(
		p string,
		fi fs.FileInfo,
		err error,
	) error {
		var want fs.FileMode
		if fi.Mode().IsRegular() {
			want = def.FilePerms
		} else {
			want = def.DirPerms
		}
		if got := fi.Mode().Perm(); got != want {
			t.Errorf(
				"%s has incorrect permissions: got:%o want:%o",
				p,
				got,
				want,
			)
		}
		return nil
	}); nil != err {
		t.Fatalf("Error walking %s: %s", s.Dir, err)
	}

}

func TestServerStop(t *testing.T) {
	s, _ := newTestServer(t)
	want := errors.New("kittens")
	got := s.Stop(want)
	if !errors.Is(got, want) {
		t.Errorf(
			"Stop returned incorrect error:\n got:%s\nwant:%s",
			got,
			want,
		)
	}
	got = s.Wait()
	if !errors.Is(got, want) {
		t.Errorf(
			"Wait returned incorrect error:\n got:%s\nwant:%s",
			got,
			want,
		)
	}
}

func TestServer_DisableExfil(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(2)
	t.Run("exfil_ok", func(t *testing.T) {
		defer wg.Done()
		t.Parallel()
		s, lb := newTestServer(t)
		have := "Kittens!"
		fn := "/kittens"
		sfn := filepath.Join(s.Dir, def.ExfilDir, fn)
		u := "http://" + s.is.HTTPListenAddr() + def.ExfilPath + fn
		lb.Reset()
		res, err := http.Post(u, "", strings.NewReader(have))
		if nil != err {
			t.Fatalf("Request error: %s", err)
		}
		defer res.Body.Close()
		if http.StatusOK != res.StatusCode {
			t.Errorf("Incorrect status: %s", res.Status)
		}
		want := `{"time":"","level":"INFO","msg":"Exfil","size":8,` +
			`"hash":"2d5964650365142d70c633a937897eddf3febdb2d47` +
			`1c496650e498ac2d15131","filename":"` + sfn + `",` +
			`"requested_path":"` + fn + `",` +
			`"host":"` + s.is.HTTPListenAddr() + `",` +
			`"method":"POST","remote_address":"127.0.0.1:x",` +
			`"url":"` + path.Join(def.ExfilPath, fn) + `"}`

		got := regexp.MustCompile(
			`("remote_address":"127.0.0.1:)\d+`,
		).ReplaceAllString(
			plog.RemoveTimestamp(lb.String()),
			"${1}x",
		)
		if got != want {
			t.Errorf(
				"Incorrect log:\n got: %s\nwant: %s",
				got,
				want,
			)
		}
		if got, err := os.ReadFile(sfn); nil != err {
			t.Errorf("Error reading exfil file %s: %s", sfn, err)
		} else if string(got) != have {
			t.Errorf(
				"Saved exfil incorrect:\n got: %s\nwant: %s\n",
				got,
				have,
			)
		}
	})
	t.Run("exfil_disabled", func(t *testing.T) {
		defer wg.Done()
		t.Parallel()
		var lb bytes.Buffer
		s := &Server{
			Dir:           t.TempDir(),
			Debug:         true,
			HTTPAddr:      "127.0.0.1:0",
			TestLogOutput: &lb,
		}
		if err := s.Start(); nil != err {
			t.Fatalf("Unable to start server: %s", err)
		}
		defer s.Stop(nil)
		have := "Kittens!"
		fn := "/kittens"
		sfn := filepath.Join(s.Dir, def.ExfilDir, fn)
		u := "http://" + s.is.HTTPListenAddr() + def.ExfilPath + fn
		lb.Reset()
		res, err := http.Post(u, "", strings.NewReader(have))
		if nil != err {
			t.Fatalf("Request error: %s", err)
		}
		defer res.Body.Close()
		if http.StatusOK != res.StatusCode {
			t.Errorf("Incorrect status: %s", res.Status)
		}
		if 0 != lb.Len() {
			t.Errorf("Unexpected log: %s", lb.String())
		}
		got, err := os.ReadFile(sfn)
		switch {
		case errors.Is(err, fs.ErrNotExist): /* Good. */
		case nil == err: /* Exfil happened. */
			t.Errorf("Exfil written to %s: %s", sfn, got)
		default: /* Some other error. */
			t.Errorf("Error reading exfil file %s: %s", sfn, err)
		}
	})
}

func TestServer_DefaultSelfsigned(t *testing.T) {
	var lb bytes.Buffer
	s := &Server{
		Dir:           t.TempDir(),
		Debug:         true,
		HTTPSAddr:     "127.0.0.1:0",
		TestLogOutput: &lb,
	}
	if err := s.Start(); nil != err {
		t.Fatalf("Starting server: %s", err)
	}
	t.Cleanup(func() { s.Stop(errors.New("test finished")) })
	t.Cleanup(func() { s.lf.Close() })

	getCert := func(sni string) (*x509.Certificate, error) {
		sa := s.is.HTTPSListenAddr()
		c, err := tls.Dial("tcp", sa, &tls.Config{
			InsecureSkipVerify: true,
			ServerName:         sni,
		})
		if nil != err {
			return nil, fmt.Errorf("dial %s: %w", sa, err)
		}
		defer c.Close()
		if err := c.Handshake(); nil != err {
			return nil, fmt.Errorf("handshake: %w", err)
		}
		return c.ConnectionState().PeerCertificates[0], nil
	}
	fc, err := getCert("")
	if nil != err {
		t.Fatalf("Error getting default cert: %s", err)
	}

	for _, c := range []string{
		"foo.com",
		"",
		"1.2.3.4",
		"::",
		"abc",
	} {
		c := c /* :( */
		t.Run(c, func(t *testing.T) {
			cert, err := getCert(c)
			if nil != err {
				t.Fatalf("Error getting cert: %s", err)
			}
			if !cert.Equal(fc) {
				t.Errorf("Cert different from first cert")
			}
		})
	}
}

func TestServer_SSDomainWhitelist(t *testing.T) {
	var lb bytes.Buffer
	s := &Server{
		Dir:               t.TempDir(),
		Debug:             true,
		HTTPSAddr:         "127.0.0.1:0",
		SSDomainWhitelist: []string{"foo.com", "*.bar.com", "bar.com"},
		TestLogOutput:     &lb,
	}
	if err := s.Start(); nil != err {
		t.Fatalf("Starting server: %s", err)
	}
	t.Cleanup(func() { s.Stop(errors.New("test finished")) })
	t.Cleanup(func() { s.lf.Close() })

	sa := s.is.HTTPSListenAddr()
	for _, c := range []struct {
		name string
		want bool
	}{
		{"foo.com", true},
		{"bar.com", true},
		{"trideg.bar.com", true},
		{"", false},
		{"kittens.com", false},
		{"moose", false},
	} {
		c := c /* :( */
		t.Run(c.name, func(t *testing.T) {
			tc, err := tls.Dial("tcp", sa, &tls.Config{
				InsecureSkipVerify: true,
				ServerName:         c.name,
			})
			if nil != tc {
				tc.Close()
			}

			if (nil == err && c.want) || (nil != err && !c.want) {
				return
			}

			if nil == err {
				t.Fatalf("Connection incorrectly succeeded")
			} else {
				t.Fatalf(
					"Connection incorrectly failed: %s",
					err,
				)
			}
		})
	}
}
