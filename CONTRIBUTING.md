# Contributing to fireauth

Thanks for your interest in contributing to `fireauth`! This document walks
through how to get a local copy of the project running, what conventions to
follow, and how to submit your work.

## Code of Conduct

By participating in this project you agree to abide by the
[Code of Conduct](./CODE_OF_CONDUCT.md). Please read it before contributing.

## Reporting bugs

Before opening a bug report, search the
[existing issues](https://github.com/andrespd99/fireauth/issues) to avoid
duplicates. If you find a match, add a comment with your context instead of
opening a new one.

When filing a new bug, use the **Bug report** issue template and include:

- The output of `fireauth version`.
- Your operating system and architecture.
- The exact command you ran and what happened.
- What you expected to happen.
- Any relevant logs (use `--verbose`) — redact sensitive data first.

> **Security-related bugs must NOT be reported through public issues.** See
> [SECURITY.md](./SECURITY.md) for the responsible disclosure process.

## Suggesting features

Feature requests are welcome. Please use the **Feature request** issue
template and describe:

- The problem you're trying to solve.
- Your proposed solution.
- Any alternatives you've considered.

Keep the scope as narrow as possible — small, focused requests are far more
likely to be accepted than sweeping redesigns.

## Development setup

`fireauth` is written in [Go](https://go.dev/) and built with
[Task](https://taskfile.dev/). You'll need both installed.

1. Fork the repository and clone your fork:

   ```bash
   git clone git@github.com:your-username/fireauth.git
   cd fireauth
   ```

2. Add an upstream remote to keep your fork in sync:

   ```bash
   git remote add upstream https://github.com/andrespd99/fireauth.git
   ```

3. Build the binary:

   ```bash
   task build
   ```

   This produces a `fireauth` binary in the project root.

4. Run it locally:

   ```bash
   ./fireauth --help
   ```

## Development workflow

1. Create a branch from `main`:

   ```bash
   git checkout -b feat/my-feature main
   ```

   Use a descriptive prefix:

   - `feat/` — new features
   - `fix/` — bug fixes
   - `docs/` — documentation
   - `chore/` — tooling, deps, cleanup

2. Make your changes. Keep commits focused and write clear commit messages
   (see [Commit messages](#commit-messages) below).

3. Make sure everything builds and passes:

   ```bash
   task build
   task test
   task lint
   ```

4. If you add a user-facing command or flag, update the `README.md` command
   table and any relevant examples.

5. Push and open a pull request targeting `main`.

## Coding conventions

- Format all Go code with `gofmt -s` (most editors can do this on save).
- Run `go vet ./...` (`task lint`) before submitting — CI will run it too.
- Follow the advice in [Effective Go](https://go.dev/doc/effective_go) and the
  [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments).
- Prefer small, focused functions and packages. Avoid "utils" grab-bags.
- Document exported declarations with doc comments, even if they seem obvious.
- Don't add comments that just restate the code — explain *why*.

## Testing

- All non-trivial code should be covered by tests. Run the suite with
  `task test` (which runs `go test ./...`).
- When fixing a bug, add a regression test that fails before your fix and passes
  after.
- Avoid tests that depend on network or external state. Mock HTTP calls and
  keep unit tests hermetic.

## Commit messages

- Start with a short, capitalized summary (max ~50 chars) in the imperative
  mood — "Add login timeout flag", not "added login timeout flag".
- Optionally prefix the summary with the type of change, matching the branch
  naming: `feat:`, `fix:`, `docs:`, `chore:`, etc.
- Add a body explaining the *why* and the *what* when the change isn't obvious
  from the diff alone.
- Reference the issue a commit closes with `Closes #123` or `Fixes #123`.

Example:

```
fix: handle expired refresh token on `token` command

Previously, a session with an expired refresh token would return a stale
ID token instead of prompting the user to re-login.

Closes #42
```

## Pull requests

- One concern per PR. If a change touches multiple unrelated things, split it.
- Keep PRs reasonably small so they can be reviewed quickly.
- Rebase onto `main` before opening, and squash or rebase noisy WIP commits into
  clean, logical units.
- Fill in the pull request template. Link the issue your PR addresses.
- CI (build, test, lint) must pass before merge. If a check fails for unrelated
  reasons, fix it in a separate PR first.
- Don't force-push while a PR is under review — it makes inline comments hard to
  track. If you must, leave a note explaining why.

## Releases

Releases are handled by maintainers via the existing GoReleaser workflow. You
don't need to worry about versioning or changelog generation for normal
contributions. If your change is user-facing, mention it in the PR description
so it can be captured in the next release notes.

## Becoming a maintainer

Maintainers are added based on sustained, high-quality contributions and good
judgment. There's no checklist — just keep contributing thoughtfully and you'll
be noticed. If you'd like to discuss it, reach out via the contact in the
[Code of Conduct](./CODE_OF_CONDUCT.md).

---

Thanks again for contributing! Every issue filed, typo fixed, and feature
added makes `fireauth` better for everyone.