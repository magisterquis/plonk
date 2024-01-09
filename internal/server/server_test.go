package server

/*
 * server_test.go
 * Tests for server.go
 * By J. Stuart McMurray
 * Created 20231129
 * Last Modified 20231207
 */

import (
	"bytes"
	"errors"
	"testing"
)

func newTestServer(t *testing.T) (*Server, *bytes.Buffer) {
	var lb bytes.Buffer
	s := &Server{
		Dir:           t.TempDir(),
		Debug:         true,
		HTTPAddr:      "127.0.0.1:0",
		testLogOutput: &lb,
	}
	if err := s.Start(); nil != err {
		t.Fatalf("Starting server: %s", err)
	}
	return s, &lb
}

func TestServer_Smoketest(t *testing.T) {
	newTestServer(t)
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
