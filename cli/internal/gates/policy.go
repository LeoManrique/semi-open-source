package gates

import (
	"path"
	"strings"

	"github.com/LeoManrique/semi-open-source/cli/internal/config"
	"github.com/LeoManrique/semi-open-source/cli/internal/gitx"
)

// secretAllowPath is where the reviewed fingerprints live inside every
// commit.
var secretAllowPath = path.Join(config.Dir, config.SecretAllow)

// allowedAt reads the fingerprints .sos/secret-allow lists as committed in
// commit, as a set. A commit without the file allows nothing.
func allowedAt(repo *gitx.Repo, commit string) (map[string]bool, error) {
	// Read from the commit, not the working tree, so every clone decides the same way.
	content, found, err := repo.ReadFileAt(commit, secretAllowPath)

	if err != nil {
		return nil, err
	}

	allowed := map[string]bool{}
	// No file in that commit, so nothing is let through.
	if !found {
		return allowed, nil
	}

	for raw := range strings.Lines(string(content)) {
		// Drop the newline strings.Lines leaves on, and any \r or stray spaces.
		line := strings.TrimSpace(raw)

		// Ignore blank lines and # comments.
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		allowed[line] = true
	}
	return allowed, nil
}

// unreviewed leaves out the findings whose fingerprint allowed holds. A
// finding with no fingerprint always stays: allowed never holds "".
func unreviewed(found []Finding, allowed map[string]bool) []Finding {
	var out []Finding
	for _, f := range found {
		// allowed never holds "", so a finding with no fingerprint always stays.
		if !allowed[f.Fingerprint] {
			out = append(out, f)
		}
	}
	return out
}
