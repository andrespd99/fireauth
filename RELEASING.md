# Releasing fireauth

Releases are triggered manually via a GitHub Actions workflow. There are no
local tag pushes — the workflow checks out `main`, creates the tag, builds,
and publishes the release all in one step.

## Prerequisites

- All changes you want in the release must be merged into `main`.
- Ensure `main` is green (CI passing).
- The `HOMEBREW_TAP_GITHUB_TOKEN` secret must be set in the repo secrets
  (a GitHub PAT with `repo` scope on `andrespd99/homebrew-fireauth`). The
  default `GITHUB_TOKEN` cannot push to another repository.

## Steps

1. Go to **Actions** → **Release** → **Run workflow** in the GitHub UI:
   [github.com/andrespd99/fireauth/actions/workflows/release.yml](https://github.com/andrespd99/fireauth/actions/workflows/release.yml)

2. Enter the version (without `v` prefix):
   - **Stable**: `0.3.0-stable`
   - **Pre-release**: `0.3.0-alpha.1`, `0.3.0-beta.1`, etc.

3. If it's a pre-release, check the **"Mark as pre-release"** checkbox so it
   won't show as the latest release (and the install script won't pick it up
   by default).

4. Click **Run workflow**.

The workflow will:
- Validate the version format (`X.Y.Z` or `X.Y.Z-prerelease`)
- Reject if a tag for that version already exists
- Check out the latest commit on `main` (always current, never stale)
- Create and push the tag automatically
- Build binaries for `darwin` and `linux` (`amd64` + `arm64`) via goreleaser
- Create a GitHub Release with all assets and checksums
- If marked as pre-release, update it accordingly

## Versioning

Follow [semantic versioning](https://semver.org):

- **Patch** (`0.3.1-stable`) — bug fixes
- **Minor** (`0.4.0-stable`) — new features, backward compatible
- **Major** (`1.0.0-stable`) — breaking changes
- **Pre-release** (`0.4.0-alpha.1`) — testing before stable

## Installing

### Homebrew (recommended)

```bash
brew tap andrespd99/fireauth
brew install fireauth
```

To upgrade later:

```bash
brew upgrade fireauth
```

### Install script

`install.sh` defaults to the latest **stable** release (tag matching
`v*.*.*-stable`). To install a specific version:

```bash
# Latest stable (default)
curl -sSL "https://raw.githubusercontent.com/andrespd99/fireauth/main/install.sh" | bash

# Specific version (including pre-releases)
curl -sSL "https://raw.githubusercontent.com/andrespd99/fireauth/main/install.sh" | bash -s -- --version 0.3.0-alpha.1
```

## Local dry run (optional)

To test goreleaser locally without publishing:

```bash
goreleaser release --clean --snapshot
```

This builds all targets and writes them to `dist/` but skips publishing.