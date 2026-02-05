---
name: fix
description: Run linting and formatting, then spawn parallel agents to fix all issues
---

# Go Project Code Quality Check

This command runs all linting and formatting tools for this Go project, collects errors, and spawns parallel agents to fix them.

## Step 1: Run Linting and Formatting Checks

Run the following commands to check for issues:

```bash
# Run go vet for static analysis
go vet ./...

# Run golangci-lint with the project's configuration
golangci-lint run ./... 2>&1 || true

# Check formatting
gofmt -l .
```

Collect all output from these commands.

## Step 2: Parse and Categorize Errors

Parse the output and group errors into domains:

- **Vet errors**: Issues from `go vet` (shadows, printf issues, etc.)
- **Lint errors**: Issues from golangci-lint linters (errcheck, staticcheck, gosimple, etc.)
- **Format errors**: Files that need formatting from `gofmt -l`

Create a list of all files with issues and the specific problems in each file.

## Step 3: Spawn Parallel Agents to Fix Issues

If there are issues to fix, spawn agents in parallel using the Task tool. Use a SINGLE response with MULTIPLE Task tool calls:

1. **format-fixer agent**: Fix all formatting issues
   - Run `gofmt -s -w .` to auto-format
   - Run `goimports -w -local github.com/unstablemind/pocket .` to fix imports

2. **lint-fixer agent**: Fix lint errors that can be auto-fixed
   - Address errcheck issues (handle returned errors)
   - Fix gosimple suggestions
   - Address staticcheck issues
   - Fix any other linter warnings

3. **vet-fixer agent**: Fix go vet issues
   - Fix shadowed variables
   - Fix printf format issues
   - Address other vet warnings

Each agent should:
1. Receive the specific list of files and errors in their domain
2. Fix all issues in their domain
3. Run the relevant check to verify fixes
4. Report what was fixed

## Step 4: Verify All Fixes

After all agents complete, run the full check again:

```bash
go vet ./...
golangci-lint run ./...
gofmt -l .
```

Report success if all checks pass, or list any remaining issues.
