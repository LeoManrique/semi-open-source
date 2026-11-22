// Package harness is what the blackbox tests stand on: throwaway git
// repositories, and a way to run the compiled sos binary in one and observe
// what it did. Nothing here may import sos's own packages. Each step a test
// takes is logged, and go test prints that log only for a failing test, or
// for every test under -v.
package harness

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	fixtureName  = "Leo Manrique"
	fixtureEmail = "leo@example.com"
	fixtureEpoch = 1577836800 // 2020-01-01T00:00:00Z
	fixtureStep  = 60         // seconds between commits
)

// Repo is a throwaway private repository under test.
type Repo struct {
	t    *testing.T
	Dir  string
	tick int // commits so far, for deterministic dates
}

// NewRepo creates an initialized git repository with everything that could
// vary between machines pinned down.
func NewRepo(t *testing.T) *Repo {
	t.Helper()

	// A directory of its own per repo, inside the one go test -artifacts keeps.
	dir, err := os.MkdirTemp(t.ArtifactDir(), "repo-")

	require.NoError(t, err)

	r := &Repo{t: t, Dir: dir}

	// Pin sha1 so hashes don't change if git's default does.
	r.Git("init", "-q", "-b", "main", "--object-format=sha1")

	// The fixture author, so no developer's identity reaches a commit.
	r.Git("config", "user.name", fixtureName)
	r.Git("config", "user.email", fixtureEmail)
	// Keep line endings exactly as written.
	r.Git("config", "core.autocrlf", "false")
	// Never prompt to sign a commit.
	r.Git("config", "commit.gpgsign", "false")
	// Track the executable bit.
	r.Git("config", "core.fileMode", "true")
	// The same commit encoding sos pins for its own commits.
	r.Git("config", "i18n.commitEncoding", "UTF-8")

	return r
}

// Git runs a git command in the repo and returns its stdout without the
// trailing newlines, failing the test if it errors. Porcelain is fine here:
// this is test setup.
func (r *Repo) Git(args ...string) string {
	r.t.Helper()
	return strings.TrimRight(runGit(r.t, r.Dir, nil, args...), "\n")
}

// runGit runs git in dir, with env added to the environment, and returns its
// stdout exactly as git wrote it, failing the test if git fails.
func runGit(t *testing.T, dir string, env []string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)

	cmd.Dir = dir
	cmd.Env = append(cmd.Environ(), env...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	require.NoError(t, err,
		"git %s (in %s)\n%s", strings.Join(args, " "), dir, stderr.String())

	return stdout.String()
}

// abs turns a slash-separated path inside the repo into a path on disk.
func (r *Repo) abs(path string) string {
	return filepath.Join(r.Dir, filepath.FromSlash(path))
}

// Write creates or replaces a file, making parent directories as needed.
func (r *Repo) Write(path, body string) {
	r.t.Helper()
	full := r.abs(path)

	err := os.MkdirAll(filepath.Dir(full), 0o755)
	require.NoError(r.t, err)

	err = os.WriteFile(full, []byte(body), 0o644)
	require.NoError(r.t, err)

	// Log the file, so a failing test shows what it was given.
	r.t.Logf("wrote %s:\n%s", path, body)
}

// Read returns a file's contents from the working tree.
func (r *Repo) Read(path string) string {
	r.t.Helper()

	data, err := os.ReadFile(r.abs(path))
	require.NoError(r.t, err)

	return string(data)
}

// Exists reports whether a path is present in the working tree. Unlike Read
// it tolerates absence, because absence is often the assertion.
func (r *Repo) Exists(path string) bool {
	r.t.Helper()

	_, err := os.Stat(r.abs(path))

	if errors.Is(err, os.ErrNotExist) {
		return false
	}
	require.NoError(r.t, err)

	return true
}

// Commit stages everything and commits it as the fixture author, at the next
// tick of the fixture clock. It returns the commit hash.
func (r *Repo) Commit(message string) string {
	r.t.Helper()
	r.Git("add", "-A")

	// Skip any hooks, and allow a commit with nothing changed.
	runGit(r.t, r.Dir, identityEnv(fixtureName, fixtureEmail, r.tickDate()),
		"commit", "-q", "--no-verify", "--allow-empty", "-m", message)

	hash := r.Git("rev-parse", "HEAD")

	// Log the commit, so a failing test shows its history.
	r.t.Logf("committed %s: %s", hash, message)

	return hash
}

// tickDate advances the fixture clock one step and returns the new date.
func (r *Repo) tickDate() string {
	date := fmt.Sprintf("%d +0000", fixtureEpoch+r.tick*fixtureStep)
	r.tick++
	return date
}

// identityEnv pins who a commit is by and when, so nothing about the developer
// running the test can reach the object.
func identityEnv(name, email, date string) []string {
	return []string{
		"GIT_AUTHOR_NAME=" + name,
		"GIT_AUTHOR_EMAIL=" + email,
		"GIT_AUTHOR_DATE=" + date,
		"GIT_COMMITTER_NAME=" + name,
		"GIT_COMMITTER_EMAIL=" + email,
		"GIT_COMMITTER_DATE=" + date,
	}
}

// AddGitlink stages a submodule pointer to commit at path, without a real
// submodule behind it. The empty directory matters: Commit runs git add -A,
// which would stage the pointer's removal if nothing were at that path.
func (r *Repo) AddGitlink(path, commit string) {
	r.t.Helper()
	// Something must be at the path, or git add -A stages the pointer's removal.
	err := os.MkdirAll(r.abs(path), 0o755)
	require.NoError(r.t, err)

	// Write the entry straight into the index, with 160000, the gitlink mode.
	r.Git("update-index", "--add", "--cacheinfo", "160000,"+commit+","+path)
}
