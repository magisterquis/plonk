package client

/*
 * events_test.go
 * Tests for events.go
 * By J. Stuart McMurray
 * Created 20231214
 * Last Modified 20231222
 */

import (
	"strings"
	"testing"
	"time"

	"github.com/magisterquis/plonk/internal/def"
)

func mustParseTime3339(t *testing.T, when string) time.Time {
	now, err := time.Parse(time.RFC3339, when)
	if nil != err {
		t.Fatalf("Failed to parse time %s: %s", when, err)
	}
	return now
}

func TestClientHandleListSeenEvent(t *testing.T) {
	_, out, _, _, c, _ := newTestClient(t)

	now := mustParseTime3339(t, "0001-01-02T00:00:00Z")
	have := def.EDSeen{{
		ID:   "i1",
		From: "f1",
		When: mustParseTime3339(t, "0001-01-01T23:59:58.5Z"),
	}, {
		ID:   "i2",
		From: "f2",
		When: mustParseTime3339(t, "0001-01-01T23:00:00Z"),
	}, {
		ID:   "i3",
		From: "f3",
		When: mustParseTime3339(t, "0001-01-01T22:30:30Z"),
	}}

	want := `
ID  From  Last Seen
--  ----  ---------
i1  f1    0001-01-01T23:59:58Z (1.5s)
i2  f2    0001-01-01T23:00:00Z (1h0m0s)
i3  f3    0001-01-01T22:30:30Z (1h29m30s)
`
	want = strings.TrimPrefix(want, "\n")

	c.handleListSeenEventAt(def.ENListSeen, have, now)
	if got := out.String(); got != want {
		t.Errorf(
			"Incorrect seen implants table:\n"+
				" got:\n%s\n"+
				"want:\n%s",
			got,
			want,
		)
	}
}
