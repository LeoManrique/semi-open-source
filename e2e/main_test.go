// Package e2e holds blackbox tests: they run the compiled sos binary against
// throwaway git repositories and assert on what a user would observe.
package e2e

import (
	"testing"

	"github.com/LeoManrique/semi-open-source/e2e/harness"
)

// Go calls TestMain instead of running the tests directly, and only in the
// test package itself, so this hands the run to the harness, which locates
// the binary before any test starts.
func TestMain(m *testing.M) {
	harness.Main(m)
}
