package gates

import (
	"fmt"

	"github.com/LeoManrique/semi-open-source/cli/internal/gitx"
	"github.com/LeoManrique/semi-open-source/cli/internal/refuse"
	"github.com/LeoManrique/semi-open-source/cli/internal/snapshot"
)

// Run inspects the public commit before it is pushed. It returns a refusal
// listing everything that must not be published, or nil when there is none.
func Run(repo *gitx.Repo, public *snapshot.Public) error {
	found, err := inspect(repo, public)

	if err != nil {
		return err
	}

	// Nothing found, so the publish goes on.

	if len(found) == 0 {
		return nil
	}

	var details []string

	for _, f := range found {
		details = append(details, f.String())
	}

	// A refusal, so main exits 1 and prints one line per finding.
	return refuse.New(fmt.Sprintf("%d problem(s) would be published", len(found)), details...)
}

// inspect collects every finding in the public commit that nobody reviewed.
func inspect(repo *gitx.Repo, public *snapshot.Public) ([]Finding, error) {
	// The fingerprints as committed at the tip, the commit being published.
	allowed, err := allowedAt(repo, public.Tip)

	if err != nil {
		return nil, err
	}

	// One scanner for every file, so its rules compile once.
	scan, err := newScanner()

	if err != nil {
		return nil, err
	}

	// The message is published too, so it is scanned like any file.
	found, err := scan.message(public.Message)

	if err != nil {
		return nil, err
	}

	for _, p := range public.Files.Paths() {
		entry := public.Files[p]
		// A submodule pointer names a commit, not a blob, and has no content.
		var content []byte
		if entry.Type() == "blob" {
			// Read from the object database by hash, never from the working tree.
			content, err = repo.ReadBlob(entry.Hash)

			if err != nil {
				return nil, err
			}
		}
		// Every entry goes through checkEntry, submodule pointers included.
		found = append(found, checkEntry(p, entry, content)...)

		// A pointer's nil content holds no secret, so scanning it finds nothing.
		batch, err := scan.file(p, content)
		if err != nil {
			return nil, err
		}
		found = append(found, batch...)
	}

	return unreviewed(found, allowed), nil
}
