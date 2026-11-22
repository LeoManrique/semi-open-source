package e2e

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/LeoManrique/semi-open-source/e2e/harness"
)

// The harness's first target: proof it can run the real binary and tell its
// three channels apart. It also pins the house rule that results go to stdout
// and a command that succeeded says nothing at all on stderr, so users can
// pipe sos into other tools.
func TestVersion_PrintsVersionOnStdout(t *testing.T) {
	repo := harness.NewRepo(t)

	res := repo.SOS("version").Succeeds()

	const want = "sos "
	assert.True(t, strings.HasPrefix(res.Stdout, want), "stdout = %q, want it to start with %q", res.Stdout, want)
	assert.Empty(t, res.Stderr)
}

// -X fails silently: a linker flag naming a variable that does not exist is
// ignored, and the binary falls back to build info as if nobody had stamped
// it. Only a build with a known stamp shows that the stamp still reaches
// `sos version`.
func TestVersion_ReportsStampedVersion(t *testing.T) {
	// No tag or commit looks like this, so only the stamp can print it.
	bin := buildStamped(t, "9.9.9-test")

	assert.Equal(t, "sos 9.9.9-test", reportedVersion(t, bin))
}

// An empty stamp is what a build nobody stamped carries, such as go install.
// It still reports a version, a commit or dev depending on where it was
// built, never the bare name.
func TestVersion_UnstampedBuildStillReportsAVersion(t *testing.T) {
	bin := buildStamped(t, "")

	got := reportedVersion(t, bin)
	// Check only the prefix: the version itself depends on where it was built.
	assert.True(t, strings.HasPrefix(got, "sos "), "version = %q, want a version after %q", got, "sos ")
}

// buildStamped builds sos with version stamped in, and returns the binary's
// path. It goes through a just recipe, so the test never runs go itself.
func buildStamped(t *testing.T, version string) string {
	t.Helper()
	// An absolute path, since the recipe runs from cli/.
	bin := filepath.Join(t.TempDir(), "sos")

	build := exec.Command("just", "build-stamped", version, bin)
	// Run from the repo root, where the justfile lives.
	build.Dir = ".."
	out, err := build.CombinedOutput()
	require.NoError(t, err, "just build-stamped\n%s", out)
	return bin
}

// reportedVersion runs `sos version` with the binary at bin and returns what
// it printed, without the newline.
func reportedVersion(t *testing.T, bin string) string {
	t.Helper()
	// Output keeps only stdout, where sos prints its version.
	out, err := exec.Command(bin, "version").Output()
	require.NoError(t, err, "%s version", bin)
	return strings.TrimSpace(string(out))
}
