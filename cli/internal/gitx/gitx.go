// Package gitx is the only part of sos that knows about git. It wraps the
// system git binary's plumbing commands.
package gitx

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"slices"
	"strings"
	"time"
)

// EmptyTree is git's hash for a tree with no entries. Every repository has it.
const EmptyTree = "4b825dc642cb6eb9a060e54bf8d69288fbee4904"

// Repo is a git repository on disk.
type Repo struct {
	Dir string // absolute path to the working tree root
}

// Error is a git invocation that exited non-zero.
type Error struct {
	Args     []string
	Stderr   string
	ExitCode int
	Err      error // what os/exec reported, so callers can still reach it
}

func (e *Error) Error() string {
	return fmt.Sprintf("git %s: %s", strings.Join(e.Args, " "), e.Stderr)
}

// Unwrap exposes the underlying exec error to errors.Is and errors.AsType.
func (e *Error) Unwrap() error { return e.Err }

// Open locates the repository containing dir and returns it rooted at the
// working tree's top level.
func Open(dir string) (*Repo, error) {
	// A temporary Repo at dir, only to ask git for the top level.
	probe := &Repo{Dir: dir}

	top, err := probe.Run("rev-parse", "--show-toplevel")

	if err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return nil, fmt.Errorf("git is not installed or not on PATH: %w", err)
		}

		// rev-parse exits 128 when dir is not inside a repository.
		if repoError, ok := errors.AsType[*Error](err); ok && repoError.ExitCode == 128 {
			return nil, fmt.Errorf("%s is not inside a git repository", dir)
		}

		return nil, fmt.Errorf("locating the repository containing %s: %w", dir, err)
	}

	return &Repo{Dir: top}, nil
}

// pinnedConfig neutralizes the one config knob that changes a commit's bytes.
// Everything sos writes must hash identically on every machine, so this is
// passed to every invocation rather than only to the ones that write.
var pinnedConfig = []string{"-c", "i18n.commitEncoding=UTF-8"}

// exec runs git and returns its raw stdout. stdin and env may be nil.
func (r *Repo) exec(stdin []byte, env []string, args ...string) ([]byte, error) {
	cmd := exec.Command(
		"git",
		// Clone so the append never writes into pinnedConfig.
		append(slices.Clone(pinnedConfig), args...)...,
	)

	cmd.Dir = r.Dir

	if stdin != nil {
		cmd.Stdin = bytes.NewReader(stdin)
	}

	// Fail instead of hanging when a remote asks for credentials.
	cmd.Env = append(cmd.Environ(), "GIT_TERMINAL_PROMPT=0")

	if env != nil {
		cmd.Env = append(cmd.Env, env...)
	}

	var stdout, stderr bytes.Buffer

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Stop waiting 10s after git exits if a child still holds its pipes.
	cmd.WaitDelay = 10 * time.Second

	err := cmd.Run()

	if err != nil {
		msg := strings.TrimSpace(stderr.String())

		// Some failures print nothing to stderr, so use exec's own error.
		if msg == "" {
			msg = err.Error()
		}

		// -1 when git never started, so there is no exit code.
		code := -1

		if cmd.ProcessState != nil {
			code = cmd.ProcessState.ExitCode()
		}

		return nil, &Error{Args: args, Stderr: msg, ExitCode: code, Err: err}
	}
	return stdout.Bytes(), nil
}

// Run returns stdout with trailing newlines trimmed, right for hashes and
// single-line answers.
func (r *Repo) Run(args ...string) (string, error) {
	out, err := r.exec(nil, nil, args...)

	return strings.TrimRight(string(out), "\n"), err
}

// RunBytes returns stdout untouched, right for file contents.
func (r *Repo) RunBytes(args ...string) ([]byte, error) {
	return r.exec(nil, nil, args...)
}
