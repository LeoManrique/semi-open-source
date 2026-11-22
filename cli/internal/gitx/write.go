package gitx

import (
	"bytes"
	"errors"
	"fmt"
	"path"
	"strings"
)

// MakeTree writes a tree object from one directory's entries and returns its
// hash. Only the final element of each entry's Path is used; git canonicalizes
// the ordering, so entries may be passed in any order.
func (r *Repo) MakeTree(entries []TreeEntry) (string, error) {
	// No entries means the empty tree, which every repository already has.
	if len(entries) == 0 {
		return EmptyTree, nil
	}

	var in bytes.Buffer
	for _, e := range entries {
		// One mktree -z record per entry. mktree wants names, not full paths.
		fmt.Fprintf(&in, "%s %s %s\t%s\x00", e.Mode, e.Type, e.Hash, path.Base(e.Path))
	}

	out, err := r.exec(in.Bytes(), nil, "mktree", "-z")

	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(out)), nil
}

// MakeCommit writes a root commit (no parents) and returns its hash. The
// message is stored verbatim, and author is also the committer, date
// included. Identities and dates are pinned through the environment and
// i18n.commitEncoding through exec's pinnedConfig, so neither the user's
// environment nor the repository's git config can influence the result,
// which is what makes the commit the same on every machine.
func (r *Repo) MakeCommit(tree, message string, author Identity) (string, error) {
	env := []string{
		"GIT_AUTHOR_NAME=" + author.Name,
		"GIT_AUTHOR_EMAIL=" + author.Email,
		"GIT_AUTHOR_DATE=" + author.Date,
		"GIT_COMMITTER_NAME=" + author.Name,
		"GIT_COMMITTER_EMAIL=" + author.Email,
		"GIT_COMMITTER_DATE=" + author.Date,
	}

	out, err := r.exec([]byte(message), env, "commit-tree", tree)

	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(out)), nil
}

// UpdateRef points a ref at a commit, creating it if needed.
func (r *Repo) UpdateRef(ref, hash string) error {
	_, err := r.Run("update-ref", ref, hash)

	return err
}

// RemoteHead asks a remote what its branch points at, without downloading
// anything. found is false when the remote has no such branch, including the
// case of a brand-new empty repository.
func (r *Repo) RemoteHead(remote, branch string) (hash string, found bool, err error) {
	out, err := r.Run("ls-remote", remote, "refs/heads/"+branch)

	if err != nil {
		return "", false, err
	}

	if out == "" {
		return "", false, nil
	}

	// ls-remote prints the hash, a tab, then the ref name.
	hash, _, _ = strings.Cut(out, "\t")

	return hash, true, nil
}

// Contains reports whether ancestor is reachable from descendant.
func (r *Repo) Contains(ancestor, descendant string) (bool, error) {
	_, err := r.Run("merge-base", "--is-ancestor", ancestor, descendant)

	if err == nil {
		return true, nil
	}

	// Exit 1 means "not an ancestor". That's an answer, not a failure.
	if exError, ok := errors.AsType[*Error](err); ok && exError.ExitCode == 1 {
		return false, nil
	}

	return false, err
}

// Fetch downloads a remote ref into a local ref. A refspec starting with +
// lets the local ref move to an unrelated commit.
func (r *Repo) Fetch(remote, refspec string) error {
	// --no-tags fetches only the ref asked for, never the remote's tags.
	_, err := r.Run("fetch", "--no-tags", remote, refspec)

	return err
}

// Push sends src, a local ref or a commit hash, to the remote's branch.
// oldTip is the hash the remote branch must still hold for the push to
// replace it. An empty oldTip means the branch does not exist yet, and the
// push is a plain one, which git refuses if someone created it meanwhile.
func (r *Repo) Push(remote, src, branch, oldTip string) error {

	args := []string{"push"}

	// Replace the remote branch only if it still points at oldTip.
	if oldTip != "" {
		args = append(args, "--force-with-lease=refs/heads/"+branch+":"+oldTip)
	}

	// Use the full ref name so git never confuses a branch with a tag.
	args = append(args, remote, src+":"+"refs/heads/"+branch)

	_, err := r.Run(args...)

	return err
}
