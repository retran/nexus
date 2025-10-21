# Nexus AI Coding Agent Instructions

## Project Vision

**Nexus** is a private, self-hosted Integrated Operations Platform designed as
the central nervous system for a household. It treats the family unit as a
small-scale enterprise, unifying Business Process Management (BPM) with
Real-Time Asset Control through both digital resource planning and physical
smart home automation.

## Architecture Overview

Nexus follows a **Code as Single Source of Truth** philosophy with
**Schema-First Design** using gRPC/Protobuf for internal APIs. The system is a
multi-layered, distributed platform built on enterprise-grade open-source tools.

### Technology Stack

- **Frontend**: React + TypeScript with Refine meta-framework for rapid admin UI
  development
- **BFF/API Gateway**: Go REST↔gRPC gateway (Backend-for-Frontend pattern)
- **Core Services**: Go gRPC services with Protobuf for type-safe internal
  communication
- **Data Layer**: PostgreSQL 16 with sqlc for type-safe SQL code generation
- **Orchestration**: Temporal for complex asynchronous business processes
- **Shop Floor Gateway**: Home Assistant (Python) for physical asset management
- **Infrastructure**: Docker Compose with Infrastructure as Code (Ansible)
- **Security**: HashiCorp Vault for secrets, Tailscale mesh network, zero-trust
  architecture

## Development Workflow

### Deployment Architecture

- **Infrastructure Stack**: `docker-compose.infra.yaml` (PostgreSQL, Temporal,
  Traefik, Home Assistant, monitoring)
- **Application Stack**: `docker-compose.app.yaml` (ui, bff, api, worker
  containers)
- **Development**: `docker-compose.dev.yaml` provides 100% production parity
  with hot reload

### Starting the Stack

```bash
# Start all services
docker-compose -f docker-compose.dev.yaml up

# Run database migrations
docker-compose -f docker-compose.dev.yaml --profile tools run migrations
```

### Hot Reload Setup

- **Backend**: Uses Air for Go hot reload (`.air.toml` configuration)
- **Frontend**: Vite dev server with HMR
- **Volumes**: Source code mounted for live editing

## Frontend Architecture (Refine.dev)

### UI Component System

- **Base UI**: Radix UI primitives in `src/components/ui/`
- **Refine UI**: Custom components in `src/components/refine-ui/`
  - `layout/`: Main layout, header, sidebar components
  - `form/`: Auth forms with react-hook-form
  - `data-table/`: Custom table components with filtering/sorting
  - `buttons/`: CRUD action buttons following Refine patterns

### Key Patterns

- **Authentication**: Google OAuth with JWT tokens in localStorage
- **Data Provider**: Currently uses fake REST API (`@refinedev/simple-rest`)
- **Theme**: Dark/light mode with Tailwind CSS + shadcn/ui
- **State**: Refine handles CRUD state, React Hook Form for forms

### Critical Files

- `App.tsx`: Main Refine configuration with auth provider
- `components/refine-ui/layout/layout.tsx`: Main app layout structure
- Package management: Uses Yarn with `refine` CLI commands

## Backend Architecture (Go)

### Service Architecture

- **BFF Layer**: REST→gRPC gateway serving the frontend
- **Core Services**: gRPC microservices with Protobuf contracts
- **Data Access**: sqlc generates type-safe Go code from SQL
- **Orchestration**: Temporal workers for complex async workflows

### Current State

- **Entry Point**: `cmd/nexus/main.go` (currently hello world)
- **Module**: `github.com/retran/nexus/backend`
- **Target**: Implement gRPC services, Temporal workers, and database layer

### Database

- **Migrations**: SQL files in `postgres/migrations/` using go-migrate
- **Schema**: Currently minimal, needs expansion for full domain model
- **Type Safety**: Will use sqlc for generating Go code from SQL queries

## Development Conventions

### File Organization

- **Frontend**: Feature-based components with UI primitives separation
- **Backend**: Standard Go project layout with `cmd/` for executables
- **Docker**: Separate dev/prod Dockerfiles with dev optimizations

### Code Style

- **Copyright Headers**: All files include Apache 2.0 license headers
- **TypeScript**: Strict configuration with ESLint
- **Go**: Standard formatting with air for development

### Environment Configuration

- Uses `.env` file shared across all Docker services
- Environment variables for database connection, auth settings
- **Security**: HashiCorp Vault for centralized secret management
- **Networking**: Tailscale mesh for zero-trust admin access
- **Public Access**: Cloudflare Tunnel (no open router ports)

## Integration Points & Data Flows

### Frontend ↔ BFF Gateway

- **Protocol**: REST API consumed by Refine.dev data providers
- **Auth**: Google OAuth with OIDC, JWT tokens in localStorage
- **Current**: Using fake REST API, needs real BFF implementation

### BFF ↔ Core Services

- **Protocol**: gRPC with Protobuf for type-safe internal communication
- **Service Discovery**: Docker's internal DNS for service resolution
- **Load Balancing**: Traefik reverse proxy for external routing

### Physical Asset Integration

1. **Apple TV (HomeKit Hub)** → Voice commands via Siri
2. **Home Assistant** → Universal device translator and shop floor gateway
3. **Webhook Chain**: Siri → HomeKit → Home Assistant → Go BFF → Temporal
   workflows
4. **Mobile Alerts**: Home Assistant Companion App for push notifications

### External System Sync

- **iPaaS Pattern**: Google Calendar, Todoist, Notion as data sources
- **Orchestration**: Temporal workflows handle external API synchronization
- **Data Flow**: External APIs → Temporal → PostgreSQL → Frontend

## Key Commands

```bash
# Frontend development
cd frontend && yarn dev

# Backend development (with air)
cd backend && air

# Database operations
docker-compose -f docker-compose.dev.yaml --profile tools run migrations up
docker-compose -f docker-compose.dev.yaml --profile tools run migrations down

# Build production images
docker-compose build
```

## Development Priorities

When working on this codebase:

1. **Backend Development**: The Go backend is currently minimal - focus on
   implementing actual API endpoints
2. **Database Schema**: Expand migrations beyond the current minimal setup
3. **Frontend Integration**: Replace fake API with real backend endpoints
4. **Temporal Workflows**: Implement actual workflow definitions and worker
   processes
5. **Authentication**: Complete the Google OAuth integration with backend
   validation

## Production Architecture Notes

- **Zero Trust Security**: No open ports, Tailscale mesh network, Vault secrets
- **GitOps Workflow**: GitHub Actions with self-hosted runners
- **Business Continuity**: Automated PostgreSQL backups to Backblaze B2,
  quarterly recovery testing
- **Monitoring**: Uptime Kuma, Prometheus + cAdvisor + node-exporter
- **Infrastructure as Code**: Ansible playbooks for host provisioning
