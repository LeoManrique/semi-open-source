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

// sosBin is the binary under test, located once per run.
var sosBin string

// Main locates the binary once, before anything runs, then runs the tests and
// exits with their status. Go only calls a TestMain that sits in the test
// package itself, so that TestMain hands its m to this.
func Main(m *testing.M) {
	path, err := findBinary()

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	sosBin = path

	os.Exit(m.Run())
}

// findBinary locates the sos binary built by `just build`. Go runs a test
// binary with its working directory set to the directory of the package under
// test, here <repo>/e2e rather than this package's, so the repo root, where
// the binary lands, is one level up.
func findBinary() (string, error) {
	// SOS_BIN runs the suite against another binary, like a release build.
	if bin := os.Getenv("SOS_BIN"); bin != "" {
		return filepath.Abs(bin)
	}

	path, err := filepath.Abs(filepath.Join("..", "sos"))

	if err != nil {
		return "", err
	}

	if _, err := os.Stat(path); err != nil {
		return "", fmt.Errorf("sos binary not found at %s\nrun `just e2e`, which builds it first", path)
	}

	return path, nil
}

// Result is one sos invocation's observable output.
type Result struct {
	t      *testing.T
	Stdout string
	Stderr string
	Code   int
}

// SOS runs the binary in the repo, with no input.
func (r *Repo) SOS(args ...string) Result {
	r.t.Helper()

	cmd := exec.Command(sosBin, args...)

	cmd.Dir = r.Dir

	var stdout, stderr bytes.Buffer

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	code := 0
	err := cmd.Run()

	// A non-zero exit is a result to check. Failing to start sos breaks the test.
	if err != nil {
		exit, ok := errors.AsType[*exec.ExitError](err)
		require.True(r.t, ok, "could not run %s: %v", sosBin, err)
		code = exit.ExitCode()
	}

	// Log the run, so a failing test shows the command and everything sos said.
	r.t.Logf("$ sos %s\nexit %d\nstdout:\n%s\nstderr:\n%s",
		strings.Join(args, " "), code, stdout.String(), stderr.String())

	return Result{t: r.t, Stdout: stdout.String(), Stderr: stderr.String(), Code: code}
}

// Succeeds asserts exit 0.
func (res Result) Succeeds() Result {
	res.t.Helper()
	return res.exits(0, "succeed")
}

// FailsUsage asserts exit 2: bad invocation or environment.
func (res Result) FailsUsage() Result {
	res.t.Helper()
	return res.exits(2, "fail with a usage error (exit 2)")
}

// Refuses asserts exit 1: sos declined on purpose.
func (res Result) Refuses() Result {
	res.t.Helper()
	return res.exits(1, "refuse (exit 1)")
}

// Says asserts that some output mentioned substr, on either stream.
func (res Result) Says(substr string) Result {
	res.t.Helper()
	require.Contains(res.t, res.Stdout+res.Stderr, substr,
		"stdout:\n%s\nstderr:\n%s", res.Stdout, res.Stderr)
	return res
}

// exits fails the test unless sos exited with want; what says what was
// expected, in words, for the failure message.
func (res Result) exits(want int, what string) Result {
	res.t.Helper()

	require.Equal(res.t, want, res.Code,
		"expected sos to %s\nstdout:\n%s\nstderr:\n%s", what, res.Stdout, res.Stderr)

	return res
}
