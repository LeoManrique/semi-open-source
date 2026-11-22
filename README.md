# sos: semi-open-source

CLI to make your projects source available without exposing all your development internals.

## Status

Work in progress

## Install

Script:
```
curl -fsSL https://raw.githubusercontent.com/LeoManrique/semi-open-source/main/install.sh | sh
```
Go:
```
go install github.com/LeoManrique/semi-open-source/cli/cmd/sos@latest
```

Or download the latest [release](https://github.com/LeoManrique/semi-open-source/releases/latest)


## Build from source

`just build` generates the binary at repo root.

## Quick start

```sh
# Step 1: initialize sos project
sos init --mirror git@github.com:you/repo.git
# Step 2: Configure files under .sos/ folder
# Step 3: Commit .sos/ folder
git add .sos/* && git commit -m "initialize sos project"
# Step 4: Publish sos project
sos publish
```

## Learn more

- [DESIGN.md](DESIGN.md): how sos behaves, and why.
- [TECHNICAL.md](TECHNICAL.md): how it works inside.
- [ROADMAP.md](ROADMAP.md): what is being built, and what comes next.

## License

Apache License 2.0, see [LICENSE](LICENSE).
