package gates

import (
	"bytes"
	"io/fs"
	"path"

	"github.com/LeoManrique/semi-open-source/cli/internal/snapshot"
)

// lfsPointerPrefix begins every git-LFS pointer file.
var lfsPointerPrefix = []byte("version https://git-lfs.github.com/spec/v1")

// checkEntry refuses one file of the public commit that a filtered push
// cannot carry correctly: a submodule pointer, a git-LFS pointer, or a
// symlink that leads out of the published tree. content is nil for a
// submodule pointer, which has none.
func checkEntry(p string, e snapshot.Entry, content []byte) []Finding {
	// 160000 is the gitlink mode, a submodule pointer.
	if e.Mode == "160000" {
		return []Finding{
			{Where: p, Detail: "submodule pointer: it names a commit in another repository, which the mirror does not have; exclude this path"},
		}
	}

	// Every LFS pointer starts with the same first line.
	if bytes.HasPrefix(content, lfsPointerPrefix) {
		return []Finding{
			{Where: p, Detail: "git-LFS pointer: the file it stands for would not be published; exclude this path"},
		}
	}

	// 120000 is the symlink mode, a blob holding the link's target.
	if e.Mode == "120000" {
		target := string(content)

		// Resolve the target from the link's own directory, cleaning away each "..".
		resolved := path.Join(path.Dir(p), target)

		// Join hides a leading /, so check target for it, and resolved for a "..".
		if path.IsAbs(target) || !fs.ValidPath(resolved) {
			return []Finding{
				{Where: p, Detail: "symlink points outside the published tree: " + target + "; exclude this path"},
			}
		}
	}
	return nil
}
