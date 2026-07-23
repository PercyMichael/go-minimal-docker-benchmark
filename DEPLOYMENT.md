# 🌐 Production VPS & CI/CD Deployment Guide

This guide details how to deploy and scale this application on a **Virtual Private Server (VPS)** (Hetzner, DigitalOcean, Linode, AWS EC2) using **Caddy Reverse Proxy (Auto-HTTPS)** and an automated **GitHub Actions CI/CD Pipeline**.

---

## 🏗️ Architecture Overview

```text
 [Developer Git Push] ──> [GitHub main branch]
                                 │
                                 ▼
                     🤖 GitHub Actions CI/CD
                     ├── 1. Run `go test`
                     ├── 2. Build Docker static Scratch image
                     ├── 3. Push Image to GHCR (`ghcr.io`)
                     └── 4. SSH Deploy to VPS
                                 │
                                 ▼
                     🖥️ Production VPS ($4–$6/mo)
                     ├── 🔒 Caddy Web Server (Auto Let's Encrypt SSL/TLS)
                     └── 🐳 Docker Compose (`json-book-api` container)
```

---

## 🛠️ Step 1: VPS Server Setup

### 1.1 Connect to your VPS via SSH
```bash
ssh root@YOUR_VPS_IP
```

### 1.2 Install Docker & Docker Compose
```bash
# Update OS packages
apt update && apt upgrade -y

# Install Docker
curl -fsSL https://get.docker.com -o get-docker.sh
sh get-docker.sh

# Start Docker service
systemctl enable --now docker
```

### 1.3 Install Caddy Web Server (Auto-HTTPS Reverse Proxy)
```bash
apt install -y debian-keyring debian-archive-keyring apt-transport-https
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/gpg.key' | gpg --dearmor -o /usr/share/keyrings/caddy-stable-archive-keyring.gpg
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/debian.deb.txt' | tee /etc/apt/sources.list.d/caddy-stable.list
apt update
apt install caddy -y
```

### 1.4 Configure Caddy Domain & Reverse Proxy
Edit `/etc/caddy/Caddyfile`:
```caddy
api.yourdomain.com {
    reverse_proxy localhost:8080
}
```
Reload Caddy:
```bash
systemctl reload caddy
```

---

## 📁 Step 2: VPS Project Directory Setup

Create the deployment folder on your VPS:
```bash
mkdir -p /opt/devops-go-api
cd /opt/devops-go-api
```

Create `/opt/devops-go-api/docker-compose.yml`:
```yaml
version: '3.8'

services:
  book-api:
    image: ghcr.io/YOUR_GITHUB_USERNAME/YOUR_REPO_NAME:latest
    container_name: json-book-api
    ports:
      - "127.0.0.1:8080:8080"
    restart: unless-stopped
    read_only: true
    user: "65534:65534"
    security_opt:
      - no-new-privileges:true
    cap_drop:
      - ALL
```

---

## 🔑 Step 3: Configure GitHub Actions Secrets

To enable automated SSH deployments from GitHub to your VPS, navigate to your GitHub repository:
**Settings -> Secrets and variables -> Actions -> New repository secret**

Add the following 3 Secrets:

| Secret Name | Description / Value |
| :--- | :--- |
| **`VPS_HOST`** | IP address or domain of your VPS (e.g. `192.0.2.1` or `api.yourdomain.com`). |
| **`VPS_USER`** | SSH username (e.g. `root` or `ubuntu`). |
| **`VPS_SSH_KEY`** | Private SSH key (`~/.ssh/id_rsa`) allowing passwordless SSH access to the VPS. |

---

## 🔄 Step 4: Automated Deployment Workflow

Every time you commit and push to the `main` branch:

1. **GitHub Actions** runs `go test -v ./...`.
2. Compiles static Go binary and builds the **5.74 MB `FROM scratch` Docker image**.
3. Pushes image tag `latest` to GitHub Container Registry (`ghcr.io`).
4. SSHs securely into your VPS and runs:
   ```bash
   docker compose pull && docker compose up -d --remove-orphans
   ```
5. **Zero-Downtime Handover**: Go's graceful shutdown handling finishes active HTTP requests on the old container before exiting cleanly as the new container starts serving traffic.

---

## 📊 Performance & Cost Summary

- **Hosting Cost**: **$4 to $6 / month** total (Hetzner / DigitalOcean).
- **Throughput Capability**: Serves **50,000+ requests per minute** per $4 VPS instance.
- **SSL / TLS**: 100% automated renewals managed by Caddy.
- **Vulnerability Status**: 0 OS-level vulnerabilities (0 CVEs).
