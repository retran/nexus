# Development Guide

This guide covers the day-to-day development workflow, common tasks, and best
practices for contributing to Nexus.

## Prerequisites

Before you begin, ensure you have:

- **Docker Desktop** - Running and accessible
- **Task** - Task runner (`brew install go-task`)
- **Go 1.25+** - (optional, Docker handles this)
- **Node.js 20+** - (optional, Docker handles this)
- **Git** - For version control

## Initial Setup

If you haven't set up your environment yet, see
[Getting Started](GETTING_STARTED.md).

Quick recap:

```bash
# Clone, setup, and start
git clone https://github.com/retran/nexus.git
cd nexus
cp .env.example .env
echo "127.0.0.1 nexus.local api.nexus.local graphql.nexus.local traefik.nexus.local" | sudo tee -a /etc/hosts
task up
```

## Development Environment

### Services Overview

When you run `task up`, these services start:

| Service    | URL                        | Hot Reload  | Purpose              |
| ---------- | -------------------------- | ----------- | -------------------- |
| UI         | <http://nexus.local>         | ✅ Vite HMR | Frontend development |
| Gateway    | <http://api.nexus.local>     | ✅ Air      | REST API testing     |
| API Server | <http://graphql.nexus.local> | ✅ Air      | GraphQL Playground   |
| Traefik    | <http://traefik.nexus.local> | -           | Routing dashboard    |
| Temporal   | <http://localhost:8088>      | -           | Workflow monitoring  |
| PostgreSQL | localhost:5432             | -           | Database access      |
| Worker     | -                          | ✅ Air      | Background jobs      |

### Hot Reload

All services have hot reload enabled:

- **Frontend**: Vite HMR - changes appear instantly
- **Backend**: Air watches Go files and rebuilds automatically
- **Database**: Schema changes require migration generation

## Common Development Tasks

### Starting/Stopping

```bash
# Start all services
task up

# Stop all services
task down

# Restart everything
task restart

# View logs (all services)
task logs

# View logs (specific service)
task logs -- gateway

# Check service status
task ps
```

### Backend Development

#### Making Code Changes

1. Edit Go files in `backend/`
2. Air automatically rebuilds the service
3. Check logs: `docker logs -f nexus-gateway-1`

Example workflow:

```bash
# Edit a file
vim backend/internal/api/rest/handlers/users.go

# Watch it rebuild automatically
docker logs -f nexus-gateway-1

# Test the change
curl http://api.nexus.local/api/users
```

#### Building Locally (Optional)

```bash
# Build all binaries
task backend:build

# Build specific service
task backend:build:gateway
task backend:build:api-server
task backend:build:worker

# Run locally (outside Docker)
cd backend
./bin/gateway
```

#### Code Generation

Nexus uses code generation for type safety:

```bash
# Generate everything (database + GraphQL)
task backend:generate

# Generate database code (sqlc)
task backend:db:generate

# Generate GraphQL server
task backend:graphql:generate

# Generate GraphQL client
task backend:graphql:client:generate
```

**When to regenerate**:

- After changing SQL queries in `backend/internal/repository/postgres/queries/`
- After editing GraphQL schema in `backend/internal/api/graphql/schema.graphql`
- After updating GraphQL queries in `backend/internal/client/graphql/queries/`

#### Testing

```bash
# Run all tests
task backend:test

# Run with coverage
task backend:test:coverage

# Run specific package
cd backend
go test ./internal/api/rest/...

# Run with verbose output
go test -v ./...
```

#### Linting & Formatting

```bash
# Format code
task backend:format

# Lint code
task backend:lint

# Auto-fix linting issues
task backend:lint:fix
```

### Frontend Development

#### Making Code Changes

1. Edit files in `frontend/src/`
2. Browser updates instantly (Vite HMR)
3. Check console for errors

Example workflow:

```bash
# Edit a component
vim frontend/src/components/UserList.tsx

# Browser auto-refreshes
# Open http://nexus.local to see changes

# Check for errors
docker logs -f nexus-ui-1
```

#### Testing

```bash
# Run tests
task frontend:test

# Run with coverage
task frontend:test:coverage

# Watch mode (auto-run on changes)
task frontend:test:watch
```

#### Type Checking

```bash
# Check types
task frontend:type-check

# The build will also type-check
task frontend:build
```

#### Linting & Formatting

```bash
# Format code (Prettier)
task frontend:format

# Check formatting
task frontend:format:check

# Lint (ESLint)
task frontend:lint

# Auto-fix linting issues
task frontend:lint:fix
```

### Database Development

#### Schema Changes

Atlas is used for schema management with HCL as the source of truth.

**Workflow**:

1. Edit `postgres/schema.hcl`
2. Generate migration
3. Apply migration
4. Regenerate Go code

```bash
# 1. Edit schema
vim postgres/schema.hcl

# 2. Generate migration
task db:schema:diff -- add_user_avatar

# 3. Apply to database
task db:migrate:apply

# 4. Regenerate Go database code
task backend:db:generate
```

#### Creating Manual Migrations

Sometimes you need a migration that can't be generated:

```bash
# Create empty migration file
task db:migrate:new -- seed_admin_user

# Edit the file
vim postgres/migrations/TIMESTAMP_seed_admin_user.sql

# Update hash
task db:migrate:hash

# Apply
task db:migrate:apply
```

#### Common Database Tasks

```bash
# Check migration status
task db:migrate:status

# Open PostgreSQL shell
task db:shell

# Reset database (WARNING: destroys data)
task db:reset

# Format schema.hcl
task db:schema:fmt

# Lint SQL files
task db:lint

# Auto-fix SQL formatting
task db:lint:fix
```

#### Connecting to Database

```bash
# Via Task
task db:shell

# Via psql directly
psql postgres://admin:YOUR_PASSWORD@localhost:5432/nexus_db

# Via GUI (TablePlus, DBeaver, etc.)
# Host: localhost
# Port: 5432
# Database: nexus_db
# User: admin (from .env)
# Password: YOUR_PASSWORD (from .env)
```

### Temporal Workflows

#### Adding a New Workflow

1. Create workflow in `backend/internal/workflows/`
2. Create activities in `backend/internal/activities/`
3. Register in `backend/cmd/worker/main.go`

```go
// Example workflow
func MyWorkflow(ctx workflow.Context, input MyInput) error {
    ao := workflow.ActivityOptions{
        StartToCloseTimeout: 10 * time.Minute,
    }
    ctx = workflow.WithActivityOptions(ctx, ao)

    var result MyResult
    err := workflow.ExecuteActivity(ctx, MyActivity, input).Get(ctx, &result)
    return err
}

// Register in worker
w.RegisterWorkflow(workflows.MyWorkflow)
w.RegisterActivity(activities.MyActivity)
```

#### Testing Workflows

```bash
# View Temporal UI
open http://localhost:8088

# Check worker logs
docker logs -f nexus-worker-1

# Trigger workflow via API (future)
curl -X POST http://api.nexus.local/api/workflows/sync
```

## Development Patterns

### Adding a New API Endpoint

#### 1. Add Database Query (if needed)

```sql
-- backend/internal/repository/postgres/queries/users.sql
-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1 LIMIT 1;
```

```bash
task backend:db:generate
```

#### 2. Add GraphQL Schema (if needed)

```graphql
# backend/internal/api/graphql/schema.graphql
extend type Query {
  userByEmail(email: String!): User
}
```

```bash
task backend:graphql:generate
```

#### 3. Implement Resolver

```go
// backend/internal/api/graphql/resolvers/schema.resolvers.go
func (r *queryResolver) UserByEmail(ctx context.Context, email string) (*model.User, error) {
    user, err := r.Queries.GetUserByEmail(ctx, email)
    // ... conversion logic
}
```

#### 4. Add GraphQL Client Query

```graphql
# backend/internal/client/graphql/queries/users.graphql
query GetUserByEmail($email: String!) {
  userByEmail(email: $email) {
    id
    email
    name
  }
}
```

```bash
task backend:graphql:client:generate
```

#### 5. Add REST Handler

```go
// backend/internal/api/rest/handlers/users.go
func (h *UserHandlers) GetUserByEmail(w http.ResponseWriter, r *http.Request) {
    email := r.URL.Query().Get("email")
    resp, err := graphql.GetUserByEmail(r.Context(), h.client, email)
    // ... handle response
}
```

#### 6. Register Route

```go
// backend/internal/api/rest/server.go
mux.HandleFunc("GET /api/users/by-email", userHandlers.GetUserByEmail)
```

#### 7. Test

```bash
curl "http://api.nexus.local/api/users/by-email?email=user@example.com"
```

### Adding a Background Job

#### 1. Create Activity

```go
// backend/internal/activities/sync.go
func (a *Activities) SyncCalendarActivity(ctx context.Context, userID string) error {
    // Fetch from Google Calendar API
    // Store in database
    return nil
}
```

#### 2. Create Workflow

```go
// backend/internal/workflows/sync.go
func SyncCalendarWorkflow(ctx workflow.Context, userID string) error {
    err := workflow.ExecuteActivity(ctx, activities.SyncCalendarActivity, userID).Get(ctx, nil)
    return err
}
```

#### 3. Register in Worker

```go
// backend/cmd/worker/main.go
w.RegisterWorkflow(workflows.SyncCalendarWorkflow)
w.RegisterActivity(activities.SyncCalendarActivity)
```

#### 4. Trigger via API

```go
// In a REST handler
c, _ := client.Dial(client.Options{HostPort: "temporal:7233"})
workflowOptions := client.StartWorkflowOptions{
    ID:        "sync-calendar-" + userID,
    TaskQueue: "nexus-task-queue",
}
c.ExecuteWorkflow(context.Background(), workflowOptions, workflows.SyncCalendarWorkflow, userID)
```

## Debugging

### Backend Debugging

#### Using Logs

```bash
# Structured JSON logs
docker logs -f nexus-gateway-1 | jq

# Filter by level
docker logs -f nexus-gateway-1 | grep ERROR

# Follow multiple services
docker-compose -f docker-compose.dev.yaml logs -f gateway api-server
```

#### Using Delve (Go Debugger)

```bash
# Add to Dockerfile.gateway.dev
RUN go install github.com/go-delve/delve/cmd/dlv@latest

# Change CMD to
CMD ["dlv", "debug", "./cmd/gateway", "--headless", "--listen=:2345", "--api-version=2"]

# Expose port in docker-compose
ports:
  - "2345:2345"

# Connect from VS Code
# .vscode/launch.json
{
  "type": "go",
  "request": "attach",
  "mode": "remote",
  "remotePath": "/app",
  "port": 2345,
  "host": "localhost"
}
```

### Frontend Debugging

#### Browser DevTools

- React DevTools extension
- Network tab for API calls
- Console for errors

#### VS Code Debugging

```json
// .vscode/launch.json
{
  "type": "chrome",
  "request": "launch",
  "name": "Launch Chrome against localhost",
  "url": "http://nexus.local",
  "webRoot": "${workspaceFolder}/frontend/src"
}
```

### Database Debugging

```bash
# Enable query logging
# In docker-compose.dev.yaml, add to postgres environment:
- POSTGRES_INITDB_ARGS=-c log_statement=all

# View query logs
docker logs -f nexus-postgres-1 | grep -i "SELECT\|INSERT\|UPDATE\|DELETE"

# Analyze slow queries
task db:shell
# Then in psql:
SELECT * FROM pg_stat_statements ORDER BY total_exec_time DESC LIMIT 10;
```

## Troubleshooting

### "Cannot connect to Docker daemon"

```bash
# Start Docker Desktop
open -a Docker

# Or on Linux
sudo systemctl start docker
```

### "Port already in use"

```bash
# Find what's using the port
lsof -i :5432  # or :8080, :8081, etc.

# Kill the process
kill -9 <PID>

# Or change port in docker-compose.dev.yaml
```

### "Migration failed"

```bash
# Check current version
task db:migrate:status

# View migration content
cat postgres/migrations/TIMESTAMP_migration_name.sql

# Rollback (if safe)
docker-compose -f docker-compose.dev.yaml --profile tools run atlas migrate down

# Fix and reapply
task db:migrate:apply
```

### "Code generation fails"

```bash
# Ensure tools are installed
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
go install github.com/99designs/gqlgen@latest

# Clear generated files
rm -rf backend/internal/repository/postgres/*.sql.go
rm -rf backend/internal/api/graphql/generated.go

# Regenerate
task backend:generate
```

### "Hot reload not working"

```bash
# Check Air is running
docker logs nexus-gateway-1 | grep "watching"

# Restart service
docker-compose -f docker-compose.dev.yaml restart gateway

# Check volume mounts
docker inspect nexus-gateway-1 | grep Mounts -A 10
```

### "Frontend won't load"

```bash
# Check if UI service is running
docker ps | grep nexus-ui

# Check logs for errors
docker logs -f nexus-ui-1

# Verify Traefik routing
curl -v http://nexus.local

# Check /etc/hosts
grep nexus.local /etc/hosts
```

## Best Practices

### Code Style

- **Go**: Follow [Effective Go](https://golang.org/doc/effective_go.html)
- **TypeScript**: Follow
  [TypeScript guidelines](https://www.typescriptlang.org/docs/handbook/declaration-files/do-s-and-don-ts.html)
- **SQL**: Use lowercase keywords, snake_case for tables/columns

### Commit Messages

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```
feat: add user avatar upload
fix: correct CORS headers
docs: update development guide
refactor: extract validation logic
test: add tests for user service
chore: update dependencies
```

### Testing

- Write tests for new features
- Maintain >80% code coverage
- Test edge cases and error scenarios
- Use table-driven tests in Go

### Documentation

- Update docs when changing APIs
- Add comments for complex logic
- Keep README and guides in sync
- Document breaking changes

## Performance Tips

### Database

- Use indexes for frequently queried columns
- Use EXPLAIN ANALYZE to understand query plans
- Batch inserts when possible
- Use connection pooling (pgx handles this)

### Backend

- Use context for cancellation
- Pool expensive resources
- Cache frequently accessed data
- Use goroutines for concurrent operations

### Frontend

- Lazy load components
- Minimize bundle size
- Use React.memo for expensive components
- Optimize images

## Related Documentation

- [Getting Started](GETTING_STARTED.md) - Initial setup
- [Architecture](ARCHITECTURE.md) - System design
- [Contributing](../CONTRIBUTING.md) - Git workflow
