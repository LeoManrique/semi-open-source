package e2e

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/LeoManrique/semi-open-source/e2e/harness"
)

// A URL that goes nowhere. sos init records it without ever contacting it,
// which is what lets you initialize a repository whose mirror does not exist
// yet.
const mirrorURL = "https://example.com/project.git"

// The three tracked files arrive together. include and config.toml are what
// every later command reads; secret-allow is scaffolded so its own comments
// document a format nobody would guess.
func TestInit_CreatesScaffold(t *testing.T) {
	repo := harness.NewRepo(t)

	repo.SOS("init", "--mirror", mirrorURL).Succeeds()

	// Each of the three tracked files must exist under .sos/.
	for _, name := range []string{"include", "config.toml", "secret-allow"} {
		assert.True(t, repo.Exists(".sos/"+name), "sos init did not create .sos/%s", name)
	}
}

// Without --name, the project name is the mirror repository's own name: the
// last part of its URL, without ".git". The common case needs no flag.
func TestInit_DefaultsNameFromMirrorURL(t *testing.T) {
	repo := harness.NewRepo(t)

	repo.SOS("init", "--mirror", mirrorURL).Succeeds()

	// mirrorURL ends in /project.git, so the name is project.
	assert.Contains(t, repo.Read(".sos/config.toml"), `name = "project"`)
}

// With no --branch, init records the branch you are standing on, so the
// common case needs no flag.
func TestInit_RecordsCurrentBranchByDefault(t *testing.T) {
	repo := harness.NewRepo(t)

	repo.SOS("init", "--mirror", mirrorURL).Succeeds()

	// NewRepo starts on main, and HEAD names it even before the first commit.
	assert.Contains(t, repo.Read(".sos/config.toml"), `branch = "main"`)
}
