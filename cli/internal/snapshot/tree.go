// Package snapshot computes the public commit: the branch tip's tree filtered
// through the allowlist committed at the tip, written as one root commit.
package snapshot

import (
	"maps"
	"slices"
)

// Entry is one file in a filtered tree: everything needed to place it in a
// git tree object, minus the name.
type Entry struct {
	Mode string // 100644, 100755, 120000, 160000
	Hash string
}

// Type is the git object type implied by the mode.
func (e Entry) Type() string {
	// 160000 is a submodule, which points at a commit in another repository.
	if e.Mode == "160000" {
		return "commit"
	}

	// Every other mode is a blob: a file, an executable or a symlink.
	return "blob"
}

// Tree is a commit's publishable content: full path -> entry. It is a
// snapshot, not a diff.
type Tree map[string]Entry

// Paths returns every path, sorted. Always use this rather than ranging over
// the map: map order is randomized, and output built in that order would
// differ from run to run.
func (t Tree) Paths() []string {
	// Sort, since map order changes from run to run.
	return slices.Sorted(maps.Keys(t))
}
