// Package matcher implements the .sos/include allowlist: gitignore pattern
// syntax, read as a list of things that MAY be published.
package matcher

import (
	"fmt"
	"io/fs"
	"path"
	"slices"
	"strings"

	"github.com/LeoManrique/semi-open-source/cli/internal/config"
)

// Matcher decides whether tree paths may be published.
type Matcher struct {
	patterns []pattern
}

// Decision is one path's fate, and why.
type Decision struct {
	Included bool
	Pattern  string // the deciding pattern as written, empty if none matched
	Line     int    // its 1-based line number, 0 if none matched
}

type pattern struct {
	text     string   // as written, for error messages and sos match -v
	line     int      // 1-based
	negated  bool     // leading !
	dirOnly  bool     // trailing /
	anchored bool     // contains a /: matches only from the repo root
	segments []string // split on /
}

// translateClasses rewrites gitignore's negated character class [!abc] into
// the [^abc] form path.Match understands. Untranslated, Go reads [!abc] as
// the literal set {!, a, b, c}, so "[!x].md" would publish x.md, the one
// file its author carved out: the dangerous direction to be wrong in.
func translateClasses(seg string) string {
	var out strings.Builder
	inClass := false

	for i := 0; i < len(seg); i++ {
		c := seg[i]
		switch {
		// A backslash escapes the next byte, so copy both as they are.
		case c == '\\' && i+1 < len(seg):
			out.WriteByte(c)
			i++
			out.WriteByte(seg[i])
			continue
		// A [ opens a class unless one is already open.
		case c == '[' && !inClass:
			inClass = true
			out.WriteByte(c)
			// gitignore negates a class with [! where path.Match wants [^.
			if i+1 < len(seg) && seg[i+1] == '!' {
				out.WriteByte('^')
				i++
			}

			continue
		// A ] closes the open class.
		case c == ']' && inClass:
			inClass = false
		}
		// Every other byte, a closing ] included, is copied as it is.
		out.WriteByte(c)
	}

	return out.String()
}

// trimTrailingSpace removes trailing blanks unless the last one is escaped.
func trimTrailingSpace(s string) string {
	end := len(s)
	// Walk back over the spaces and tabs at the end.
	for end > 0 && (s[end-1] == ' ' || s[end-1] == '\t') {
		// A backslash before the blank escapes it, so it stays, as in .gitignore.
		if end >= 2 && s[end-2] == '\\' {
			break
		}
		end--
	}
	return s[:end]
}

// Parse reads an allowlist. Malformed patterns are an error, never a skipped
// line: an allowlist that doesn't mean what it says is how leaks happen.
func Parse(content []byte) (*Matcher, error) {
	m := &Matcher{}
	for i, raw := range strings.Split(string(content), "\n") {
		p, ok, err := parseLine(raw, i+1)

		if err != nil {
			return nil, err
		}

		// Blank and comment lines leave no pattern.
		if ok {
			m.patterns = append(m.patterns, p)
		}
	}
	return m, nil
}

func parseLine(raw string, line int) (pattern, bool, error) {
	// Drop the \r that Windows line endings (\r\n) leave, then trailing blanks.
	text := trimTrailingSpace(strings.TrimSuffix(raw, "\r"))

	// Ignore blank lines and # comments.
	if text == "" || strings.HasPrefix(text, "#") {
		return pattern{}, false, nil
	}

	p := pattern{text: text, line: line}

	// A leading ! negates the line, so drop it before reading the rest.
	if strings.HasPrefix(text, "!") {
		p.negated = true
		text = text[1:]
	}

	// A trailing / means directory-only. Strip it first, so docs/ matches at any depth.
	if strings.HasSuffix(text, "/") {
		p.dirOnly = true
		text = strings.TrimSuffix(text, "/")
	}

	// Nothing is left once the markers are gone, so refuse an empty pattern.
	if text == "" {
		return pattern{}, false, fmt.Errorf("%s:%d: empty pattern", config.IncludePath, line)
	}

	// A / anywhere anchors the pattern to the root, as in .gitignore.
	p.anchored = strings.Contains(text, "/")

	// The leading / was only a marker, and tree paths never start with /.
	text = strings.TrimPrefix(text, "/")

	// Split on /, so each segment is matched on its own.
	p.segments = strings.Split(text, "/")

	for i, seg := range p.segments {
		// A doubled slash leaves an empty segment no path has, so refuse the line.
		if seg == "" {
			return pattern{}, false, fmt.Errorf("%s:%d: %q has an empty path segment",
				config.IncludePath, line, p.text)
		}

		// Translate before the check, so path.Match only ever sees [^.
		p.segments[i] = translateClasses(seg)

		// Match against any name, only to catch a malformed pattern while parsing.
		if _, err := path.Match(p.segments[i], "probe"); err != nil {
			return pattern{}, false,
				fmt.Errorf("%s:%d: %q is not a valid pattern: %v", config.IncludePath, line, p.text, err)
		}
	}

	return p, true, nil
}

func (p pattern) matches(candidate string) bool {
	segs := strings.Split(candidate, "/")

	// An anchored pattern matches only from the root.
	if p.anchored {
		return matchSegments(p.segments, segs)
	}

	// Floating: the pattern may start at any depth, but must reach the end.
	for i := range segs {
		if matchSegments(p.segments, segs[i:]) {
			return true
		}
	}
	return false
}

// matchSegments matches a segmented pattern against a segmented path, where
// "**" stands for any number of segments: none or more inside the pattern,
// one or more at its end.
func matchSegments(pat, name []string) bool {
	for len(pat) > 0 {
		// path.Match reads ** as a single *, so handle it here.
		if pat[0] == "**" {
			// A trailing ** needs a segment to swallow, so docs/** never matches docs.
			if len(pat) == 1 {
				return len(name) > 0
			}

			// Let ** swallow none, one, up to every segment, then match the rest.
			for i := 0; i <= len(name); i++ {
				if matchSegments(pat[1:], name[i:]) {
					return true
				}
			}

			return false
		}

		// A pattern segment is left but the path ran out, so no match.
		if len(name) == 0 {
			return false
		}

		ok, err := path.Match(pat[0], name[0])

		// Parse already caught bad patterns, but no match is the safe answer.
		if err != nil || !ok {
			return false
		}

		// Both segments matched, so move on to the next pair.
		pat, name = pat[1:], name[1:]
	}
	// The pattern ran out, so it matches only if the path ran out too.
	return len(name) == 0
}

// Include reports whether a tree path may be published.
func (m *Matcher) Include(p string) bool {
	return m.Decide(p).Included
}

// Decide reports a path's fate and the pattern responsible.
func (m *Matcher) Decide(p string) Decision {
	// Only clean tree paths, or secret/../.sos/config.toml gets past the .sos/ check.
	if !fs.ValidPath(p) || p == "." {
		return Decision{Included: false}
	}

	// Nothing under .sos/ is ever published, whatever the patterns say.
	if p == config.Dir || strings.HasPrefix(p, config.Dir+"/") {
		return Decision{Included: false}
	}

	// A pattern may match the path or any directory above it.
	candidates := ancestorsAndSelf(p)

	// The last candidate is the path itself, the others its directories.
	self := len(candidates) - 1

	// Walk from the last line up, so the first match is the last matching line, which wins.
	for _, pat := range slices.Backward(m.patterns) {
		if pat.matchesAny(candidates, self) {
			return Decision{Included: !pat.negated, Pattern: pat.text, Line: pat.line}
		}
	}
	// No pattern matched, so the path stays private.
	return Decision{}
}

// ancestorsAndSelf turns "a/b/c.md" into ["a", "a/b", "a/b/c.md"].
func ancestorsAndSelf(p string) []string {
	segs := strings.Split(p, "/")

	out := make([]string, 0, len(segs))

	for i := range segs {
		out = append(out, strings.Join(segs[:i+1], "/"))
	}
	return out
}

// matchesAny reports whether the pattern matches the path or any ancestor.
// Directory-only patterns never match the path itself (index self).
func (p pattern) matchesAny(candidates []string, self int) bool {
	for i, c := range candidates {
		// A directory pattern never matches the path itself, which is a file.
		if p.dirOnly && i == self {
			continue
		}
		if p.matches(c) {
			return true
		}
	}
	return false
}
