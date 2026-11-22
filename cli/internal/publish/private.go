package publish

import (
	"fmt"
	"slices"
	"strings"

	"github.com/LeoManrique/semi-open-source/cli/internal/gitx"
	"github.com/LeoManrique/semi-open-source/cli/internal/refuse"
)

// sharedReason closes every refusal of the private-remote rule.
const sharedReason = "the mirror must show what the private remote holds, so every clone publishes the same commit"

// RequirePrivateRemoteTip refuses unless the branch tip is the private
// remote's tip, and names the one step that settles each difference. A
// repository with no remote at all is a solo project and is exempt.
func RequirePrivateRemoteTip(repo *gitx.Repo, branch string) error {
	remote, configured, err := privateRemote(repo, branch)

	if err != nil {
		return err
	}

	// No remote at all means a solo project, which is exempt.
	if !configured {
		return nil
	}

	// Use the full ref name so git never confuses a branch with a tag.
	local, found, err := repo.ResolveCommit("refs/heads/" + branch)

	if err != nil {
		return err
	}

	// Name the missing branch, since this runs before snapshot.Build would.
	if !found {
		return fmt.Errorf("branch %s does not exist", branch)
	}

	// Ask the remote for its tip, which downloads nothing.
	remoteTip, present, err := repo.RemoteHead(remote, branch)

	if err != nil {
		return fmt.Errorf("reading the private remote %s: %w", remote, err)
	}

	// Never pushed, so no teammate could compute this commit.
	if !present {
		return refuse.New(
			fmt.Sprintf("branch %s is not on the private remote %s", branch, remote),
			"push it first: git push -u "+remote+" "+branch,
			sharedReason,
		)
	}

	// Same tip, so every clone computes the same public commit.
	if remoteTip == local {
		return nil
	}

	// A tip this clone never fetched can't be compared, so check for it first.
	_, have, err := repo.ResolveCommit(remoteTip)

	if err != nil {
		return err
	}

	// Refuse rather than guess, and name the fetch that makes it comparable.
	if !have {
		return refuse.New(
			fmt.Sprintf("the private remote %s has commits you have not fetched", remote),
			"run: git fetch "+remote+", then publish again",
			sharedReason,
		)
	}

	// The local tip in the remote's history means this clone is behind.
	behind, err := repo.Contains(local, remoteTip)

	if err != nil {
		return err
	}

	// Behind refuses too: publishing would put an older state back.
	if behind {
		return refuse.New(
			fmt.Sprintf("branch %s is behind the private remote %s", branch, remote),
			"run: "+pullCommand(repo, remote, branch)+", then publish again",
			"publishing an older state would replace a newer commit a teammate may have published",
		)
	}

	// The remote's tip in the local history means this clone is ahead.
	ahead, err := repo.Contains(remoteTip, local)

	if err != nil {
		return err
	}

	if ahead {
		return refuse.New(
			fmt.Sprintf("branch %s has commits that are not on the private remote %s",
				branch, remote),
			"push them first: git push "+remote+" "+branch,
			sharedReason,
		)
	}

	// Neither tip contains the other, so the two have diverged.
	return refuse.New(
		fmt.Sprintf("branch %s and the private remote %s have diverged", branch, remote),
		"integrate them first, for example with git pull "+remote+" "+branch+
			", then git push "+remote+" "+branch,
		sharedReason,
	)
}

// privateRemote returns the remote the branch tracks, falling back to origin,
// and then to the first remote there is. Only a listed remote counts: a branch
// can also track "." (another local branch) or a bare URL. configured is false
// when the repository has no remote at all.
func privateRemote(repo *gitx.Repo, branch string) (string, bool, error) {
	remotes, err := remoteNames(repo)

	if err != nil {
		return "", false, err
	}

	// No remote at all, so the repository is a solo project.
	if len(remotes) == 0 {
		return "", false, nil
	}

	// git config --get exits 1 when the branch tracks no remote.
	name, err := repo.Run("config", "--get", "branch."+branch+".remote")

	// Only a listed remote counts, never "." or a bare URL.
	if err == nil && slices.Contains(remotes, name) {
		return name, true, nil
	}

	// Otherwise fall back to origin, the usual name of the shared remote.
	if slices.Contains(remotes, "origin") {
		return "origin", true, nil
	}

	// Otherwise the first remote, sorted by name.
	return remotes[0], true, nil
}

// remoteNames lists the repository's remotes, sorted by name.
func remoteNames(repo *gitx.Repo) ([]string, error) {
	list, err := repo.Run("remote")

	if err != nil {
		return nil, err
	}

	// Splitting "" would give one empty name, so no remotes is nil.
	if list == "" {
		return nil, nil
	}

	// git remote prints one name per line.
	return strings.Split(list, "\n"), nil
}

// pullCommand is the command that brings branch up to date with remote when
// branch is only behind it: git pull when branch is checked out, and
// otherwise a fetch straight into branch, since git pull would update the
// checked-out branch instead.
func pullCommand(repo *gitx.Repo, remote, branch string) string {
	current, err := repo.CurrentBranch()

	// git pull only updates the checked-out branch, so suggest it only for that one.
	if err == nil && current == branch {
		return "git pull " + remote + " " + branch
	}

	// Otherwise, detached HEAD included, fetch straight into branch.
	return "git fetch " + remote + " " + branch + ":" + branch
}
