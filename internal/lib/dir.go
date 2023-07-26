package lib

/*
 * dir.go
 * Manage our working directory
 * By J. Stuart McMurray
 * Created 20230726
 * Last Modified 20230726
 */

import "path/filepath"

// WorkingDir is our working directory.  It should not be used until set in
// main.
var WorkingDir string = "."

// AbsPath is like filepath.Abs, but uses workingDir as the working directory.
func AbsPath(path string) string {
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	return filepath.Join(WorkingDir, path)
}
