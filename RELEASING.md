# Releasing cashea-auth

Releases are fully automated via [goreleaser](https://goreleaser.com) and a
GitHub Actions workflow. To cut a new release you only need to push a tag.

## Prerequisites

- All changes you want in the release must be merged into `main`.
- The working tree should be clean and up to date with `origin/main`.

## Steps

1. Ensure `main` is up to date:

   ```bash
   git checkout main
   git pull origin main
   ```

2. Create a tag matching the `v*` pattern (the release workflow only fires on
   tags starting with `v`):

   ```bash
   git tag v0.3.0
   ```

   Use [semantic versioning](https://semver.org):
   - **Patch** (`v0.3.1`) — bug fixes
   - **Minor** (`v0.4.0`) — new features, backward compatible
   - **Major** (`v1.0.0`) — breaking changes

3. Push the tag:

   ```bash
   git push origin v0.3.0
   ```

4. The [Release workflow](.github/workflows/release.yml) runs automatically:
   - goreleaser builds binaries for `darwin` and `linux` (`amd64` + `arm64`)
   - Archives are uploaded as `cashea-auth_<os>_<arch>.tar.gz`
   - A `checksums.txt` file is generated
   - A GitHub Release is created with the tag name and all assets

5. Verify the release on
   [github.com/cashea-bnpl/auth-devtools/releases](https://github.com/cashea-bnpl/auth-devtools/releases).

## Local dry run (optional)

To test goreleaser locally without publishing:

```bash
goreleaser release --clean --snapshot
```

This builds all targets and writes them to `dist/` but skips publishing.

## How the install script works

`install.sh` fetches `/releases/latest` from the GitHub API, finds the asset
matching the user's OS/arch, and downloads it. Because of this, `install.sh`
always installs the most recent tag — no changes needed when cutting a new
release.