# cashea-auth

CLI tool for authenticating against Firebase and managing bearer tokens for REST API testing.

Instead of manually grabbing tokens from the frontend, run a command and get everything you need. Supports multiple simultaneous user sessions for multi-account testing and **multiple Firebase projects** for different environments and/or applications.

## Prerequisites

- Firebase Web API Key (from Firebase Console > Project Settings > General)
- Firebase service account JSON key (from Firebase Console > Project Settings > Service accounts)

## Install

Since this is a private repo, you need GitHub access. The easiest way is via the `gh` CLI:

```bash
# If you have gh CLI installed and authenticated:
curl -sSL "https://raw.githubusercontent.com/cashea-bnpl/auth-devtools/main/install.sh" \
  -H "Authorization: token $(gh auth token)" | bash
```

Or with a personal access token:

```bash
export GITHUB_TOKEN=ghp_your_token_here
curl -sSL "https://raw.githubusercontent.com/cashea-bnpl/auth-devtools/main/install.sh" \
  -H "Authorization: token $GITHUB_TOKEN" | bash
```

The script auto-detects your OS/architecture, downloads the right binary, and installs it. No Go needed.

### Build from source

If you prefer to build from source (requires [Go 1.21+](https://go.dev/dl/) and [Task](https://taskfile.dev/)):

```bash
git clone git@github.com:cashea-bnpl/auth-devtools.git
cd auth-devtools
task build
cp cashea-auth /usr/local/bin/
```

## Setup

Run the setup wizard to configure your first Firebase project:

```bash
cashea-auth init
```

This will prompt for:
1. Your Firebase Web API Key
2. Path to the service account JSON file

The service account JSON is copied into `~/.cashea-auth/projects/<name>/` so the original can be safely deleted from Downloads. The project name defaults to the `project_id` from the service account JSON.

For non-interactive setup (e.g., scripting):

```bash
cashea-auth init --api-key "AIzaSy..." --service-account ~/path/to/service-account.json
```

You can also specify the project name explicitly:

```bash
cashea-auth init staging --api-key "AIzaSy..." --service-account ~/path/to/staging-sa.json
cashea-auth init production --api-key "AIzaSy..." --service-account ~/path/to/prod-sa.json
```

### Migrating from single-project

If you were already using `cashea-auth` with the old single-project config, your existing setup is automatically migrated to a `default` project the first time you run any command. No action needed.

## Usage

### Projects

You can configure as many Firebase projects as you need (e.g., staging and production):

```bash
# List all configured projects
cashea-auth project list

# Switch the active project (interactive picker)
cashea-auth project use

# Switch directly
cashea-auth project use production

# Remove a project
cashea-auth project remove staging

# Rename a project
cashea-auth project rename staging dev
```

You can also override the active project for a single command using the global `--project` flag — perfect for scripting:

```bash
# Get a token from production without switching
cashea-auth --project production token

# Log in against staging
cashea-auth --project staging login

# Check who you are in production
cashea-auth --project production me
```

### Sign in

```bash
cashea-auth login
```

Or non-interactively:

```bash
cashea-auth login --email user@example.com --password "..."
```

### Get a bearer token

```bash
# Print token to stdout (pipe-friendly)
cashea-auth token

# Use directly with curl
curl -H "Authorization: Bearer $(cashea-auth token)" https://api.cashea.com/users/me

# Get a token from a specific project
curl -H "Authorization: Bearer $(cashea-auth --project production token)" https://api.cashea.com/users/me

# Print as full header
cashea-auth token --header

# Copy to clipboard (macOS)
cashea-auth token --copy

# Force refresh even if not expired
cashea-auth token --refresh
```

Tokens auto-refresh when expired or within 5 minutes of expiry.

### View current user

```bash
cashea-auth me

# JSON output
cashea-auth me --json
```

Also available as `cashea-auth whoami`.

### Manage multiple sessions

Sessions are per-project. Switching projects preserves the sessions within each project.

```bash
# List all stored sessions for the active project
cashea-auth sessions

# Switch active session (interactive picker)
cashea-auth switch

# Switch directly
cashea-auth switch other@example.com

# Remove a session
cashea-auth logout
cashea-auth logout other@example.com
```

### Updating

```bash
# Update to the latest version
cashea-auth update

# Check for updates without installing
cashea-auth update --check
```

Requires `GITHUB_TOKEN` or `gh` CLI (same as install).

### Use with Postman

> [!NOTE]
> Coming soon... Postman does not allow child_process calls in pre-request scripts, so it cannot call this tool automatically. But there's a work-around: Because Postman pre-request scripts can make HTTP request, we can start an http server and call a served path to get the bearer token. But I think this tool is already a good MVP and I'm tired boss 🚬

### Debugging

Add `--verbose` to any command to see debug logs (HTTP calls, file I/O, token refresh decisions):

```bash
cashea-auth --verbose login
cashea-auth --verbose token
cashea-auth --verbose project list
```

## Commands

| Command          | Description                                     |
| ---------------- | ----------------------------------------------- |
| `init`           | Set up or add a Firebase project                |
| `project list`   | List all configured projects                    |
| `project use`    | Switch the active project                       |
| `project remove` | Remove a project                                |
| `project rename` | Rename a project                               |
| `login`          | Sign in with email and password                 |
| `token`          | Print the current bearer token                  |
| `me`             | Show current user details (Firebase Admin SDK)  |
| `sessions`       | List all stored sessions for the active project |
| `switch`         | Switch active session                           |
| `logout`         | Remove a stored session                         |
| `update`         | Self-update to the latest release               |

## Local storage

All data is stored in `~/.cashea-auth/` with restricted permissions (0700/0600):

```
~/.cashea-auth/
├── config.json                          # Global config (active project)
└── projects/
    ├── staging/
    │   ├── project.json                 # API key, service account path, active session
    │   ├── sessions.json                # Stored user sessions (tokens, UIDs)
    │   └── service-account.json         # Firebase service account credentials
    └── production/
        ├── project.json
        ├── sessions.json
        └── service-account.json
```