# fireauth

CLI tool for authenticating against Firebase and managing bearer tokens for REST API testing.

Instead of manually grabbing tokens from the frontend, run a command and get everything you need. Supports multiple simultaneous user sessions for multi-account testing and **multiple Firebase projects** for different environments and/or applications.

## Prerequisites

- Firebase Web API Key (from Firebase Console > Project Settings > General)
- Firebase service account JSON key (from Firebase Console > Project Settings > Service accounts)

## Install

```bash
curl -sSL "https://raw.githubusercontent.com/andrespd99/fireauth/main/install.sh" | bash
```

The script auto-detects your OS/architecture, downloads the right binary, and installs it. No Go needed.

### Build from source

If you prefer to build from source (requires [Go 1.21+](https://go.dev/dl/) and [Task](https://taskfile.dev/)):

```bash
git clone https://github.com/andrespd99/fireauth.git
cd fireauth
task build
cp fireauth /usr/local/bin/
```

## Setup

Run the setup wizard to configure your first Firebase project:

```bash
fireauth init
```

This will prompt for:
1. Your Firebase Web API Key
2. Path to the service account JSON file

The service account JSON is copied into `~/.fireauth/projects/<name>/` so the original can be safely deleted from Downloads. The project name defaults to the `project_id` from the service account JSON.

For non-interactive setup (e.g., scripting):

```bash
fireauth init --api-key "AIzaSy..." --service-account ~/path/to/service-account.json
```

You can also specify the project name explicitly:

```bash
fireauth init staging --api-key "AIzaSy..." --service-account ~/path/to/staging-sa.json
fireauth init production --api-key "AIzaSy..." --service-account ~/path/to/prod-sa.json
```

### Migrating from single-project

If you were already using `fireauth` with the old single-project config, your existing setup is automatically migrated to a `default` project the first time you run any command. No action needed.

## Usage

### Projects

You can configure as many Firebase projects as you need (e.g., staging and production):

```bash
# List all configured projects
fireauth project list

# Switch the active project (interactive picker)
fireauth project use

# Switch directly
fireauth project use production

# Remove a project
fireauth project remove staging

# Rename a project
fireauth project rename staging dev
```

You can also override the active project for a single command using the global `--project` flag — perfect for scripting:

```bash
# Get a token from production without switching
fireauth --project production token

# Log in against staging
fireauth --project staging login

# Check who you are in production
fireauth --project production me
```

### Sign in

```bash
fireauth login
```

Or non-interactively:

```bash
fireauth login --email user@example.com --password "..."
```

### Get a bearer token

```bash
# Print token to stdout (pipe-friendly)
fireauth token

# Use directly with curl
curl -H "Authorization: Bearer $(fireauth token)" https://api.example.com/users/me

# Get a token from a specific project
curl -H "Authorization: Bearer $(fireauth --project production token)" https://api.example.com/users/me

# Print as full header
fireauth token --header

# Copy to clipboard (macOS)
fireauth token --copy

# Force refresh even if not expired
fireauth token --refresh
```

Tokens auto-refresh when expired or within 5 minutes of expiry.

### View current user

```bash
fireauth me

# JSON output
fireauth me --json
```

Also available as `fireauth whoami`.

### Manage multiple sessions

Sessions are per-project. Switching projects preserves the sessions within each project.

```bash
# List all stored sessions for the active project
fireauth sessions

# Switch active session (interactive picker)
fireauth switch

# Switch directly
fireauth switch other@example.com

# Remove a session
fireauth logout
fireauth logout other@example.com
```

### Updating

```bash
# Update to the latest version
fireauth update

# Check for updates without installing
fireauth update --check
```

### Use with Postman

Since Postman pre-request scripts cannot spawn child processes, `fireauth`
includes a built-in HTTP server you can start locally. Postman scripts call
the server over HTTP to fetch the bearer token automatically.

#### 1. Start the server

```bash
fireauth serve
```

By default it listens on `http://127.0.0.1:9876` (localhost only — no remote
access). Use `--addr` to change the port:

```bash
fireauth serve --addr 127.0.0.1:9877
```

#### 2. Endpoints

| Method | Path       | Description                                              |
| ------ | ---------- | -------------------------------------------------------- |
| `GET`  | `/health`  | Health check (`{"status":"ok","version":"..."}`)       |
| `GET`  | `/token`   | Returns the bearer token for the active session          |
| `GET`  | `/me`      | Returns JSON user details for the active session         |

All endpoints accept an optional `?project=` query parameter to override the
active project for that request.

**`/token` query parameters:**

| Param     | Default | Description                                              |
| --------- | ------- | -------------------------------------------------------- |
| `project` | (active)| Override the active project                              |
| `refresh` | `false` | Force token refresh (`true`/`false`)                     |
| `format`  | (bare)  | Set to `header` to get `Authorization: Bearer <token>`   |

#### 3. Postman pre-request script

Add this to your collection's **Pre-request Script** tab (or per-request if
you prefer):

```javascript
// Fetch a fresh bearer token from fireauth and set it as the Authorization header.
pm.sendRequest({
    url: "http://127.0.0.1:9876/token",
    method: "GET"
}, function (err, response) {
    if (err) {
        console.log("fireauth: request failed — is the server running? (fireauth serve)");
        throw err;
    }
    if (response.code !== 200) {
        console.log("fireauth: " + response.text());
        throw new Error("Failed to fetch token from fireauth");
    }
    pm.request.headers.upsert({
        key: "Authorization",
        value: "Bearer " + response.text()
    });
});
```

> [!TIP]
> If you work with multiple Firebase projects, add `?project=production` (or
> the project name) to the URL in the script to target a specific project
> without switching the active one.

### Debugging

Add `--verbose` to any command to see debug logs (HTTP calls, file I/O, token refresh decisions):

```bash
fireauth --verbose login
fireauth --verbose token
fireauth --verbose project list
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
| `serve`          | Start a local HTTP server for Postman           |
| `update`         | Self-update to the latest release               |

## Local storage

All data is stored in `~/.fireauth/` with restricted permissions (0700/0600):

```
~/.fireauth/
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