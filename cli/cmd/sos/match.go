package main

import (
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/LeoManrique/semi-open-source/cli/internal/config"
	"github.com/LeoManrique/semi-open-source/cli/internal/gitx"
	"github.com/LeoManrique/semi-open-source/cli/internal/matcher"
	"github.com/LeoManrique/semi-open-source/cli/internal/refuse"
)

// loadWorkingAllowlist reads .sos/include from the working tree.
func loadWorkingAllowlist() (*matcher.Matcher, error) {
	repo, err := gitx.Open(".")

	if err != nil {
		return nil, err
	}

	// Read through a root so a .sos symlink can't point outside the repo.
	root, err := os.OpenRoot(repo.Dir)

	if err != nil {
		return nil, err
	}

	defer func() {
		_ = root.Close()
	}()

	content, err := root.ReadFile(config.IncludePath)

	// No allowlist yet, so say that instead of a raw file-not-found error.
	if errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("no %s in this repository", config.IncludePath)
	}

	if err != nil {
		return nil, err
	}

	m, err := matcher.Parse(content)

	if err != nil {
		// A line sos can't read is a refusal, so the run stops with exit 1.
		return nil, refuse.New(err.Error())
	}
	return m, nil
}

// normalizePath turns user input into a git tree path: no ./, no leading /.
func normalizePath(p string) string {
	// Use / on every OS and clean up ./, .. and extra slashes.
	p = path.Clean(filepath.ToSlash(p))
	// Tree paths never start with /, so /a.txt is the same as a.txt.
	return strings.TrimPrefix(p, "/")
}

func cmdMatch(args []string) error {
	// Not named fs, which would hide the io/fs import.
	flags := flag.NewFlagSet("match", flag.ContinueOnError)

	// Flag errors and help go to stderr, keeping stdout for results.
	flags.SetOutput(os.Stderr)

	// A pointer, filled in by Parse, so read it as *verbose afterwards.
	verbose := flags.Bool("v", false, "show the pattern that decided each path")

	flags.Usage = func() {
		fmt.Fprintln(os.Stderr,
			"usage: sos match [-v] <path>...\n\nEvaluate paths against .sos/include in the working tree.")
		flags.PrintDefaults()
	}

	if err := flags.Parse(args); err != nil {
		return err
	}

	paths := flags.Args()

	if len(paths) == 0 {
		flags.Usage()
		return fmt.Errorf("no paths given")
	}

	// Load after the argument checks, so a bad command line is reported first.
	m, err := loadWorkingAllowlist()

	if err != nil {
		return err
	}

	for _, arg := range paths {
		p := normalizePath(arg)

		// A path outside the repo has no verdict, so it's a usage error.
		if !fs.ValidPath(p) || p == "." {
			return fmt.Errorf("%s is not a path inside the repository", arg)
		}

		d := m.Decide(p)

		// Every path starts excluded until the allowlist includes it.
		verdict := "exclude"

		if d.Included {
			verdict = "include"
		}

		if *verbose {
			// Line is 0 when no pattern matched, so start with that reason.
			reason := "(no pattern matched)"

			// Name the deciding line as file:line, so an editor can jump to it.
			if d.Line > 0 {
				reason = fmt.Sprintf("%s:%d: %s",
					config.IncludePath, d.Line, d.Pattern)
			}

			// A tab puts the reason in a column of its own.
			fmt.Printf("%-8s %s\t%s\n", verdict, p, reason)
		} else {
			// Pad the verdict to 8 columns so the paths line up.
			fmt.Printf("%-8s %s\n", verdict, p)
		}
	}
	return nil
}
