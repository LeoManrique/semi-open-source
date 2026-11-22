package snapshot

import (
	"fmt"

	"github.com/LeoManrique/semi-open-source/cli/internal/config"
	"github.com/LeoManrique/semi-open-source/cli/internal/gitx"
)

// Public is the commit the mirror receives, with what went into it.
type Public struct {
	Tip     string // the private commit it was computed from
	Commit  string // the public commit's hash
	Files   Tree   // the filtered tree it holds
	Message string // its message: the project name
}

// Build computes the public commit for branch: the tip's filtered tree in one
// root commit, with the tip's author as author and committer and the project
// name committed at the tip as its message. It writes trees and the commit
// into the private repository's object database, which is harmless: nothing
// is public until it is pushed.
func Build(repo *gitx.Repo, branch string) (*Public, error) {
	// Use the full ref name so git never confuses a branch with a tag.
	tip, found, err := repo.ResolveCommit("refs/heads/" + branch)

	if err != nil {
		return nil, err
	}

	if !found {
		return nil, fmt.Errorf("branch %s does not exist", branch)
	}

	// Read the name from the commit, so every clone writes the same message.
	committed, err := config.LoadAt(repo, tip)

	if err != nil {
		return nil, err
	}

	files, err := FilteredTree(repo, tip)

	if err != nil {
		return nil, err
	}

	// Write only trees, since every file's blob is already in the repository.
	tree, err := WriteTree(repo, files)

	if err != nil {
		return nil, err
	}

	commit, err := repo.ReadCommit(tip)

	if err != nil {
		return nil, err
	}

	// Use the tip's identity, so every clone gets the same hash.
	public, err := repo.MakeCommit(tree, committed.Name+"\n", commit.Author)

	if err != nil {
		return nil, err
	}
	return &Public{Tip: tip, Commit: public, Files: files, Message: committed.Name}, nil
}
