package gitx

import (
	"errors"
	"fmt"
	"strings"
)

// CurrentBranch returns the checked-out branch, or an error on a detached HEAD.
func (r *Repo) CurrentBranch() (string, error) {
	name, err := r.Run("symbolic-ref", "--short", "HEAD")

	if err != nil {
		return "", fmt.Errorf("no branch checked out (detached HEAD?)")
	}

	return name, nil
}

// ResolveCommit turns any revision (branch, tag, hash) into a full commit hash.
// found is false when the revision simply doesn't exist.
func (r *Repo) ResolveCommit(rev string) (hash string, found bool, err error) {
	// ^{commit} turns a tag into its commit and rejects anything else.
	out, err := r.Run("rev-parse", "--verify", "--quiet", rev+"^{commit}")

	if err != nil {
		// With --quiet, a missing revision exits 1 and prints nothing.
		if exitErr, ok := errors.AsType[*Error](err); ok && exitErr.ExitCode == 1 {
			return "", false, nil
		}

		return "", false, err
	}
	return out, true, nil
}

// Identity is an author or committer: who, and when.
// Date is git's raw form, Unix seconds and a UTC offset, e.g. "1577836800 +0000".
type Identity struct {
	Name  string
	Email string
	Date  string
}

// Commit is the part of a commit object sos reads.
type Commit struct {
	Tree   string
	Author Identity
}

// ReadCommit parses a commit object.
func (r *Repo) ReadCommit(hash string) (*Commit, error) {
	raw, err := r.RunBytes("cat-file", "commit", hash)

	if err != nil {
		return nil, err
	}

	c := &Commit{}
	text := string(raw)

	// Headers end at the first blank line. The message comes after.
	headers, _, found := strings.Cut(text, "\n\n")

	if !found {
		return nil, fmt.Errorf("commit %s: no message separator", hash)
	}

	for line := range strings.SplitSeq(headers, "\n") {
		// A leading space continues the previous header, like a gpgsig block.
		if strings.HasPrefix(line, " ") {
			continue
		}

		key, value, _ := strings.Cut(line, " ")

		switch key {
		case "tree":
			c.Tree = value
		case "author":
			c.Author = parseIdentity(value)
		}
	}
	return c, nil
}

// parseIdentity splits `Leo Manrique <leo@example.com> 1577836800 +0000`.
func parseIdentity(s string) Identity {
	// Search from the end, since the name itself may contain " <".
	open := strings.LastIndex(s, " <")
	shut := strings.LastIndex(s, "> ")

	// Unexpected shape, so keep it all as the name instead of failing.
	if open < 0 || shut < open {
		return Identity{Name: s, Email: "", Date: ""}
	}

	return Identity{
		Name:  s[:open],
		Email: s[open+2 : shut],
		Date:  s[shut+2:],
	}
}

// TreeEntry is one line of a tree object.
// Path is the full path from the repo root when produced by ListTree; when
// passed to MakeTree only the final element is used.
type TreeEntry struct {
	Mode string // 100644, 100755, 120000, 160000, 040000
	Type string // blob, tree, commit
	Hash string
	Path string
}

// ListTree returns every file in a commit (blobs, symlinks and gitlinks) with
// full paths. Directories are not returned; git has no empty directories, so
// the file list fully describes the tree.
func (r *Repo) ListTree(rev string) ([]TreeEntry, error) {
	// -r goes into subtrees. -z ends records with NUL and leaves paths unquoted.
	out, err := r.RunBytes("ls-tree", "-r", "-z", rev)

	if err != nil {
		return nil, err
	}

	var entries []TreeEntry
	for record := range strings.SplitSeq(string(out), "\x00") {
		// The output ends with a NUL, which leaves one empty record.
		if record == "" {
			continue
		}

		// A tab separates the metadata from the path, which may contain spaces.
		meta, path, ok := strings.Cut(record, "\t")

		if !ok {
			return nil, fmt.Errorf("unparsable ls-tree record %q", record)
		}

		fields := strings.Fields(meta)

		if len(fields) != 3 {
			return nil, fmt.Errorf("unparsable ls-tree metadata %q", meta)
		}

		entries = append(entries, TreeEntry{Mode: fields[0], Type: fields[1], Hash: fields[2], Path: path})
	}
	return entries, nil
}

// ReadBlob returns a blob's contents.
func (r *Repo) ReadBlob(hash string) ([]byte, error) {
	return r.RunBytes("cat-file", "blob", hash)
}

// ReadFileAt returns a file's contents as of a commit. found is false when the
// path doesn't exist there. Pass a commit hash you have already resolved: a
// bogus revision is indistinguishable from a missing path.
func (r *Repo) ReadFileAt(commit, path string) (content []byte, found bool, err error) {
	out, err := r.RunBytes("cat-file", "blob", commit+":"+path)

	if err != nil {
		// cat-file exits 128 when the path isn't in that commit.
		if pError, ok := errors.AsType[*Error](err); ok && pError.ExitCode == 128 {
			return nil, false, nil
		}
		return nil, false, err
	}
	return out, true, nil
}
