package main

/*
 * interact_test.go
 * Tests for interact.go
 * By J. Stuart McMurray
 * Created 20230224
 * Last Modified 20230423
 */

import (
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

func TestUnmarshalRLog(t *testing.T) {
	/* Placeholder HTTP request. */
	req, err := http.NewRequest(
		http.MethodGet,
		"http://example.com",
		nil,
	)
	if nil != err {
		t.Errorf("UnmarshalRLog: http.NewRequest err:%s", err)
		return
	}

	for _, c := range []struct {
		have   any
		id     string
		sameid bool
	}{{
		have: OutputLog{
			ID:     "foo",
			Output: "it worked",
			Err:    "it worked a little",
		},
		id:     "foo",
		sameid: true,
	}, {
		have: TaskLog{
			ID:   "bar",
			Task: "Do it",
			Err:  "Do it less",
		},
		id:     "bar",
		sameid: true,
	}, {
		have: TaskLog{
			ID: "foo",
		},
		id:     "bar",
		sameid: false,
	}, {
		have: TaskLog{
			ID: "",
		},
		id:     "foo",
		sameid: false,
	}, {
		have: TaskLog{
			ID: "foo",
		},
		id:     "",
		sameid: false,
	}, {
		have: TaskLog{
			ID: "",
		},
		id:     "",
		sameid: true,
	}} {
		b, err := json.Marshal(c.have)
		if nil != err {
			t.Errorf(
				"UnmarshalRLog: json.Marshal(%#v): err:%s",
				c.have,
				err,
			)
			continue
		}
		logMsg := strings.TrimPrefix(
			rLogMarshal(req, MessageTypeUnknown, string(b)),
			"["+MessageTypeUnknown+"] ",
		)

		gotp := reflect.New(reflect.TypeOf(c.have)).Interface()
		err = UnmarshalRLog(c.id, gotp, []byte(logMsg))
		got := reflect.ValueOf(gotp).Elem().Interface()

		if !c.sameid && !errors.Is(err, errWrongID) {
			t.Errorf(
				"unmarshalRLog: did not detect wrong ID\n"+
					"sameid: %t\n"+
					"    id: %q\n"+
					"  have: %#v\n"+
					"   got: %#v\n"+
					"   err: %s",
				c.sameid,
				c.id,
				c.have,
				got,
				err,
			)
			continue
		}

		if nil != err && !(errors.Is(err, errWrongID) && !c.sameid) {
			t.Errorf(
				"unmarshalRLog: error\n"+
					"sameid: %t\n"+
					"    id: %q\n"+
					"  have: %#v\n"+
					"   got: %#v\n"+
					"   err: %s",
				c.sameid,
				c.id,
				c.have,
				got,
				err,
			)
			continue
		}

		if got != c.have {
			t.Errorf(
				"unmarshalRLog: discrepancy\n"+
					"have: %#v\n"+
					" got: %#v",
				c.have,
				got,
			)
			continue
		}
	}
}
