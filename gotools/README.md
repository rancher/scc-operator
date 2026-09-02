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

The `scripts/gotools-manage` script provides a convenient wrapper for managing Go tools with isolated `go.mod` files.

**List installed tools**

```sh
scripts/gotools-manage list
```

**Add a new tool**

```sh
# Using known tool definitions (controller-gen, mockgen)
scripts/gotools-manage init controller-gen

# With specific version
scripts/gotools-manage init controller-gen sigs.k8s.io/controller-tools/cmd/controller-gen@v0.21.0

# Custom tool with module path
scripts/gotools-manage init mytool example.com/path/to/tool@v1.0.0
```

**Update a tool to a new version**

```sh
# Update to latest version
scripts/gotools-manage update controller-gen

# Update to specific version
scripts/gotools-manage update controller-gen v0.22.0
```

**Reinitialize a tool (clean dependencies)**

This is useful when you want to clean up the `go.mod`/`go.sum` files and regenerate them:

```sh
# Reinitialize with current version (clean deps)
scripts/gotools-manage reinit controller-gen

# Reinitialize and update to latest version
scripts/gotools-manage reinit controller-gen --latest
```

**Remove a tool**

```sh
scripts/gotools-manage clean controller-gen
```

**Using a tool**

```sh
go tool -modfile <path to modfile> <tool>
```

For example, to use controller-gen:

```sh
go tool -modfile gotools/controller-gen/go.mod controller-gen -h
```

**Check tool version**

```sh
# Check installed version
go run -modfile=gotools/controller-gen/go.mod sigs.k8s.io/controller-tools/cmd/controller-gen --version
go run -modfile=gotools/mockgen/go.mod go.uber.org/mock/mockgen -version

# List available versions
go list -m -versions sigs.k8s.io/controller-tools
go list -m -versions go.uber.org/mock
```
