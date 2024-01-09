package pbuf

/*
 * pbuf_test.go
 * Tests for pbuf.go
 * By J. Stuart McMurray
 * Created 20231215
 * Last Modified 20231215
 */

import (
	"testing"
)

func TestBufferReadWrite(t *testing.T) {
	b := new(Buffer)

	have := "kittens"
	n, err := b.Write([]byte(have))
	if nil != err {
		t.Fatalf("Write failed: %s", err)
	} else if n != len(have) {
		t.Fatalf(
			"Wrote incorrect number of bytes: got:%d want:%d",
			n,
			len(have),
		)
	}

	rb := make([]byte, 10*len(have))
	n, err = b.Read(rb)
	if nil != err {
		t.Fatalf("Read failed")
	} else if n != len(have) {
		t.Fatalf("Read incorrect size: got:%d want:%d", n, len(have))
	} else if got := string(rb[:n]); got != have {
		t.Fatalf("Read incorrect: got:%s want:%s", got, have)
	}
}
