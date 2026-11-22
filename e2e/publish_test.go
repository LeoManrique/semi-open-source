package e2e

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/LeoManrique/semi-open-source/e2e/harness"
)

// initialized gives a test a repository wired to a fresh mirror by sos init,
// with the project named demo and the allowlist written but not committed.
// sos reads .sos/include from the commit it publishes, so each test commits
// it along with its own files.
func initialized(t *testing.T, allowlist string) (*harness.Repo, *harness.Bare) {
	t.Helper()
	repo := harness.NewRepo(t)
	mirror := harness.NewBare(t)

	repo.SOS("init", "--mirror", mirror.URL(), "--name", "demo").Succeeds()
	// Not committed yet: each test commits it along with its own files.
	repo.Write(".sos/include", allowlist)
	return repo, mirror
}

// The first publish puts exactly one commit on the mirror: a root commit
// holding the allowed files and nothing else, named after the project, and
// authored and committed as the tip was. The excluded file's blob is not on
// the mirror at all, which is the claim sos makes: not that the mirror does
// not list the file, but that its bytes never crossed the wire.
func TestPublish_FirstPublishCreatesOneCommit(t *testing.T) {
	repo, mirror := initialized(t, "README.md\ndocs/\n")
	repo.Write("README.md", "# Demo\n")
	repo.Write("docs/guide.md", "How to use it.\n")
	repo.Write("secret.txt", "shh\n")
	repo.Commit("Add the first files")
	// The secret's blob hash, to look for on the mirror.
	secret := repo.Git("rev-parse", "HEAD:secret.txt")

	repo.SOS("publish").Succeeds()

	// A root commit, so the branch reaches exactly one commit.
	assert.Equal(t, "1", mirror.Git("rev-list", "--count", "refs/heads/main"), "commits on the mirror's branch")
	// Exactly the allowed files, and nothing from .sos/.
	assert.Equal(t, []string{"README.md", "docs/guide.md"}, mirror.Files("main"))
	// The secret's bytes never reached the mirror, not only its name.
	assert.False(t, mirror.HasObject(secret), "the excluded blob %s reached the mirror", secret)

	got := mirror.Git("log", "-1", "--date=raw", "--format=%P|%s|%an <%ae> %ad|%cn <%ce> %cd", "refs/heads/main")
	// No parent, the project name, and the fixture author at the clock's first tick.
	want := "|demo|Leo Manrique <leo@example.com> 1577836800 +0000|Leo Manrique <leo@example.com> 1577836800 +0000"
	assert.Equal(t, want, got, "the published commit's parents, subject, author and committer")
}

// Republishing replaces the mirror's commit rather than adding to it: the
// mirror always holds exactly one commit, the current state. A publish with
// nothing new pushes nothing, says so, and leaves the commit where it is.
func TestPublish_RepublishReplacesTheOnlyCommit(t *testing.T) {
	repo, mirror := initialized(t, "README.md\ndocs/\n")
	repo.Write("README.md", "# Demo\n")
	repo.Commit("first")
	repo.SOS("publish").Succeeds()
	first := mirror.Head("main")

	// A new included file, so the next publish has something new.
	repo.Write("docs/guide.md", "How to use it.\n")
	repo.Commit("Add a guide")
	repo.SOS("publish").Succeeds()

	second := mirror.Head("main")
	assert.NotEqual(t, first, second, "the second publish left the first commit on the mirror")
	// A root commit, so the branch reaches exactly one commit.
	assert.Equal(t, "1", mirror.Git("rev-list", "--count", "refs/heads/main"), "commits on the mirror's branch")
	// Exactly the allowed files, and nothing from .sos/.
	assert.Equal(t, []string{"README.md", "docs/guide.md"}, mirror.Files("main"))

	// Nothing committed since the last publish, so there is nothing to push.
	repo.SOS("publish").Succeeds().Says("Mirror is up to date.")

	assert.Equal(t, second, mirror.Head("main"), "a publish with nothing new moved the mirror")
}

// A remote that points at the mirror is one git push away from publishing the
// whole private history. sos refuses while it exists, names the setting, says
// how to remove it, and publishes nothing.
func TestPublish_RefusesWhenTheMirrorIsARemote(t *testing.T) {
	repo, mirror := initialized(t, "README.md\n")
	repo.Write("README.md", "# Demo\n")
	repo.Commit("first")
	// Point a remote at the mirror, with the URL exactly as sos init got it.
	repo.Git("remote", "add", "public", mirror.URL())

	// Exit 1 names the setting and the command that removes it.
	repo.SOS("publish").Refuses().
		Says("remote.public.url points at the public mirror").
		Says("git remote remove public")

	// The refusal comes before the push, so nothing reached the mirror.
	assert.True(t, mirror.IsEmpty(), "sos published with the mirror configured as a remote")
}

// The branch has a private remote but was never pushed to it, so no teammate
// could compute the commit about to be published. sos refuses before touching
// the mirror, and names the push that settles it.
func TestPublish_RefusesWhenBranchNotPushedToPrivateRemote(t *testing.T) {
	repo, mirror := initialized(t, "README.md\n")
	repo.Write("README.md", "# Demo\n")
	repo.Commit("first")
	// A private remote named origin, with nothing pushed to it yet.
	private := harness.NewBare(t)
	repo.Git("remote", "add", "origin", private.URL())

	// Exit 1 names the branch, the remote and the push that settles it.
	repo.SOS("publish").Refuses().
		Says("branch main is not on the private remote origin").
		Says("git push -u origin main")

	// The refusal comes before the push, so nothing reached the mirror.
	assert.True(t, mirror.IsEmpty(), "the refused publish still wrote to the mirror")
}

// detectableSecret is a real, detectable secret: it comes from gitleaks' test
// fixtures and matches Betterleaks' aws-access-token rule.
const detectableSecret = `aws_key = "AKIALALEMEL33243OLIA"`

// leaky is a Go file whose third line holds detectableSecret. The line number
// is part of the fingerprint the tests look for.
func leaky() string { return "package main\n\nconst " + detectableSecret + "\n" }

// A secret in a file sos is about to publish stops the publish before
// anything reaches the mirror. The refusal names the file, the line and the
// fingerprint to review, and never the secret itself: printed into a CI log,
// it would leak there instead.
func TestPublish_RefusesDetectedSecret(t *testing.T) {
	// The allowlist includes config.go, so the file is published and scanned.
	repo, mirror := initialized(t, "config.go\n")
	repo.Write("config.go", leaky())
	repo.Commit("Add the config")

	// The 1-based line, then the fingerprint to copy into .sos/secret-allow.
	res := repo.SOS("publish").Refuses().
		Says("config.go:3").
		Says("[config.go:aws-access-token:3]")

	// Neither stream may print the key, or a CI log would leak it.
	assert.NotContains(t, res.Stdout+res.Stderr, "AKIALALEMEL33243OLIA", "the refusal printed the secret")
	// The gates run before the push, so nothing reached the mirror.
	assert.True(t, mirror.IsEmpty(), "the refused publish still wrote to the mirror")
}

// A false alarm is let through by its fingerprint, committed in
// .sos/secret-allow: a reviewed decision, tracked like any other change.
// Comments in the file record why.
func TestPublish_AllowlistedFingerprintPublishes(t *testing.T) {
	// The allowlist includes config.go, so the file is published and scanned.
	repo, mirror := initialized(t, "config.go\n")
	repo.Write("config.go", leaky())
	// The fingerprint the refusal printed, under a comment on why it is safe.
	repo.Write(".sos/secret-allow", "# reviewed: a key from gitleaks' own fixtures\nconfig.go:aws-access-token:3\n")
	repo.Commit("Add the config")

	repo.SOS("publish").Succeeds()

	// The fingerprint let the file through, and .sos/ itself is never published.
	assert.Equal(t, []string{"config.go"}, mirror.Files("main"))
}

// A submodule pointer names a commit in a repository the mirror's visitors
// cannot fetch, so the published tree would not check out. sos refuses it
// and names the path, and the mirror stays empty.
func TestPublish_RefusesSubmoduleInPublishedTree(t *testing.T) {
	// Include everything, so the pointer is in the published tree.
	repo, mirror := initialized(t, "*\n")
	repo.Write("README.md", "# Demo\n")
	// git never checks a pointer's commit exists, so the repo's own first one will do.
	pointee := repo.Commit("first")
	repo.AddGitlink("vendor/lib", pointee)
	repo.Commit("Add a submodule pointer")

	// A whole-file finding: the path, with no line and no fingerprint.
	repo.SOS("publish").Refuses().Says("vendor/lib: submodule pointer")

	// The gates run before the push, so nothing reached the mirror.
	assert.True(t, mirror.IsEmpty(), "the refused publish still wrote to the mirror")
}
