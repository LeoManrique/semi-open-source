package e2e

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// The preview a user reads before publishing: the next publish would add
// exactly one file, and the listing names it and nothing the mirror already
// has. Looking is free: the mirror is where it was.
func TestDiff_ShowsAddedPaths(t *testing.T) {
	repo, mirror := initialized(t, "README.md\ndocs/\n")
	repo.Write("README.md", "# Demo\n")
	repo.Commit("first")
	// Publish first, so the mirror already holds README.md.
	repo.SOS("publish").Succeeds()
	// Keep the mirror's commit, to prove diff leaves it where it was.
	published := mirror.Head("main")

	repo.Write("docs/guide.md", "How to use it.\n")
	repo.Commit("Add a guide")

	res := repo.SOS("diff").Succeeds()

	// Only the new file is listed, since README.md is already on the mirror.
	want := "The next publish to " + mirror.URL() + " (main) would change:\n  + docs/guide.md\n"
	assert.Equal(t, want, res.Stdout)
	assert.Equal(t, published, mirror.Head("main"), "sos diff moved the mirror")
}

// --tree answers a different question: not what changes next, but
// everything the mirror would hold. Nothing outside the allowlist may show
// up in it.
func TestDiff_TreeFlagListsFullPublishedFileList(t *testing.T) {
	repo, _ := initialized(t, "docs/\n")
	repo.Write("docs/a.md", "A\n")
	repo.Write("docs/b/c.md", "C\n")
	// Outside the allowlist, so it must never show up.
	repo.Write("secret.txt", "shh\n")
	repo.Commit("first")

	// No publish first: --tree never asks the mirror.
	res := repo.SOS("diff", "--tree").Succeeds()

	// Both files under docs/, sorted, the nested one included, and nothing else.
	assert.Equal(t, "docs/a.md\ndocs/b/c.md\n", res.Stdout)
}
