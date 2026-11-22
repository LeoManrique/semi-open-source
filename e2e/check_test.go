package e2e

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// sos check runs the checks a publish runs on this machine and pushes
// nothing: a clean repository passes with exit 0, and the mirror stays
// empty.
func TestCheck_CleanRepositoryExitsZero(t *testing.T) {
	repo, mirror := initialized(t, "README.md\n")
	repo.Write("README.md", "# Demo\n")
	repo.Commit("first")

	repo.SOS("check").Succeeds().Says("No problems found")

	// check stops before the network, so it never reaches the mirror.
	assert.True(t, mirror.IsEmpty(), "sos check wrote to the mirror")
}
