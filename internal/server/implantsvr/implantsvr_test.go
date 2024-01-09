package implantsvr

/*
 * implantsvr_test.go
 * Listen for and handle implant requests
 * By J. Stuart McMurray
 * Created 20231207
 * Last Modified 20231213
 */

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/magisterquis/plonk/internal/def"
	"github.com/magisterquis/plonk/internal/server/state"
	"github.com/magisterquis/plonk/lib/plog"
)

func resrec() (*httptest.ResponseRecorder, *bytes.Buffer) {
	var b bytes.Buffer
	rr := httptest.NewRecorder()
	rr.Body = &b
	return rr, &b
}

func newTestServer(t *testing.T) (
	*Server,
	*bytes.Buffer,
) {
	d := t.TempDir()
	_, lb, sl := plog.NewTestLogger()
	s := &Server{
		Dir:      d,
		SL:       sl,
		SM:       state.NewTestState(t),
		HTTPAddr: "127.0.0.1:0",
		ExfilMax: 1024,
		noSeen:   true,
	}
	if err := s.Start(); nil != err {
		t.Fatalf("Error starting server: %s", err)
	}
	t.Cleanup(func() { s.Stop(nil) })
	lb.Reset()

	return s, lb
}

func TestServer_Smoketest(t *testing.T) {
	newTestServer(t)
}

func TestServer_Output(t *testing.T) {
	s, lb := newTestServer(t)
	id := "kittens"
	u := "http://" + s.HTTPListenAddr() + path.Join(
		"/",
		def.OutputPath,
		id,
	)
	res, err := http.Post(u, "", nil)
	if nil != err {
		t.Fatalf("Error sending request: %s", err)
	}
	defer res.Body.Close()
	wantStatus := http.StatusOK
	if got := res.StatusCode; got != wantStatus {
		t.Errorf(
			"Incorrect status: got:%d want:%d",
			got,
			wantStatus,
		)
	}
	b, err := io.ReadAll(res.Body)
	if nil != err {
		t.Errorf("Error reading body: %s", err)
	}
	if 0 != len(b) {
		if '\n' != b[len(b)-1] {
			b = append(b, '\n')
		}
		t.Errorf("Unexpected body:\n%s", b)
	}
	wantLog := `{"time":"","level":"DEBUG","msg":"Output","id":"kittens",` +
		`"host":"127.0.0.1:0","method":"POST",` +
		`"remote_address":"127.0.0.1:0","url":"/o/kittens"}`
	gotLog := regexp.MustCompile(
		`"127\.0\.0\.1:\d{1,5}"`,
	).ReplaceAllString(
		plog.RemoveTimestamp(lb.String()),
		`"127.0.0.1:0"`,
	)
	if gotLog != wantLog {
		t.Errorf("Log incorrect:\n got: %s\nwant: %s", gotLog, wantLog)
	}
}

func TestServer_DefaultFile(t *testing.T) {
	s, _ := newTestServer(t)
	u := "http://" + s.HTTPListenAddr() + path.Join("/okittens")
	wantStatus := http.StatusOK
	res, err := http.Get(u)
	if nil != err {
		t.Fatalf("Error sending first request: %s", err)
	}
	if got := res.StatusCode; got != wantStatus {
		t.Errorf(
			"Incorrect first status: got:%d want:%d",
			got,
			wantStatus,
		)
	}
	b, err := io.ReadAll(res.Body)
	if nil != err {
		t.Errorf("Error reading first body: %s", err)
	}
	if 0 != len(b) {
		t.Errorf("First body not empty: %s", b)
	}
	res.Body.Close()

	content := "It works :)"
	fn := filepath.Join(s.Dir, def.DefaultFile)
	if err := os.WriteFile(
		fn,
		[]byte(content),
		def.FilePerms,
	); nil != err {
		t.Fatalf("Error writing %s: %s", fn, err)
	}

	res, err = http.Get(u)
	if nil != err {
		t.Fatalf("Error sending second request: %s", err)
	}
	if got := res.StatusCode; got != wantStatus {
		t.Errorf(
			"Incorrect second status: got:%d want:%d",
			got,
			wantStatus,
		)
	}
	b, err = io.ReadAll(res.Body)
	if nil != err {
		t.Errorf("Error reading second body: %s", err)
	}
	if content != string(b) {
		t.Errorf("Second body not empty: %s", b)
	}
	res.Body.Close()
}
