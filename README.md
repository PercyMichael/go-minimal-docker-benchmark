# 🚀 High-Performance Go JSON API & Containerization Benchmark

A production-ready, ultra-lightweight REST API written in Go, structured with **Package-by-Feature** architecture, powered by **Chi Router**, **`sqlc`**, and containerized using multi-stage Docker builds. Serves structured JSON payloads with zero unnecessary dependencies, hardened security, and sub-millisecond response times.

---

## 🎯 Purpose & Target Audience

This repository serves as a **production blueprint and microservice benchmark** for:

- **Backend & Software Engineers**: A reference implementation for building ultra-fast Go REST APIs with sub-millisecond response times, clean feature-based packaging (`internal/`), and minimal memory footprint (<10 MB RAM).
- **DevOps & Security Engineers**: A benchmark for microservice container optimization, zero-CVE image security, non-root execution context (`UID 65534`), automated VPS deployment, and reducing Docker image footprint by **58.4%** (from 13.8 MB to 5.74 MB).

---

## 📋 Features & Architecture

- **Ultra-Fast & Lightweight**: Built with Go 1.25, Chi Router (`github.com/go-chi/chi/v5`), and `sqlc` PostgreSQL code generation.
- **Production-Grade Reliability**:
  - **Graceful Shutdown**: Listens for `SIGTERM`/`SIGINT` signals with a 10-second context timeout.
  - **DoS & Slowloris Protection**: Explicit HTTP timeouts (`ReadTimeout: 10s`, `WriteTimeout: 15s`, `IdleTimeout: 120s`).
  - **Liveness Probe**: Integrated `/healthz` endpoint for Kubernetes/AWS ECS load balancers.
- **Security Hardening**:
  - `-trimpath` removes host filesystem paths from binary metadata.
  - `-ldflags="-s -w"` strips debugging symbols and symbol tables.
  - Runs as an unprivileged non-root user (`UID 65534` / `nobody`).
  - Read-only root filesystem compatible (`read_only: true`).
- **Automated CI/CD**: Fully automated test, build, containerize, and SSH VPS deployment pipeline via [GitHub Actions](file:///.github/workflows/deploy.yml).

---

## 🏗️ Architectural & Tooling Choices

| Category | Tool / Choice | Rationale & Trade-offs |
| :--- | :--- | :--- |
| **HTTP Router** | **`Chi` (`go-chi/chi/v5`)** | 100% `net/http` compatible, zero-allocation overhead, and clean middleware grouping (`r.Group(...)`) without locking into a proprietary context framework like Gin. |
| **Database & Queries** | **`sqlc` + PostgreSQL** | Compiles raw `.sql` queries into 100% type-safe Go code at build time. Avoids ORM performance traps (like N+1 queries in GORM) while eliminating manual `rows.Scan` boilerplate. |
| **Schema Migrations** | **`Goose` (Sequential `.sql` Files)** | Database migrations are split into individual, versioned SQL files (`00001_create_users_table.sql`, `00002_create_notes_table.sql`) for clean schema history and incremental rollback capability. |
| **App Architecture** | **Package-by-Feature (`internal/`)** | Domain features are grouped into self-contained packages (`internal/user`, `internal/note`). Keeps handlers, services, and queries co-located, reducing context-switching across deep folder hierarchies. |
| **Configuration** | **`godotenv` + `os.Getenv`** | Automatically loads `.env` files in local development while falling back to platform OS environment variables in production (Docker, Kubernetes, AWS). |
| **Task Automation** | **`Makefile`** | Provides simple, unified developer commands (`make build`, `make run`, `make test`, `make generate`). |

---

## 📁 Project Structure

```text
.
├── cmd/
│   └── api/
│       └── main.go                 # App entrypoint: loads config, boots DB, runs Chi server
├── internal/                       # Compiler-enforced private application code
│   ├── config/
│   │   └── config.go               # Environment configuration parser & fallback defaults
│   ├── db/
│   │   ├── migrations/             # Sequential raw SQL migration files (Goose target)
│   │   │   ├── 00001_create_users_table.sql
│   │   │   ├── 00002_create_notes_table.sql
│   │   │   └── 00003_create_sessions_table.sql
│   │   ├── queries/                # Raw SQL query definitions (sqlc target)
│   │   │   ├── users.sql
│   │   │   └── notes.sql
│   │   └── db.go                   # PostgreSQL connection pool & auto-migration engine
│   ├── middleware/
│   │   └── auth.go                 # Authentication & session token middleware
│   ├── models/
│   │   └── models.go               # Shared domain structs & API response envelopes
│   ├── user/                       # User domain feature (handlers, service)
│   └── note/                       # Note domain feature (handlers, service)
├── .env                            # Active local environment variables (Git ignored)
├── .env.example                    # Environment variable template for developer setup
├── .github/workflows/deploy.yml    # GitHub Actions production CI/CD pipeline
├── Dockerfile                      # Active production Dockerfile (v3 Scratch base)
├── Dockerfile.v1-alpine            # Version 1: Alpine Linux Build (13.80 MB)
├── Dockerfile.v2-distroless        # Version 2: Google Distroless Build (7.61 MB)
├── Dockerfile.v3-scratch           # Version 3: Pure Scratch Build (5.74 MB)
├── docker-compose.yml              # Container & DB composition setup
├── Makefile                        # Task runner script (build, run, test, generate)
├── sqlc.yaml                       # sqlc compiler configuration
├── go.mod                          # Go module definition
├── main.go                         # Root server entrypoint
├── DEPLOYMENT.md                   # VPS deployment & Caddy Auto-SSL guide
└── README.md                       # Project documentation
```

---

## 🛠️ The Job of the Dockerfile

The [Dockerfile](file:///Volumes/B/projects/devops%20go/Dockerfile) uses a **Multi-Stage Build** pattern to separate compilation from execution, keeping the final production image minimal and secure.

### Stage 1: The Builder Stage (`golang:1.25-alpine`)
1. **Environment Setup**: Sets `CGO_ENABLED=0` and `GOOS=linux` to produce a 100% statically linked Go binary without external C library dependencies.
2. **Dependency & Source Copy**: Sets `WORKDIR /app` and copies `go.mod` and source code.
3. **Static Binary Compilation**: Executes `go build -trimpath -ldflags="-s -w" -o /app/server ./cmd/api`.

### Stage 2: The Final Runtime Stage (`FROM scratch`)
1. **Zero OS Base**: Starts from `scratch` (an empty 0-byte image layer with no shell, no package manager, and zero CVE vulnerability attack surface).
2. **SSL / TLS Support**: Copies CA certificates (`/etc/ssl/certs/ca-certificates.crt`) from the builder stage for outbound HTTPS calls.
3. **Binary Copy**: Copies the static `/app/server` binary into `/app/server`.
4. **Least-Privilege Security**: Enforces non-root user execution (`USER 65534:65534`).

---

## 📊 Detailed 3-Build Comparison & Benchmark Evolution

During development, the container went through **3 distinct build iterations**, reducing image size by **58.4%** while eliminating OS-level security vulnerabilities.

### Build 1: Alpine Runtime (`Dockerfile.v1-alpine`)
- **Base Image**: `alpine:3.20`
- **Image Size**: **`13.80 MB`**
- **Characteristics**: Uses standard Alpine Linux base with `apk` package manager and `sh` shell.
- **Pros/Cons**: Easy to debug via `docker exec -it sh`, but includes unnecessary OS packages and potential CVE surface.

### Build 2: Google Distroless (`Dockerfile.v2-distroless`)
- **Base Image**: `gcr.io/distroless/static-debian12:nonroot`
- **Image Size**: **`7.61 MB`** (-44.8% reduction)
- **Characteristics**: Stripped-down Debian 12 base maintained by Google Security team.
- **Pros/Cons**: Contains no shell/package manager, includes pre-configured CA certs and non-root user (`UID 65532`).

### Build 3: Pure Scratch Base (`Dockerfile.v3-scratch` / `Dockerfile`) 🏆 *(Current Production)*
- **Base Image**: `scratch` (0-byte empty layer)
- **Image Size**: **`5.74 MB`** (**-58.4% reduction from Build 1**)
- **Characteristics**: Statically compiled binary + copied CA certificates + `WORKDIR /app` running under non-root user `65534` (`nobody`).
- **Pros/Cons**: Absolute smallest image size, **0 OS-level CVE vulnerabilities**, fastest container pull/startup time.

---

### 📉 Comparative Summary Table

| Build # | File Source | Runtime Base Image | Image Size | Size Reduction | OS Attack Surface / CVEs | User Context | Shell (`sh`) |
| :---: | :--- | :--- | :---: | :---: | :--- | :---: | :---: |
| **Build 1** | [`Dockerfile.v1-alpine`](file:///Volumes/B/projects/devops%20go/Dockerfile.v1-alpine) | `alpine:3.20` | `13.80 MB` | Baseline | Moderate (Busybox, `apk`, `sh`) | `root` / Custom | Enabled |
| **Build 2** | [`Dockerfile.v2-distroless`](file:///Volumes/B/projects/devops%20go/Dockerfile.v2-distroless) | `gcr.io/distroless/static-debian12:nonroot` | `7.61 MB` | -44.8% | Near Zero (No shell/OS tools) | `nonroot` (`65532`) | Disabled |
| **Build 3** | [`Dockerfile.v3-scratch`](file:///Volumes/B/projects/devops%20go/Dockerfile.v3-scratch) 🏆 | **`FROM scratch`** | **`5.74 MB`** | **-58.4%** | **Absolute Zero (0 OS CVEs)** | `nobody` (`65534`) | Disabled |

---

## 🔌 API Endpoints Table

| Method | Endpoint | Auth Required | Description |
| :--- | :--- | :---: | :--- |
| `POST` | `/api/auth/register` | ❌ No | Register new user account |
| `POST` | `/api/auth/login` | ❌ No | Authenticate user & set session token |
| `POST` | `/api/auth/logout` | ❌ No | Revoke active session token |
| `GET` | `/api/auth/me` | 🔓 Optional | Fetch current logged-in user profile |
| `GET` | `/api/notes` | 🔐 Yes | List user notes |
| `POST` | `/api/notes` | 🔐 Yes | Create a new user note |
| `PUT` | `/api/notes?id={id}` | 🔐 Yes | Update an existing note |
| `DELETE` | `/api/notes?id={id}` | 🔐 Yes | Delete a note |
| `GET` | `/healthz` | ❌ No | Liveness health check probe |

---

## 🌐 VPS & CI/CD Deployment Guide

For full instructions on setting up a $4/month VPS, installing Caddy for automatic HTTPS, setting up GitHub Secrets, and deploying automatically, see the **[DEPLOYMENT.md](file:///Volumes/B/projects/devops%20go/DEPLOYMENT.md)** guide.

---

## 🚀 Quick Start Guide

### 1. Running Locally (Without Docker)

Copy local environment config:
```bash
cp .env.example .env
```

Run the local API server:
```bash
make run
```

Run all unit tests:
```bash
make test
```

### 2. Building & Running with Docker Compose
```bash
docker compose up -d --build
```

### 3. Testing the Endpoints
```bash
# Test Healthcheck endpoint
curl http://localhost:8080/healthz
```
