# Gryd (⚡)

[![Go Version](https://img.shields.io/badge/go-%3E%3D1.21-00ADD8.svg?style=flat&logo=go)](https://golang.org)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Platform](https://img.shields.io/badge/platform-macOS%20%7C%20Linux-lightgrey.svg)](https://github.com/Sidharth-Prabhu/Gryd)

**Gryd** is a high-performance, developer-first command-line orchestrator and terminal user interface (TUI) for local multi-service development environments.

Instead of juggling 5+ terminal tabs for your API backend, frontend, worker queues, MCP servers, and databases—or wrestling with heavy container overhead—Gryd allows developers to configure, launch, monitor, test, tunnel, and manage all services with a single command.

---

## ⚡ Highlights

- **🖥️ Responsive Full-Screen TUI**: Live multi-pane terminal interface powered by Bubble Tea & Lip Gloss, featuring Catppuccin-themed status badges, service navigation, and live scrollable log viewing.
- **🛡️ Process Group Isolation & Port Freeing**: Uses native Unix process groups (`pgid`). Stopping services terminates all child processes (`node`, `python`, `uvicorn`, `go`) without leaving orphaned port listeners (`EADDRINUSE`).
- **🔍 In-TUI Log Search & Regex Filtering (`/`)**: Real-time log searching with automatic fallback from regex to substring search and keyword highlighting.
- **🏷️ Log Level Cycling (`l`) & Color Differentiation**: Instantly filter logs by level (`ERROR`, `WARN`, `INFO`) with color-coded syntax highlighting across the TUI and CLI.
- **📑 Structured JSON Log Parser (`J`)**: Auto-detects structured JSON logs (Zap, Zerolog, Pino) with toggleable viewing modes (`compact` single-line, `pretty` multiline, or `raw`).
- **📍 Visual Timestamp Markers (`m`)**: Injects high-visibility visual divider markers into the log stream to cleanly track what happened after an action or API request.
- **📊 Real-Time Resource Telemetry & Process Tree (`i` / `p`)**: Displays live CPU% and RSS memory usage badges next to each service in the sidebar; inspect child process trees, threads, and file descriptors.
- **🌐 Instant Public Tunneling (`t` / `gryd tunnel`)**: Expose local ports to public HTTPS URLs via Cloudflare, ngrok, or localtunnel for quick webhook testing.
- **👁️ Built-in File Watcher (`watch` / `ignore`)**: Declarative hot-reload rules per service with 300ms debouncing—no external tools like `nodemon` or `air` required.
- **🔄 Hot Reload (`r`) vs. Hard Restart (`R`)**: Graceful code reload (`r`) keeps container sidecars running, while hard restart (`R`) completely cleans processes, frees ports, and restarts fresh.
- **🔀 Multi-Environment Profiles (`--profile` / `-p`)**: Run customized subsets of services and dependencies with cascading `.env` file support (`.env`, `.env.local`, `.env.<profile>`).
- **🔌 Remote Unix Socket IPC Daemon**: Control running services from another terminal tab (`gryd service restart <svc>`, `gryd service stop <svc>`, `gryd service exec <svc> -- <cmd>`).
- **🧪 Pre-Flight Boot Verification (`gryd test`)**: Automated integration test boot that verifies port readiness, safely terminates all child processes, and outputs standard CI exit codes or machine-readable JSON (`--json`).
- **🌿 Integrated Version Control (`gryd vc`)**: Full-screen Git branch selector with live search, push workflows, and branch merging.

---

## 📦 Installation

### From Source (Go 1.21+)

```bash
git clone https://github.com/Sidharth-Prabhu/Gryd.git
cd Gryd
go build -o /usr/local/bin/gryd ./cmd/gryd
```

### Quick Install (macOS / Linux)

```bash
go install github.com/loom-cli/gryd/cmd/gryd@latest
```

Verify your installation:
```bash
gryd --version
```

---

## 🚀 Quick Start

### 1. Scaffold your configuration
Inside your project root:
```bash
# Launch the interactive configuration wizard
gryd init --wizard

# Or generate a starter template directly
gryd init my-app
```

### 2. Verify everything boots cleanly
```bash
gryd test
```

### 3. Launch the interactive development dashboard
```bash
gryd run
```

---

## ⚙️ Configuration Reference (`gryd.yaml`)

Gryd uses a human-friendly YAML file in your project root:

```yaml
name: cleo
working_directory: .

services:
  backend:
    directory: api
    command: uv run relationos-api
    port: 8000
    env:
      ENVIRONMENT: development
      PORT: "8000"
    watch:
      - "src/**"
      - "config/**"
    ignore:
      - "*.log"
      - ".git/**"

  worker:
    directory: mcp
    command: uv run python scripts/worker.py
    depends_on:
      - backend

  frontend:
    directory: web
    command: npm run dev
    port: 3000
    depends_on:
      - backend

# Optional containerized sidecars (PostgreSQL, Redis, RabbitMQ, Ollama, etc.)
dependencies:
  postgres:
    image: postgres:16-alpine
    ports: ["5432:5432"]
    env:
      POSTGRES_DB: app
      POSTGRES_USER: dev
      POSTGRES_PASSWORD: password
    volumes:
      - ./data/postgres:/var/lib/postgresql/data
  redis:
    image: redis:7-alpine
    ports: ["6379:6379"]

# Named Environment Profiles
profiles:
  minimal:
    services: ["backend"]
    dependencies: ["postgres"]
    env:
      DEBUG: "true"
  full-stack:
    services: ["backend", "worker", "frontend"]
    dependencies: ["postgres", "redis"]
    env_file: ".env.fullstack"

settings:
  health_check: true
  restart_on_failure: false
```

### Field Definitions

| Field | Type | Description |
|---|---|---|
| `name` | `string` | Unique project identifier. |
| `working_directory` | `string` | Root folder path (defaults to `.`). |
| `services.<name>.directory` | `string` | Subdirectory where the service command runs. |
| `services.<name>.command` | `string` | Shell command executed for the service. |
| `services.<name>.port` | `int` | *(Optional)* Port used for readiness verification & port freeing. |
| `services.<name>.depends_on`| `list` | *(Optional)* Prerequisite services or sidecars required before startup. |
| `services.<name>.env` | `map` | *(Optional)* Environment variables injected into the service process. |
| `services.<name>.watch` | `list` | *(Optional)* Directories or file globs to watch for hot reloads. |
| `services.<name>.ignore` | `list` | *(Optional)* File patterns ignored by the file watcher. |
| `dependencies.<name>.image` | `string` | Container image to run as a sidecar (via Docker or Podman). |
| `dependencies.<name>.ports` | `list` | Port mappings (`host:container`). |
| `dependencies.<name>.env` | `map` | Environment variables injected into the container. |
| `dependencies.<name>.volumes` | `list` | Volume mounts (`host_dir:container_dir`). |
| `profiles.<name>.services` | `list` | Subset of services to run under this profile. |
| `profiles.<name>.dependencies` | `list` | Subset of sidecars to run under this profile. |
| `profiles.<name>.env` | `map` | Environment overrides for this profile. |
| `profiles.<name>.env_file` | `string` | Path to custom `.env` file for this profile. |
| `settings.health_check` | `bool` | Verifies whether configured ports are open and listening. |
| `settings.restart_on_failure` | `bool` | Automatically restarts services that exit with non-zero codes. |

---

## 📖 Command Reference

### Global Flags
- `-c, --config <path>`: Path to configuration file (default is `./gryd.yaml`).
- `-p, --profile <name>`: Target environment profile defined in `gryd.yaml` (e.g. `minimal`, `full-stack`).
- `-d, --debug`: Enable verbose/debug logging output.
- `-h, --help`: Display help and syntax for any command.

---

### `gryd run [project]`
Starts all services defined in `gryd.yaml` concurrently.

```bash
# Launch default environment with full-screen TUI
gryd run

# Launch with a specific profile
gryd run --profile minimal

# Run a specific project from outside its directory
gryd run my-project
```

---

### `gryd logs [service]`
View, filter, stream, or export service logs directly from the terminal.

```bash
# View combined logs for all services
gryd logs

# Filter logs for a specific service
gryd logs backend

# Live follow/stream logs (like tail -f)
gryd logs backend -f

# Show last N lines
gryd logs backend -n 100

# Export logs to a file
gryd logs backend -n 500 -o backend.log
```

---

### `gryd tunnel <service>`
Expose a service's local port to a public HTTPS URL via Cloudflare Tunnels, ngrok, or localtunnel.

```bash
# Expose backend port with auto-detected provider
gryd tunnel backend

# Expose using a specific provider
gryd tunnel web --provider ngrok

# Stop an active tunnel
gryd tunnel stop backend
```

---

### `gryd service`
Remotely inspect or control individual services via the background IPC daemon without stopping the whole project.

```bash
# Run only specific services from configuration
gryd service run backend worker

# Restart a single service via daemon IPC
gryd service restart backend
# (Shortcut: gryd restart backend)

# Stop a single service via daemon IPC
gryd service stop worker
# (Shortcut: gryd stop worker)

# Execute a command in a service's directory
gryd service exec backend -- pnpm test
```

---

### `gryd test [project]`
Runs an integration boot test, monitors socket listeners, streams live logs, and verifies clean startup and shutdown.

```bash
# Standard test boot
gryd test

# Test with a specific profile
gryd test --profile minimal

# Machine-readable JSON output for CI pipelines
gryd test --json
```

---

### `gryd status [project]`
Displays execution status, PIDs, active ports, container sidecars, and uptimes.

```bash
# Human-friendly card view
gryd status

# Machine-readable JSON output for CI/CD or scripts
gryd status --json
```

---

### `gryd ports`
Scans all project-assigned ports and identifies active listening PIDs and process names.

```bash
gryd ports
```

---

### `gryd look [project]`
Attaches to an already-running background project and opens the live interactive TUI dashboard.

```bash
gryd look
gryd look my-project
```

---

### `gryd reload [project]`
Performs a graceful hot reload on running services without destroying state or stopping container sidecars.

```bash
gryd reload
```

---

### `gryd restart [project]`
Performs a complete hard restart: terminates all processes and process groups (`SIGTERM` + `SIGKILL`), frees all occupied ports, cleans up state, and boots the project fresh.

```bash
gryd restart
```

---

### `gryd stop [project]`
Stops all running services, terminates container sidecars, and frees occupied network ports.

```bash
gryd stop
```

---

### `gryd list`
Lists all known projects, their configuration file paths, and current execution states.

```bash
gryd list
```

---

### `gryd vc` (Version Control)
Full-screen interactive Git branch selector, conflict resolver, and push manager.

```bash
# Open interactive Git management dashboard
gryd vc

# Commit and push all changes with a single command
gryd vc push -m "feat: new feature"

# Merge another branch safely
gryd vc merge feature/login
```

---

### `gryd version`
Prints version, release phase, platform, and Go runtime information.

```bash
gryd version
gryd --version
```

---

## ⌨️ TUI Keyboard Shortcuts

When running in the full-screen dashboard (`gryd run` or `gryd look`):

| Key | Action | Description |
|---|---|---|
| `/` | **Search** | Enter live log search mode (supports plain text & regex). `Enter` to confirm, `Esc` to clear. |
| `l` | **Level Filter** | Cycle log level filter: `ALL` $\rightarrow$ `ERROR` $\rightarrow$ `WARN` $\rightarrow$ `INFO` $\rightarrow$ `ALL`. |
| `J` | **JSON Mode** | Cycle structured JSON log viewing mode: `compact` $\rightarrow$ `pretty` $\rightarrow$ `raw`. |
| `m` | **Marker** | Drop a visual timestamp divider line across the log stream. |
| `t` | **Tunnel** | Toggle a public HTTPS tunnel for the selected service. |
| `i` / `p` | **Inspect Tree** | Toggle detailed process tree view, child PIDs, threads, and file descriptors. |
| `r` | **Hot Reload** | Gracefully reload services in-place without destroying container sidecars. |
| `R` | **Hard Restart** | Hard-terminate all processes and ports and boot the project fresh. |
| `↑` / `↓` / `Tab` | **Navigate** | Switch between individual service logs or combined log view. |
| `u` / `d` | **Scroll** | Scroll up/down by 8 lines (`g` jumps to top, `G` jumps to bottom). |
| `c` | **Detach** | Detach to background mode. Services continue running; re-attach with `gryd look`. |
| `q` / `Ctrl+C` | **Quit** | Gracefully terminate all processes, release ports, and exit. |

---

## 📄 License

Gryd is licensed under the [MIT License](LICENSE).
