package harness

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// Bare is a bare repository standing in for a hosted mirror. Its directory
// path doubles as its URL: git takes a local path as a remote.
type Bare struct {
	t   *testing.T
	Dir string
}

// NewBare creates an empty bare repository.
func NewBare(t *testing.T) *Bare {
	t.Helper()

	// A directory of its own, inside the one go test -artifacts keeps, like NewRepo's.
	parent, err := os.MkdirTemp(t.ArtifactDir(), "bare-")
	require.NoError(t, err)

	dir := filepath.Join(parent, "remote.git")

	// Pin sha1 like NewRepo, since git refuses a push between object formats.
	runGit(t, parent, nil, "init", "-q", "--bare", "-b", "main", "--object-format=sha1", dir)

	return &Bare{t: t, Dir: dir}
}

// URL is what sos init --mirror is given.
func (b *Bare) URL() string { return b.Dir }

// Git runs a git command inside the bare repository and returns its stdout
// without the trailing newlines, like Repo.Git.
func (b *Bare) Git(args ...string) string {
	b.t.Helper()
	return strings.TrimRight(runGit(b.t, b.Dir, nil, args...), "\n")
}

// IsEmpty reports whether nothing has been published yet: no branch at all.
func (b *Bare) IsEmpty() bool {
	b.t.Helper()
	// With no branch at all, for-each-ref prints nothing.
	return b.Git("for-each-ref", "--format=%(refname)", "refs/heads/") == ""
}

// Head returns the commit hash the branch points at.
func (b *Bare) Head(branch string) string {
	b.t.Helper()
	// Use the full ref name so git never confuses a branch with a tag.
	return b.Git("rev-parse", "refs/heads/"+branch)
}

// HasObject reports whether an object exists in the repository. Unlike Git it
// tolerates one failure, git's "no such object", because a false answer is
// the point: the strongest leak assertion there is says an excluded blob
// never arrived here at all. Any other failure fails the test, so a broken
// mirror can never pass for one without the blob.
func (b *Bare) HasObject(hash string) bool {
	b.t.Helper()
	// Not runGit, since one non-zero exit here is the answer, not a failure.
	cmd := exec.Command("git", "cat-file", "-e", hash)

	cmd.Dir = b.Dir

	err := cmd.Run()

	// cat-file -e answers through its exit code, and 0 means the object exists.
	if err == nil {
		return true
	}

	// Exit 1 means no such object. That's an answer, not a failure.
	if exErr, ok := errors.AsType[*exec.ExitError](err); ok && exErr.ExitCode() == 1 {
		return false
	}

	// Anything else, like 128, means git couldn't ask, so fail closed.
	require.NoError(b.t, err, "git cat-file -e %s (in %s)", hash, b.Dir)

	return false
}

// Files returns every file path at a revision, in byte order, which is the
// order git lists a tree in.
//
// -z is not optional here: without it git C-quotes any path containing a
// non-ASCII byte or a quote, and whether it does depends on the developer's
// core.quotePath. This is the harness's main leak assertion, and it must give
// the same answer on every machine.
func (b *Bare) Files(rev string) []string {
	b.t.Helper()
	var files []string
	// -r goes into subtrees. -z ends records with NUL and leaves paths unquoted.
	out := b.Git("ls-tree", "-r", "-z", "--name-only", rev)

	for name := range strings.SplitSeq(out, "\x00") {
		// The output ends with a NUL, which leaves one empty record.
		if name != "" {
			files = append(files, name)
		}
	}

	return files
}
