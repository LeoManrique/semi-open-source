# sos — Design

## Why

`sos` is short for semi-open-source, but also a way to help developers publish their work without it being completely stolen or slop-forked.

## What the public sees

The initial version will only publish a single commit, including the selected files only, the original authorship and the project's name as its message. A new publish will replace the single commit (until a better design is implemented).

Only the committed files from the branch are published, uncommitted modifications are ignored.

Nothing is public unless it is explicitely allowed in `.sos/include`. `.sos/` is never published in the public repo, even when `.sos/include` is `*`.

## When sos refuses

`sos` should actively help the developer not leak anything that is not intended to be published.

`sos` will also refuse to publish under these conditions:
- The local repo's branch to be published does not match the private remote's content.
- A public mirror is configured as a remote of the private repo.
- Sensitive data, such as passwords or private keys, is about to be published to the public mirror.
- The repo contains submodules, Git-LFS files and symlinks pointing outside the repository.
