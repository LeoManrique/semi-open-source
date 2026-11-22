package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/LeoManrique/semi-open-source/cli/internal/config"
	"github.com/LeoManrique/semi-open-source/cli/internal/gitx"
	"github.com/LeoManrique/semi-open-source/cli/internal/snapshot"
)

// fetchedRef is where diff fetches the mirror's commit to, in the private
// repository, to read its files.
const fetchedRef = "refs/sos/fetched"

func cmdDiff(args []string) error {
	flags := flag.NewFlagSet("diff", flag.ContinueOnError)
	// Flag errors and help go to stderr, keeping stdout for results.
	flags.SetOutput(os.Stderr)
	// A pointer, filled in by Parse, so read it as *showTree afterwards.
	showTree := flags.Bool("tree", false, "list every file the mirror would hold")

	flags.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: sos diff [--tree]")
		flags.PrintDefaults()
	}
	if err := flags.Parse(args); err != nil {
		return err
	}
	// Parse stops at the first non-flag word, so refuse it instead of ignoring it.
	if flags.NArg() > 0 {
		flags.Usage()
		return fmt.Errorf("diff takes no arguments, got %q", flags.Arg(0))
	}

	repo, cfg, err := openInitialized()

	if err != nil {
		return err
	}

	// The same commit publish would push, so the preview can't differ from it.
	public, err := snapshot.Build(repo, cfg.Branch)

	if err != nil {
		return err
	}

	// --tree lists what the mirror would hold, without asking the mirror.
	if *showTree {
		for _, p := range public.Files.Paths() {
			fmt.Println(p)
		}
		return nil
	}

	before, err := mirrorTree(repo, cfg)

	if err != nil {
		return err
	}

	added, modified, removed := snapshot.Diff(before, public.Files)

	// Nothing would change, so say so instead of printing an empty list.
	if len(added)+len(modified)+len(removed) == 0 {
		fmt.Println("No published file would change.")
		return nil
	}

	fmt.Printf("The next publish to %s (%s) would change:\n", cfg.Mirror, cfg.Branch)
	for _, p := range added {
		fmt.Printf("  + %s\n", p)
	}
	for _, p := range modified {
		fmt.Printf("  ~ %s\n", p)
	}
	for _, p := range removed {
		fmt.Printf("  - %s\n", p)
	}
	return nil
}

// mirrorTree reads the files of the commit the mirror's branch holds,
// fetching it into fetchedRef. A mirror without the branch holds nothing.
func mirrorTree(repo *gitx.Repo, cfg *config.Config) (snapshot.Tree, error) {
	// Ask for the branch tip first, which downloads nothing.
	_, found, err := repo.RemoteHead(cfg.Mirror, cfg.Branch)

	if err != nil {
		return nil, fmt.Errorf("reading the mirror: %w", err)
	}

	// No branch on the mirror yet, so every file will be an addition.
	if !found {
		return snapshot.Tree{}, nil
	}

	// + forces the ref, since each publish replaces the mirror's commit.
	refspec := "+refs/heads/" + cfg.Branch + ":" + fetchedRef

	// Fetch into its own ref, so refs/sos/mirror keeps this clone's last publish.
	if err := repo.Fetch(cfg.Mirror, refspec); err != nil {
		return nil, fmt.Errorf("fetching the mirror: %w", err)
	}
	return snapshot.TreeAt(repo, fetchedRef)
}
