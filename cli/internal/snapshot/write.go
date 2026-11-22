package snapshot

import (
	"strings"

	"github.com/LeoManrique/semi-open-source/cli/internal/gitx"
)

// node is a directory under construction while a flat Tree is turned into
// nested git tree objects.
type node struct {
	files map[string]Entry
	dirs  map[string]*node
}

func newNode() *node {
	// Make both maps, since storing into a nil map panics.
	return &node{files: map[string]Entry{}, dirs: map[string]*node{}}
}

// insert places e at the path made of segments, creating the directories on
// the way down.
func (n *node) insert(segments []string, e Entry) {
	// One segment left means it is the file's own name.
	if len(segments) == 1 {
		n.files[segments[0]] = e
		return
	}

	child, ok := n.dirs[segments[0]]

	if !ok {
		// Assign with =, so the child declared above is the one that recurses.
		child = newNode()
		n.dirs[segments[0]] = child
	}

	// Hand the rest of the path down one level.
	child.insert(segments[1:], e)
}

// WriteTree writes a filtered tree as git tree objects, one per directory,
// and returns the root tree's hash. It never writes a blob.
func WriteTree(repo *gitx.Repo, t Tree) (string, error) {
	root := newNode()
	for _, p := range t.Paths() {
		// Split each path into its directories and the file name.
		root.insert(strings.Split(p, "/"), t[p])
	}

	return writeNode(repo, root)
}

// writeNode writes one directory's tree object, its subdirectories first,
// and returns its hash.
func writeNode(repo *gitx.Repo, n *node) (string, error) {
	var entries []gitx.TreeEntry
	// Map order is fine here: git sorts a tree's entries itself.
	for name, e := range n.files {
		entries = append(entries, gitx.TreeEntry{Mode: e.Mode, Hash: e.Hash, Type: e.Type(), Path: name})
	}
	for name, child := range n.dirs {
		// Write the subdirectory first, since its hash goes into this tree.
		hash, err := writeNode(repo, child)

		if err != nil {
			return "", err
		}

		// 040000 is the mode git gives a directory.
		entries = append(entries, gitx.TreeEntry{Mode: "040000", Type: "tree", Hash: hash, Path: name})
	}

	return repo.MakeTree(entries)
}
