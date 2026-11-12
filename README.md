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
  - [`gryd service run`](#gryd-service-run-service1-service2)
  - [`gryd test`](#gryd-test-project)
  - [`gryd look`](#gryd-look-project)
  - [`gryd status`](#gryd-status-project)
  - [`gryd ports`](#gryd-ports)
  - [`gryd vc --status`](#gryd-vc---status)
  - [`gryd vc branches`](#gryd-vc-branches)
  - [`gryd vc remote`](#gryd-vc-remote)
  - [`gryd vc commit`](#gryd-vc-commit)
  - [`gryd vc push`](#gryd-vc-push)
  - [`gryd vc merge`](#gryd-vc-merge-source_branch-base_branch)
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

### `gryd init [name] [--wizard]`
Generates a starter `gryd.yaml` configuration in the current working directory.
Use `--wizard` (or `-w`) to launch an interactive step-by-step terminal wizard:

* Prompts for **Project Name** and **Working Directory** (defaults to current directory name and `.`).
* Prompts for **Health Checks** and **Auto-Restart on Failure** settings.
* Iteratively prompts for each service:
  - **Service Identifier** (e.g. `backend`, `frontend`, `worker`).
  - **Subdirectory** (relative or absolute).
  - **Command to Execute** (`npm run dev`, `uv run api`, `go run .`).
  - **Port** (optional).
  - **Dependencies (`depends_on`)** with auto-suggested configured services.
  - **Custom Environment Variables** (`KEY=val,PORT=8000`).
* Asks whether to add another service, previews the generated layout in a styled card, and writes out a clean `gryd.yaml`.

```bash
# Interactive step-by-step wizard
gryd init --wizard

# Quick default template
gryd init
gryd init MyPlatform
```

---

### `gryd run [project]`
Starts all services defined in the configuration concurrently.
By default in an interactive terminal, launches the responsive full-screen TUI dashboard:
* **Header Bar**: Live project name, status badge, active service count, and stack uptime.
* **Left Sidebar**: Real-time service status indicators (`●:port` for running, `✖` for stopped) with keyboard navigation (`↑`/`↓` or `Tab`).
* **Main Log Pane**: Dedicated live-streaming log viewer for the selected service or all combined services, with full viewport scrolling (`PgUp`/`PgDown`, `u`/`d`, `g`/`G`).
* **Hot Keybindings**:
  - `r`: Hot reload all project services on the fly.
  - `c`: Detach to background and keep services running.
  - `q` / `Ctrl+C`: Gracefully stop all services, kill process groups, and release listening ports.

*(Note: When redirected to a pipe or run in non-interactive CI environments, `gryd run` automatically falls back to raw multiplexed stdout streaming).*

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

### `gryd service run <service1> [service2...]`
Starts only the specified subset of services from your project configuration without booting the entire environment.

```bash
# Run a single service
gryd service run worker

# Run multiple specific services together
gryd service run worker frontend

# Target a project from outside its directory
gryd service run worker frontend --project Campora
```

All interactive live-stream controls (`r` to reload selected services, `c` to run in background, `Ctrl+C` to quit) remain fully functional.

---

### `gryd test [project]`
Executes a pre-flight test boot of all project services. Monitors them until their configured ports are listening and background tasks initialize stably, then terminates all child processes cleanly and releases the ports.

Returns an exit code of `0` on success and `1` on boot failure or timeout.

```bash
# Inside a project directory
gryd test

# From any directory
gryd test Campora

# Custom readiness timeout
gryd test --timeout 30s
```

**Sample Output:**
```text
[gryd test] Starting test run for project 'MockApp' (timeout: 25s)...
Working directory: /Users/developer/Projects/MockApp

  ✓ Service 'web' listening on port 9911 (252ms)
  ✓ Service 'worker' started and running stably (2.002s)

✓ All 2 services for 'MockApp' booted successfully!
All child processes have been cleanly terminated and ports released.

  SERVICE          STATUS     PORT     BOOT TIME    COMMAND
  ────────────────────────────────────────────────────────────────────
  web              Passed     9911     252ms        python3 -m http.server 9911
  worker           Passed     -        2.002s       sleep 5

✓ Project 'MockApp' is verified and ready to run with: gryd run
```

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

### `gryd ports`
Scans all active and configured Gryd projects and presents a neat, structured overview of allocated ports, listening states, and the exact operating system processes/PIDs bound to them.

```bash
gryd ports
```

**Sample Output:**
```text
Gryd Port Allocations
────────────────────────────────────────────────────────────────────

  PORT     PROJECT          SERVICE        STATUS                   PIDS         PROCESS
  ────────────────────────────────────────────────────────────────────────────────────────
  3000     cleo             frontend       Active (Listening)       88926        next-server (v16)
  8000     cleo             backend        Active (Listening)       88889        uvicorn
  8100     cleo             mcp            Active (Listening)       88890        python

  Run gryd stop <project> to terminate a project and release its ports.
```

---

### `gryd vc --status` (or `gryd vc status`)
Checks the version control environment: validates whether Git is installed on the system, checks configured global credentials (`user.name`, `user.email`), verifies GitHub CLI authentication status (`gh auth status`), and displays active repository and branch details if inside a Git repo.

```bash
gryd vc --status
# or
gryd vc status
```

**Sample Output:**
```text
Version Control System Status
──────────────────────────────────────────────────

  ✓ Git Installed:       git version 2.53.0
  ✓ User Identity:       Sidharth Prabhu <mail@sidharthprabhu.co.in>
  ✓ GitHub CLI (gh):     Logged in as Sidharth-Prabhu

  Current Repository:
    • Branch: main
    • Remote: https://github.com/Sidharth-Prabhu/Gryd.git
```

If Git is not installed, Gryd displays an informative error message with direct platform installation instructions.

> **Note:** `gryd vc` commands require an initialized Gryd project (a `gryd.yaml` configuration in the directory tree). If Gryd has not been initialized, running `gryd vc` will prompt you to run `gryd init` first.

---

### `gryd vc branches`
Lists all local and remote tracking branches in the current repository, highlighting the active branch (`HEAD`) in green with commit hashes and tracking messages.

```bash
gryd vc branches
```

**Sample Output:**
```text
Git Branches
──────────────────────────────────────────────────

* main (HEAD)                         49cf8ed [origin/main] Features added
  remotes/origin/main                 49cf8ed Features added
```

---

### `gryd vc remote`
Displays all configured remote URLs for the repository (both fetch and push). If no remote is configured yet, Gryd interactively prompts you to add one:

```bash
gryd vc remote
```

**Interactive Prompt (When no remote is set):**
```text
No remote repository configured. Would you like to add one? (y/n): y
Enter remote repository URL (e.g. https://github.com/user/repo.git): https://github.com/example/my-project.git

✓ Successfully added remote 'origin': https://github.com/example/my-project.git
You can now push using: git push -u origin <branch>
```

---

### `gryd vc commit`
Presents an interactive branch selector (with arrow-key navigation or direct typing to create a new branch), stages all repository changes (`git add -A`), and commits with your message.

```bash
gryd vc commit
# or with inline message
gryd vc commit -m "Add new authentication flow"
```

**Interactive Branch Selection:**
```text
Select a target Git branch:
Use ↑/↓ arrows to select, type to create a new branch, Enter to confirm

❯ main (current)
  dev
  + Create new branch...
```

---

### `gryd vc push`
Pushes to a selected remote repository on your chosen branch.
1. Prompts you to pick an existing branch or type a new one to switch/create.
2. If uncommitted changes exist, prompts to commit them first.
3. Presents an interactive arrow-key selector to choose the remote repository before pushing.

```bash
gryd vc push
```

**Interactive Remote Selector:**
```text
Select a remote repository to push to:
Use ↑/↓ arrow keys to navigate, Enter to select, q/Ctrl+C to cancel

❯ origin         (https://github.com/Sidharth-Prabhu/Gryd.git)
  backup         (git@gitlab.com:sidharth/backup.git)
```

#### Smart Branch Conflict Resolution
If switching branches would overwrite uncommitted local changes, Gryd halts and presents an interactive resolution menu with explanations of each choice:

```text
⚠ Uncommitted local changes would be overwritten by switching to 'main'.
How would you like to resolve this?
Use ↑/↓ arrow keys to choose an action, Enter to confirm, Esc to abort

❯ 1. Stash changes, switch branch, and re-apply (Recommended)
    Shelves your uncommitted work into git stash, switches to 'main', and automatically pops/restores your changes there.

  2. Commit changes on current branch first
    Prompts for a commit message and commits your uncommitted work to current branch before switching to 'main'.

  3. Stash changes and switch (Keep stash saved)
    Safely saves your uncommitted changes into git stash and switches to 'main', leaving the working tree completely clean.

  4. Force switch and discard uncommitted changes
    Permanently discards conflicting uncommitted local changes and forces checkout of 'main'. (Warning: Changes will be lost).

  5. Cancel / Abort
    Cancels the operation and stays on your current branch without modifying anything.
```

---

### `gryd vc merge [source_branch] [base_branch]`
Merges a feature or source branch into a target base branch (e.g. `main` or `development`).
If branch arguments are omitted, Gryd interactively prompts you to choose the source branch and destination base branch using arrow keys.

```bash
# Fully interactive branch selection
gryd vc merge

# Merge a specific branch into the current branch
gryd vc merge feature/login

# Explicitly specify source and base branch
gryd vc merge feature/login main

# Flags:
# --no-ff: Create a merge commit even if fast-forward is possible
# --squash: Squash commits into working tree without committing immediately
gryd vc merge feature/login main --no-ff
```

**Sample Output:**
```text
[gryd vc] Switching to base branch 'main'...
[gryd vc] Merging 'feature/login' into 'main'...

Updating 0eb22ed..4afc70b
Fast-forward
 login.txt | 1 +
 1 file changed, 1 insertion(+)

✓ Successfully merged 'feature/login' into 'main'!
You can now push the updated base branch using: gryd vc push
```

If merge conflicts occur, Gryd clearly lists the conflicted files and provides immediate guidance on resolving them.

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
