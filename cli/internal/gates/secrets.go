package gates

import (
	"context"
	"fmt"
	"regexp"

	"github.com/betterleaks/betterleaks/v2/config"
	"github.com/betterleaks/betterleaks/v2/report"
	"github.com/betterleaks/betterleaks/v2/scan"
	"github.com/betterleaks/betterleaks/v2/sources"
)

// binaryFormats are the only paths sos does not scan: formats where a regex
// hit is always noise. Everything else, lockfiles, vendored code and go.sum
// included, is published, so it is scanned.
var binaryFormats = regexp.MustCompile(
	`(?i)\.(?:bmp|gif|jpe?g|png|svg|tiff?|eot|[ot]tf|woff2?|pdf|bin|exe|dll|gem)$`)

// scanner finds secrets with Betterleaks' built-in rules.
type scanner struct {
	engine *scan.Scanner
}

// text is content sos already holds, found at where, as a Betterleaks source.
// Betterleaks' own file sources would read the working tree and skip the
// paths its prefilter lists; text does neither.
type text struct {
	where   string
	content string
}

// Fragments hands Betterleaks the whole content as one fragment.
func (t text) Fragments(_ context.Context, yield sources.FragmentsFunc) error {
	// The path lets path-aware rules match, and line numbers count from the first line.
	return yield(sources.Fragment{Raw: t.content, StartLine: 1, Attributes: map[string]string{sources.AttrPath: t.where}}, nil)
}

func newScanner() (*scanner, error) {
	cfg, err := config.Default()

	if err != nil {
		return nil, fmt.Errorf("loading secret-scanning rules: %w", err)
	}

	// The default prefilter skips lockfiles, go.sum and vendor/, which sos
	// publishes. Clear it in case a later Betterleaks applies it here too.
	cfg.PrefilterExpr = ""

	// The default filter drops values that look like placeholders. A secret
	// that looks like one would be published, so sos keeps every match.
	cfg.FilterExpr = ""

	// An inline betterleaks:allow or gitleaks:allow comment would itself be
	// published. Exceptions belong in .sos/secret-allow, where someone reviews them.
	engine, err := scan.New(cfg, scan.WithAllowSignatures())

	if err != nil {
		return nil, fmt.Errorf("compiling secret-scanning rules: %w", err)
	}
	return &scanner{engine: engine}, nil
}

// file scans one published file. Its findings are reviewable: a false alarm
// in a file is let through by its fingerprint.
func (s *scanner) file(path string, content []byte) ([]Finding, error) {
	// A regex hit in an image, a font or a compiled binary is always noise.
	if binaryFormats.MatchString(path) {
		return nil, nil
	}

	// Reviewable, so each finding gets a fingerprint.
	return s.findings(path, string(content), true)
}

// findings runs the scanner over content, found at where, and turns what it
// reports into findings. A reviewable finding gets a fingerprint, the line
// .sos/secret-allow lists to let it through. Every finding's detail ends with
// how to clear it.
func (s *scanner) findings(where, content string, reviewable bool) ([]Finding, error) {
	var out []Finding

	// Betterleaks calls keep once per match, never two at a time.
	keep := func(f report.Finding) error {
		// The rule's description, never f.Match, which holds the secret itself.
		finding := Finding{Where: where, Line: f.Location.StartLine, Detail: f.Description}

		// f.Match.Fingerprint hashes the secret alone, so make sos's own, which names where it is.
		if reviewable {
			finding.Fingerprint = fmt.Sprintf("%s:%s:%d", where, f.RuleID, finding.Line)

			// A false alarm in a file is let through once someone reviews it.
			finding.Detail += "; remove it, or once reviewed commit its fingerprint to .sos/secret-allow"
		} else {
			// Nothing lets a secret in the project name through, so the name must change.
			finding.Detail += "; rename the project"
		}

		out = append(out, finding)

		return nil
	}

	_, err := s.engine.Scan(context.Background(), text{where: where, content: content}, keep)

	if err != nil {
		return nil, fmt.Errorf("scanning %s for secrets: %w", where, err)
	}
	return out, nil
}

// messageWhere names the public commit's message in a finding: it is the
// project name, from .sos/config.toml.
const messageWhere = "the project name in .sos/config.toml"

// message scans the public commit's message. Its findings are not
// reviewable: a project name holding a secret is renamed, not let through.
func (s *scanner) message(msg string) ([]Finding, error) {
	// Not reviewable, so no finding gets a fingerprint.
	return s.findings(messageWhere, msg, false)
}
