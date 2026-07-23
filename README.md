# 🚀 High-Performance Go JSON API & Containerization Benchmark

A production-ready, ultra-lightweight REST API written in Go and containerized using multi-stage Docker builds. Serves structured JSON payloads with zero dependencies, hardened security, and sub-millisecond response times.

---

## 📋 Features & Architecture

- **Ultra-Fast & Lightweight**: Built with Go 1.24 standard library (`net/http`).
- **Production-Grade Reliability**:
  - **Graceful Shutdown**: Listens for `SIGTERM`/`SIGINT` signals with a 10-second context timeout.
  - **DoS & Slowloris Protection**: Explicit HTTP timeouts (`ReadTimeout: 5s`, `WriteTimeout: 10s`, `IdleTimeout: 120s`).
  - **Liveness Probe**: Integrated `/healthz` endpoint for Kubernetes/AWS ECS load balancers.
- **Security Hardening**:
  - `-trimpath` removes host filesystem paths from binary metadata.
  - `-ldflags="-s -w"` strips debugging symbols and symbol tables.
  - Runs as an unprivileged non-root user (`UID 65534` / `nobody`).
  - Read-only root filesystem compatible (`read_only: true`).

---

## 📁 Project Structure

```text
.
├── Dockerfile                  # Active production Dockerfile (v3 Scratch)
├── Dockerfile.v1-alpine        # Version 1: Alpine Linux Build (13.80 MB)
├── Dockerfile.v2-distroless     # Version 2: Google Distroless Build (7.61 MB)
├── Dockerfile.v3-scratch        # Version 3: Pure Scratch Build (5.74 MB)
├── docker-compose.yml          # Container configuration
├── go.mod                      # Go module definition
├── main.go                     # Production Go REST API
├── .dockerignore               # Docker build ignore rules
├── .gitignore                  # Git repository ignore rules
└── README.md                   # Project documentation
```

---

## 🛠️ The Job of the Dockerfile

The [Dockerfile](file:///Volumes/B/projects/devops%20go/Dockerfile) uses a **Multi-Stage Build** pattern to separate compilation from execution, keeping the final production image minimal and secure.

### Stage 1: The Builder Stage (`golang:1.24-alpine`)
1. **Environment Setup**: Sets `CGO_ENABLED=0` and `GOOS=linux` to produce a 100% statically linked Go binary without external C library dependencies.
2. **Dependency & Source Copy**: Sets `WORKDIR /app` and copies `go.mod` and source code.
3. **Static Binary Compilation**: Executes `go build -trimpath -ldflags="-s -w" -o /app/server .`.

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

| Method | Endpoint | Description | Sample Response |
| :--- | :--- | :--- | :--- |
| `GET` | `/` | Returns hardcoded Book JSON payload | `{"title":"The DevOps Handbook","author":"...","year_of_publication":2016,"number_of_pages":480}` |
| `GET` | `/healthz` | Liveness health check probe | `{"status":"UP"}` |

---

## 🚀 Quick Start Guide

### 1. Running Locally (Without Docker)
```bash
/usr/local/go/bin/go run main.go
```

### 2. Building & Running with Docker Compose
```bash
docker compose up -d --build
```

### 3. Testing the Endpoints
```bash
# Test Book JSON payload
curl http://localhost:8080/

# Test Healthcheck endpoint
curl http://localhost:8080/healthz
```
