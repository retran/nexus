# Getting Started

This guide will help you set up Nexus for local development in under 10 minutes.

## Prerequisites

Before you begin, make sure you have:

- **Docker Desktop** - Download from [docker.com](https://docker.com) and make
  sure it's running
- **Task** - Task runner for running commands
  - macOS: `brew install go-task`
  - Linux:
    `sh -c "$(curl --location https://taskfile.dev/install.sh)" -- -d -b ~/.local/bin`
- **5GB free disk space** - For Docker images and containers

## Quick Start

### 1. Clone Repository

```bash
git clone https://github.com/retran/nexus.git
cd nexus
```

### 2. Configure Environment

```bash
# Create .env file from template
cp .env.example .env

# Edit .env and change at minimum:
# - POSTGRES_PASSWORD (choose a strong password)
nano .env  # or use your preferred editor
```

**Important**: Review the `.env` file and update any values needed. The defaults
work for local development, but you should change `POSTGRES_PASSWORD`.

### 3. Set Up Local DNS

Nexus uses subdomain routing for clean service separation. Add these to your
hosts file:

```bash
echo "127.0.0.1 nexus.local api.nexus.local graphql.nexus.local traefik.nexus.local" | sudo tee -a /etc/hosts
```

### 4. Start Services

```bash
# This will build images and start all services
# First run takes 2-3 minutes
task up
```

You should see Docker building images and starting containers for:

- PostgreSQL database
- Traefik reverse proxy
- Backend services (gateway, api-server, worker)
- Frontend UI
- Temporal workflow engine

### 5. Verify Installation

Open your browser and check:

- Frontend: <http://nexus.local>
- GraphQL Playground: <http://graphql.nexus.local>
- Traefik Dashboard: <http://traefik.nexus.local>

Or use the command line:

```bash
# Check API health
curl http://api.nexus.local/health

# View running containers
docker ps --filter "name=nexus-"

# Check logs
task logs
```

You should see 7 running containers:

- `nexus-postgres-1`
- `nexus-traefik-1`
- `nexus-gateway-1`
- `nexus-api-server-1`
- `nexus-worker-1`
- `nexus-ui-1`
- `nexus-temporal-1`

## Services Overview

| Service    | URL                          | Description                     |
| ---------- | ---------------------------- | ------------------------------- |
| Frontend   | <http://nexus.local>         | React UI with hot reload        |
| REST API   | <http://api.nexus.local>     | Backend gateway (BFF pattern)   |
| GraphQL    | <http://graphql.nexus.local> | Internal API server             |
| Traefik    | <http://traefik.nexus.local> | Reverse proxy dashboard         |
| PostgreSQL | `localhost:5432`             | Database (connect via psql/GUI) |
| Temporal   | `localhost:8088`             | Workflow engine UI              |

## Making Your First Change

All services have hot reload enabled, so changes appear immediately.

### Backend Change

```bash
# Edit any Go file
vim backend/cmd/gateway/main.go

# Air automatically rebuilds
# Watch the rebuild: docker logs -f nexus-gateway-1
```

### Frontend Change

```bash
# Edit any React file
vim frontend/src/App.tsx

# Browser auto-refreshes (Vite HMR)
# No restart needed!
```

### Database Change

```bash
# Edit schema
vim postgres/schema.hcl

# Generate and apply migration
task db:schema:diff -- my_change_name
task db:migrate:apply

# Regenerate Go code
task backend:db:generate
```

## Essential Commands

```bash
# Start/stop services
task up              # Start everything
task down            # Stop everything
task restart         # Restart all services
task ps              # Show running containers

# View logs
task logs            # All services
task logs -- gateway # Specific service

# Development tasks
task backend:build   # Build Go services
task frontend:test   # Run frontend tests
task backend:test    # Run backend tests

# Database operations
task db:shell        # Open PostgreSQL shell
task db:migrate:status    # Check migration status
task db:reset        # Reset database (WARNING: destroys data)

# See all available commands
task --list-all
```

## Troubleshooting

### Docker daemon not running

```bash
# Start Docker Desktop (macOS)
open -a Docker

# Or start docker service (Linux)
sudo systemctl start docker
```

### Port already in use

If you see "port is already allocated":

```bash
# Find process using the port
lsof -i :5432  # or :8080, :8081, etc.

# Kill the process
kill -9 <PID>
```

### Cannot resolve nexus.local

```bash
# Check /etc/hosts has the entries
grep nexus.local /etc/hosts

# If missing, add them
echo "127.0.0.1 nexus.local api.nexus.local graphql.nexus.local traefik.nexus.local" | sudo tee -a /etc/hosts
```

### Services won't start

```bash
# Check logs for errors
task logs

# Try rebuilding from scratch
task down
docker system prune -f
task up
```

### Database connection fails

- Verify PostgreSQL is running: `docker ps | grep postgres`
- Check password in `.env` matches what you're using
- Try connecting: `task db:shell`

## Next Steps

Now that you have Nexus running:

1. **Read Development Guide**: [docs/DEVELOPMENT.md](DEVELOPMENT.md) -
   Day-to-day development workflows
2. **Understand Architecture**: [docs/ARCHITECTURE.md](ARCHITECTURE.md) - System
   design and patterns
3. **Contribution Guidelines**: [CONTRIBUTING.md](../CONTRIBUTING.md) - Git
   workflow and code standards

## Need Help?

- Check existing [GitHub Issues](https://github.com/retran/nexus/issues)
- Review documentation in [docs/](.) folder
- Read inline code comments for implementation details

Happy coding! 🚀
