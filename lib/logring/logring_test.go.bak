package logring

/*
 * logring_test.go
 * Tests for logring.go
 * By J. Stuart McMurray
 * Created 20231219
 * Last Modified 20231219
 */

import (
	"fmt"
	"strings"
	"testing"
)

func trimTimestamp(t *testing.T, s string) string {
	parts := strings.SplitN(s, " ", 3)
	if 3 != len(parts) {
		t.Fatalf("Invalid log line: %q", s)
	}
	return parts[2]

}

func TestRingMessages(t *testing.T) {
	for _, c := range []struct {
		size  int
		prep  func(r *Ring)
		wants []string
	}{{
		size: 5,
		prep: func(r *Ring) {
			for i := 0; i < 5; i++ {
				r.Printf("%d", i)
			}
		},
		wants: []string{"0", "1", "2", "3", "4"},
	}, {
		size: 3,
		prep: func(r *Ring) {
			for i := 0; i < 2; i++ {
				r.Printf("%d", i)
			}
		},
		wants: []string{"0", "1"},
	}, {
		size: 4,
		prep: func(r *Ring) {
			for i := 0; i < 6; i++ {
				r.Printf("%d", i)
			}
		},
		wants: []string{"2", "3", "4", "5"},
	}} {
		c := c /* :( */
		t.Run("", func(t *testing.T) {
			r := New(c.size)
			c.prep(r)

			gots := r.Messages()
			wantl := len(c.wants)
			if l := len(gots); l != wantl {
				t.Fatalf(
					"Incorrect number of messages: "+
						"got:%d want:%d",
					l,
					wantl,
				)
			}

			for i, got := range gots {
				got = trimTimestamp(t, got)

				want := c.wants[i]
				if got != want {
					t.Errorf(
						"Message %d incorrect:\n"+
							" got: %s\n"+
							"want: %s",
						i,
						got,
						want,
					)
				}
			}

			if got := r.Cap(); got != c.size {
				t.Errorf(
					"Buffer capacity incorrect:\n"+
						" got: %d\n"+
						"want: %d",
					got,
					c.size,
				)
			}

			if got := r.Len(); len(c.wants) != got {
				t.Errorf(
					"Buffer used length incorrect:\n"+
						" got: %d\n"+
						"want: %d",
					got,
					len(c.wants),
				)
			}
		})
	}
}

func TestRingMessagesAndClear(t *testing.T) {
	n := 5
	r := New(n)
	for i := 0; i < n; i++ {
		r.Printf("%d", i)
	}
	gots := r.MessagesAndClear()
	if l := len(gots); l != n {
		t.Fatalf("Incorrect number of messages: got:%d want:%d", l, n)
	}
	for i, got := range gots {
		got = trimTimestamp(t, got)
		want := fmt.Sprintf("%d", i)
		if got != want {
			t.Errorf(
				"Message %d incorrect:\n got: %s\nwant: %s",
				i,
				got,
				want,
			)
		}
	}

	if got := r.Cap(); got != n {
		t.Errorf(
			"Buffer capacity incorrect:\n got: %d\nwant: %d",
			got,
			n,
		)
	}

	if got := r.Len(); 0 != got {
		t.Errorf("Buffer reported non-zero used length %d", got)
	}

	gots = r.Messages()
	if l := len(gots); 0 != l {
		t.Errorf("Buffer not clear; got %d messages", len(gots))
	}
}
