# Nexus

**Private, self-hosted Integrated Operations Platform** — the central nervous
system for your household.

Nexus treats the family unit as a small-scale enterprise, unifying Business
Process Management (BPM) with Real-Time Asset Control through both digital
resource planning and physical smart home automation.

## Quick Start

### Prerequisites

- Docker & Docker Compose
- Go 1.25.2+
- Node.js 23+ & Yarn
- Task (taskfile.dev)

### Bootstrap Development Environment

```bash
# 1. Generate .env file from template
./scripts/generate-env.sh

# 2. Bootstrap all services (one command does everything)
task setup:bootstrap
```

This single command will:

- Start Vault and seed all secrets
- Initialize databases with migrations
- Start all infrastructure services (Postgres, Redis, Temporal, Ory stack)
- Install dependencies for backend and frontend
- Start all application services

**Note**: All secrets are managed by HashiCorp Vault. The bootstrap process
automatically:

1. Initializes Vault in dev mode
2. Seeds secrets from .env into Vault KV storage
3. Services load secrets from Vault at startup

### Development Services

Once the environment is running, the following services are available:

> **Note**: Add these domains to your `/etc/hosts` for domain-based access. Run
> `task setup:hosts` to automatically configure:
>
> ```text
> 127.0.0.1 nexus.local api.nexus.local auth.nexus.local adminer.nexus.local redis.nexus.local mail.nexus.local temporal.nexus.local traefik.nexus.local grafana.nexus.local prometheus.nexus.local jaeger.nexus.local dozzle.nexus.local
> ```

### Production-like URLs (via Traefik + Oathkeeper)

> **Security**: All URLs below are protected by:
>
> - **Authentication**: Kratos SSO (cookie session)
> - **Authorization**: Admin tools require `admin` role in Keto
> - Built-in authentication is disabled in all admin tools

#### Application

- **Frontend**: <http://nexus.local>
- **API Gateway**: <http://api.nexus.local>
- **Auth (Kratos)**: <http://auth.nexus.local>

#### Admin & Monitoring (requires admin role)

- **Database (Adminer)**: <http://adminer.nexus.local>
- **Redis Commander**: <http://redis.nexus.local>
- **Email Testing (MailHog)**: <http://mail.nexus.local>
- **Workflows (Temporal UI)**: <http://temporal.nexus.local>
- **Reverse Proxy (Traefik)**: <http://traefik.nexus.local>
- **Grafana**: <http://grafana.nexus.local> - Dashboards and visualization
- **Prometheus**: <http://prometheus.nexus.local> - Metrics time-series database
- **Jaeger**: <http://jaeger.nexus.local> - Distributed tracing UI
- **Dozzle**: <http://dozzle.nexus.local> - Real-time Docker logs viewer

### Development/Debug URLs (Direct Access)

#### Application Services

- **Frontend Dev Server**: <http://localhost:5173>
- **API Gateway**: <http://localhost:8080>
- **Data API (GraphQL)**: <http://localhost:8081> - Direct GraphQL endpoint for
  debugging
- **System API (Internal)**: <http://localhost:8082> - Webhooks and internal
  operations

#### Admin & Monitoring Tools

- **Adminer**: <http://localhost:8089>
- **Redis Commander**: <http://localhost:8087>
- **MailHog**: <http://localhost:8025>
- **Temporal UI**: <http://localhost:8088>
- **Grafana**: <http://localhost:3001>
- **Prometheus**: <http://localhost:9090>
- **Jaeger**: <http://localhost:16686>
- **Dozzle**: <http://localhost:8086>
- **Traefik Dashboard**: <http://localhost:8888>

#### Ory Stack (Direct Access)

- **Kratos Public API**: <http://localhost:4433>
- **Kratos Admin API**: <http://localhost:4434>
- **Kratos Self-Service UI**: <http://localhost:4455>
- **Keto Read API**: <http://localhost:4466>
- **Keto Write API**: <http://localhost:4467>
- **Oathkeeper Proxy**: <http://localhost:4455>
- **Oathkeeper Admin API**: <http://localhost:4456>

#### Core Infrastructure

- **PostgreSQL**: `localhost:5432`
- **Redis**: `localhost:6379`
- **Vault**: <http://localhost:8200> (Token: `root`)
- **Temporal**: `localhost:7233`
- **SMTP (MailHog)**: `localhost:1025`

## License

Copyright 2025 Andrew Vasilyev Licensed under the Apache License, Version 2.0
