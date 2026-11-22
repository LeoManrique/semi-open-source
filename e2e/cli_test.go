package e2e

import (
	"testing"

	"github.com/LeoManrique/semi-open-source/e2e/harness"
)

// An unrecognized verb is always an error that names itself, never a quiet
// no-op: exit 2 keeps "you invoked me wrong" apart from the exit 1 that means
// sos refused on purpose.
func TestUnknownCommand_ExitsTwo(t *testing.T) {
	repo := harness.NewRepo(t)

	repo.SOS("bogus").FailsUsage().Says("unknown command")
}
