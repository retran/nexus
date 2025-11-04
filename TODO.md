# PRODUCTION CONFIGURATION REFACTORING PLAN

## Context & Goal

Transform current dev-focused configuration into production-ready setup for
single Mac Mini deployment at home, while preserving ability to run dev
environment on developer machines.

## Current State Analysis

### Development-Centric Issues

1. **Port Exposure**: PostgreSQL (5432), Redis (6379), Vault (8200) exposed to
   host
2. **Hot Reload**: Air for Go, Vite HMR for frontend - unnecessary overhead in
   prod
3. **Dev Dockerfiles**: Volume mounts for live code reload
4. **Insecure Settings**: Traefik dashboard without auth, Vault dev mode, debug
   logging
5. **Local Domains**: `.local` TLD with /etc/hosts, not real domains
6. **Secrets**: Stored in .env file, not production secret management
7. **tmpfs for certs**: Ephemeral certificate storage (good for dev, bad for
   restarts)
8. **Single Compose**: No separation between infra and application stacks

### Production Requirements (Mac Mini @ Home)

1. **Real Domains**: Actual domain names (e.g., nexus.yourdomain.com)
2. **HTTPS/TLS**: Let's Encrypt certificates via Traefik ACME
3. **External Access**: Cloudflare Tunnel (no open router ports)
4. **Persistent Certs**: Volume-backed certificate storage
5. **Production Images**: Multi-stage builds, no dev tooling
6. **Secure Vault**: File backend with real seal keys (not dev mode)
7. **Secrets Management**: Vault as single source of truth
8. **Monitoring**: Prometheus, Grafana, Uptime Kuma
9. **Backups**: Automated PostgreSQL backups to Backblaze B2
10. **Zero Trust**: Tailscale mesh network for admin access

## Proposed Structure

```text
docker-compose.yaml              # Main production compose
docker-compose.override.yaml     # Local dev overrides (git-ignored)
docker-compose.dev.example.yaml  # Template for dev overrides
.env.prod.example               # Production env template
.env.dev.example                # Development env template (current)
backend/
  Dockerfile                    # Production multi-stage build
  Dockerfile.dev                # Development (current)
frontend/
  Dockerfile                    # Production multi-stage build
  Dockerfile.dev                # Development (current)
ory/
  prod/                         # Production configs
    kratos.yml
    keto.yml
    oathkeeper.yml
    access-rules.yml
  dev/                          # Development configs (current)
vault/
  config.prod.hcl              # Production Vault config
  config.dev.hcl               # Development Vault config (current)
```

## Implementation Plan

### Phase 1: Production Docker Images (5 commits)

**Goal**: Create production-ready multi-stage Dockerfiles

1. **backend/Dockerfile (multi-stage for all services)**
   - Stage 1: Build (golang:1.25-alpine)
   - Stage 2: Runtime (alpine:latest)
   - No Air, no volume mounts, static binaries
   - Separate targets: gateway, internal-api, worker, data-api

2. **frontend/Dockerfile (production)**
   - Stage 1: Build (node:22-alpine, yarn build)
   - Stage 2: Runtime (nginx:alpine)
   - Static files only, no Vite dev server

3. **Update docker-compose.yaml for prod images**
   - Use production Dockerfiles
   - Remove volume mounts for source code
   - Keep config mounts (ory, vault agent)

4. **Create docker-compose.override.yaml template**
   - Volume mounts for dev
   - Port exposures for dev
   - Override with .dev Dockerfiles
   - Air/HMR commands

5. **Test both modes**
   - `docker-compose up` = production
   - `docker-compose -f docker-compose.yaml -f docker-compose.override.yaml up`
     = dev

### Phase 2: Configuration Split (4 commits)

1. **Create ory/prod/ configs**
   - kratos.yml: production URLs, secure cookies, real SMTP
   - keto.yml: same as dev (stateless)
   - oathkeeper.yml: production URLs
   - access-rules.yml: same logic, different domains
2. **Create vault/config.prod.hcl**
   - TLS listener (not tcp with tls_disable)
   - File storage with proper paths
   - No dev mode, requires seal keys
   - Cluster configuration for HA readiness
3. **Environment file structure**
   - .env.prod.example with placeholders
   - Document secrets that must be in Vault
   - Real domain examples
   - ACME/Let's Encrypt settings
4. **Update Taskfile.yml**
   - `task up:prod` vs `task up:dev`
   - Separate compose file references
   - Production checks (no debug, sealed vault, etc.)

### Phase 3: External Access & Security (3 commits)

1. **Traefik ACME configuration**
   - Add ACME resolver for Let's Encrypt
   - Certificate storage volume
   - HTTP-01 or DNS-01 challenge (DNS via Cloudflare API)
   - Secure dashboard with SSO
2. **Cloudflare Tunnel setup**
   - Add cloudflared service
   - Tunnel configuration
   - Route specific hosts through tunnel
   - Keep admin interfaces on Tailscale only
3. **Remove debug/dev features from prod**
   - No exposed ports except 80/443 to tunnel
   - Log level INFO/WARN
   - Disable Kratos --dev flag
   - Disable Traefik --api.insecure

### Phase 4: Persistence & Reliability (3 commits)

1. **Certificate persistence**
   - Named volumes for Vault certs (not tmpfs)
   - Graceful cert rotation
   - Backup strategy for Vault storage
2. **PostgreSQL backups**
   - Add backup sidecar service
   - Cron-based pg_dump
   - Upload to Backblaze B2
   - Retention policy
3. **Monitoring stack**
   - Prometheus service
   - Grafana with SSO
   - cAdvisor for container metrics
   - node-exporter for host metrics
   - Uptime Kuma for uptime checks

### Phase 5: Deployment Automation (2 commits)

1. **GitOps preparation**
   - GitHub Actions workflow structure
   - Self-hosted runner setup docs
   - Image build and push
   - Deployment trigger
2. **Ansible playbooks (skeleton)**
   - Host provisioning playbook
   - Docker installation
   - Service deployment
   - Backup configuration
   - Tailscale setup

## Success Criteria

- [ ] `docker-compose up` runs production config on Mac Mini
- [ ] Developer can use
      `docker-compose -f docker-compose.yaml -f docker-compose.override.yaml up`
      for dev
- [ ] All secrets in Vault (no .env in production)
- [ ] HTTPS via Let's Encrypt working
- [ ] Cloudflare Tunnel provides external access
- [ ] Admin interfaces only accessible via Tailscale
- [ ] Automated PostgreSQL backups to B2
- [ ] Monitoring dashboards accessible
- [ ] Can deploy updates via git push + GitHub Actions

## Non-Goals (For Later)

- Kubernetes migration
- Multi-node HA setup
- Advanced observability (traces, logs aggregation)
- Disaster recovery automation
- Home Assistant integration details

---------------- NEXT commits are for reference, do not implement them

## Commit 31: Add structured logging library (slog)

**Research**:

- Review `slog` package documentation: <https://pkg.go.dev/log/slog>
- Check JSON handler configuration and best practices
- Review context-aware logging patterns

**Files**:

- `backend/internal/logger/logger.go`
- `backend/internal/logger/logger_test.go`
- `backend/go.mod`

**Changes**:

- Create shared logging library using `slog.JSONHandler`
- Configure automatic `service` field injection
- Write to `stdout` for Docker log collection
- Support context propagation for request tracing

**Test**: Logger outputs valid JSON with required fields

**Commit**: `feat(logging): step-31 - add structured logging library (slog)`

---

## Commit 32: Integrate structured logging in internal-api

**Research**:

- Review structured logging best practices for REST APIs
- Check middleware patterns for request logging

**Files**:

- `backend/cmd/internal-api/main.go`
- `backend/internal/api/internal/handlers/*.go`
- `backend/internal/api/internal/middleware/logging.go`

**Changes**:

- Replace all `log.Printf` with `slog` calls
- Add logging middleware for all HTTP requests
- Add structured fields: `user_id`, `handler_name`, `error`
- Log webhook events with full context

**Test**: All logs are valid JSON with structured fields

**Commit**:
`refactor(logging): step-32 - integrate structured logging in internal-api`

---

## Commit 33: Integrate structured logging in data-api

**Research**:

- Review GraphQL operation logging patterns
- Check `gqlgen` integration with custom loggers

**Files**:

- `backend/cmd/data-api/main.go`
- `backend/internal/api/graphql/resolvers/*.go`

**Changes**:

- Replace all logging with `slog`
- Add GraphQL context: `operation_name`, `error_code`, `resolver`
- Log query execution time
- Log GraphQL errors with full stack trace

**Test**: GraphQL operations logged with structured context

**Commit**:
`refactor(logging): step-33 - integrate structured logging in data-api`

---

## Commit 34: Integrate structured logging in gateway

**Research**:

- Review HTTP middleware logging patterns
- Check latency measurement best practices

**Files**:

- `backend/cmd/gateway/main.go`
- `backend/internal/api/rest/middleware/logging.go`
- `backend/internal/api/rest/handlers/*.go`

**Changes**:

- Replace all logging with `slog`
- Add HTTP request logger middleware
- Log fields: `method`, `path`, `status_code`, `latency_ms`
- Log all outgoing service calls with timing

**Test**: All HTTP requests logged with timing and status

**Commit**:
`refactor(logging): step-34 - integrate structured logging in gateway`

---

## Commit 35: Integrate structured logging in worker

**Research**:

- Review Temporal SDK logging integration
- Check `slog` adapter patterns for Temporal

**Files**:

- `backend/cmd/worker/main.go`
- `backend/internal/workflows/*.go`
- `backend/internal/activities/*.go`

**Changes**:

- Replace all logging with `slog`
- Integrate `slog` adapter for Temporal SDK
- Add workflow context: `workflow_id`, `run_id`, `activity_name`
- Log workflow state transitions

**Test**: Temporal workflows logged with structured context

**Commit**:
`refactor(logging): step-35 - integrate structured logging in worker`

---

## Commit 36: Add VictoriaMetrics core services

**Research**:

- Review VictoriaMetrics documentation: <https://docs.victoriametrics.com/>
- Check VictoriaLogs setup: <https://docs.victoriametrics.com/victorialogs/>
- Review VictoriaTraces (OpenTelemetry compatible)

**Files**:

- `docker-compose.dev.yaml`
- `observability/vm-config.yaml`

**Changes**:

- Add `victoriametrics` service (port 8428) - TSDB
- Add `victorialogs` service (port 9428) - Logs
- Add `victoriatraces` service (port 4318) - Traces
- Configure retention policies for dev environment

**Test**: All Victoria services start and accept data

**Commit**: `feat(obs): step-36 - add VictoriaMetrics core services`

---

## Commit 37: Add Promtail with JSON log parsing

**Research**:

- Review Promtail configuration:
  <https://grafana.com/docs/loki/latest/send-data/promtail/>
- Check `pipeline_stages` for JSON parsing
- Review Docker log collection patterns

**Files**:

- `docker-compose.dev.yaml`
- `observability/promtail-config.yaml`

**Changes**:

- Add `promtail` service for log shipping
- Configure Docker log collection
- Add `pipeline_stages` with `json` parser
- Extract fields: `level`, `service`, `error`, `user_id`
- Ship logs to VictoriaLogs

**Test**: Promtail parses JSON logs and sends to VictoriaLogs

**Commit**: `feat(obs): step-37 - add Promtail log shipping with JSON parsing`

---

## Commit 38: Add metric exporters

**Research**:

- Review postgres-exporter:
  <https://github.com/prometheus-community/postgres_exporter>
- Review redis-exporter: <https://github.com/oliver006/redis_exporter>

**Files**:

- `docker-compose.dev.yaml`

**Changes**:

- Add `postgres-exporter` service
- Add `redis-exporter` service
- Configure exporters to connect to respective services

**Test**: Exporters expose metrics on their endpoints

**Commit**: `feat(obs): step-38 - add metric exporters`

---

## Commit 39: Configure metric scraping

**Research**:

- Review Prometheus scrape configuration
- Check VictoriaMetrics compatibility with Prometheus config

**Files**:

- `observability/prometheus-config.yaml`
- `docker-compose.dev.yaml`

**Changes**:

- Create Prometheus scrape configuration
- Add scrape jobs for: Postgres, Redis, Traefik, Temporal
- Add `prometheus.io/scrape` labels to services
- Configure VictoriaMetrics to use this config

**Test**: VictoriaMetrics scrapes all configured targets

**Commit**: `feat(obs): step-39 - configure metric scraping`

---

## Commit 40: Add Grafana service

**Research**:

- Review Grafana Docker setup:
  <https://grafana.com/docs/grafana/latest/setup-grafana/installation/docker/>
- Check Grafana SSO with generic OAuth2

**Files**:

- `docker-compose.dev.yaml`
- `observability/grafana/grafana.ini`

**Changes**:

- Add `grafana` service (port 3001)
- Configure domain `grafana.nexus.local`
- Setup Grafana SSO via Kratos OAuth2
- Configure auto-admin role for `admin` users

**Test**: Can access Grafana UI via SSO

**Commit**: `feat(obs): step-40 - add Grafana service with SSO`

---

## Commit 41: Auto-provision datasources and dashboards

**Research**:

- Review Grafana provisioning:
  <https://grafana.com/docs/grafana/latest/administration/provisioning/>
- Check VictoriaMetrics datasource configuration
- Review LogQL query patterns for VictoriaLogs

**Files**:

- `observability/grafana/provisioning/datasources/victoria.yaml`
- `observability/grafana/provisioning/dashboards/dashboards.yaml`
- `observability/grafana/provisioning/dashboards/*.json`

**Changes**:

- Auto-configure VictoriaMetrics datasource
- Auto-configure VictoriaLogs datasource
- Auto-configure VictoriaTraces datasource
- Create 5 dashboards:
  - `nexus-overview.json` - System overview
  - `nexus-postgres.json` - PostgreSQL metrics
  - `nexus-redis.json` - Redis metrics
  - `nexus-temporal.json` - Temporal workflow metrics
  - `nexus-traefik.json` - HTTP traffic and errors

**Test**: All datasources and dashboards load in Grafana

**Commit**:
`feat(obs): step-41 - auto-provision Grafana datasources and dashboards`

---

## Commit 42: Add observability documentation

**Research**:

- Review documentation best practices
- Check LogQL query examples

**Files**:

- `observability/README.md`
- `observability/QUICKSTART.md`
- `observability/LOCAL_DNS_SETUP.md`

**Changes**:

- Document observability architecture
- Create quickstart guide for accessing Grafana
- Document LogQL queries for common scenarios
- Add local DNS setup instructions
- Document dashboard usage

**Test**: Documentation is clear and actionable

**Commit**: `docs(obs): step-42 - add observability stack documentation`
