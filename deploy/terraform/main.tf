terraform {
  required_version = ">= 1.5.0"
  required_providers {
    digitalocean = {
      source  = "digitalocean/digitalocean"
      version = "~> 2.30"
    }
    cloudflare = {
      source  = "cloudflare/cloudflare"
      version = "~> 4.0"
    }
  }
}

variable "do_token" {
  description = "DigitalOcean API Personal Access Token"
  type        = string
  sensitive   = true
}

variable "cloudflare_api_token" {
  description = "Cloudflare API Token"
  type        = string
  sensitive   = true
}

variable "cloudflare_zone_id" {
  description = "Cloudflare Zone ID for domain management"
  type        = string
}

variable "domain_name" {
  description = "Production Domain Name (e.g., api.yourrideshare.ug)"
  type        = string
  default     = "api.yourrideshare.ug"
}

variable "region" {
  description = "DigitalOcean datacenter region (fra1 = Frankfurt, lowest latency to Uganda)"
  type        = string
  default     = "fra1"
}

provider "digitalocean" {
  token = var.do_token
}

provider "cloudflare" {
  api_token = var.cloudflare_api_token
}

# Private VPC Network for Secure Inter-Server Communication
resource "digitalocean_vpc" "app_vpc" {
  name   = "ride-share-vpc"
  region = var.region
}

# DigitalOcean SSH Key for deployment access
resource "digitalocean_ssh_key" "deployer" {
  name       = "ride-share-deployer-key"
  public_key = file("~/.ssh/id_rsa.pub")
}

# 1. Dedicated Go Web App API Droplet Server
resource "digitalocean_droplet" "api_server" {
  image              = "ubuntu-24-04-x64"
  name               = "ride-share-api-prod"
  region             = var.region
  size               = "s-2vcpu-4gb"
  vpc_uuid           = digitalocean_vpc.app_vpc.id
  monitoring         = true
  ipv6               = true
  ssh_keys           = [digitalocean_ssh_key.deployer.fingerprint]

  tags = ["ride-share", "production", "api"]
}

# 2. Dedicated Managed PostgreSQL 16 + PostGIS Database Cluster
resource "digitalocean_database_cluster" "postgres" {
  name       = "ride-share-postgres-db"
  engine     = "pg"
  version    = "16"
  size       = "db-s-1vcpu-1gb"
  node_count = 1
  region     = var.region
  vpc_uuid   = digitalocean_vpc.app_vpc.id

  tags = ["ride-share", "production", "database"]
}

# 3. Dedicated Managed Redis 7 Spatial GEO Cache Cluster
resource "digitalocean_database_cluster" "redis" {
  name       = "ride-share-redis-cache"
  engine     = "redis"
  version    = "7"
  size       = "db-s-1vcpu-1gb"
  node_count = 1
  region     = var.region
  vpc_uuid   = digitalocean_vpc.app_vpc.id

  tags = ["ride-share", "production", "cache"]
}

# Database Firewall: Allow DB connections ONLY from Go Web App via Private VPC
resource "digitalocean_database_firewall" "postgres_fw" {
  cluster_id = digitalocean_database_cluster.postgres.id

  rule {
    type  = "droplet"
    value = digitalocean_droplet.api_server.id
  }
}

# Redis Firewall: Allow Redis connections ONLY from Go Web App via Private VPC
resource "digitalocean_database_firewall" "redis_fw" {
  cluster_id = digitalocean_database_cluster.redis.id

  rule {
    type  = "droplet"
    value = digitalocean_droplet.api_server.id
  }
}

# DigitalOcean Cloud Web Firewall
resource "digitalocean_firewall" "web_firewall" {
  name = "ride-share-firewall"

  droplet_ids = [digitalocean_droplet.api_server.id]

  # Allow HTTP, HTTPS, and SSH inbound
  inbound_rule {
    protocol         = "tcp"
    port_range       = "80"
    source_addresses = ["0.0.0.0/0", "::/0"]
  }

  inbound_rule {
    protocol         = "tcp"
    port_range       = "443"
    source_addresses = ["0.0.0.0/0", "::/0"]
  }

  inbound_rule {
    protocol         = "tcp"
    port_range       = "22"
    source_addresses = ["0.0.0.0/0", "::/0"]
  }

  # Allow all outbound traffic
  outbound_rule {
    protocol              = "tcp"
    port_range            = "1-65535"
    destination_addresses = ["0.0.0.0/0", "::/0"]
  }

  outbound_rule {
    protocol              = "udp"
    port_range            = "1-65535"
    destination_addresses = ["0.0.0.0/0", "::/0"]
  }
}

# Cloudflare DNS Record (A Record pointing to Droplet IP with Cloudflare Proxy Enabled)
resource "cloudflare_record" "api_dns" {
  zone_id = var.cloudflare_zone_id
  name    = var.domain_name
  value   = digitalocean_droplet.api_server.ipv4_address
  type    = "A"
  proxied = true # Enables Cloudflare Free CDN, SSL & Anti-DDoS Shield
  ttl     = 1    # Auto TTL when proxied
}

output "server_ip" {
  description = "Public IPv4 address of production Web Droplet"
  value       = digitalocean_droplet.api_server.ipv4_address
}

output "postgres_uri" {
  description = "Managed PostgreSQL Private VPC Connection URI"
  value       = digitalocean_database_cluster.postgres.private_uri
  sensitive   = true
}

output "redis_uri" {
  description = "Managed Redis Private VPC Connection URI"
  value       = digitalocean_database_cluster.redis.private_uri
  sensitive   = true
}

output "domain_url" {
  description = "Production API Domain URL"
  value       = "https://${var.domain_name}"
}
