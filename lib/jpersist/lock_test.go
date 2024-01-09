package jpersist

/*
 * lock_test.go
 * Tests for lock.go
 * By J. Stuart McMurray
 * Created 20231007
 * Last Modified 20231208
 */

import (
	"os"
	"testing"
	"time"
)

func ensureEnoughTime(t *testing.T, reserve time.Duration) {
	dl, ok := t.Deadline()
	if ok && time.Until(dl) < reserve {
		t.SkipNow()
	}
}

func TestManagerUnlockAndWrite(t *testing.T) {
	f := tempFile(t)
	mgr, err := NewManager[testStruct](&Config{File: f.Name()})
	if nil != err {
		t.Fatalf("Error creating Manager: %s", err)
	}

	mgr.Lock()
	mgr.C.S = "kittens"
	mgr.C.N = 3
	if err := mgr.UnlockAndWrite(); nil != err {
		t.Fatalf("UnlockAndWrite error: %s", err)
	}

	want := `{
	"S": "kittens",
	"N": 3
}` /* A bit fragile. */
	got, err := os.ReadFile(f.Name())
	if nil != err {
		t.Fatalf("Error reading file after UnlockAndWrite: %s", err)
	}
	if string(got) != want {
		t.Fatalf(
			"UnlockAndWrite wrote unexpected data:\n"+
				"have: %#v\n"+
				"  got: %s\n"+
				" want: %s",
			*mgr.C,
			got,
			want,
		)
	}
}

func TestManagerUnlock_WithDelay(t *testing.T) {
	ech := make(chan error, 1)

	delay := time.Second / 2
	maxTime := delay / 2

	/* Don't run this if we haven't enough time. */
	ensureEnoughTime(t, delay*2)

	f := tempFile(t)
	mgr, err := NewManager[testStruct](&Config{
		File:       f.Name(),
		WriteDelay: delay,
		OnError: func(err error) {
			ech <- err
		},
	})
	if nil != err {
		t.Fatalf("Error creating Manager: %s", err)
	}

	start := time.Now()

	mgr.Lock()
	mgr.C.S = "kittens"
	mgr.C.N = 3
	if err := mgr.UnlockAndWrite(); nil != err {
		t.Fatalf("UnlockAndWrite error: %s", err)
	}

	want := `{
	"S": "kittens",
	"N": 3
}` /* A bit fragile. */
	got, err := os.ReadFile(f.Name())
	if nil != err {
		t.Fatalf("Error reading file after UnlockAndWrite: %s", err)
	}
	d := time.Since(start)
	if maxTime <= d {
		t.Fatalf(
			"Read after UnlockAndWrite too slow.  Took %s, max %s",
			d,
			maxTime,
		)
	}
	if string(got) != want {
		t.Fatalf(
			"UnlockAndWrite wrote unexpected data:\n"+
				"have: %#v\n"+
				"  got: %s\n"+
				" want: %s",
			*mgr.C,
			got,
			want,
		)
	}

	start = time.Now()
	mgr.Lock()
	mgr.C.N = 4
	if err := mgr.Unlock(); nil != err {
		t.Fatalf("First Unlock error: %s", err)
	}

	got, err = os.ReadFile(f.Name())
	if nil != err {
		t.Fatalf("Error reading file after first Unlock: %s", err)
	}
	d = time.Since(start)
	if maxTime <= d {
		t.Fatalf(
			"Read after first Unlock too slow.  Took %s, max %s",
			d,
			maxTime,
		)
	}
	if string(got) != want {
		t.Fatalf(
			"First Unlock wrote unexpected data:\n"+
				"have: %#v\n"+
				"  got: %s\n"+
				" want: %s",
			*mgr.C,
			got,
			want,
		)
	}

	mgr.Lock()
	mgr.C.N = 5
	if err := mgr.Unlock(); nil != err {
		t.Fatalf("Second Unlock error: %s", err)
	}
	got, err = os.ReadFile(f.Name())
	if nil != err {
		t.Fatalf("Error reading file after second Unlock: %s", err)
	}
	d = time.Since(start)
	if maxTime <= d {
		t.Fatalf(
			"Read after second Unlock too slow.  Took %s, max %s",
			d,
			maxTime,
		)
	}
	if string(got) != want {
		t.Fatalf(
			"Second Unlock wrote unexpected data:\n"+
				"have: %#v\n"+
				"  got: %s\n"+
				" want: %s",
			*mgr.C,
			got,
			want,
		)
	}

	/* Wait for timer to expire. */
	select {
	case err := <-ech:
		t.Fatalf("Error: %s", err)
	case <-mgr.writeWaiter.WaitChan():
		/* Good. */
	case <-time.After(2 * delay):
		t.Fatalf("Unlock didn't happen within %s", 2*delay)
	}

	want = `{
	"S": "kittens",
	"N": 5
}` /* A bit fragile. */
	got, err = os.ReadFile(f.Name())
	if nil != err {
		t.Fatalf("Error reading file after delayed write: %s", err)
	}
	if string(got) != want {
		t.Fatalf(
			"Delayed write wrote unexpected data:\n"+
				"have: %#v\n"+
				"  got: %s\n"+
				" want: %s",
			*mgr.C,
			got,
			want,
		)
	}
}
