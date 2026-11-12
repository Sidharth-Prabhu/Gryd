# Gryd (⚡)

[![Go Version](https://img.shields.io/badge/go-%3E%3D1.21-00ADD8.svg?style=flat&logo=go)](https://golang.org)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Platform](https://img.shields.io/badge/platform-macOS%20%7C%20Linux-lightgrey.svg)](https://github.com/Sidharth-Prabhu/Gryd)

**Gryd** is a high-performance, developer-first command-line orchestrator and terminal user interface (TUI) for multi-service local environments.

Instead of opening 5 terminal tabs for your API backend, frontend, worker, MCP server, and databases—or wrestling with heavy container overhead—Gryd allows developers to configure, launch, monitor, test, and manage all services with a single command.

---

## ⚡ Highlights

- **🖥️ Responsive Full-Screen TUI**: Live multi-pane terminal interface powered by Bubble Tea & Lip Gloss, featuring Catppuccin-themed status badges, service navigation, and live scrollable log viewing.
- **🚀 One-Command Orchestration**: Concurrently launch microservices, frontends, background workers, and scripts with clean process isolation.
- **🛡️ Process Group Isolation & Port Freeing**: Uses native Unix process groups (`pgid`). When stopping services, child processes (`node`, `python`, `uvicorn`) are completely terminated without leaving orphaned port listeners.
- **🧙 Interactive Setup Wizard (`gryd init --wizard`)**: Step-by-step interactive CLI guide to scaffold `gryd.yaml` without editing YAML manually.
- **🧪 Pre-Flight Boot Verification (`gryd test`)**: Automated integration test boot that monitors port listening readiness, verifies clean startups, safely terminates, and exits with standard CI exit codes.
- **🔍 Port Conflict Auditor (`gryd ports`)**: Scans all project-assigned ports and identifies active listening PIDs and process names.
- **🔄 Zero-Disruption Hot Reload (`r`) & Background Detach (`c`)**: Restart running services on the fly or detach to background mode without stopping child processes.
- **🌿 Integrated Version Control (`gryd vc`)**: Full-screen Git branch selector with live search, push workflows, conflict resolvers, and branch merging.

---

## 📋 Table of Contents

- [Installation](#-installation)
- [Quick Start](#-quick-start)
- [Configuration Reference (`gryd.yaml`)](#-configuration-reference-grydyaml)
- [Command Reference](#-command-reference)
  - [`gryd init [--wizard]`](#gryd-init-name---wizard)
  - [`gryd run`](#gryd-run-project)
  - [`gryd service run`](#gryd-service-run-service1-service2)
  - [`gryd test`](#gryd-test-project)
  - [`gryd status`](#gryd-status-project)
  - [`gryd ports`](#gryd-ports)
  - [`gryd look`](#gryd-look-project)
  - [`gryd reload`](#gryd-reload-project)
  - [`gryd stop`](#gryd-stop-project)
  - [`gryd list`](#gryd-list)
  - [`gryd vc` (Version Control)](#-gryd-vc-version-control)
- [Architecture & Safety](#-architecture--safety)
- [Contributing](#-contributing)
- [License](#-license)

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
gryd init --wizard
```
Follow the interactive prompts to define your project name, working directory, and services.

Alternatively, generate a default starter configuration:
```bash
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

Gryd uses a human-friendly YAML file placed in your project root:

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

  worker:
    directory: mcp
    command: uv run python scripts/worker.py
    depends_on:
      - backend

  mcp:
    directory: mcp
    command: env RELOS_MCP_TRANSPORT=streamable-http FASTMCP_HOST=127.0.0.1 FASTMCP_PORT=8100 uv run relationos-mcp
    port: 8100
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

settings:
  health_check: true
  restart_on_failure: false
```

### Field Definitions

| Field | Type | Description |
|---|---|---|
| `name` | `string` | Unique project identifier. |
| `working_directory` | `string` | Root directory path (defaults to `.`). |
| `services.<name>.directory` | `string` | Subdirectory where the service command runs. |
| `services.<name>.command` | `string` | Shell command to execute. |
| `services.<name>.port` | `int` | *(Optional)* Port used by the service for readiness verification & port freeing. |
| `services.<name>.depends_on`| `list` | *(Optional)* Prerequisite services or sidecars required before startup. |
| `services.<name>.env` | `map` | *(Optional)* Environment variables injected into the service process. |
| `dependencies.<name>.image` | `string` | Container image to run as a sidecar (via Docker or Podman). |
| `dependencies.<name>.ports` | `list` | Port mappings (`host:container`). |
| `dependencies.<name>.env` | `map` | Environment variables injected into the container. |
| `dependencies.<name>.volumes` | `list` | Volume mounts (`host_dir:container_dir`). |
| `settings.health_check` | `bool` | Verifies whether configured ports are open and listening. |
| `settings.restart_on_failure` | `bool` | Automatically restarts services that exit with non-zero codes. |

---

## 📖 Command Reference

### `gryd init [name] [--wizard]`
Scaffolds a `gryd.yaml` configuration file in the current directory.

```bash
# Launch the interactive configuration wizard
gryd init --wizard

# Create a starter template directly
gryd init MyProject
```

The interactive wizard asks for:
- Project Name and Working Directory.
- Health Check and Auto-Restart preferences.
- Each service's name, directory, command, port, dependencies, and environment variables.
- Prompts before overwriting existing files and generates a formatted preview.

---

### `gryd run [project]`
Starts all services defined in `gryd.yaml` concurrently.

By default in an interactive terminal, launches the responsive full-screen TUI:
- **Left Sidebar**: Real-time service status indicators (`●:port` for listening, `✖` for stopped) with keyboard navigation (`↑`/`↓` or `Tab`).
- **Main Pane**: Dedicated live-streaming log viewer for the selected service or all combined streams.
- **Viewport Scrolling**: Navigate output using `u`/`d`, `PgUp`/`PgDown`, or `g`/`G`.

#### Keybindings inside `gryd run`:
| Key | Action |
|---|---|
| `/` | **Interactive Search**: Filter log stream in real time with keyword highlighting |
| `Enter` | Keep filtered search active and resume navigation |
| `Esc` | Clear search filter and reset viewport to full stream |
| `↑` / `↓` / `Tab` | Switch active service log stream |
| `u` / `d` / `PgUp` / `PgDown` | Scroll viewport up / down |
| `g` / `G` | Jump to top / bottom of log stream |
| `r` | Hot reload all project services on the fly |
| `c` | Detach to background and keep services running |
| `q` / `Ctrl+C` | Gracefully shut down all services and free ports |

*(Note: When redirected to a pipe or run in non-interactive CI environments, `gryd run` automatically falls back to raw multiplexed stdout streaming).*

---

### `gryd service run <service1> [service2...]`
Starts only the specified subset of services without launching the entire stack.

```bash
# Run a single service
gryd service run worker

# Run multiple specific services together
gryd service run worker frontend

# Target a project from outside its directory
gryd service run worker frontend --project cleo
```

---

### `gryd test [project]`
Executes a pre-flight integration test boot of all project services:
1. Spawns all declared services concurrently.
2. Monitors port listeners and process health until all services are ready.
3. Automatically terminates all child processes and releases sockets.
4. Outputs an itemized summary table and exits with standard status codes (`0` for success, `1` on failure or timeout).

```bash
gryd test
gryd test --timeout 30s
```

---

### `gryd status [project]`
Displays a formatted status card showing overall project health, uptime, process IDs, ports, and execution commands.

```bash
# Inside project directory
gryd status

# From any directory
gryd status cleo
```

---

### `gryd ports`
Audits all ports declared across running or configured Gryd projects. Identifies listening sockets via TCP probe and `lsof`, displaying active PIDs and binary names.

```bash
gryd ports
```

---

### `gryd look [project]`
Attaches to live streaming logs of a background project started with `gryd run` (after pressing `c` to detach) or `gryd reload`.

```bash
gryd look
gryd look cleo
```

---

### `gryd reload [project]`
Performs a zero-disruption restart of a project's services in the background: terminates existing process groups, frees listening sockets, re-reads `gryd.yaml`, and starts services fresh.

```bash
gryd reload
```

---

### `gryd stop [project]`
Forcefully terminates all services for a project using process group signals (`SIGTERM` followed by `SIGKILL`) and aggressively frees any lingering sockets.

```bash
gryd stop
gryd stop cleo
```

---

### `gryd list`
Scans known directories and registries, listing all active background projects and configured repositories.

```bash
gryd list
```

---

## 🌿 `gryd vc` (Version Control)

Gryd includes built-in version control commands to streamline Git workflows within your project environment:

### `gryd vc status`
Inspects local Git installation, configured author identity (`user.name`, `user.email`), GitHub CLI login status (`gh`), and current repository branch/remote.

### `gryd vc branches`
Displays a clean, structured table separating local branches (with `* (HEAD)`, commit hash, tracking branch, and commit subject) and remote tracking branches.

### `gryd vc push`
Interactive branch switcher and remote pusher:
- Presents an interactive Bubble Tea branch selector with live fuzzy search and arrow-key navigation.
- Detects uncommitted changes and prompts for a commit message.
- If switching branches causes conflicts with local uncommitted work, provides an interactive 5-option resolution menu (`stash-pop`, `commit`, `stash-keep`, `force`, `cancel`).
- Lets you select the destination remote and pushes with upstream tracking (`git push -u <remote> <branch>`).

### `gryd vc merge [source_branch] [base_branch]`
Interactively selects source and destination branches, checks out the base branch, pulls latest changes, and merges the source branch cleanly.

---

## 🔒 Architecture & Safety

- **Process Group Isolation**: Gryd launches each service inside its own process group (`syscall.SysProcAttr{Setpgid: true}`). Signals are dispatched directly to the process group (`-PID`), ensuring sub-processes (such as Python child workers or Webpack compilers) do not linger as orphans.
- **Port Auditing & Cleanup**: Whenever services stop or restart, Gryd audits listening TCP ports and clears lingering processes to prevent `EADDRINUSE` errors.
- **State Storage**:
  - `~/.gryd/run/<project>.json`: Active runtime process metadata.
  - `~/.gryd/logs/<project>.log`: Complete multiplexed log history.
  - `~/.gryd/projects.json`: Known project catalog.

---

## 🤝 Contributing

Contributions are welcome!

1. Fork the repository.
2. Create your feature branch (`git checkout -b feature/amazing-feature`).
3. Ensure all tests pass (`go test ./...`).
4. Commit your changes (`git commit -m 'Add amazing feature'`).
5. Push to the branch (`git push origin feature/amazing-feature`).
6. Open a Pull Request.

---

## 📄 License

Distributed under the MIT License. See [LICENSE](LICENSE) for more information.
