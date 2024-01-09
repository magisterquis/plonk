package client

/*
 * client_test.go
 * Tests for client.go
 * By J. Stuart McMurray
 * Created 20231214
 * Last Modified 20231222
 */

import (
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/magisterquis/plonk/internal/server"
	"github.com/magisterquis/plonk/lib/pbuf"
)

const testOpName = "testop"

func newTestClient(t *testing.T) (
	in *io.PipeWriter, /* Client stdin. */
	out *pbuf.Buffer, /* Client stdout. */
	slb *pbuf.Buffer, /* Server logs. */
	rch <-chan error, /* Return from c.Run(). */
	c *Client, /* Client itself. */
	svr *server.Server, /* Server. */
) {
	var (
		dir      = t.TempDir()
		inr, inw = io.Pipe()
		ch       = make(chan error, 1)
	)
	out = new(pbuf.Buffer)
	slb = new(pbuf.Buffer)

	svr = &server.Server{
		Dir:           dir,
		HTTPAddr:      "127.0.0.1:0",
		TestLogOutput: slb,
	}
	if err := svr.Start(); nil != err {
		t.Fatalf("Error starting test server: %s", err)
	}
	t.Cleanup(func() {
		svr.Stop(errors.New("test finished"))
		svr.Wait()
		svr.CloseLogfile()
	})

	client := Client{
		Dir:    dir,
		Debug:  true,
		Name:   testOpName,
		Stdin:  inr,
		Stdout: out,
		Stderr: out,
	}
	if err := client.Start(); nil != err {
		t.Fatalf("Error starting client: %s", err)
	}
	go func() { ch <- client.Wait(); close(ch) }()
	t.Cleanup(func() { inw.Close(); client.Wait() })

	/* Remove the connected message. */
	for !strings.HasSuffix(out.String(), "\n") {
		time.Sleep(time.Nanosecond) /* More a yield than anything. */
	}
	if !strings.HasSuffix(
		out.String(),
		"[OPERATOR] Connected: "+testOpName+" (cnum:1)\n",
	) {
		t.Fatalf("Incorrect first log line: %s", out.String())
	}
	out.Reset()

	return inw, out, slb, ch, &client, svr
}

func TestClient_Smoketest(t *testing.T) {
	newTestClient(t)
}

func TestClient_CloseInput(t *testing.T) {
	in, out, slb, rch, _, _ := newTestClient(t)
	if err := in.Close(); nil != err {
		t.Fatalf("Error closing input stream: %s", err)
	}
	if err := <-rch; nil != err {
		t.Fatalf(
			"Error after closing stdin: %s\n"+
				"Client output:\n%s\n"+
				"Server logs:\n%s",
			err,
			out.String(),
			slb.String(),
		)
	}
}

func TestClient_Valediction(t *testing.T) {
	have := errors.New("test error")
	_, out, _, _, c, svr := newTestClient(t)
	svr.Stop(have)
	if err := c.Wait(); nil != err {
		t.Fatalf("Client error: %s", err)
	}
	want := `Server sent a valediction:

Error: test error

`
	l := out.String()
	parts := strings.SplitN(l, " ", 3)
	if 3 != len(parts) {
		t.Fatalf("Invalid log:\n%s", l)
	}
	if got := parts[2]; got != want {
		t.Fatalf(
			"Incorrect client output:\n got:\n%s\nwant:\n%s",
			got,
			want,
		)
	}
}
