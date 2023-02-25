package main

/*
 * env_test.go
 * Tests for env.go
 * By J. Stuart McMurray
 * Created 20230225
 * Last Modified 20230225
 */

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

func TestLoadEnv(t *testing.T) {
	/* If env.go's init()'s going to panic, it would have by now. */
	cs := []struct {
		name string
		envp *string
		val  string
	}{{
		envp: &Env.LogFile,
		val:  time.Now().Format(time.RFC3339Nano),
	}, {
		envp: &Env.StaticFilesDir,
		val:  time.Now().Format(time.RFC3339Nano),
	}}

	/* Set the environment and reload Env. */
	for _, c := range cs {
		os.Setenv(EnvVarName(c.envp), c.val)
	}
	loadEnv() /* It'll panic on error. */

	/* Make sure it worked. */
	for _, c := range cs {
		if *c.envp != c.val {
			t.Errorf(
				"LoadEnv env:%p name:%s envp:%p val:%q got:%q",
				&Env,
				EnvVarName(c.envp),
				c.envp,
				c.val,
				*c.envp,
			)
		}
	}
}

func TestEnvVarName(t *testing.T) {
	/* If env.go's init()'s going to panic, it would have by now. */

	var x string
	for _, c := range []struct {
		have *string
		want string
	}{{
		have: &Env.LogFile,
		want: envPrefix + "LOGFILE",
	}, {
		have: &x,
		want: fmt.Sprintf("BUG: no field at %p in Env (%p)", &x, &Env),
	}} {
		got := EnvVarName(c.have)
		if got != c.want {
			t.Errorf(
				"EnvVarName: have:%p got:%s want:%s",
				c.have,
				got,
				c.want,
			)
			continue
		}
	}
}

func TestEnvDefaults(t *testing.T) {
	/* Make sure we don't have anything in the environment to mess up
	defaults. */
	envs := os.Environ()
	for _, env := range envs {
		v, _, found := strings.Cut(env, "=")
		if !found {
			t.Errorf(
				"EnvDefaults: environment variable "+
					"without =: %q",
				env,
			)
			continue
		}
		if !strings.HasPrefix(v, envPrefix) {
			continue
		}
		if err := os.Unsetenv(v); nil != err {
			t.Errorf(
				"EnvDefaults: error removing %q from "+
					"environment: %s",
				v,
				err,
			)
		}
	}
	loadEnv()
	for _, c := range []struct {
		ptr  *string
		want string
	}{
		{ptr: &Env.LogFile, want: "log"},
		{ptr: &Env.StaticFilesDir, want: "files"},
	} {
		if *c.ptr != c.want {
			t.Errorf(
				"EnvDefaults: var:%s ptr:%p got:%s want:%s",
				EnvVarName(c.ptr),
				c.ptr,
				*c.ptr,
				c.want,
			)
		}
	}
}
