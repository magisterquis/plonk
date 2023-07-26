package lib

/*
 * env.go
 * Config from environment variables
 * By J. Stuart McMurray
 * Created 20230225
 * Last Modified 20230726
 */

import (
	"fmt"
	"log"
	"os"
	"reflect"
	"strconv"
	"strings"
	"text/tabwriter"

	"golang.org/x/exp/slices"
)

const (
	// envPrefix is prepended to environment variable names before querying
	// the environment.
	envPrefix = "PLONK_"
	// defaultTag is the struct tag used used to specify a default in Env.
	defaultTag = "default"
)

// Env holds strings from the environment.
var Env struct {
	LogFile        string `default:"log"`
	StaticFilesDir string `default:"files"`
	LocalCertDir   string `default:"certs"` /* Non-LE certs. */
	LECertDir      string `default:"lecerts"`
	TaskFile       string `default:"tasking.json"`
	DefaultFile    string `default:"index.html"`
	HTTPTimeout    string `default:"1m"`
	FilesPrefix    string `default:"f"`
	TaskPrefix     string `default:"t"`
	OutputPrefix   string `default:"o"`
	OutputMax      string `default:"10485760"` /* 10MB */
	ExfilPrefix    string `default:"p"`
	ExfilMax       string `default:"104857600"` /* 100MB */
	ExfilDir       string `default:"exfil"`
}

// EnvVarName gets the environment variable name for the field in Env at f.  If
// f isn't in Env, EnvVarName returns a string indicating there's a bug.
func EnvVarName(f *string) string {
	v := reflect.ValueOf(&Env).Elem()
	t := v.Type()
	p := reflect.ValueOf(f) /* To compare to fields in v. */
	/* Look through the fields in Env to see if we've a match. */
	for i := 0; i < v.NumField(); i++ {
		if v.Field(i).Addr().Equal(p) {
			return envNameify(t.Field(i).Name)
		}
	}
	/* No match.  Got a bad pointer. */
	return fmt.Sprintf("BUG: no field at %p in Env (%p)", f, &Env)
}

// PrintEnv prints the environment variables and their current values to stdout.
func PrintEnv() {
	var vs [][2]string

	/* Gather the variables. */
	v := reflect.ValueOf(Env)
	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		vs = append(vs, [2]string{
			envNameify(t.Field(i).Name),
			v.Field(i).String(),
		})
	}
	slices.SortFunc(vs, func(a, b [2]string) bool {
		if a[0] == b[0] {
			/* Unpossible */
			return a[1] < b[1]
		}
		return a[0] < b[0]
	})

	/* Make it all nice and tabular. */
	tw := tabwriter.NewWriter(os.Stdout, 2, 8, 2, ' ', 0)
	defer tw.Flush()
	for _, v := range vs {
		fmt.Fprintf(tw, "%s\t%s\n", v[0], v[1])
	}
}

// init loads the config from the environment into Env.
func init() {
	loadEnv()
}

// loadEnv loads Env from the environment.
func loadEnv() {
	v := reflect.ValueOf(&Env).Elem()
	t := v.Type()
	/* Populate each field with an environemnt variable of a same
	name or, failing that, the default in the tag. */
	for i := 0; i < v.NumField(); i++ {
		/* If it's in the environment, life's easy. */
		if ev := os.Getenv(envNameify(t.Field(i).Name)); "" != ev {
			v.Field(i).Set(reflect.ValueOf(ev))
			continue
		}
		/* If not, use the default. */
		v.Field(i).Set(reflect.ValueOf(t.Field(i).Tag.Get(defaultTag)))
		/* If we still don't have anything, something's gone wrong. */
		if "" == v.Field(i).String() {
			panic(fmt.Sprintf(
				"environment config %q not set",
				t.Field(i).Name,
			))
		}
	}
}

// envNameify returns envPrefix prepended to a capitalized form of s.
func envNameify(s string) string { return envPrefix + strings.ToUpper(s) }

// MustParseEnvInt parses an int from an environment variable.  The value must
// be greater than 0.
func MustParseEnvInt(ev *string) int64 {
	n, err := parseEnvInt(*ev)
	if nil != err {
		log.Fatalf(
			"[%s] Parsing %s (%q): %s",
			MessageTypeError,
			EnvVarName(ev),
			*ev,
			err,
		)
	}
	return n
}

// parseEnvInt parses an int from an environment variable.  The value must be
// greater than 0.
func parseEnvInt(s string) (int64, error) {
	/* Intify. */
	n, err := strconv.ParseInt(s, 0, 64)
	if nil != err {
		return 0, err
	} else if 0 >= n {
		return 0, fmt.Errorf("must be greater than 0, not %d", n)
	}
	return n, nil
}
