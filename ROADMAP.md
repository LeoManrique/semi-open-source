# sos — Roadmap

### v0.1 — one snapshot commit, published end to end
- [x] 01 Toolchain and skeleton
- [x] 02 gitx I: running git
- [x] 03 gitx II: reading objects
- [x] 04 gitx III: writing objects and remotes
- [x] 05 Harness I: throwaway repos
- [x] 06 Harness II: running sos
- [x] 07 Matcher I: `sos match`, literal names
- [x] 08 Matcher II: globs, classes, `**`
- [x] 09 Matcher III: anchoring, directories, negation, `.sos`, `-v`
- [x] 10 `sos init` I: scaffold and name
- [x] 11 `sos init` II: branch, second init, reading config
- [x] 12 The snapshot: the mirror fixture, the filtered tip tree
- [x] 13 Publishing the tip
- [x] 14 Republishing replaces the commit
- [x] 15 The mirror is never a remote
- [x] 16 The private remote
- [x] 17 Secret gate
- [x] 18 Content gates and `sos check`
- [x] 19 `sos diff`
- [ ] 20 Release and install

### v0.2 — real history
- [ ] Per-commit replay from the last snapshot
- [ ] Mapping cache in `.git/sos/`
- [ ] Mirror verification with divergence reporting
- [ ] `sos repair`

### v0.3 — squash and status
- [ ] `sos squash`
- [ ] `sos status`

### v0.4 — adoption
- [ ] `sos adopt`

### Later
- [ ] Uniform committer identity (`[committer]` in the config)
- [ ] Warnings: commit-message lint, published content referencing excluded paths, `sos check --strict`
- [ ] Warn when a `!` line in `.sos/include` is overridden by a later line that includes the same paths
- [ ] Identity-aware mirror comparison
- [ ] Restore full test coverage: refusal edge cases, unit tests
- [ ] Tests for `install.sh`: its checksum refusal and its `sosx` branch
- [ ] Release tags pushed by sos, and multiple branches
- [ ] Retroactive redaction of published history
- [ ] `--with-history`: publish a newly included path's full private history
- [ ] Author-identity privacy mapping
- [ ] Custom secret-scan rules
- [ ] Contribution backflow: public PR to private patch across the redaction boundary
