package subcom

/*
 * subcom_test.go
 * Tests for subcom.go
 * By J. Stuart McMurray
 * Created 20231020
 * Last Modified 20231216
 */

import (
	"errors"
	"slices"
	"testing"
)

type retCtx struct {
	gps []string
	gas []string
}

func TestAdd(t *testing.T) {
	hps := []string{"p1", "p2"}
	wps := []string{"p1", "p2", "h"}
	has := []string{"h", "a1", "a2"}
	was := []string{"a1", "a2"}
	cdr := New[*retCtx](nil)
	cdr.Add("h", "a", "d", func(ctx *retCtx, parents, args []string) error {
		ctx.gps = slices.Clone(parents)
		ctx.gas = slices.Clone(args)
		return nil
	})
	var ctx retCtx
	if err := cdr.Call(&ctx, hps, has); nil != err {
		t.Fatalf("Error: %s", err)
	}
	if !slices.Equal(ctx.gps, wps) {
		t.Errorf(
			"Incorrect parents\nhave: %s\n got: %s\nwant: %s",
			hps,
			ctx.gps,
			wps,
		)
	}
	if !slices.Equal(ctx.gas, was) {
		t.Errorf(
			"Incorrect parents\nhave: %s\n got: %s\nwant: %s",
			has,
			ctx.gas,
			was,
		)
	}
}

func TestNewWithSpecs(t *testing.T) {
	h1 := func(ctx *int, name, args []string) error { *ctx = 1; return nil }
	h2 := func(ctx *int, name, args []string) error { *ctx = 2; return nil }
	re := errors.New("success")

	cdr := New([]Spec[*int]{{
		Name:        "c1",
		ArgHelp:     "a1",
		Description: "d1",
		Handler:     h1,
	}, {
		Name:        "c2",
		ArgHelp:     "a2",
		Description: "d2",
		Handler:     h2,
	}, {
		Name:        "ce",
		ArgHelp:     "ae",
		Description: "de",
		Handler:     func(_ *int, _, _ []string) error { return re },
	}})

	var ctx int

	if err := cdr.Call(&ctx, nil, []string{"c1"}); nil != err {
		t.Errorf("Error calling c1: %s", err)
	} else if ctx != 1 {
		t.Errorf("Wrong number: got:%d want:%d", ctx, 1)
	}

	if err := cdr.Call(&ctx, nil, []string{"c2"}); nil != err {
		t.Errorf("Error calling c2: %s", err)
	} else if ctx != 2 {
		t.Errorf("Wrong number: got:%d want:%d", ctx, 2)
	}

	if err := cdr.Call(&ctx, nil, []string{"ce"}); !errors.Is(err, re) {
		t.Errorf("Incorrect ce error: got:%s want:%s", re, err)
	}
}