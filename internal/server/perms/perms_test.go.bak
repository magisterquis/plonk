package perms

/*
 * perms.go
 * Process wide process permissions and such
 * By J. Stuart McMurray
 * Created 20231214
 * Last Modified 20231214
 */

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/magisterquis/plonk/internal/def"
)

func TestMustSetProcessPerms(t *testing.T) {
	MustSetProcessPerms()
	t.Run("perms/directory", func(t *testing.T) {
		t.Parallel()
		d := t.TempDir()
		name := filepath.Join(d, "kittens")
		if err := os.MkdirAll(name, def.DirPerms); nil != err {
			t.Fatalf("Error making %s: %s", name, err)
		}
		fi, err := os.Stat(name)
		if nil != err {
			t.Fatalf("Error stat()ing %s: %s", name, err)
		}
		want := fs.FileMode(def.DirPerms)
		if got := fi.Mode().Perm(); def.DirPerms != got {
			t.Fatalf("Incorrect perms: got:%o want:%o", got, want)
		}
	})
	t.Run("perms/file", func(t *testing.T) {
		t.Parallel()
		d := t.TempDir()
		name := filepath.Join(d, "kittens")
		f, err := os.OpenFile(
			name,
			os.O_CREATE|os.O_RDONLY|os.O_EXCL,
			def.FilePerms,
		)
		if nil != err {
			t.Fatalf("Error making %s: %s", name, err)
		}
		defer f.Close()
		fi, err := f.Stat()
		if nil != err {
			t.Fatalf("Error stat()ing %s: %s", name, err)
		}
		want := fs.FileMode(def.FilePerms)
		if got := fi.Mode().Perm(); want != got {
			t.Fatalf("Incorrect perms: got:%o want:%o", got, want)
		}
	})
}
