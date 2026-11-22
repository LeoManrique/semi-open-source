// Package publish holds the checks a publish runs before it computes
// anything: where a git push can go, and whether the branch is shared.
package publish

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/LeoManrique/semi-open-source/cli/internal/gitx"
	"github.com/LeoManrique/semi-open-source/cli/internal/refuse"
)

// RequireMirrorNotAPushTarget refuses while a setting a git push follows names
// the mirror's URL. sos pushes the mirror by URL so that no git command
// reaches it by accident; such a setting undoes that, and one plain git push
// would publish the whole private history.
func RequireMirrorNotAPushTarget(repo *gitx.Repo, mirror string) error {
	routes, err := pushRoutes(repo)

	if err != nil {
		return err
	}

	for _, r := range routes {
		// Compare raw strings, so only the mirror's URL as written matches.
		if r.url != mirror {
			continue
		}

		// Refuse at the first route to the mirror, saying how to remove it.
		return refuse.New(
			r.name+" points at the public mirror",
			"any git push through it would publish private history",
			"remove it: "+r.remove,
		)
	}
	return nil
}

// route is one setting a git push can follow to another repository.
type route struct {
	name   string // how a refusal names the setting
	remove string // the command that removes it
	url    string // the URL it names, exactly as written
}

// pushRoutes lists every route a git push can follow out of the repository:
// the push settings in its config, and its legacy remote files.
func pushRoutes(repo *gitx.Repo) ([]route, error) {
	routes, err := configRoutes(repo)

	if err != nil {
		return nil, err
	}

	legacy, err := legacyRoutes(repo)

	if err != nil {
		return nil, err
	}

	return append(routes, legacy...), nil
}

// legacyRoutes lists the remotes git still reads from files in .git/remotes
// and .git/branches, a format git remote does not list. Any word in such a
// file can be the URL, and in .git/branches a #branch may follow it.
func legacyRoutes(repo *gitx.Repo) ([]route, error) {
	var routes []route
	for _, kind := range []string{"remotes", "branches"} {
		// Ask git for the path, since .git can be a file, as in a worktree.
		dir, err := repo.Run("rev-parse", "--git-path", kind)

		if err != nil {
			return nil, err
		}

		// git prints it relative to the repository's root, where gitx runs git.
		if !filepath.IsAbs(dir) {
			dir = filepath.Join(repo.Dir, dir)
		}

		entries, err := os.ReadDir(dir)

		// No such directory is the usual case, not an error.
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}

		if err != nil {
			return nil, err
		}

		for _, entry := range entries {
			file := filepath.Join(dir, entry.Name())

			content, err := os.ReadFile(file)

			if err != nil {
				return nil, err
			}

			// Any word can be the URL. One that isn't never equals the mirror's.
			for word := range strings.FieldsSeq(string(content)) {
				// In .git/branches a #branch may follow the URL, so drop it.
				url, _, _ := strings.Cut(word, "#")

				// git remote remove can't remove it, so say to delete the file.
				routes = append(routes,
					route{name: "the remote file " + file, remove: "rm " + file, url: url})
			}
		}
	}
	return routes, nil
}

// pushSettings matches the keys of the settings that say where a push goes:
// a remote's url and pushurl, a branch's remote and pushremote, and
// remote.pushdefault, all in the lower case git config prints them in.
const pushSettings = `^(remote\..*\.(url|pushurl)|branch\..*\.(remote|pushremote)|remote\.pushdefault)$`

// configRoutes lists the push settings in git's config, each value read raw:
// git remote get-url would apply url.<base>.insteadOf first, and the
// rewritten URL could hide one that names the mirror exactly.
func configRoutes(repo *gitx.Repo) ([]route, error) {
	// Read the values raw, before any insteadOf rewrite, from every config file.
	out, err := repo.Run("config", "--get-regexp", pushSettings)

	// Exit 1 means no key matched. That's an answer, not a failure.
	if gitErr, ok := errors.AsType[*gitx.Error](err); ok && gitErr.ExitCode == 1 {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	var routes []route
	for line := range strings.SplitSeq(out, "\n") {
		// --get-regexp prints the key, a space, then the value.
		key, value, _ := strings.Cut(line, " ")

		routes = append(routes, route{name: key, remove: removal(key), url: value})
	}
	return routes, nil
}

// removal is the command that removes a push setting: the whole remote for a
// remote's url, the one setting for anything else.
func removal(key string) string {
	// Cut both known ends off, so a remote name with dots stays whole.
	name, isRemote := strings.CutPrefix(key, "remote.")
	name, isURL := strings.CutSuffix(name, ".url")

	// For a remote's url, remove the whole remote, never leave it without one.
	if isRemote && isURL {
		return "git remote remove " + name
	}

	// Anything else is one setting. --unset-all clears every value it holds.
	return "git config --unset-all " + key
}
