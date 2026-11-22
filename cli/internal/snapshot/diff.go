package snapshot

import "github.com/LeoManrique/semi-open-source/cli/internal/gitx"

// TreeAt reads every file of commit, a hash or a ref, unfiltered. It is how
// sos reads the mirror's commit, which it filtered before pushing.
func TreeAt(repo *gitx.Repo, commit string) (Tree, error) {
	entries, err := repo.ListTree(commit)

	if err != nil {
		return nil, err
	}

	tree := Tree{}
	// Keep every entry as it is: sos filtered this commit before pushing it.
	for _, e := range entries {
		tree[e.Path] = Entry{Mode: e.Mode, Hash: e.Hash}
	}
	return tree, nil
}

// Diff compares two trees: the paths only after holds, the paths both hold
// with a different entry, and the paths only before holds. Each list is
// sorted.
func Diff(before, after Tree) (added, modified, removed []string) {
	// Walk after for what is new or changed. Paths keeps each list sorted.
	for _, p := range after.Paths() {
		old, existed := before[p]

		if !existed {
			added = append(added, p)
			continue
		}

		// != compares mode and hash, so a chmod +x counts as a change.
		if old != after[p] {
			modified = append(modified, p)
		}
	}
	// Walk before for what is gone, deleted or no longer included.
	for _, p := range before.Paths() {
		if _, ok := after[p]; !ok {
			removed = append(removed, p)
		}
	}
	return added, modified, removed
}
