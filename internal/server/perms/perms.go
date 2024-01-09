// Package perms - Process wide process permissions and such
package perms

/*
 * perms.go
 * Process wide process permissions and such
 * By J. Stuart McMurray
 * Created 20231214
 * Last Modified 20231214
 */

import (
	"sync"

	"github.com/magisterquis/plonk/internal/def"
	"golang.org/x/sys/unix"
)

// setPermsOnce ensures that permissions things are only set once and lets
// SetPerms be called multiple times.
var setPermsOnce = sync.OnceValue(setPerms)

// MustSetProcessPerms sets up process-wide permissions.  It may be called more
// than once, // but calls after the first have no effect.  MustSetPerms panics
// on error.
func MustSetProcessPerms() {
	if err := setPermsOnce(); nil != err {
		panic("setting process-wide security settings: " + err.Error())
	}
}

// setPerms sets the umask to allow only ug=rw.
func setPerms() error {
	/* Set the umask to allow ug=rw. */
	unix.Umask(0777 & ^(def.FilePerms | def.DirPerms))

	return nil
}
