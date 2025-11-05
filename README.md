# Nexus

**Private, self-hosted Integrated Operations Platform** — the central nervous
system for your household.

Nexus treats the family unit as a small-scale enterprise, unifying Business
Process Management (BPM) with Real-Time Asset Control through both digital
resource planning and physical smart home automation.

## Quick Start

### Bootstrap Development Environment

```bash
# 1. Setup local domains (requires sudo)
task hosts:setup

# 2. Bootstrap all services
task bootstrap:dev
```

This will set up all infrastructure, databases, and services.

### Development Services

Once the environment is running, the following services are available:

> **Note**: Add these domains to your `/etc/hosts` for domain-based access:
>
> ```text
> 127.0.0.1 nexus.local api.nexus.local auth.nexus.local adminer.nexus.local redis.nexus.local mail.nexus.local temporal.nexus.local traefik.nexus.local
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

### Common Tasks

```bash
# Setup local domains
task hosts:setup      # Add *.nexus.local to /etc/hosts
task hosts:remove     # Remove *.nexus.local from /etc/hosts

# Check health of all services
task health

# View logs
task logs              # All services
task logs:backend      # Backend services only
task logs:infra        # Infrastructure only

# Run tests
task test

# Lint and format
task lint
task fmt

# Stop all services
task dev:stop

# Destroy everything (clean slate)
task destroy
```

## License

Copyright 2025 Andrew Vasilyev Licensed under the Apache License, Version 2.0
