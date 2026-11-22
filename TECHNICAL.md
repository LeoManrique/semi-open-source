# sos — Technical Reference

This document covers how is the project designed from a technical perspective.

## Requirements

`sos` assumes that the following CLI tool is already installed:

- `git`: >=2.54

_Older versions might work but are not officially supported_

It also assumes that all the required git credentials are already set up.

## sos project

### `.sos/`

Folder that includes the configuration for a semi-open-source project.

### `.sos/include`

Following a format similar to .gitignore, states which files and folders will be included in the source available version.

Later matching lines decide if a specific file or directory stays public or private, even for already excluded parent directories.

To exclude specific patterns use  a "!" prefix.

### `.sos/config.toml`

Set additional configuration about how to publish the repository, like the branch to be published, the mirror, and the project name.

## How a publish works

The files included by the configuration are written into a new git tree, leaving excluded files untouched..
