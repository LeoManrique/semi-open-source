package e2e

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/LeoManrique/semi-open-source/e2e/harness"
)

// Named so a test and the line of code that implements it read the same way.
// A bare true/false at a call site says nothing.
const (
	included = true
	excluded = false
)

// allowlist writes .sos/include and returns the repo.
func allowlist(t *testing.T, lines string) *harness.Repo {
	t.Helper()
	repo := harness.NewRepo(t)
	repo.Write(".sos/include", lines)
	return repo
}

// assertMatch runs sos match and checks the verdict for one path.
func assertMatch(t *testing.T, repo *harness.Repo, path string, wantIncluded bool) {
	t.Helper()
	res := repo.SOS("match", path).Succeeds()

	// sos match prints the verdict first, "include" or "exclude".
	got := strings.HasPrefix(res.Stdout, "include")
	assert.Equal(t, wantIncluded, got, "sos match %s printed %q", path, strings.TrimSpace(res.Stdout))
}

// The starting state of every repository: an allowlist that says nothing
// publishes nothing. If the default ran the other way, every other rule of
// the dialect would be decoration.
func TestMatch_EmptyAllowlistExcludesEverything(t *testing.T) {
	repo := allowlist(t, "")
	assertMatch(t, repo, "a.txt", excluded)
}

// The other half of the default: a literal name does publish the file it
// names. Fail-closed is only useful if there is still a way to open a door.
func TestMatch_ExactRootFileIncluded(t *testing.T) {
	repo := allowlist(t, "README.md\n")
	assertMatch(t, repo, "README.md", included)
}

// The everyday wildcard: "*" stands for any run of characters inside one path
// segment, so a file can be published by extension rather than by name.
func TestMatch_StarMatchesWithinSegment(t *testing.T) {
	repo := allowlist(t, "*.md\n")
	assertMatch(t, repo, "a.md", included)
}

// "**" is not a single level with a longer name. The recursion has to keep
// swallowing segments until the rest of the pattern fits, however far down
// the file turns out to be.
func TestMatch_DoubleStarMatchesManyDirectories(t *testing.T) {
	repo := allowlist(t, "docs/**/a.md\n")
	// Three directories between docs and a.md, more than one * could cover.
	assertMatch(t, repo, "docs/x/y/z/a.md", included)
}

// A trailing slash publishes a whole subtree, not just its immediate
// children: the pattern matches the ancestor "docs", never the file itself.
// Without this every file in the tree would need a line of its own.
func TestMatch_DirectoryPatternIncludesNestedFile(t *testing.T) {
	repo := allowlist(t, "docs/\n")
	// Two levels below docs, so only the ancestor docs can match.
	assertMatch(t, repo, "docs/a/b.md", included)
}

// "!" carves an exception out of a directory published wholesale: the
// everyday way to keep one secret inside an otherwise public tree. If it did
// not beat the broader line above it, the file would ship.
func TestMatch_NegationExcludesPreviouslyIncluded(t *testing.T) {
	// docs/ includes the file first, then the ! line below it excludes it again.
	repo := allowlist(t, "docs/\n!docs/secret.md\n")
	assertMatch(t, repo, "docs/secret.md", excluded)
}
