package waiter

/*
 * waiter_test.go
 * Tests for waiter.go
 * By J. Stuart McMurray
 * Created 20231008
 * Last Modified 20231112
 */

import (
	"sync"
	"testing"
	"time"
)

func TestWaiter(t *testing.T) {
	e := "kittens"
	var w Waiter[string]
	var (
		r1, r2, r3 string
		ok1, ok2   bool
	)
	var ready, done sync.WaitGroup
	ready.Add(2)
	done.Add(3)
	go func() {
		r3 = w.Wait()
		done.Done()
	}()
	go func() {
		ch := w.WaitChan()
		ready.Done()
		r1, ok1 = <-ch
		done.Done()
	}()
	go func() {
		ch := w.WaitChan()
		ready.Done()
		r2, ok2 = <-ch
		done.Done()
	}()

	ch3 := w.WaitChan()

	/* This is cheating... */
	ready.Wait()
	for {
		w.l.Lock()
		n := len(w.waiters)
		w.l.Unlock()
		if 4 == n {
			break
		}
		time.Sleep(time.Millisecond)

	}
	w.Broadcast(e)
	done.Wait()

	for i, v := range []string{r1, r2, r3} {
		if e != v {
			t.Errorf("Incorrect r%d: %s", i+1, v)
		}
	}
	for i, v := range []bool{ok1, ok2} {
		if !v {
			t.Errorf("Unexpectedly closed channel %d", i+1)
		}
	}

	v, ok := <-ch3
	if e != v {
		t.Errorf("Incorrect receive from ch3: %s", v)
	}
	if !ok {
		t.Errorf("Unexpectedly closed ch3")
	}
}

func TestWaiterAlwaysBroadcast(t *testing.T) {
	want := "kittens"
	var w Waiter[string]
	w.AlwaysBroadcast(want)
	got := w.Wait()
	if want != got {
		t.Errorf("First wait incorrect got:%q want:%q", got, want)
	}
	got = w.Wait()
	if want != got {
		t.Errorf("Second wait incorrect got:%q want:%q", got, want)
	}
}
