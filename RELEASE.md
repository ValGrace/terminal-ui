# Release Process

This document describes the release process for Command History Tracker.

## Pre-Release Checklist

- [ ] All tests pass on all platforms (Windows, macOS, Linux)
- [ ] Code is formatted and linted (`make check`)
- [ ] Documentation is up to date (README.md, API.md, INTEGRATION.md)
- [ ] CHANGELOG.md is updated with release notes
- [ ] Version number is updated in `internal/version/version.go`
- [ ] All dependencies are up to date (`go mod tidy`)
- [ ] Security vulnerabilities are addressed (`go list -m -json all | nancy sleuth`)

## Release Steps

### 1. Prepare Release Branch

```bash
# Create release branch
git checkout -b release/v0.1.0

# Update version
# Edit internal/version/version.go and set Version = "0.1.0"

# Update CHANGELOG.md
# Add release date and finalize release notes

# Commit changes
git add .
git commit -m "Prepare release v0.1.0"
git push origin release/v0.1.0
```

### 2. Run Tests

```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage

# Run linting
make vet
```

### 3. Build Release Binaries

```bash
# Clean previous builds
make clean

# Build for all platforms
make release

# Or use build scripts directly
./build.sh --all
# or on Windows
.\build.ps1 -All
```

### 4. Test Release Binaries

Test each binary on its target platform:

- Windows: `dist/tracker-windows-amd64.exe version`
- Linux: `dist/tracker-linux-amd64 version`
- macOS: `dist/tracker-darwin-amd64 version`

### 5. Create Release Archives

```bash
# Create archives with checksums
make release-archives
```

### 6. Create Git Tag

```bash
# Create annotated tag
git tag -a v0.1.0 -m "Release v0.1.0"

# Push tag to trigger release workflow
git push origin v0.1.0
```

### 7. GitHub Release

The GitHub Actions workflow will automatically:
- Build binaries for all platforms
- Create release archives
- Generate checksums
- Create GitHub release with artifacts

Alternatively, create manually:
1. Go to GitHub Releases page
2. Click "Draft a new release"
3. Choose the tag (v0.1.0)
4. Add release title: "Release v0.1.0"
5. Copy release notes from CHANGELOG.md
6. Upload release archives and checksums
7. Publish release

### 8. Post-Release

- [ ] Merge release branch to main
- [ ] Update develop branch
- [ ] Announce release (if applicable)
- [ ] Update documentation site (if applicable)
- [ ] Close milestone (if using GitHub milestones)

## Version Numbering

We follow [Semantic Versioning](https://semver.org/):

- **MAJOR** version for incompatible API changes
- **MINOR** version for new functionality in a backwards compatible manner
- **PATCH** version for backwards compatible bug fixes

Examples:
- `0.1.0` - Initial release
- `0.2.0` - New features added
- `0.2.1` - Bug fixes
- `1.0.0` - First stable release

## Hotfix Process

For critical bugs in production:

```bash
# Create hotfix branch from main
git checkout main
git checkout -b hotfix/v0.1.1

# Fix the bug
# Update version to 0.1.1
# Update CHANGELOG.md

# Commit and tag
git commit -m "Fix critical bug"
git tag -a v0.1.1 -m "Hotfix v0.1.1"

# Push
git push origin hotfix/v0.1.1
git push origin v0.1.1

# Merge back to main and develop
git checkout main
git merge hotfix/v0.1.1
git push origin main

git checkout develop
git merge hotfix/v0.1.1
git push origin develop
```

## Build Flags

The build process uses the following ldflags:

```
-X command-history-tracker/internal/version.Version=<version>
-X command-history-tracker/internal/version.GitCommit=<commit>
-X command-history-tracker/internal/version.BuildDate=<date>
```

These are automatically set by the build scripts and Makefile.

## Platform-Specific Notes

### Windows
- Test on both PowerShell and Command Prompt
- Verify shell integration works correctly
- Test installer scripts (install.ps1)

### macOS
- Test on both Intel and Apple Silicon
- Verify code signing (if applicable)
- Test shell integration for Bash and Zsh

### Linux
- Test on major distributions (Ubuntu, Fedora, Arch)
- Verify shell integration for Bash and Zsh
- Test installer scripts (install.sh)

## Troubleshooting

### Build Failures

If builds fail:
1. Check Go version (requires 1.21+)
2. Verify all dependencies are available
3. Check for platform-specific issues
4. Review build logs for errors

### Test Failures

If tests fail:
1. Run tests locally on the failing platform
2. Check for platform-specific test issues
3. Review test logs for details
4. Fix issues and re-run tests

### Release Workflow Failures

If GitHub Actions workflow fails:
1. Check workflow logs
2. Verify secrets are configured
3. Check permissions
4. Re-run workflow if transient failure
