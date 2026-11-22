package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/LeoManrique/semi-open-source/cli/internal/refuse"
)

// version is stamped in by the linker: `just build` passes -X main.version.
// It stays empty in a build nobody stamped, such as one made by go install.
var version = ""

const usage = `sos: publish a redacted public mirror of a private repository.

usage:
  sos <command> [arguments]

commands:
  init      scaffold .sos/ and record the mirror
  match     evaluate paths against .sos/include
  publish   push the filtered branch tip to the mirror
  check     run the publish checks without pushing
  diff      show what the next publish would change on the mirror
  version   print the sos version
  help      print this message`

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	// No command given, so print the usage and exit 2.
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, usage)
		return 2
	}

	name := args[0]
	// Accept --version as an alias, but version is the real command.
	if name == "--version" {
		name = "version"
	}

	var err error
	switch name {
	case "init":
		err = cmdInit(args[1:])
	case "match":
		err = cmdMatch(args[1:])
	case "publish":
		err = cmdPublish(args[1:])
	case "check":
		err = cmdCheck(args[1:])
	case "diff":
		err = cmdDiff(args[1:])
	case "version":
		fmt.Println("sos", buildVersion())
	case "help", "-h", "--help":
		fmt.Println(usage)
	default:
		fmt.Fprintf(os.Stderr, "sos: unknown command %q\n\n%s\n", name, usage)
		return 2
	}

	return report(err)
}

func report(err error) int {
	if err == nil {
		return 0
	}

	// sos refused on purpose, so exit 1 and list each reason.
	if r, ok := errors.AsType[*refuse.Error](err); ok {
		fmt.Fprintf(os.Stderr, "sos: %s\n", r.Msg)
		for _, d := range r.Details {
			fmt.Fprintf(os.Stderr, "  %s\n", d)
		}
		return 1
	}

	// Anything else is a usage or environment error, so exit 2.
	fmt.Fprintf(os.Stderr, "sos: %v\n", err)
	return 2
}
