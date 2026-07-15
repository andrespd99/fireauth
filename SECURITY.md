# Security Policy

## Supported versions

`fireauth` is a small, single-binary CLI under active development. Security
fixes are released as patch versions and only the **latest released version** is
maintained. Please update to the newest release before reporting or testing a
vulnerability — older versions won't receive backports.

| Version | Supported          |
| ------- | ------------------ |
| Latest  | :white_check_mark: |
| < Latest | :x:               |

## Reporting a vulnerability

If you believe you've found a security vulnerability in `fireauth`, **please do
not open a public GitHub issue**. Instead, report it privately so we can
investigate and fix it before details are public.

Preferred reporting methods, in order:

1. **GitHub private vulnerability reporting** — use
   [Report a vulnerability](https://github.com/andrespd99/fireauth/security/advisories/new)
   on the Security tab. This is the fastest path and keeps the report
   confidential.
2. **Email** — send details to **hello@andrespacheco.dev** with
   `[fireauth security]` in the subject line.

Please include:

- A description of the issue and why you believe it's exploitable.
- The affected version (`fireauth version`) and your OS/architecture.
- Steps to reproduce, a proof of concept, or a minimal example.
- Any relevant logs (redact sensitive data). Running with `--verbose` is
  helpful.

You should receive an initial response within **72 hours**. We'll keep you
updated as we investigate, patch, and release a fix, and we'll credit you in the
advisory unless you'd prefer to remain anonymous.

## Disclosure process

1. We acknowledge receipt of the report.
2. We investigate and validate the issue, then coordinate a fix with you.
3. A fix is prepared in a private branch and tested.
4. A patch release is published and a GitHub Security Advisory is published at
   the same time.
5. Credit (if desired) is given in the advisory.

Please give us reasonable time to fix the issue before disclosing it publicly.
We aim to release fixes within **30 days** of validation for high-severity
issues, sooner when possible.

## Scope

In scope:

- The `fireauth` CLI source code in this repository (`main.go`, `cmd/`,
  `internal/`, `install.sh`).
- The local HTTP server started by `fireauth serve`.
- How credentials (service account JSON, tokens, sessions) are stored and used
  on disk under `~/.fireauth/`.

Out of scope:

- Vulnerabilities in third-party dependencies that aren't reachable from
  `fireauth`'s code. A CVE in a dependency alone isn't a `fireauth`
  vulnerability — please demonstrate that the vulnerable code path is actually
  reachable from `fireauth`.
- Issues that require the attacker to already have read access to the user's
  `~/.fireauth/` directory (which is created with `0700` permissions). Local
  privilege escalation that depends on already-compromised user files is out of
  scope.
- Firebase or Google Cloud APIs themselves — report those to Google via the
  [Google VRP](https://bughunter.withgoogle.com/).

## Hardening notes

- All local files under `~/.fireauth/` are created with restrictive permissions
  (`0700` for directories, `0600` for files). Don't relax these in contributions.
- The `serve` HTTP server binds to `127.0.0.1` only — it is not network
  accessible by default. Only bind it to other interfaces if you understand the
  implications.
- `fireauth` never writes credentials to stdout or logs in non-`--verbose`
  mode. With `--verbose`, HTTP traffic is logged but bearer tokens are
  redacted. If you're reporting a leak, please double check it isn't caused by
  `--verbose` output that you shared publicly.

Thanks for helping keep `fireauth` and its users safe.