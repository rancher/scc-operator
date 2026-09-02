# Gotools

This directory contains Go-based tools to use with [go
tool](https://tip.golang.org/doc/modules/managing-dependencies#tools).

Each tool is within its own directory with its own `go.mod` file to avoid
dependency conflicts.

## Keeping go versions in sync

To keep the `go` and `toolchain` versions synchronized between the main `go.mod` and all tool modules:

```sh
# Sync versions from main go.mod to all gotools
scripts/gotools-sync

# Validate that all versions are in sync
scripts/gotools-validate
```

## Managing tools

Use `scripts/gotools-manage` to manage tools. Run `scripts/gotools-manage --help` for details.

**Common tasks:**

```sh
# List tools
scripts/gotools-manage list

# Update CVE patches (golang.org/x/* only for controller-gen)
scripts/gotools-update-all-deps

# Update tool version
scripts/gotools-manage update mockgen v0.7.0

# Add new tool
scripts/gotools-manage init <name> <module>[@version]
```

**Update dependencies for CVE patches:**

```sh
# All tools at once
scripts/gotools-update-all-deps

# Individual tool (controller-gen: golang.org/x/* only by default)
scripts/gotools-manage update-deps controller-gen

# Include k8s.io/* for controller-gen (may change codegen behavior)
scripts/gotools-manage update-deps controller-gen --all
```

**Using tools:**

```sh
go tool -modfile gotools/controller-gen/go.mod controller-gen -h
```
