package main

import "runtime/debug"

// buildVersion reports what this binary was built from: the stamped
// version, else the module version the go command recorded, else the commit
// the binary was built at, else dev.
func buildVersion() string {
	// A stamp was chosen on purpose, so it always wins.
	if version != "" {
		return version
	}

	info, ok := debug.ReadBuildInfo()

	// A binary built without module support carries no build info at all.
	if !ok {
		return "dev"
	}

	// go install records the tag it fetched. A local build says (devel).
	if v := info.Main.Version; v != "" && v != "(devel)" {
		return v
	}

	// A local build records its commit and whether the tree had changes.
	var revision, dirty string
	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			revision = setting.Value
		case "vcs.modified":
			// vcs.modified is "true" when the tree had uncommitted changes.
			if setting.Value == "true" {
				dirty = "-dirty"
			}
		}
	}

	// Cut the commit to 12 characters, like a short hash. min keeps a
	// shorter value from making the slice panic.
	if revision != "" {
		return revision[:min(len(revision), 12)] + dirty
	}

	// No stamp, no module version and no commit, so fall back to dev.
	return "dev"
}
