# sos — all build/test workflows go through these recipes.
#
# Two Go modules: cli/ is the application, e2e/ is the blackbox test suite.
# Every recipe cd's into the one it needs; nothing is run from the repo root.

# The version stamped into binaries: the current release tag, else the commit,
# else dev. Only tags starting with v are releases; cli/v… tags are the Go
# module's, and belong on the public mirror.
version := `git describe --tags --match 'v*' --always --dirty 2>/dev/null || echo dev`

# List available recipes
default:
    @just --list

# Build the sos binary at the repo root
build:
    cd cli && go build -ldflags "-X main.version={{version}}" -o ../sos ./cmd/sos

# Build with an explicit version stamped in, to a given path
build-stamped version out:
    cd cli && go build -ldflags "-X main.version={{version}}" -o {{out}} ./cmd/sos

# Print the version that would be stamped
show-version:
    @echo {{version}}

# Format and vet both modules
fmt:
    cd cli && gofmt -l -w . && go vet ./...
    cd e2e && gofmt -l -w . && go vet ./...

# `gofmt -w` rewrites in place and exits 0 whether or not it changed something,
# so the `fmt` recipe can never fail a build. This recipe reports instead of
# fixing.
#
# Gate formatting and vet without rewriting anything
fmt-check:
    #!/usr/bin/env bash
    set -euo pipefail
    unformatted="$( (cd cli && gofmt -l .) ; (cd e2e && gofmt -l .) )"
    if [ -n "$unformatted" ]; then
      echo "not gofmt'd:" >&2
      echo "$unformatted" >&2
      exit 1
    fi
    cd cli && go vet ./...
    cd ../e2e && go vet ./...

# `just build` only compiles what cmd/sos imports. A package nothing imports
# yet (gitx before any verb uses it) is only type-checked by ./..., and with
# several packages go build writes no binary, only errors.
#
# Compile every package in the application module
compile:
    cd cli && go build ./...

# -count=1 is required, not stylistic. The binary under test lives at the repo
# root, outside this module, so `go test` cannot see it change and will happily
# re-serve a cached PASS for a binary that would now fail.
#
# Blackbox e2e tests — build first; they exercise the compiled binary
e2e: build
    cd e2e && go test ./... -count=1

# Run one e2e test by name (e.g. just e2e-one TestPublish_FirstPublishCreatesOneCommit)
e2e-one name: build
    cd e2e && go test ./... -count=1 -run '{{name}}' -v

# -artifacts keeps every directory a test made under t.ArtifactDir, in
# _artifacts/ below -outputdir. That is the repo root, outside both modules, so
# gofmt and go vet never walk into a kept fixture. The last run's are cleared
# first, so only this run's repositories are there.
#
# Run one e2e test by name and keep its repositories in _artifacts/
e2e-keep name: build
    rm -rf _artifacts
    cd e2e && go test ./... -count=1 -run '{{name}}' -v -artifacts -outputdir '{{justfile_directory()}}'

# Add a Go dependency to the application module
add pkg:
    cd cli && go get {{pkg}} && go mod tidy

# Add a Go dependency to the e2e test module
e2e-add pkg:
    cd e2e && go get {{pkg}} && go mod tidy

# Reconcile go.mod/go.sum with the imports actually in the code
tidy:
    cd cli && go mod tidy
    cd e2e && go mod tidy

# -trimpath and -buildvcs=false keep this machine's paths and the private
# repository's commit hash out of binaries that are published.
#
# Cross-compile release binaries and checksums into dist/
release:
    #!/usr/bin/env bash
    set -euo pipefail
    version="{{version}}"
    rm -rf dist && mkdir -p dist

    for target in darwin/arm64 darwin/amd64 linux/amd64 linux/arm64; do
      os="${target%/*}"; arch="${target#*/}"
      out="../dist/sos_${version}_${os}_${arch}"
      echo "building sos_${version}_${os}_${arch}"
      (cd cli && GOOS="$os" GOARCH="$arch" CGO_ENABLED=0 \
        go build -trimpath -buildvcs=false -ldflags "-s -w -X main.version=${version}" -o "$out" ./cmd/sos)
    done

    cd dist && shasum -a 256 sos_* > SHA256SUMS
    echo "built $(ls sos_* | wc -l | tr -d ' ') binaries"
