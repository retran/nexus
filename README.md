# Nexus

**Private, self-hosted Integrated Operations Platform** — the central nervous
system for your household.

Nexus treats the family unit as a small-scale enterprise, unifying Business
Process Management (BPM) with Real-Time Asset Control through both digital
resource planning and physical smart home automation.

## Features

- 🏠 **Smart Home Integration** - Control physical devices through Home
  Assistant
- 📅 **Lifecycle Management** - Task tracking, calendar sync, and notifications
- � **Workflow Automation** - Temporal-based async job processing
- 🔐 **Self-Hosted** - Complete data ownership and privacy
- 🚀 **Modern Stack** - React, Go, PostgreSQL, GraphQL

## Quick Start

```bash
# Prerequisites: Docker, Task runner
brew install go-task

# Clone and setup
git clone https://github.com/retran/nexus.git
cd nexus
cp .env.example .env

# Add local DNS
echo "127.0.0.1 nexus.local api.nexus.local graphql.nexus.local traefik.nexus.local" | sudo tee -a /etc/hosts

# Start everything
task up

# Access at http://nexus.local
```

**📚 [Full Setup Guide →](docs/GETTING_STARTED.md)**

## Architecture

```
Frontend (React) → REST Gateway (Go) → GraphQL API (Go) → PostgreSQL 16
                                     ↓
                              Temporal ← Worker (Go)
```

**Services:**

- **UI** (`nexus.local`) - React + Refine.dev
- **Gateway** (`api.nexus.local`) - REST BFF with authentication
- **API Server** (`graphql.nexus.local`) - GraphQL internal API
- **Database** - PostgreSQL 16 with type-safe queries (sqlc)
- **Worker** - Temporal workflow processor
- **Traefik** - Reverse proxy for subdomain routing

**[Architecture Details →](docs/ARCHITECTURE.md)**

## Development

### Common Commands

```bash
# Development
task up                      # Start all services
task down                    # Stop all services
task logs                    # View logs (add -- <service>)

# Backend
task backend:build           # Build binaries
task backend:generate        # Generate code (sqlc, gqlgen, genqlient)
task backend:test            # Run tests

# Frontend
task frontend:dev            # Dev server (or visit http://nexus.local)
task frontend:test           # Run tests

# Database
task db:schema:diff -- name  # Create migration from schema changes
task db:migrate:apply        # Apply migrations
```

**[Development Guide →](docs/DEVELOPMENT.md)**

**[Contributing →](CONTRIBUTING.md)**

## Documentation

- [Getting Started](docs/GETTING_STARTED.md) - Setup and installation
- [Development Guide](docs/DEVELOPMENT.md) - Workflow and patterns
- [Architecture](docs/ARCHITECTURE.md) - System design
- [Contributing](CONTRIBUTING.md) - How to contribute

## License

Copyright 2025 Andrew Vasilyev

Licensed under the Apache License, Version 2.0
