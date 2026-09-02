# Updating Go Tools

This guide explains how to keep the Go tools in `gotools/` up to date.

## Quick Reference

```bash
# List all installed tools
scripts/gotools-manage list

# Update a specific tool to latest
scripts/gotools-manage update controller-gen

# Reinitialize a tool (clean dependencies, keep version)
scripts/gotools-manage reinit controller-gen

# Reinitialize and upgrade to latest
scripts/gotools-manage reinit controller-gen --latest
```

## When to Use Each Command

### `update` - Upgrade to a New Version

Use when you want to upgrade a tool to a newer version:

```bash
# Update to latest
scripts/gotools-manage update controller-gen

# Update to specific version
scripts/gotools-manage update controller-gen v0.22.0
```

This modifies the existing `go.mod` and `go.sum` incrementally.

### `reinit` - Clean Dependencies

Use when you want to clean up the `go.mod` and `go.sum` files by starting fresh:

```bash
# Reinitialize with current version (recommended for cleanup)
scripts/gotools-manage reinit controller-gen

# Reinitialize AND upgrade to latest
scripts/gotools-manage reinit controller-gen --latest
```

The `reinit` command:
1. Captures the current version (unless `--latest` is used)
2. Removes the tool directory completely
3. Recreates it from scratch with a fresh `go.mod`
4. Reinstalls the tool at the specified version
5. Syncs go/toolchain versions with main `go.mod`
6. Runs `go mod tidy`

This is particularly useful when:
- Dependencies have accumulated cruft over time
- You want a clean slate for the `go.sum` file
- The `go.mod` has indirect dependencies you don't need

## Workflow for Updating All Tools

To update all tools and clean up their dependencies:

```bash
# List tools to see what you have
scripts/gotools-manage list

# For each tool, reinitialize with latest version
scripts/gotools-manage reinit controller-gen --latest
scripts/gotools-manage reinit mockgen --latest

# Verify everything is in sync
scripts/gotools-validate
```

Or if you want to keep current versions but clean dependencies:

```bash
scripts/gotools-manage reinit controller-gen
scripts/gotools-manage reinit mockgen
scripts/gotools-validate
```

## Adding New Tools

```bash
# If the tool is known (controller-gen, mockgen)
scripts/gotools-manage init <tool-name>

# For custom tools
scripts/gotools-manage init <tool-name> <module-path>[@version]

# Example
scripts/gotools-manage init golangci-lint github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

## Keeping Go Versions in Sync

After updating tools, ensure the `go` and `toolchain` versions match the main `go.mod`:

```bash
# Sync versions from main go.mod
scripts/gotools-sync

# Validate everything is in sync
scripts/gotools-validate
```

Note: `gotools-manage` automatically syncs versions during `init`, `update`, and `reinit` operations.
