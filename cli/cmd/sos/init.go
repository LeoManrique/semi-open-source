package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/LeoManrique/semi-open-source/cli/internal/config"
	"github.com/LeoManrique/semi-open-source/cli/internal/gitx"
)

func cmdInit(args []string) error {
	flags := flag.NewFlagSet("init", flag.ContinueOnError)

	// Flag errors and help go to stderr, keeping stdout for results.
	flags.SetOutput(os.Stderr)

	mirror := flags.String("mirror", "", "URL of the public mirror repository (required)")

	name := flags.String("name", "", "project name (default: the mirror repository's name)")

	branch := flags.String("branch", "", "branch to publish (default: the current branch)")

	flags.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: sos init --mirror <url> [--name <name>] [--branch <branch>]")
		flags.PrintDefaults()
	}

	if err := flags.Parse(args); err != nil {
		return err
	}

	// Parse stops at the first non-flag word, so refuse it instead of ignoring it.
	if flags.NArg() > 0 {
		flags.Usage()
		return fmt.Errorf("init takes no arguments, got %q", flags.Arg(0))
	}

	// A missing --mirror is a usage error, never a prompt a script could hang on.
	if *mirror == "" {
		flags.Usage()

		return fmt.Errorf("--mirror is required")
	}

	cfg := config.Config{Mirror: *mirror}

	cfg.Name = *name

	// No --name, so take the name from the mirror's URL.
	if cfg.Name == "" {
		cfg.Name = nameFromMirror(cfg.Mirror)
	}

	// An empty name would be an empty public commit message, so refuse it.
	if cfg.Name == "" {
		return fmt.Errorf("cannot tell the project name from %s: pass --name", cfg.Mirror)
	}

	// Open after the argument checks, so a bad command line is reported first.
	repo, err := gitx.Open(".")

	if err != nil {
		return err
	}

	// Refuse a second init with a clear message, before anything is written.
	if config.Exists(repo.Dir) {
		return fmt.Errorf("%s/%s already exists: this repository is already initialized",
			config.Dir, config.File)
	}

	if *branch == "" {
		// No --branch, so publish the branch that is checked out.
		current, err := repo.CurrentBranch()

		// A detached HEAD has no branch to name, so point at --branch.
		if err != nil {
			return fmt.Errorf("%w: pass --branch to say which branch to publish", err)
		}

		// Store it behind the pointer, so the code below reads one variable.
		*branch = current
	}

	cfg.Branch = *branch

	if err = config.Scaffold(repo.Dir, cfg); err != nil {
		return err
	}

	fmt.Printf("Initialized sos in %s/\n", config.Dir)
	fmt.Printf("  mirror: %s\n", cfg.Mirror)
	fmt.Printf("  name:   %s\n", cfg.Name)
	fmt.Printf("  branch: %s\n", cfg.Branch)
	fmt.Printf("\nReview %s, then commit %s/: nothing publishes until you do.\n",
		config.IncludePath, config.Dir)
	return nil
}

// nameFromMirror takes the project name from the mirror's URL: its last path
// element without ".git". https://host/you/project.git and
// git@host:you/project.git both give "project".
func nameFromMirror(mirror string) string {
	// Drop a trailing / so project.git/ still gives project.
	mirror = strings.TrimSuffix(mirror, "/")

	// The name follows the last / or :, so https:// and git@host: URLs both work.
	i := strings.LastIndexAny(mirror, "/:")

	// With no / or :, i is -1 and this keeps the whole string.
	last := mirror[i+1:]

	// Drop the .git that clone URLs usually end in.
	return strings.TrimSuffix(last, ".git")
}
