package implantsvr

/*
 * handlers_test.go
 * Tests for handlers.go
 * By J. Stuart McMurray
 * Created 20231207
 * Last Modified 20231208
 */

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/magisterquis/plonk/internal/def"
	"github.com/magisterquis/plonk/lib/plog"
)

func TestHandleTasking(t *testing.T) {
	haveID := "kittens"
	rpath := def.TaskPath + "/" + haveID
	s, lb := newTestServer(t)
	rr, rb := resrec()
	s.SM.Lock()
	ls0 := s.SM.C.LastSeen[0]
	s.SM.Unlock()

	/* No tasking. */
	s.handleTasking(rr, httptest.NewRequest(http.MethodGet, rpath, nil))
	if http.StatusOK != rr.Code {
		t.Fatalf("Pre-q request: non-ok response %d", rr.Code)
	}
	if 0 != rb.Len() {
		t.Errorf("Pre-q request: got body %q", rb.String())
	}
	wantLog := `{"time":"","level":"DEBUG","msg":"Task request","qlen":0,` +
		`"id":"` + haveID + `","host":"example.com","method":"GET",` +
		`"remote_address":"192.0.2.1:1234","url":"/t/` + haveID + `"}`
	if gotLog := plog.RemoveTimestamp(lb.String()); gotLog != wantLog {
		t.Errorf(
			"Pre-q request log incorrect:\n got: %s\nwant: %s",
			gotLog,
			wantLog,
		)
	}
	s.SM.Lock()
	ls1 := s.SM.C.LastSeen[0]
	if ls1 == ls0 {
		t.Errorf("Pre-q last seen not updated")
	}
	ls0 = ls1
	s.SM.Unlock()
	lb.Reset()

	/* Add a task and try again. */
	rr = httptest.NewRecorder()
	haveTask := "moose"
	s.SM.Lock()
	s.SM.C.TaskQ[haveID] = append(s.SM.C.TaskQ[haveID], haveTask)
	s.SM.Unlock()
	s.handleTasking(rr, httptest.NewRequest(http.MethodGet, rpath, nil))
	if http.StatusOK != rr.Code {
		t.Fatalf("Post-q request: non-ok response %d", rr.Code)
	}
	if b, err := io.ReadAll(rr.Result().Body); nil != err {
		panic(err)
	} else if got := string(b); haveTask != got {
		t.Errorf(
			"Post-q tasking incorrect:\n got:%s\nwant: %s",
			got,
			haveTask,
		)
	}
	wantLog = `{"time":"","level":"INFO","msg":"Task request","qlen":0,` +
		`"task":"` + haveTask + `","id":"` + haveID + `",` +
		`"host":"example.com","method":"GET",` +
		`"remote_address":"192.0.2.1:1234","url":"/t/` + haveID + `"}`
	if gotLog := plog.RemoveTimestamp(lb.String()); gotLog != wantLog {
		t.Errorf(
			"Post-q request log incorrect:\n got: %s\nwant: %s",
			gotLog,
			wantLog,
		)
	}
	s.SM.Lock()
	ls1 = s.SM.C.LastSeen[0]
	if ls1 == ls0 {
		t.Errorf("Post-q last seen not updated")
	}
	if !ls1.When.After(ls0.When) {
		t.Errorf(
			"Post-q last seen updated, but times out of order:\n"+
				"old: %v\n"+
				"new: %v",
			ls0,
			ls1,
		)
	}
	ls0 = ls1
	s.SM.Unlock()
	lb.Reset()

	/* Make sure the queue is empty. */
	rr = httptest.NewRecorder()
	s.handleTasking(rr, httptest.NewRequest(http.MethodGet, rpath, nil))
	if http.StatusOK != rr.Code {
		t.Fatalf("Post-tasking request: non-ok response %d", rr.Code)
	}
	if b, err := io.ReadAll(rr.Result().Body); nil != err {
		panic(err)
	} else if 0 != len(b) {
		t.Errorf("Post-tasking request: got body %q", b)
	}
	wantLog = `{"time":"","level":"DEBUG","msg":"Task request","qlen":0,` +
		`"id":"` + haveID + `","host":"example.com","method":"GET",` +
		`"remote_address":"192.0.2.1:1234","url":"/t/` + haveID + `"}`
	if gotLog := plog.RemoveTimestamp(lb.String()); gotLog != wantLog {
		t.Errorf(
			"Post-tasking request log incorrect:\n got: %s\nwant: %s",
			gotLog,
			wantLog,
		)
	}
	s.SM.Lock()
	ls1 = s.SM.C.LastSeen[0]
	if ls1 == ls0 {
		t.Errorf("Post-tasking last seen not updated")
	}
	if !ls1.When.After(ls0.When) {
		t.Errorf(
			"Post-tasking last seen updated, "+
				"but times out of order:\n"+
				"old: %v\n"+
				"new: %v",
			ls0,
			ls1,
		)
	}
	ls0 = ls1
	s.SM.Unlock()
}

func TestHandleTasking_MultipleTasks(t *testing.T) {
	haveID := "kittens"
	rpath := def.TaskPath + "/" + haveID
	s, lb := newTestServer(t)
	nTask := 100
	tasks := make([]string, 0, 5)
	for i := 0; i < nTask; i++ {
		tasks = append(tasks, fmt.Sprintf("t%d", i))
	}
	s.SM.Lock()
	s.SM.C.TaskQ[haveID] = slices.Clone(tasks)
	s.SM.Unlock()

	for i := 0; i < nTask; i++ {
		rr := httptest.NewRecorder()
		var got def.EDLMTaskRequest
		wantT := tasks[0]
		tasks = slices.Delete(tasks, 0, 1)
		want := def.EDLMTaskRequest{
			ID:   haveID,
			Task: wantT,
			QLen: len(tasks),
		}
		rr.Body = new(bytes.Buffer)
		s.handleTasking(
			rr,
			httptest.NewRequest(http.MethodGet, rpath, nil),
		)
		if http.StatusOK != rr.Code {
			t.Errorf("[%d] Non-ok response %d", i, rr.Code)
			continue
		}
		if got := rr.Body.String(); got != wantT {
			t.Errorf(
				"[%d] Incorrect task: got:%s want:%s",
				i,
				got,
				wantT,
			)
		}
		if err := json.Unmarshal(lb.Bytes(), &got); nil != err {
			t.Errorf("[%d] Error unJSONing log: %s", i, err)
			continue
		}
		lb.Reset()
		if got != want {
			t.Errorf(
				"[%d] Incorrect log:\n got:%+v\nwant:%+v",
				i,
				got,
				want,
			)
		}

	}
}

func TestHandleOutput(t *testing.T) {
	haveID := "kittens"
	haveOutput := "This is Output!\n"
	rpath := def.OutputPath + "/" + haveID
	s, lb := newTestServer(t)
	rr, _ := resrec()

	s.handleOutput(rr, httptest.NewRequest(
		http.MethodPost,
		rpath,
		strings.NewReader(haveOutput),
	))
	if http.StatusOK != rr.Code {
		t.Errorf("Non-ok response %d", rr.Code)
	}

	wantLog := `{"time":"","level":"INFO","msg":"Output",` +
		`"output":"This is Output!","id":"kittens",` +
		`"host":"example.com","method":"POST",` +
		`"remote_address":"192.0.2.1:1234","url":"/o/kittens"}`
	if gotLog := plog.RemoveTimestamp(lb.String()); gotLog != wantLog {
		t.Errorf(
			"Log incorrect:\n got: %s\nwant: %s",
			gotLog,
			wantLog,
		)
	}

	s.SM.Lock()
	ls0 := s.SM.C.LastSeen[0]
	s.SM.Unlock()
	if haveID != ls0.ID {
		t.Errorf(
			"Last seen implant ID incorrect: got:%s want:%s",
			haveID,
			ls0.ID,
		)
	}
}

func TestLogIfNew(t *testing.T) {
	id := "kittens"
	s, lb := newTestServer(t)
	s.noSeen = false
	for i := 0; i < 3; i++ {
		rr := httptest.NewRecorder()
		s.handleTasking(rr, httptest.NewRequest(
			http.MethodGet,
			def.TaskPath+"/"+id,
			nil,
		))
		if http.StatusOK != rr.Code {
			t.Errorf("Incorrect task status %d", rr.Code)
		}
		rr = httptest.NewRecorder()
		s.handleOutput(rr, httptest.NewRequest(
			http.MethodPost,
			def.OutputPath+"/"+id,
			nil,
		))
		if http.StatusOK != rr.Code {
			t.Errorf("Incorrect output status %d", rr.Code)
		}
	}

	var (
		want = `{"time":"","level":"INFO","msg":"New implant",` +
			`"id":"kittens"}`
		sawLine bool
	)
	for i, l := range strings.Split(lb.String(), "\n") {
		if "" == l {
			continue
		}
		l = plog.RemoveTimestamp(l)
		if l == want {
			if sawLine {
				t.Errorf(
					"Duplicate new implant log line %d",
					i,
				)
			}
			sawLine = true
		}
	}
	if !sawLine {
		t.Errorf("Did not get New implant line")
	}
}

func TestHandleExfil(t *testing.T) {
	havePath := "foo/kittens/bar"
	haveExfil := "This is Exfil!\n"
	rpath := def.ExfilPath + "/" + havePath
	s, lb := newTestServer(t)
	rr, _ := resrec()

	s.handleExfil(rr, httptest.NewRequest(
		http.MethodPost,
		rpath,
		strings.NewReader(haveExfil),
	))
	if http.StatusOK != rr.Code {
		t.Errorf("Incorrect status %d", rr.Code)
	}

	wantLog := `{"time":"","level":"INFO","msg":"Exfil","size":15,` +
		`"filename":"` + s.Dir + `/exfil/foo/kittens/bar",` +
		`"hash":"d640c6c638d986f092b2688ee0aec215f6b5c9e200f4daa26fb` +
		`22ee6a5f2e9b4","host":"example.com","method":"POST",` +
		`"remote_address":"192.0.2.1:1234","url":"/p/foo/kittens/bar"}`
	if got := plog.RemoveTimestamp(lb.String()); got != wantLog {
		t.Errorf("Incorrect log:\n got: %s\nwant: %s", got, wantLog)
	}

	fn := filepath.Join(s.Dir, def.ExfilDir, havePath)
	if got, err := os.ReadFile(fn); nil != err {
		t.Errorf("Error reading exfil file %s: %s", fn, err)
	} else if string(got) != haveExfil {
		t.Errorf("Exfil incorrect\n got: %q\nwant: %q", got, haveExfil)
	}

}

func TestHandleStaticFile(t *testing.T) {
	for _, c := range []struct {
		path         string
		wantBody     string
		wantStatus   int    /* If not 200 */
		wantLocation string /* Location header. */
		wantLog      string
		prep         func(dir string) error
	}{{
		wantLog: `{"time":"","level":"INFO","msg":"Static file requested","status_code":200,"size":13,"filename":"","host":"example.com","method":"GET","remote_address":"192.0.2.1:1234","url":"/f"}`,
	}, {
		path:    "/",
		wantLog: `{"time":"","level":"INFO","msg":"Static file requested","status_code":200,"size":13,"filename":"/","host":"example.com","method":"GET","remote_address":"192.0.2.1:1234","url":"/f/"}`,
	}, {
		path:         "/d",
		wantStatus:   http.StatusMovedPermanently,
		wantLocation: "d/",
		wantLog:      `{"time":"","level":"INFO","msg":"Static file requested","status_code":301,"location":"d/","filename":"/d","host":"example.com","method":"GET","remote_address":"192.0.2.1:1234","url":"/f/d"}`,
		prep: func(d string) error {
			return os.MkdirAll(filepath.Join(d, "d"), def.DirPerms)
		},
	}, {
		path:    "/d/",
		wantLog: `{"time":"","level":"INFO","msg":"Static file requested","status_code":200,"size":13,"filename":"/d/","host":"example.com","method":"GET","remote_address":"192.0.2.1:1234","url":"/f/d/"}`,
		prep: func(d string) error {
			return os.MkdirAll(filepath.Join(d, "d"), def.DirPerms)
		},
	}, {
		path:     "/dlist/",
		wantBody: "<a href=\"f1\">f1</a>\n<a href=\"f2\">f2</a>",
		wantLog:  `{"time":"","level":"INFO","msg":"Static file requested","status_code":200,"size":53,"filename":"/dlist/","host":"example.com","method":"GET","remote_address":"192.0.2.1:1234","url":"/f/dlist/"}`,
		prep: func(d string) error {
			dn := filepath.Join(d, "dlist")
			if err := os.MkdirAll(dn, def.DirPerms); nil != err {
				return err
			}
			for _, fn := range []string{"f1", "f2"} {
				fn = filepath.Join(dn, fn)
				if err := os.WriteFile(
					fn,
					[]byte(fn),
					def.FilePerms,
				); nil != err {
					return err
				}
			}
			return nil
		},
	}, {
		path:     "/dlist/f2",
		wantBody: "Body of f2",
		wantLog:  `{"time":"","level":"INFO","msg":"Static file requested","status_code":200,"size":10,"filename":"/dlist/f2","host":"example.com","method":"GET","remote_address":"192.0.2.1:1234","url":"/f/dlist/f2"}`,
		prep: func(d string) error {
			dn := filepath.Join(d, "dlist")
			if err := os.MkdirAll(dn, def.DirPerms); nil != err {
				return err
			}
			for _, fn := range []string{"f1", "f2"} {
				ffn := filepath.Join(dn, fn)
				if err := os.WriteFile(
					ffn,
					[]byte("Body of "+fn),
					def.FilePerms,
				); nil != err {
					return err
				}
			}
			return nil
		},
	}} {
		c := c /* :( */
		t.Run(c.path, func(t *testing.T) {
			t.Parallel()
			s, lb := newTestServer(t)
			rr, rb := resrec()
			if nil != c.prep {
				if err := c.prep(filepath.Join(
					s.Dir,
					def.StaticFilesDir,
				)); nil != err {
					t.Fatalf("Prep failed: %s", err)
				}
			}
			s.handleStaticFile(rr, httptest.NewRequest(
				http.MethodGet,
				def.FilePath+c.path,
				nil,
			))
			if 0 == c.wantStatus {
				c.wantStatus = http.StatusOK
			}
			if c.wantStatus != rr.Code {
				t.Errorf("Incorrect status %d", rr.Code)
			}
			if got := rr.Result().Header.Get(
				"Location",
			); got != c.wantLocation {
				t.Errorf(
					"Location header incorrect:\n"+
						" got: %s\n"+
						"want: %s",
					got,
					c.wantLocation,
				)
			}
			got := rb.String()
			got = strings.TrimSpace(got)
			got = strings.TrimPrefix(got, "<pre>")
			got = strings.TrimSuffix(got, "</pre>")
			got = strings.TrimSpace(got)
			got = strings.TrimPrefix(got, "<pre>")
			if got != c.wantBody {
				t.Errorf(
					"Body incorrect:\n got: %q\nwant: %q",
					got,
					c.wantBody,
				)
			}
			if got := plog.RemoveTimestamp(
				lb.String(),
			); got != c.wantLog {
				t.Errorf(
					"Log incorrect:\n got: %s\nwant: %s",
					got,
					c.wantLog,
				)
			}
		})
	}
}
