package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/LeoManrique/semi-open-source/cli/internal/gates"
	"github.com/LeoManrique/semi-open-source/cli/internal/publish"
	"github.com/LeoManrique/semi-open-source/cli/internal/snapshot"
)

func cmdCheck(args []string) error {
	// No flags at all, so any flag is a usage error rather than ignored.
	flags := flag.NewFlagSet("check", flag.ContinueOnError)
	// Flag errors and help go to stderr, keeping stdout for results.
	flags.SetOutput(os.Stderr)
	flags.Usage = func() { fmt.Fprintln(os.Stderr, "usage: sos check") }
	if err := flags.Parse(args); err != nil {
		return err
	}
	// Parse stops at the first non-flag word, so refuse it instead of ignoring it.
	if flags.NArg() > 0 {
		flags.Usage()
		return fmt.Errorf("check takes no arguments, got %q", flags.Arg(0))
	}

	repo, cfg, err := openInitialized()

	if err != nil {
		return err
	}

	// Step 1, as in a publish. Step 2 asks the private remote over the
	// network, which check never does.
	if err := publish.RequireMirrorNotAPushTarget(repo, cfg.Mirror); err != nil {
		return err
	}

	// Step 3: compute the public commit, and run the gates over it.
	public, err := snapshot.Build(repo, cfg.Branch)

	if err != nil {
		return err
	}

	// A refusal reaches main as it is, so check exits 1, the code CI relies on.
	if err := gates.Run(repo, public); err != nil {
		return err
	}

	fmt.Printf("No problems found in %s, the public commit of %s.\n", public.Commit[:8], cfg.Branch)
	return nil
}
