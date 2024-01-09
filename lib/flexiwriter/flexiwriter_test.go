package flexiwriter

/*
 * flexiwriter_test.go
 * Tests for flexiwriter.go
 * By J. Stuart McMurray
 * Created 20231006
 * Last Modified 20231208
 */

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"sync"
	"testing"
)

func TestWriter(t *testing.T) {
	fw := New()
	fw.WriteString("one")

	/* Simple write. */
	var b1 bytes.Buffer
	fw.Add(&b1, nil)
	s := "two"
	fw.WriteString(s)
	if got := b1.String(); s != got {
		t.Fatalf("After one writer: got:%q want:%q", got, s)
	}

	/* Removal */
	var b2 bytes.Buffer
	var wg sync.WaitGroup
	wg.Add(1)
	var rerr error
	fw.Add(&b2, func(err error) { rerr = err; wg.Done() })
	fw.Remove(&b2)
	wg.Wait()
	if nil != rerr {
		t.Fatalf("Remove error: %s", rerr)
	}

	/* Multiple Write */
	var b3 bytes.Buffer
	fw.Add(&b3, nil)
	b1.Reset()
	s = "three"
	fw.WriteString(s)
	if got := b1.String(); s != got {
		t.Fatalf("With two writers, b1: got:%q want:%q", got, s)
	}
	if got := b3.String(); s != got {
		t.Fatalf("With two writers, b3: got:%q want:%q", got, s)
	}
	if "" != b2.String() {
		t.Fatalf("After removal, b2: got:%q", b2.String())
	}

	/* Write Error. */
	werr := errors.New("test error")
	rerr = nil
	pr, pw := io.Pipe()
	pr.CloseWithError(werr)
	wg.Add(1)
	fw.Add(pw, func(err error) { rerr = err; wg.Done() })
	s = "four"
	b1.Reset()
	b3.Reset()
	fw.WriteString(s)
	if got := b1.String(); s != got {
		t.Fatalf("After write error, b1: got:%q want:%q", got, s)
	}
	if got := b3.String(); s != got {
		t.Fatalf("After write error, b3: got:%q want:%q", got, s)
	}
	wg.Wait()
	if !errors.Is(rerr, werr) {
		t.Fatalf(
			"onremove got wrong error: got:%s want:%s",
			rerr,
			werr,
		)
	}
}

func TestNew(t *testing.T) {
	var b1, b2 bytes.Buffer
	fw := New(&b1, &b2)
	s := "one"
	if _, err := fw.Write([]byte(s)); nil != err {
		t.Fatalf("Write error: %s", err)
	}
	if got := b1.String(); s != got {
		t.Fatalf("After write, b1: got:%q want:%q", got, s)
	}
	if got := b2.String(); s != got {
		t.Fatalf("After write, b2: got:%q want:%q", got, s)
	}
}

func TestAdd_nil(t *testing.T) {
	fw := New(nil, nil)
	if 0 != len(fw.ws) {
		t.Fatalf("Adding nil writer updated writer map")
	}
	if _, err := fw.Write([]byte("one")); nil != err {
		t.Fatalf("Write error: %s", err)
	}
	if fw.Remove(nil) {
		t.Fatalf("Removing nil writer returned true")
	}
}

func TestRemove(t *testing.T) {
	_, pw1 := io.Pipe()
	_, pw2 := io.Pipe()
	_, pw3 := io.Pipe()
	fw := New(pw3)
	fw.Add(pw1, nil)
	if fw.Remove(pw2) {
		t.Errorf("Removing non-Added writer returned true")
	}
	if !fw.Remove(pw1) {
		t.Errorf("Removing Added writer returned false")
	}
	if !fw.Remove(pw3) {
		t.Errorf("Removing writer added by New returned false")
	}
}

func TestWrite_RemoveMany(t *testing.T) {
	var (
		nParallel = 1000
		fw        = New()
	)
	for i := 0; i < nParallel; i++ {
		i := i /* :( */
		pr, pw := io.Pipe()
		pr.Close()
		fw.Add(pw, func(err error) {
			if errors.Is(err, io.ErrClosedPipe) {
				return
			}
			t.Errorf("[%d] Write error: %s", i, err)
		})
	}
	fmt.Fprintf(fw, "kittens")
}
