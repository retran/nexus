# Nexus Architecture

This document describes the overall system architecture, design patterns, and
technical decisions.

## System Overview

Nexus is a monorepo containing multiple services that work together to provide a
private household operations platform.

```
┌─────────────────────────────────────────────────────────────────┐
│                          Internet                                │
└──────────────────────┬──────────────────────────────────────────┘
                       │
              ┌────────▼────────┐
              │  Cloudflare     │ (Production only)
              │  Tunnel         │
              └────────┬────────┘
                       │
              ┌────────▼────────┐
              │    Traefik      │  Reverse Proxy / API Gateway
              │  (Port 80/443)  │  - Subdomain routing
              └─────────────────┘  - SSL termination
                       │          - Rate limiting
         ┌─────────────┼─────────────┐
         │             │             │
    ┌────▼───┐    ┌───▼────┐   ┌───▼────┐
    │   UI   │    │Gateway │   │API Srv │
    │ :5173  │    │ :8080  │   │ :8081  │
    │(React) │    │(Go BFF)│   │(GraphQL│
    └────────┘    └───┬────┘   └───┬────┘
                      │            │
                      │    ┌───────┴────────┐
                      │    │                │
                  ┌───▼────▼───┐       ┌───▼────┐
                  │ PostgreSQL │◄──────┤ Worker │
                  │   :5432    │       │  (Go)  │
                  └────────────┘       └───┬────┘
                                           │
                                      ┌────▼────┐
                                      │Temporal │
                                      │ :7233   │
                                      └─────────┘
```

## Service Responsibilities

### Frontend UI (React + Refine.dev)

- **Port**: 5173 (dev), exposed via Traefik
- **URL**: `http://nexus.local`
- **Purpose**: User interface for all household operations
- **Tech**: React 18, TypeScript, Vite, Refine.dev, shadcn/ui
- **Features**:
  - Google OAuth authentication
  - Responsive design (mobile-first)
  - Real-time updates via GraphQL subscriptions (future)
  - Offline-capable PWA (future)

### Gateway (REST BFF)

- **Port**: 8080, exposed via Traefik
- **URL**: `http://api.nexus.local`
- **Purpose**: Backend-for-Frontend pattern, translates REST→GraphQL
- **Tech**: Go 1.25, Chi router, genqlient GraphQL client
- **Responsibilities**:
  - REST API for frontend
  - Authentication/authorization middleware
  - Request validation
  - Response transformation
  - CORS handling
  - Rate limiting

### API Server (GraphQL)

- **Port**: 8081, internal only (exposed in dev)
- **URL**: `http://graphql.nexus.local` (dev only)
- **Purpose**: Internal GraphQL API for data operations
- **Tech**: Go 1.25, gqlgen
- **Responsibilities**:
  - GraphQL schema and resolvers
  - Business logic
  - Database queries via sqlc
  - Data validation
  - Authorization (future: fine-grained permissions)

### Worker (Temporal)

- **Port**: N/A (internal)
- **Purpose**: Background job processing and workflows
- **Tech**: Go 1.25, Temporal SDK
- **Responsibilities**:
  - External API sync (Google Calendar, Todoist, etc.)
  - Scheduled tasks
  - Long-running operations
  - Retry logic and error handling
  - State management

### PostgreSQL

- **Port**: 5432
- **Purpose**: Primary data store
- **Tech**: PostgreSQL 16
- **Features**:
  - JSONB for flexible data
  - Full-text search
  - Triggers for updated_at timestamps
  - Row-level security (future)

### Temporal

- **Port**: 7233 (gRPC), 8088 (Web UI)
- **Purpose**: Workflow orchestration
- **Tech**: Temporal Server
- **Features**:
  - Durable execution
  - Automatic retries
  - Workflow versioning
  - Activity timeout handling

### Traefik

- **Port**: 80 (HTTP), 8090 (dashboard)
- **Purpose**: Reverse proxy and API gateway
- **Tech**: Traefik v3.0
- **Features**:
  - Subdomain-based routing
  - Automatic service discovery (Docker labels)
  - SSL/TLS termination (production)
  - Rate limiting and circuit breakers

## Data Flow

### Read Operation (User views data)

```
1. User → nexus.local → Traefik
2. Traefik → UI (React serves HTML/JS)
3. UI → api.nexus.local/users → Traefik → Gateway
4. Gateway → graphql:8081 (GraphQL query) → API Server
5. API Server → PostgreSQL (SQL via sqlc)
6. Response flows back: PostgreSQL → API → Gateway → UI
```

### Write Operation (User creates data)

```
1. UI → POST api.nexus.local/users → Gateway
2. Gateway validates → GraphQL mutation → API Server
3. API Server validates → PostgreSQL INSERT
4. API Server triggers Temporal workflow (if needed)
5. Worker picks up workflow, executes activities
6. Response returns to UI
```

### Background Job (Scheduled sync)

```
1. Temporal schedule triggers workflow
2. Worker executes workflow steps:
   a. Fetch data from external API (activity)
   b. Transform data (workflow logic)
   c. Store in PostgreSQL via API Server (activity)
3. On failure: automatic retry with backoff
4. Send notification via Home Assistant (activity)
```

## Design Patterns

### Backend for Frontend (BFF)

- Gateway tailored for frontend needs
- Aggregates multiple GraphQL queries
- Transforms data to frontend-friendly format
- Handles authentication before reaching internal APIs

### Code Generation

- **sqlc**: SQL → type-safe Go code
- **gqlgen**: GraphQL schema → Go resolvers
- **genqlient**: GraphQL queries → Go client
- Benefits: Type safety, reduced boilerplate, compile-time errors

### Schema-First Design

- Database schema in `postgres/schema.hcl` (single source of truth)
- GraphQL schema in `backend/internal/api/graphql/schema.graphql`
- Migrations generated from schema diffs
- Code generated from schemas

### Repository Pattern

- `internal/repository/postgres/` contains all database access
- Clean separation: API layer → Repository → Database
- Easy to mock for testing
- Can swap database implementation

### Middleware Stack (Gateway)

```go
Handler
  ← Recovery (panic handling)
    ← Logger (request/response logging)
      ← CORS (cross-origin requests)
        ← Auth (JWT validation) [future]
          ← Rate Limiter [future]
```

## Technology Choices

### Why Go?

- Excellent concurrency (goroutines)
- Fast compilation and execution
- Strong typing
- Great ecosystem for backend services
- Easy deployment (single binary)

### Why GraphQL (internal)?

- Strongly typed schema
- Flexible queries (no over-fetching)
- Excellent tooling (gqlgen, GraphQL Playground)
- Type generation for clients

### Why REST (external)?

- Simpler for frontend developers
- Better caching (HTTP semantics)
- Easier to secure (API keys, rate limiting)
- Standards-compliant

### Why PostgreSQL?

- ACID compliance
- Rich data types (JSONB, arrays)
- Full-text search
- Battle-tested reliability
- Great performance

### Why Temporal?

- Durable workflows (survives crashes)
- Built-in retry logic
- Workflow versioning
- Excellent observability
- State management for long-running tasks

### Why Refine.dev?

- Rapid admin UI development
- Built-in CRUD operations
- Extensible components
- Great TypeScript support
- Works with shadcn/ui

## Security Architecture

### Development

- All services accessible via subdomains
- No authentication required (local environment)
- CORS allows `nexus.local` origin

### Production (Future)

- **External Access**: Only UI exposed via Cloudflare Tunnel
- **Internal Network**: All other services on private Docker network
- **Authentication**: Google OAuth + JWT tokens
- **Authorization**: Role-based access control (RBAC)
- **Secrets**: HashiCorp Vault for centralized secrets
- **Network**: Tailscale mesh for admin access
- **TLS**: Automatic via Cloudflare
- **Rate Limiting**: Traefik middleware
- **Database**: Connection pooling, prepared statements

## Deployment

### Development

```bash
# Docker Compose with hot reload
task up
# Services: traefik, postgres, temporal, api-server, gateway, worker, ui
```

### Production (Future)

- **Host**: Self-hosted server (Intel NUC, Raspberry Pi 4, etc.)
- **OS**: Ubuntu Server with Ansible provisioning
- **Containers**: Docker Compose (not Kubernetes - too complex for home)
- **Networking**: Tailscale mesh + Cloudflare Tunnel
- **Backups**: PostgreSQL → Backblaze B2 (automated)
- **Monitoring**: Uptime Kuma, Prometheus, Grafana
- **Logs**: Centralized via Loki
- **CI/CD**: GitHub Actions with self-hosted runner

## Scalability Considerations

### Current (Single Node)

- Suitable for household use (~1-10 users)
- All services on one machine
- PostgreSQL handles thousands of requests/sec
- Temporal handles hundreds of workflows

### Future (If Needed)

- **Database**: Read replicas for reporting
- **Gateway**: Multiple instances behind Traefik
- **Worker**: Scale workers horizontally
- **Temporal**: Separate Temporal cluster
- **Cache**: Redis for session/rate limit data

## Observability

### Logging

- **Format**: Structured JSON logs
- **Levels**: DEBUG (dev), INFO (production)
- **Destination**: stdout → Docker logs
- **Aggregation**: Loki (future)

### Metrics

- **Temporal**: Built-in metrics and Web UI
- **PostgreSQL**: pg_stat_statements
- **Application**: Prometheus metrics (future)
- **Traefik**: Access logs and metrics

### Tracing

- **Future**: OpenTelemetry for distributed tracing
- **Useful for**: Debugging slow requests across services

## Directory Structure

```
nexus/
├── backend/                    # Go services
│   ├── cmd/                   # Executables
│   │   ├── api-server/        # GraphQL API
│   │   ├── gateway/           # REST BFF
│   │   └── worker/            # Temporal worker
│   ├── internal/              # Internal packages
│   │   ├── api/               # API implementations
│   │   ├── client/            # External clients
│   │   ├── repository/        # Data access
│   │   ├── activities/        # Temporal activities
│   │   └── workflows/         # Temporal workflows
│   └── Taskfile.yml           # Backend tasks
├── frontend/                   # React UI
│   ├── src/
│   │   ├── components/        # React components
│   │   ├── pages/             # Page components
│   │   └── hooks/             # Custom hooks
│   └── Taskfile.yml           # Frontend tasks
├── postgres/                   # Database
│   ├── schema.hcl             # Atlas schema (source of truth)
│   └── migrations/            # SQL migrations
├── docs/                       # Documentation
├── docker-compose.dev.yaml    # Development services
└── Taskfile.yml               # Root tasks
```

## Future Enhancements

### Phase 1 (MVP - Current)

- ✅ Basic CRUD operations
- ✅ User management
- ✅ Database schema
- ✅ Development environment

### Phase 2 (Home Integration)

- Home Assistant integration
- Device control workflows
- Automation rules
- Push notifications

### Phase 3 (Lifecycle Management)

- Google Calendar sync
- Todoist integration
- Recurring task management
- Location-based triggers

### Phase 4 (Advanced Features)

- Multi-user with permissions
- Mobile app (React Native)
- Voice commands via Siri
- AI-powered suggestions

## Related Documentation

- [Getting Started](GETTING_STARTED.md) - Setup guide
- [Development](DEVELOPMENT.md) - Development workflow
- [Contributing](../CONTRIBUTING.md) - Contribution guidelines
