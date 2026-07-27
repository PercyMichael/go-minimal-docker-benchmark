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

# DigitalOcean SSH Key for deployment access
resource "digitalocean_ssh_key" "deployer" {
  name       = "ride-share-deployer-key"
  public_key = file("~/.ssh/id_rsa.pub")
}

# DigitalOcean Droplet (Ubuntu 24.04, 2 vCPU, 4GB RAM)
resource "digitalocean_droplet" "api_server" {
  image              = "ubuntu-24-04-x64"
  name               = "ride-share-api-prod"
  region             = var.region
  size               = "s-2vcpu-4gb"
  monitoring         = true
  ipv6               = true
  ssh_keys           = [digitalocean_ssh_key.deployer.fingerprint]

  tags = ["ride-share", "production"]
}

# DigitalOcean Cloud Firewall
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
  description = "Public IPv4 address of production Droplet"
  value       = digitalocean_droplet.api_server.ipv4_address
}

output "domain_url" {
  description = "Production API Domain URL"
  value       = "https://${var.domain_name}"
}
