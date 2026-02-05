---
name: update-app
description: Update dependencies, fix deprecations and warnings
---

# Dependency Update & Deprecation Fix

## Step 1: Check for Updates

```bash
go list -u -m all
```

## Step 2: Update Dependencies

```bash
go get -u ./...
go mod tidy
```

## Step 3: Check for Deprecations & Warnings

Run build and check output:
```bash
go build ./...
```

Read ALL output carefully. Look for:
- Deprecation warnings
- Import path changes
- Breaking API changes

## Step 4: Fix Issues

For each warning/deprecation:
1. Research the recommended replacement or fix
2. Update code/dependencies accordingly
3. Re-run build
4. Verify no warnings remain

## Step 5: Run Quality Checks

```bash
go vet ./...
go build ./...
gofmt -l .
```

Fix all errors before completing.

## Step 6: Verify Clean Build

Ensure a fresh build works:
```bash
go clean -cache
go mod download
go build ./...
```

Verify ZERO warnings/errors.
