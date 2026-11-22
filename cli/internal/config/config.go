// Package config reads and writes .sos/, sos's tracked configuration.
package config

import (
	"errors"
	"fmt"
	"os"
	"path"

	"github.com/LeoManrique/semi-open-source/cli/internal/gitx"
	"github.com/pelletier/go-toml/v2"
)

// Names of the tracked files. Nothing under Dir is ever published.
const (
	Dir         = ".sos"
	Include     = "include"
	File        = "config.toml"
	SecretAllow = "secret-allow"

	// IncludePath is the allowlist's tree path, the same on disk and in every commit.
	IncludePath = Dir + "/" + Include
)

// ErrNotInitialized reports a repository with no .sos/config.toml.
var ErrNotInitialized = errors.New("not initialized: run sos init")

// Config is .sos/config.toml.
type Config struct {
	Mirror string `toml:"mirror"`
	Branch string `toml:"branch"`
	Name   string `toml:"name"`
}

// open returns a root confined to the repository, plus the repo-relative
// slash path of one of sos's files inside it.
func open(repoDir, name string) (*os.Root, string, error) {
	// Read through a root so a .sos symlink can't point outside the repo.
	root, err := os.OpenRoot(repoDir)

	if err != nil {
		return nil, "", err
	}

	// Roots take / paths, so use path.Join, not filepath.Join.
	return root, path.Join(Dir, name), nil
}

// Exists reports whether the repository has been initialized.
func Exists(repoDir string) bool {
	root, file, err := open(repoDir, File)

	if err != nil {
		return false
	}

	defer func() {
		_ = root.Close()
	}()

	// Stat fails when the file is missing, so a nil error means it exists.
	_, err = root.Stat(file)

	return err == nil
}

// Load reads .sos/config.toml from the working tree.
func Load(repoDir string) (*Config, error) {
	root, file, err := open(repoDir, File)

	if err != nil {
		return nil, err
	}

	defer func() {
		_ = root.Close()
	}()

	data, err := root.ReadFile(file)

	// A missing file means init never ran, so return the named error.
	if errors.Is(err, os.ErrNotExist) {
		return nil, ErrNotInitialized
	}

	if err != nil {
		return nil, err
	}

	return parse(data, file)
}

// LoadAt reads .sos/config.toml as committed in commit, a resolved commit
// hash. Everything that shapes the public commit is read through this, never
// from the working tree.
func LoadAt(repo *gitx.Repo, commit string) (*Config, error) {
	// Git tree paths use / on every OS, so use path.Join.
	data, found, err := repo.ReadFileAt(commit, path.Join(Dir, File))

	if err != nil {
		return nil, err
	}

	// Never fall back to the working tree: publishing needs .sos/ committed.
	if !found {
		return nil, fmt.Errorf("%s/%s is not committed at %s: commit %s/ first", Dir, File, commit, Dir)
	}

	// Name the commit too, so an error never points at the working-tree copy.
	return parse(data, fmt.Sprintf("%s/%s at %s", Dir, File, commit))
}

// parse reads config.toml's bytes and checks every field is set. where names
// the copy being read, for error messages.
func parse(data []byte, where string) (*Config, error) {
	var c Config

	// Say which copy is broken, the working tree's or a commit's.
	if err := toml.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("%s: %w", where, err)
	}

	// Unmarshal leaves a missing key empty, so check every field here.
	if err := c.validate(); err != nil {
		return nil, fmt.Errorf("%s: %w", where, err)
	}

	return &c, nil
}

// validate checks that every field is set: an empty one would turn up later
// as an empty mirror URL or an empty commit message.
func (c *Config) validate() error {
	if c.Mirror == "" {
		return errors.New("mirror is not set")
	}

	if c.Branch == "" {
		return errors.New("branch is not set")
	}

	if c.Name == "" {
		return errors.New("name is not set")
	}

	return nil
}
