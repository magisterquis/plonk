package client

/*
 * help_test.go
 * Tests for help.go
 * By J. Stuart McMurray
 * Created 20231228
 * Last Modified 20231228
 */

import (
	"maps"
	"regexp"
	"strings"
	"testing"
)

func TestHelpTopicsCount(t *testing.T) {
	/* Make sure we have the right number of topics. */
	got := maps.Clone(helpTopics)
	seen := make(map[string]struct{})
	nre := regexp.MustCompile(`^-- (.*) --$`)

	for _, v := range strings.Split(string(topicTxtar), "\n") {
		ms := nre.FindStringSubmatch(strings.TrimSuffix(v, "\n"))
		if 2 != len(ms) {
			continue
		}
		n := ms[1]
		if _, ok := seen[n]; ok {
			t.Errorf("Duplicate topic: %s", n)
			continue
		}
		seen[n] = struct{}{}

		if _, ok := got[n]; !ok {
			t.Errorf("Missing topic: %s", n)
			continue
		}
		delete(got, n)
	}

	for n := range got {
		t.Errorf("Unexpected topic: %s", n)
	}
}

func TestUnTxtarHelp(t *testing.T) {
	have := `
Instructions.  They're nice.
-- t1 --
d1

b1l1
b2l2
b3l3
-- t2 --
d2

b2l1
	`
	got := make(map[string]helpTopic)
	want := map[string]helpTopic{
		"t1": helpTopic{
			Description: "d1",
			Text:        "b1l1\nb2l2\nb3l3",
		},
		"t2": helpTopic{
			Description: "d2",
			Text:        "b2l1",
		},
	}
	if err := unTxtarHelp(got, []byte(have)); nil != err {
		t.Errorf("Error: %s", err)
	}
	for wn, wv := range want {
		gv, ok := got[wn]
		if !ok {
			t.Errorf("Missing topic: %s", wn)
			continue
		}
		if gv.Description != wv.Description {
			t.Errorf(
				"Incorrect description for %s:\n"+
					" got: %s\n"+
					"want: %s",
				wn,
				gv.Description,
				wv.Description,
			)
		}
		if gv.Text != wv.Text {
			t.Errorf(
				"Incorrect text for %s:\n"+
					" got: %s\n"+
					"want: %s",
				wn,
				gv.Text,
				wv.Text,
			)
		}
		delete(got, wn)
	}

	for gn := range got {
		t.Errorf("Unexpected topic: %s", gn)
	}
}
