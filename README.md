# cashea-auth

CLI tool for authenticating against Firebase and managing bearer tokens for REST API testing.

Instead of manually grabbing tokens from the frontend, run a command and get everything you need. Supports multiple simultaneous user sessions for multi-account testing.

## Prerequisites

- Firebase Web API Key (from Firebase Console > Project Settings > General)
- Firebase service account JSON key (from Firebase Console > Project Settings > Service accounts)

## Install

```bash
curl -sSL https://raw.githubusercontent.com/cashea-bnpl/auth-devtools/main/install.sh | bash
```

This downloads the latest pre-built binary for your OS/architecture. No Go installation needed.

### Build from source

If you prefer to build from source (requires [Go 1.21+](https://go.dev/dl/) and [Task](https://taskfile.dev/)):

```bash
git clone git@github.com:cashea-bnpl/auth-devtools.git
cd auth-devtools
task build
cp cashea-auth /usr/local/bin/
```

## Setup

Run the one-time setup wizard:

```bash
cashea-auth init
```

This will prompt for:
1. Your Firebase Web API Key
2. Path to the service account JSON file

The service account JSON is copied into `~/.cashea-auth/` so the original can be safely deleted from Downloads.

For non-interactive setup (e.g., scripting):

```bash
cashea-auth init --api-key "AIzaSy..." --service-account ~/path/to/service-account.json
```

## Usage

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

```bash
# List all stored sessions
cashea-auth sessions

# Switch active session (interactive picker)
cashea-auth switch

# Switch directly
cashea-auth switch other@example.com

# Remove a session
cashea-auth logout
cashea-auth logout other@example.com
```

### Use with Postman

> [!NOTE]
> Coming soon... Postman does not allow child_process calls in pre-request scripts, so it cannot call this tool automatically. But there's a work-around: Because Postman pre-request scripts can make HTTP request, we can start an http server and call a served path to get the bearer token. But I think this tool is already a good MVP and I'm tired boss 🚬

### Debugging

Add `--verbose` to any command to see debug logs (HTTP calls, file I/O, token refresh decisions):

```bash
cashea-auth --verbose login
cashea-auth --verbose token
```

## Commands

| Command    | Description                                    |
| ---------- | ---------------------------------------------- |
| `init`     | One-time setup wizard                          |
| `login`    | Sign in with email and password                |
| `token`    | Print the current bearer token                 |
| `me`       | Show current user details (Firebase Admin SDK) |
| `sessions` | List all stored sessions                       |
| `switch`   | Switch active session                          |
| `logout`   | Remove a stored session                        |

## Development

```bash
task build          # Build binary
task test           # Run tests
task test-verbose   # Run tests with verbose output
task lint           # Run go vet
task clean          # Remove build artifacts
```

## Local storage

All data is stored in `~/.cashea-auth/` with restricted permissions (0700/0600):

- `config.json` — API key, service account path, active session
- `sessions.json` — Stored user sessions (tokens, UIDs)
- `service-account.json` — Firebase service account credentials
