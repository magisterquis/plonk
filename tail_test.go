package main

/*
 * tail_test.go
 * Tests for tail.go
 * By J. Stuart McMurray
 * Created 20230423
 * Last Modified 20230523
 */

import (
	"os/exec"
	"testing"
)

func TestGetIDFromSeenLine(t *testing.T) {
	for _, c := range []struct {
		Name string
		Have string
		ID   string
		Ok   bool
	}{{
		Name: "ok/with_task",
		Have: `2023/02/28 23:12:45 [CALLBACK] 127.0.0.1:49784 NoSNI ` +
			`GET /t/kittens ` +
			`{"ID":"kittens","Task":"ps auxwww"}`,
		ID: "kittens",
		Ok: true,
	}, {
		Name: "ok/with_output",
		Have: `2023/05/23 22:42:03 [OUTPUT] 127.0.0.1:9721 NoSNI ` +
			`GET /o/kittens {"ID":"kittens"}`,
		ID: "kittens",
		Ok: true,
	}} {
		c := c /* :C */
		t.Run(c.ID, func(t *testing.T) {
			t.Parallel()
			id, ok := getIDFromSeenLine([]byte(c.Have))
			if id == c.ID && ok == c.Ok {
				return
			}
			t.Errorf("got id:%q ok:%t", id, ok)
		})
	}
}

func TestKilled(t *testing.T) {
	t.Run("withkill", func(t *testing.T) {
		t.Parallel()
		cat := exec.Command("cat")
		pw, err := cat.StdinPipe()
		if nil != err {
			t.Errorf("getting stdin pipe: %s", err)
			return
		}
		if err := cat.Start(); nil != err {
			t.Errorf("staring child: %s", err)
			return
		}
		if err := cat.Process.Kill(); nil != err {
			if err := cat.Process.Release(); nil != err {
				t.Errorf("releasing: %s", err)
			}
			t.Errorf("killing child: %s", err)
			return
		}
		if err := pw.Close(); nil != err {
			defer cat.Wait()
			t.Errorf("closing stdin pipe: %s", err)
			return
		}
		err = cat.Wait()
		if nil == err {
			t.Errorf("no error waiting")
			return
		}
		if !killed(err) {
			t.Errorf("unexpectedly not killed")
		}
	})
	t.Run("nokill", func(t *testing.T) {
		t.Parallel()
		echo := exec.Command("echo", "-n")
		if err := echo.Start(); nil != err {
			t.Errorf("starting child")
		}
		err := echo.Wait()
		if killed(err) {
			t.Errorf("unexpectedly killed")
		}
	})
}
