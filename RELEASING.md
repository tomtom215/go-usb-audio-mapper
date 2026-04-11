<!-- SPDX-License-Identifier: MIT -->
<!-- Copyright 2025 Tom F. (https://github.com/tomtom215) -->

# Release Process

## Prerequisites

- Commit access to `main`
- All CI checks passing
- Go 1.22+ installed locally
- [GoReleaser](https://goreleaser.com/) installed (for local testing)

## Release Checklist

### 1. Prepare the Release

```bash
# Create a release branch
git checkout -b release/vX.Y.Z

# Update version in config.go
# AppVersion = "X.Y.Z"

# Update CHANGELOG.md
# Move [Unreleased] items to [X.Y.Z] with today's date

# Run full test suite
make all
```

### 2. Verify Quality Gates

```bash
# All of these must pass
go vet ./...
gofmt -l .                    # Must produce no output
go test -race -count=1 ./...  # All tests pass
go build -o /dev/null ./...   # Clean build
```

### 3. Merge to Main

```bash
git add -A
git commit -m "Release vX.Y.Z"
git push origin release/vX.Y.Z
# Open PR, get review, merge to main
```

### 4. Tag and Release

```bash
git checkout main
git pull origin main
git tag -a vX.Y.Z -m "Release vX.Y.Z"
git push origin vX.Y.Z
```

Pushing the tag triggers the GitHub Actions release workflow which:

1. Validates the tag format
2. Runs the full CI matrix
3. Builds multi-arch binaries (amd64, arm64, armv7)
4. Creates a GitHub Release with binaries and checksums
5. Generates release notes from CHANGELOG.md

### 5. Verify

- Check the [Releases page](https://github.com/tomtom215/go-usb-audio-mapper/releases)
- Verify binaries are attached
- Verify release notes are correct
- Download and test the binary on a target system

## Versioning

This project follows [Semantic Versioning 2.0.0](https://semver.org/):

- **MAJOR**: Incompatible CLI or behavior changes
- **MINOR**: New features, backward-compatible
- **PATCH**: Bug fixes, backward-compatible

## Hotfix Process

For critical security fixes:

1. Branch from the release tag: `git checkout -b hotfix/vX.Y.Z+1 vX.Y.Z`
2. Apply the fix
3. Follow steps 2-5 above with an expedited review
