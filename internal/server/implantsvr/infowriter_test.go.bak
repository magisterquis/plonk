package implantsvr

/*
 * infowriter_test.go
 * Tests for infowriter.go
 * By J. Stuart McMurray
 * Created 20231208
 * Last Modified 20231208
 */

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestInfoWriter(t *testing.T) {
	noCode := -1
	for _, c := range []struct {
		Code int
		Body string
	}{
		{noCode, "bodytest"},
		{http.StatusOK, "Kittens"},
		{http.StatusNotFound, ""},
	} {
		c := c /* :( */
		t.Run("", func(t *testing.T) {
			var (
				iw infoWriter
				rr = httptest.NewRecorder()
			)
			http.HandlerFunc(func(
				w http.ResponseWriter,
				r *http.Request,
			) {
				iw.Wrapped = w
				if noCode != c.Code {
					iw.WriteHeader(c.Code)
				}
				io.WriteString(&iw, c.Body)
			}).ServeHTTP(rr, httptest.NewRequest("", "/", nil))

			want := c.Code
			if noCode == want {
				want = http.StatusOK
			}
			if got := int(iw.StatusCode.Load()); got != want {
				t.Errorf(
					"Incorrect status code:\n"+
						" got: %d\n"+
						"want: %d\n"+
						" res: %d",
					got,
					want,
					rr.Code,
				)
			}
			if got, _ := io.ReadAll(
				rr.Result().Body,
			); c.Body != string(got) {
				t.Errorf(
					"Incorrect body:\n got: %s\nwant: %s",
					got,
					c.Body,
				)
			}
			if got := int(iw.Written.Load()); got != len(c.Body) {
				t.Errorf(
					"Incorrect length: got:%d want:%d",
					got,
					len(c.Body),
				)
			}
		})
	}
}
