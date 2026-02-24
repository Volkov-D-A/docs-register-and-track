---
trigger: always_on
---

# 1. System & Execution Rules (CRITICAL)
- ENVIRONMENT: Windows 11 Host.
- TERMINAL FIX: You MUST prefix all terminal commands, build scripts, and CLI executions with `cmd /c` to ensure proper process termination and EOF signaling. 
  - Good: `cmd /c go run main.go`
  - Bad: `go run main.go`
- NON-INTERACTIVE: Never execute commands that wait for user input (e.g., Y/N prompts). Always use auto-confirm flags (like `-y` or `--force`).

# 2. Go & Wails Framework Context
- PROJECT TYPE: Go application using the Wails framework.
- WAILS CLI: When running Wails commands (like building or generating bindings), strictly use `cmd /c wails build` or `cmd /c wails dev`.
- FRONTEND (Rollup): Be aware that frontend dependencies are managed separately. To run frontend scripts, always change the directory to the frontend folder first, then use `cmd /c npm run build` (or your specific package manager command).

# 3. Docker & Infrastructure
- DOCKER COMPOSE: Always run containers in detached mode to avoid blocking the agent's pipeline (`cmd /c docker compose up -d`).
- LINE ENDINGS: When generating or modifying `.sh` scripts or Dockerfiles meant for Linux (Ubuntu) containers, strictly enforce `LF` (Unix) line endings, avoiding Windows `CRLF` to prevent container build crashes.

# 4. Database (PostgreSQL)
- MIGRATIONS: When executing PostgreSQL queries or migrations via CLI, pass queries directly (e.g., `cmd /c psql -c "..."`) or use non-interactive tools to prevent the shell from waiting for input.