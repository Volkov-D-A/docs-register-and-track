---
trigger: always_on
---

# 1. System & Environment Context
- OS: Ubuntu Linux (running via WSL 2 on a Windows host).
- TERMINAL: Native `bash`. Do NOT use `cmd.exe`, `powershell`, or prefix commands with `cmd /c`.
- PATHS: Strictly use Linux forward-slash paths (e.g., `/home/user/projects/...`). Never use Windows `C:\` paths.
- LINE ENDINGS: Ensure all generated bash scripts, Dockerfiles, and config files use Unix (LF) line endings, not Windows (CRLF).

# 2. Project Context & Architecture
- DOMAIN: Document registration and tracking system.
- BACKEND: Go (Golang).
- FRONTEND: Wails framework with Rollup for bundling. 
- DATABASE: PostgreSQL running locally in a Docker container.

# 3. Execution & Build Rules (CRITICAL)
- MAKEFILE FIRST: A `Makefile` is present in the root. Always prefer running `make dev`, `make build-linux`, or `make build-windows` over raw Wails CLI commands.
- WAILS DEV: If running Wails manually in this environment, you MUST append the webkit tag for Ubuntu 24.04 compatibility: `wails dev -tags webkit2_41`.
- DEPENDENCIES: Frontend dependencies are managed via `npm`. If frontend tasks are required, always `cd frontend` first.
- BACKGROUND TASKS: When running long-living processes (like `npm run watch` or `docker compose up`), use detached mode (`-d`) or background execution (`&`) so the terminal is not blocked.

# 4. Database & Docker Interactions
- DB CONNECTION: The PostgreSQL database is available at `localhost:5432`.
- CLI QUERIES: When asked to execute SQL, use non-interactive commands (e.g., `psql -U doc_admin -d doc_registration -c "..."`). Do not open interactive `psql` shells that wait for user input.
- DOCKER COMPOSE: Use `docker compose` (V2 syntax), not the legacy `docker-compose`.

# 5. Go Best Practices for this Workspace
- GO MODULES: Always run `go mod tidy` after adding new dependencies.
- EXCEL & FILES: When writing logic for document parsing or generation (e.g., Excel files), ensure file I/O operations use relative paths from the project root or explicitly defined absolute Linux paths.