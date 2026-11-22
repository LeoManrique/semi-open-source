package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/LeoManrique/semi-open-source/cli/internal/config"
	"github.com/LeoManrique/semi-open-source/cli/internal/gates"
	"github.com/LeoManrique/semi-open-source/cli/internal/gitx"
	"github.com/LeoManrique/semi-open-source/cli/internal/publish"
	"github.com/LeoManrique/semi-open-source/cli/internal/snapshot"
)

// mirrorRef is where the last published commit is parked in the private
// repository. It keeps the commit reachable, so git's garbage collection
// never deletes it, and release tags are pushed from it.
const mirrorRef = "refs/sos/mirror"

func cmdPublish(args []string) error {
	// No flags at all, so any flag is a usage error rather than ignored.
	flags := flag.NewFlagSet("publish", flag.ContinueOnError)
	// Flag errors and help go to stderr, keeping stdout for results.
	flags.SetOutput(os.Stderr)
	flags.Usage = func() { fmt.Fprintln(os.Stderr, "usage: sos publish") }
	if err := flags.Parse(args); err != nil {
		return err
	}
	// Parse stops at the first non-flag word, so refuse it instead of ignoring it.
	if flags.NArg() > 0 {
		flags.Usage()
		return fmt.Errorf("publish takes no arguments, got %q", flags.Arg(0))
	}

	repo, cfg, err := openInitialized()

	if err != nil {
		return err
	}

	// Step 1: no setting a git push follows may lead to the mirror.
	if err = publish.RequireMirrorNotAPushTarget(repo, cfg.Mirror); err != nil {
		return err
	}

	// Step 2: the branch tip must be the private remote's tip.
	if err = publish.RequirePrivateRemoteTip(repo, cfg.Branch); err != nil {
		return err
	}

	// Step 3: compute the public commit, and run the gates over it.
	public, err := snapshot.Build(repo, cfg.Branch)

	if err != nil {
		return err
	}

	// Before step 4, so a refused publish never asks the mirror anything.
	if err = gates.Run(repo, public); err != nil {
		return err
	}

	// Step 4: ask the mirror where its branch is, and replace its commit
	// unless it is this one already.
	mirrorTip, found, err := repo.RemoteHead(cfg.Mirror, cfg.Branch)

	if err != nil {
		return fmt.Errorf("reading the mirror: %w", err)
	}

	// The same tip gives the same hash, so an equal hash means nothing new.
	upToDate := found && mirrorTip == public.Commit

	if !upToDate {
		// An empty mirrorTip makes a plain push. Any other replaces only that commit.
		err = repo.Push(cfg.Mirror, public.Commit, cfg.Branch, mirrorTip)

		if err != nil {
			return fmt.Errorf("pushing to the mirror: %w", err)
		}
	}

	// Step 5: park the commit the mirror now holds.
	if err = repo.UpdateRef(mirrorRef, public.Commit); err != nil {
		return err
	}

	// Nothing was pushed, so don't claim a publish.
	if upToDate {
		fmt.Println("Mirror is up to date.")
		return nil
	}

	fmt.Printf("Published %s to %s (%s)\n", public.Commit[:8], cfg.Mirror, cfg.Branch)
	return nil
}

// openInitialized is the preamble every command that reads the config
// shares: find the repository, then read the working tree's config.
func openInitialized() (*gitx.Repo, *config.Config, error) {
	repo, err := gitx.Open(".")

	if err != nil {
		return nil, nil, err
	}

	// With no config yet, Load's error says to run sos init.
	cfg, err := config.Load(repo.Dir)

	if err != nil {
		return nil, nil, err
	}
	return repo, cfg, nil
}
