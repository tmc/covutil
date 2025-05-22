# Syncing to Upstream Go Coverage

This document describes how to sync the internal/coverage package with upstream Go's implementation.

## Overview

The `internal/coverage` directory is maintained as a git subtree from the upstream Go repository. This allows us to track changes and updates from Go's internal coverage implementation while maintaining our own modifications for import path compatibility.

## Initial Setup (Already Done)

The initial setup was completed using git subtree commands documented in git notes. To view the complete setup history:

```bash
git notes show HEAD
```

## Updating from Upstream

To update the coverage package from upstream Go:

### 1. Prepare the Update

```bash
# Ensure golang remote exists and is up to date
git remote -v | grep golang || git remote add golang https://github.com/golang/go.git --no-tags
git fetch golang --no-tags
```

### 2. Update the Subtree

```bash
# Update the subtree from upstream
git subtree pull --prefix=internal/coverage golang master --squash -m "Update coverage package from upstream Go"
```

### 3. Fix Import Paths

After updating, all import paths need to be rewritten to use our module path:

```bash
# Install fiximports if not available
go install golang.org/x/tools/cmd/fiximports@latest

# Fix base import paths
fiximports -w -r 'internal/coverage -> github.com/tmc/covutil/internal/coverage' internal/coverage/

# Fix wildcard imports
fiximports -w -r 'internal/coverage/... -> github.com/tmc/covutil/internal/coverage/...' internal/coverage/

# Fix specific runtime imports that may be introduced
fiximports -w -r 'internal/runtime/atomic -> sync/atomic' internal/coverage/
```

### 4. Handle Missing Dependencies

Check for any new internal package dependencies that may have been introduced:

```bash
# Build to identify missing dependencies
go build ./...
```

If new internal packages are referenced, create stub implementations in `internal/runtime/` or find suitable standard library replacements.

### 5. Update Public API

Review changes in `internal/coverage/defs.go` and other core files to ensure the public API in `coverage/package.go` is up to date:

```bash
# Check for new exported types/constants that should be re-exported
grep -n "^type\|^const\|^var" internal/coverage/defs.go
```

### 6. Test the Update

```bash
# Ensure everything builds
go build ./...

# Run tests
go test ./...
```

## Troubleshooting

### Import Path Issues

If you encounter import path errors after an update:

1. Use `fiximports` with the appropriate rewrite rules
2. Check for new internal package dependencies
3. Ensure overlay files in `internal/testprogram/overlays/` use correct import paths

### Build Failures

Common issues after upstream updates:

1. **New internal dependencies**: Create stub implementations or find standard library alternatives
2. **API changes**: Update the public `coverage/package.go` to match new internal APIs
3. **Test failures**: May require updating test data or test expectations

### Git Subtree Issues

If git subtree commands fail:

1. Ensure the golang remote is properly configured
2. Try using `--strategy=subtree` option
3. Consider manual merge if automatic subtree merge fails

## Files That Require Manual Attention

After each update, review these files for potential manual fixes:

- `internal/coverage/cfile/hooks.go` - Uses our custom exithook implementation
- `internal/testprogram/overlays/runtime/coverage/coverage.go` - May need import path fixes
- `coverage/package.go` - Public API re-exports may need updates

## Documentation

- Git subtree commands are documented in git notes on the HEAD commit
- This document should be updated when new update procedures are discovered
- Keep track of any custom modifications we make that need to be preserved across updates