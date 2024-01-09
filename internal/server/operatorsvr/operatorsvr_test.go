package operatorsvr

/*
 * operatorsvr_test.go
 * Tests for operatorsvr.go
 * By J. Stuart McMurray
 * Created 20231129
 * Last Modified 20231208
 */

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/magisterquis/plonk/internal/def"
	"github.com/magisterquis/plonk/internal/server/state"
	"github.com/magisterquis/plonk/lib/estream"
	"github.com/magisterquis/plonk/lib/flexiwriter"
	"github.com/magisterquis/plonk/lib/plog"
)

const testClientName = "test_client"

func lastLine(b *bytes.Buffer) string {
	ls := strings.Split(strings.TrimRight(b.String(), "\n"), "\n")
	if 0 == len(ls) {
		return ""
	}
	return ls[len(ls)-1]
}

func waitForLine(b *bytes.Buffer) {
	for s := b.Bytes(); 0 == len(s) || '\n' != s[len(s)-1]; s = b.Bytes() {
		time.Sleep(10 * time.Nanosecond)
	}
}

func newTestServer(t *testing.T) (*Server, *bytes.Buffer, net.Conn) {
	var (
		lb     = new(bytes.Buffer)
		fw     = flexiwriter.New(lb)
		lv, sl = plog.NewJSONLogger(fw)
		s      = &Server{
			Dir: t.TempDir(),
			SL:  sl,
			FW:  fw,
			SM:  state.NewTestState(t),
		}
	)
	lv.Set(slog.LevelDebug)

	if err := s.Start(); nil != err {
		t.Fatalf("Error starting server: %s", err)
	}

	c, err := net.Dial("unix", s.l.Addr().String())
	if nil != err {
		t.Fatalf("Error connecting to server: %s", err)
	}
	t.Cleanup(func() {
		if err := s.Stop("test end"); nil != err {
			t.Errorf("Error stopping server: %s", err)
		}
		if err := c.Close(); nil != err &&
			!errors.Is(err, net.ErrClosed) {
			t.Fatalf("Error closing test connection: %s", err)
		}
	})

	/* Send our name. */
	if err := estream.New(c).Send(
		def.ENName,
		def.EDName(testClientName),
	); nil != err {
		t.Fatalf("Error sending name to server: %s", err)
	}

	/* We should get a line on c that says we've connected.  We read it
	this way so as to avoid buffering. */
	getLine := func() string {
		var (
			cb bytes.Buffer
			rb = make([]byte, 1)
		)
		for {
			n, err := c.Read(rb)
			if 0 == n && nil == err {
				err = errors.New("0-byte read")
			}
			if nil != err {
				t.Fatalf(
					"Error reading from server: %s\n"+
						"Log:\n%s",
					err,
					lb.String(),
				)
			}
			if '\n' == rb[0] {
				return cb.String()
			}
			cb.WriteByte(rb[0])
		}
	}
	b, err := json.Marshal(def.LMOpConnected)
	if nil != err {
		t.Fatalf("Failed to marshal %q: %s", def.LMOpConnected, err)
	}
	want := string(b)
	if got := getLine(); got != want {
		t.Fatalf(
			"Incorrect op connected message type:\n"+
				" got: %q\n"+
				"want: %q",
			got,
			want,
		)
	}
	want = `{"time":"","level":"INFO","msg":"Operator connected","opname":"` + testClientName + `","cnum":1}`
	if got := plog.RemoveTimestamp(getLine()); got != want {
		t.Fatalf(
			"Incorrect op connected message:\n"+
				" got: %s\n"+
				"want: %s",
			got,
			want,
		)
	}

	/* Remove the not interesting messages from the log buffer as well. */
	lb.Reset()

	return s, lb, c
}

func TestLastLine(t *testing.T) {
	for _, c := range []struct {
		have string
		want string
		skip int
	}{{
		have: "foo\nbar\ntridge",
		want: "tridge",
	}, {
		have: "foo\nbar\ntridge\n",
		want: "tridge",
	}, {
		have: "foo\nbar\ntridge\n\n\n",
		want: "tridge",
	}, {
		have: "\n\n\n\n",
		want: "",
	}, {
		have: "",
		want: "",
	}} {
		c := c /* :( */
		t.Run("", func(t *testing.T) {
			b := bytes.NewBufferString(c.have)
			got := lastLine(b)
			if got == c.want {
				return
			}
			t.Fatalf(
				"Incorrect last line:\n"+
					"have: %q\n"+
					" got: %q\n"+
					"want: %q",
				c.have,
				got,
				c.want,
			)
		})
	}
}

func TestServer_Smoketest(t *testing.T) {
	newTestServer(t)
}

func TestServerStop(t *testing.T) {
	have := "test reason"
	s, _, c := newTestServer(t)
	if err := s.Stop(have); nil != err {
		t.Fatalf("Error on stop: %s", err)
	}

	want := strings.Join([]string{
		`"goodbye"`,
		`{"Message":"test reason"}`,
		``,
	}, "\n")
	b, err := io.ReadAll(c)
	if nil != err {
		t.Errorf("Error reading from conn: %s", err)
	}
	if got := string(b); got != want {
		t.Errorf(
			"Incorrect shutdown message:\n"+
				"have: %s\n"+
				"got: %q\n"+
				"want: %q",
			have,
			got,
			want,
		)
	}
}
