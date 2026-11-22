package snapshot

import (
	"fmt"

	"github.com/LeoManrique/semi-open-source/cli/internal/config"
	"github.com/LeoManrique/semi-open-source/cli/internal/gitx"
	"github.com/LeoManrique/semi-open-source/cli/internal/matcher"
	"github.com/LeoManrique/semi-open-source/cli/internal/refuse"
)

// FilteredTree computes what the public sees of a commit, a resolved commit
// hash: every file in it that the allowlist committed in it includes.
func FilteredTree(repo *gitx.Repo, commit string) (Tree, error) {
	// Read the allowlist from the commit, never from the working tree.
	m, err := allowlistAt(repo, commit)

	if err != nil {
		return nil, err
	}

	entries, err := repo.ListTree(commit)

	if err != nil {
		return nil, err
	}

	tree := Tree{}
	for _, e := range entries {
		// Keep only what the allowlist includes, which never covers .sos/.
		if !m.Include(e.Path) {
			continue
		}

		tree[e.Path] = Entry{Mode: e.Mode, Hash: e.Hash}
	}
	return tree, nil
}

// allowlistAt parses the allowlist as committed in commit.
func allowlistAt(repo *gitx.Repo, commit string) (*matcher.Matcher, error) {
	content, found, err := repo.ReadFileAt(commit, config.IncludePath)

	if err != nil {
		return nil, err
	}

	// No committed allowlist is a setup mistake, so exit 2 and say what to commit.
	if !found {
		return nil, fmt.Errorf("%s is not committed at %s: commit it first", config.IncludePath, commit)
	}

	m, err := matcher.Parse(content)

	if err != nil {
		// A broken allowlist is a refusal, never a guess at what a line meant.
		return nil, refuse.New(err.Error())
	}
	return m, nil
}
