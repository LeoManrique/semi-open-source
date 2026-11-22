package config

import (
	"fmt"
	"os"
	"path"
)

const includeTemplate = `# .sos/include: the allowlist.
#
# Nothing is published unless a pattern here matches it. The syntax is
# .gitignore's, read as "may be published": a leading ! excludes again, and
# the last matching pattern wins. Nothing under .sos/ is ever published.
#
# Publishing reads this file as committed at the branch tip, so a change
# takes effect once it is committed. sos match reads it as it is on disk.
#
# Examples:
#   cmd/
#   docs/
#   !docs/internal/
#   README.md
#   LICENSE
`

const secretAllowTemplate = `# .sos/secret-allow: reviewed secret-scan exceptions.
#
# One fingerprint per line, exactly as sos publish and sos check print it:
#   <path>:<rule-id>:<line>
#
# Each line records a human decision that a finding is a false alarm.
# Anything not listed here blocks publishing.
`

const configTemplate = `# sos configuration. Tracked in the private repository; never published.

# Where the public mirror lives.
mirror = %q

# The branch whose tip is published.
branch = %q

# The project name, the message of the public commit.
name = %q
`

// Scaffold writes a fresh .sos/ directory. It fails if one already exists.
func Scaffold(repoDir string, c Config) error {
	// Write through a root so a .sos symlink can't point outside the repo.
	root, err := os.OpenRoot(repoDir)

	if err != nil {
		return err
	}

	defer func() {
		_ = root.Close()
	}()

	// Unlike os.MkdirAll, this fails when .sos is a symlink out of the repo.
	if err = root.MkdirAll(Dir, 0o755); err != nil {
		return fmt.Errorf("creating %s/: %w", Dir, err)
	}

	// A slice, not a map: the order files are created in is observable when
	// one of these writes fails. Config first, then the file you must edit.
	files := []struct{ name, body string }{
		{File, fmt.Sprintf(configTemplate, c.Mirror, c.Branch, c.Name)},
		{Include, includeTemplate},
		{SecretAllow, secretAllowTemplate},
	}

	for _, f := range files {
		// Roots take / paths, so use path.Join, not filepath.Join.
		if err = writeNew(root, path.Join(Dir, f.name), f.body); err != nil {
			return err
		}
	}
	return nil
}

// writeNew creates a file inside root, refusing to clobber an existing one.
func writeNew(root *os.Root, name, body string) error {
	// O_EXCL fails if the file exists, so a second init never overwrites it.
	f, err := root.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)

	if err != nil {
		return err
	}

	// WriteString errors on a short write, so the byte count isn't needed.
	_, err = f.WriteString(body)

	if err != nil {
		// Close anyway, but report the write's error, since it came first.
		_ = f.Close()
		return err
	}

	// A failed close can mean the written bytes never reached the disk.
	return f.Close()
}
