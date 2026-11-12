# Gryd ⚡

> **A cross-platform CLI developer utility to configure, orchestrate, and launch entire multi-service development environments with a single command.**

Eliminate the chaos of opening half a dozen terminal tabs just to run `npm run dev`, your backend API, worker processes, and database dependencies. With **Gryd**, declare your services in a single `gryd.yaml` file, boot the environment in one command, stream logs with unified multiplexing, and manage background execution effortlessly.

---

## 📑 Table of Contents

- [Features](#-features)
- [Installation](#-installation)
  - [Prerequisites](#prerequisites)
  - [Install from Source](#install-from-source)
- [Quick Start](#-quick-start)
- [Configuration Reference (`gryd.yaml`)](#-configuration-reference-grydyaml)
- [Command Reference](#-command-reference)
  - [`gryd init`](#gryd-init-name)
  - [`gryd run`](#gryd-run-project)
  - [`gryd look`](#gryd-look-project)
  - [`gryd status`](#gryd-status-project)
  - [`gryd reload`](#gryd-reload-project)
  - [`gryd stop`](#gryd-stop-project)
  - [`gryd list`](#gryd-list)
  - [`gryd --version`](#gryd---version)
- [Interactive Controls (Live Stream)](#-interactive-controls-live-stream)
- [Context-Aware Project Detection](#-context-aware-project-detection)
- [Architecture & Internals](#-architecture--internals)
- [Contributing](#-contributing)
- [License](#-license)

---

## ✨ Features

- 🚀 **One-Command Boot**: Launch frontend, backend, workers, and mock servers concurrently with `gryd run`.
- 📁 **Smart Context-Aware Resolution**: Run commands bare (`gryd status`, `gryd run`, `gryd stop`) while inside any project directory or subdirectory; reference any registered project by name from anywhere else.
- 🎨 **Unified Log Multiplexing**: Synchronized, line-by-line color-coded stdout/stderr streaming preventing interleaved lines.
- 📌 **Pinned Bottom Controls**: Live logs scroll cleanly *above* an interactive terminal footer that never scrolls away.
- 🔄 **Hot Reloading**: Press `r` in the log viewer or run `gryd reload` to cleanly terminate old processes, re-read configuration, and restart services without disrupting your flow.
- 📴 **Background Execution & Detachment**: Press `c` during a live run to return instantly to your terminal prompt while leaving services running in the background.
- 🔍 **Live Log Attaching (`gryd look`)**: Re-attach to background logs at any time from any directory.
- 🛑 **Clean Port Freeing & Graceful Shutdown**: Process group isolation (`Setpgid`) ensures no orphaned child processes (`node`, JVM, Python) linger on ports like `3000` or `8080`.
- 📊 **Detailed Health & Process Status (`gryd status`)**: Real-time view of individual PIDs, ports, commands, and project uptime.

---

## 📦 Installation

### Prerequisites

- **Go 1.21+** installed on your system.

### Install from Source

```bash
# Clone the repository
git clone https://github.com/loom-cli/gryd.git
cd gryd

# Build and install the binary
go install ./cmd/gryd

# (Optional) Ensure Go binary path is in your PATH
export PATH="$HOME/go/bin:$HOME/.local/bin:$PATH"
```

Verify your installation:

```bash
gryd --version
```

Output:
```text
gryd version 0.1.0-dev (none)
Release:    Phase 1 MVP
Go version: go1.26.2
Platform:   darwin/arm64
```

---

## ⚡ Quick Start

### 1. Initialize a Project
Navigate to your project repository and generate a starter configuration:

```bash
cd ~/Projects/my-app
gryd init my-app
```

This creates a `gryd.yaml` file in the current directory.

### 2. Configure Your Services
Edit `gryd.yaml`:

```yaml
name: my-app
working_directory: .

services:
  backend:
    directory: backend
    command: go run main.go
    port: 8080

  frontend:
    directory: frontend
    command: npm run dev
    port: 5173
    depends_on:
      - backend

settings:
  health_check: true
  restart_on_failure: false
```

### 3. Launch the Environment

```bash
gryd run
```

All services start concurrently, and unified logs begin streaming to your console.

---

## 🛠 Configuration Reference (`gryd.yaml`)

Gryd uses a declarative YAML file (`gryd.yaml` or legacy `loom.yaml`):

```yaml
# Unique project identifier
name: Campora

# Root directory for relative service paths. Supports ~ home expansion.
# Defaults to the folder containing gryd.yaml if omitted.
working_directory: ~/Projects/Campora

services:
  # Service identifier
  backend:
    # Relative to working_directory or absolute path
    directory: backend
    # Exact shell command to execute
    command: ./mvnw spring-boot:run
    # Port service listens on (used for port auditing & automatic cleanup)
    port: 8080
    # Custom environment variables
    env:
      ENVIRONMENT: development
      PORT: "8080"

  frontend:
    directory: frontend
    command: npm run dev
    port: 5173
    # Startup dependency ordering
    depends_on:
      - backend

  worker:
    directory: worker
    command: python -m worker.main

settings:
  health_check: true
  restart_on_failure: false
```

---

## 📖 Command Reference

### `gryd init [name]`
Generates a starter `gryd.yaml` configuration in the current working directory.

```bash
gryd init
# or specify a project name
gryd init MyPlatform
```

---

### `gryd run [project]`
Starts all services defined in the configuration concurrently.

```bash
# Inside a project directory
gryd run

# From any other directory
gryd run Campora

# Specify a custom config file
gryd run --config /path/to/custom.yaml
```

**Flags:**
- `-c, --config string`: Path to specific configuration file.
- `-d, --debug`: Enable verbose debug logging.

---

### `gryd look [project]`
Attaches to and streams real-time logs for a background project.

```bash
# Inside project directory
gryd look

# From any directory
gryd look Campora
```

*Press `c` or `Ctrl+C` to detach and return to the terminal without stopping services.*

---

### `gryd status [project]`
Displays an itemized health and runtime status table showing service PIDs, port bindings, overall state, and uptime.

```bash
# Inspect local project or overview of all active projects
gryd status

# Inspect a specific project
gryd status Campora
```

**Sample Output:**
```text
Project:          Campora
Working Dir:      /Users/developer/Projects/Campora
Config File:      /Users/developer/Projects/Campora/gryd.yaml
Log File:         /Users/developer/.gryd/logs/Campora.log
Overall Status:   Running
Uptime:           18m24s (started 12:00:15)

  SERVICE          STATUS       PID      PORT     COMMAND
  ────────────────────────────────────────────────────────────────────
  backend          Running      62140    8080     ./mvnw spring-boot:run
  frontend         Running      62141    5173     npm run dev
  worker           Running      62142    -        python -m worker.main

View live logs with: gryd look Campora
```

---

### `gryd reload [project]`
Restarts all services in the background without opening the log stream.
- Re-reads `gryd.yaml` from disk.
- Terminates previous processes and clears ports.
- Spawns fresh instances with new PIDs.

```bash
# Inside project directory
gryd reload

# From any directory
gryd reload Campora
```

---

### `gryd stop [project]`
Completely shuts down all project services and aggressively audits ports to ensure no orphaned processes linger.

```bash
# Inside project directory
gryd stop

# From any directory
gryd stop Campora
```

**Sample Output:**
```text
[gryd] Stopping project 'Campora'...
  • Service 'frontend': freed port 5173 (killed process PIDs [62141])

✓ Successfully stopped project 'Campora'.
  • Terminated 2 running service processes.
  • Released and confirmed ports: [5173, 8080]
```

---

### `gryd list`
Scans and displays all currently running background projects along with known local and global project configurations.

```bash
gryd list
```

---

### `gryd --version`
Displays version, release cycle, Go toolchain version, and OS/architecture.

```bash
gryd --version
# or
gryd -v
```

---

## 🎮 Interactive Controls (Live Stream)

While running `gryd run` or `gryd look`, the bottom of your terminal is pinned with interactive controls:

```text
──────────────────────────────────────────────────────────
Press 'r' to reload  |  Press 'c' to run in background  |  Ctrl+C to quit
──────────────────────────────────────────────────────────
```

| Key | Action | Behavior |
| :---: | :--- | :--- |
| **`r`** | **Reload** | Gracefully stops all child processes, re-parses config, and restarts services fresh. |
| **`c`** | **Run in Background** | Detaches the live log view and returns to your shell immediately. Services remain active. |
| **`Ctrl+C`** | **Quit / Stop** | Traps interrupt signal, stops all services gracefully, frees ports, and cleans state. |

---

## 🧭 Context-Aware Project Detection

You never need to remember where a project lives:

1. **Inside Project Directory**:
   Gryd walks up parent directories (like Git searching for `.git`) to locate `gryd.yaml`. Running `gryd status`, `gryd run`, or `gryd reload` operates on that project automatically.

2. **From Any Other Directory**:
   When you run a project with Gryd, its path is recorded in `~/.gryd/projects.json`. You can control it from anywhere:
   ```bash
   cd ~/Downloads
   gryd status Campora
   gryd reload Campora
   gryd stop Campora
   ```

---

## 🏗 Architecture & Internals

- **Process Supervision (`internal/runner`)**: Uses `os/exec` with process group isolation (`Setpgid: true` on Unix). Sending signals to `-PID` kills entire process trees (e.g. `npm` -> `node` -> `esbuild`), preventing zombie background processes.
- **Port Management (`internal/runner/port.go`)**: Queries socket listeners and audits open ports on shutdown and reload using TCP dials and `lsof`, killing lingering holders if necessary.
- **State Storage (`internal/state`)**:
  - `~/.gryd/run/<project>.json`: Active runtime metadata, PIDs, and start timestamps.
  - `~/.gryd/logs/<project>.log`: Multiplexed plain-text logs for background streaming.
  - `~/.gryd/projects.json`: Global catalog of discovered project paths.
- **Terminal Rendering (`internal/runner/footer.go`)**: Emits DECSTBM (`\033[top;bottomr`) escape sequences to split the terminal buffer into a scrollable log region (rows `1` to `H-2`) and a fixed status footer (rows `H-1` to `H`).

---

## 🤝 Contributing

Contributions, bug reports, and feature suggestions are welcome!

1. Fork the repository.
2. Create your feature branch (`git checkout -b feature/amazing-feature`).
3. Commit your changes (`git commit -m 'Add amazing feature'`).
4. Ensure all tests pass (`go test -v ./...`).
5. Push to the branch (`git push origin feature/amazing-feature`).
6. Open a Pull Request.

---

## 📄 License

Distributed under the MIT License. See `LICENSE` for more information.
