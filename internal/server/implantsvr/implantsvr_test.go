package implantsvr

/*
 * implantsvr_test.go
 * Listen for and handle implant requests
 * By J. Stuart McMurray
 * Created 20231207
 * Last Modified 20231208
 */

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"path"
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

func TestServer_Index(t *testing.T) {
	s, _ := newTestServer(t)
	res, err := http.Get("http://" + s.httpAddr)
	if nil != err {
		t.Fatalf("Error sending request: %s", err)
	}
	defer res.Body.Close()
	wantStatus := http.StatusNotFound
	if got := res.StatusCode; got != wantStatus {
		t.Errorf(
			"Incorrect status: got:%d want:%d",
			got,
			wantStatus,
		)
	}
	b, err := io.ReadAll(res.Body)
	if nil != err {
		t.Fatalf("Error reading body: %s", err)
	}
	want := "404 page not found\n"
	if got := string(b); got != want {
		t.Fatalf(
			"Body incorrect:\n got: %s\nwant: %s",
			got,
			want,
		)
	}
}

func TestServer_Output(t *testing.T) {
	s, lb := newTestServer(t)
	id := "kittens"
	u := "http://" + s.httpAddr + path.Join("/", def.OutputPath, id)
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
