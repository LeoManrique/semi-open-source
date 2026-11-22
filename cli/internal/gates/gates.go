// Package gates inspects the public commit before it is pushed, and refuses
// what must not be published.
package gates

import "fmt"

// Finding is one thing in the public commit that must not be published.
type Finding struct {
	Where       string // a published path, or the project name
	Line        int    // 1-based; 0 when the finding is not about one line
	Detail      string
	Fingerprint string // what .sos/secret-allow lists to let it through; "" when nothing can
}

// String renders a finding as "where:line: detail [fingerprint]", leaving out
// the line and the fingerprint when there are none.
func (f Finding) String() string {
	location := f.Where

	// Line 0 means the finding is about a whole file, so leave the line out.
	if f.Line > 0 {
		location = fmt.Sprintf("%s:%d", f.Where, f.Line)
	}

	// No fingerprint means nothing can let the finding through, so print none.
	if f.Fingerprint != "" {
		return fmt.Sprintf("%s: %s [%s]", location, f.Detail, f.Fingerprint)
	}

	return location + ": " + f.Detail
}
