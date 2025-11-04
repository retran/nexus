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

## Implementation Plan - REFACTOR EXISTING ONLY

### Phase 1: Rename & Restructure (3 commits)

**Goal**: Rename current dev compose to main, prepare structure for prod/dev
split

1. **Rename docker-compose.dev.yaml → docker-compose.yaml**
   - This becomes the production compose file
   - Rename volumes: `postgres_dev_data` → `postgres_data`, etc.
   - Update Taskfile.yml references
   - Update .gitignore to ignore `docker-compose.override.yaml`

2. **Create docker-compose.override.yaml.example**
   - Template for local development overrides
   - Volume mounts for hot reload
   - Port exposures (5432, 6379, 8200)
   - Command overrides (air, vite)
   - Reference to .dev Dockerfiles
   - Developer copies to `docker-compose.override.yaml` (git-ignored)

3. **Reorganize config directories**
   - Rename `ory/dev/` → `ory/config/` (single config, parameterized via env)
   - Rename `vault/config.hcl` → `vault/config.dev.hcl`
   - Keep configs environment-agnostic via environment variables
   - Update volume mounts in compose

### Phase 2: Production-Ready Docker Images (4 commits)

**Goal**: Create production multi-stage Dockerfiles, keep .dev variants

1. **backend/Dockerfile (production multi-stage)**
   - Stage 1: Build all services in single Dockerfile with targets
   - Stage 2: Runtime with only binaries
   - Targets: gateway, internal-api, worker, data-api
   - Update compose to use `target:` for each service

2. **frontend/Dockerfile (production)**
   - Stage 1: Build with yarn (node:22-alpine)
   - Stage 2: Serve with nginx (nginx:alpine)
   - Copy nginx.conf for SPA routing

3. **Update docker-compose.yaml to use prod images**
   - Change `dockerfile: Dockerfile.*.dev` → `dockerfile: Dockerfile`
   - Remove `command: ["air", ...]` - use CMD from Dockerfile
   - Remove source volume mounts (`./backend:/app`)
   - Keep config mounts (`./ory/config`, vault agent templates)

4. **Document dev override pattern**
   - Add README.dev.md with setup instructions
   - Override services to use .dev Dockerfiles
   - Override commands to use air/vite
   - Add back source volume mounts

### Phase 3: Environment & Security Hardening (4 commits)

**Goal**: Production-ready settings in main compose, dev overrides for local

1. **Remove exposed ports from docker-compose.yaml**
   - Change `ports: ["5432:5432"]` → `expose: [5432]`
   - Only Traefik keeps port 80/443
   - Document: devs add back in override for direct access

2. **Secure production defaults**
   - Traefik: Remove `--api.insecure`, add ACME placeholder
   - Traefik: Change log level DEBUG → INFO
   - Kratos: Remove `--dev --watch-courier`
   - Vault: Comment about dev mode (needs seal keys in prod)

3. **Create .env.prod.example**
   - Real domain placeholders (nexus.yourdomain.com)
   - ACME email for Let's Encrypt
   - No default passwords (must be set)
   - Reference to Vault for secret storage
   - Clear separation from .env.dev.example

4. **Update vault/config.hcl for production path**
   - Keep current as config.dev.hcl
   - Create config.hcl (symlink/copy decision via docs)
   - Document seal key setup
   - Document TLS setup (can come later)

### Phase 4: Certificate & Volume Persistence (2 commits)

**Goal**: Make certificates persistent, prepare for production restarts

1. **Replace tmpfs with named volumes for certificates**
   - Change all `*-secrets` volumes from tmpfs → named volumes
   - Keeps certificates across container restarts
   - Document: Vault agent will regenerate if missing
   - Update volume cleanup docs (these are now persistent)

2. **Add Traefik certificate persistence**
   - Add `traefik_acme` named volume for Let's Encrypt certs
   - Mount to `/letsencrypt` in Traefik
   - Add ACME configuration (commented out for dev)
   - Document: Uncomment for production with real domain

### Phase 5: Documentation & Validation (2 commits)

**Goal**: Complete documentation, validate both modes work

1. **Update all documentation**
   - README.md: Production setup instructions
   - README.dev.md: Development setup (NEW)
   - Taskfile.yml: Add `up:prod` and `up:dev` tasks
   - .github/copilot-instructions.md: Update architecture section
   - Document override pattern clearly

2. **Test matrix validation**
   - Test: `docker-compose up` (production mode, should work for Mac Mini)
   - Test:
     `docker-compose -f docker-compose.yaml -f docker-compose.override.yaml up`
     (dev mode)
   - Document environment differences
   - Validate builds work without source mounts
   - Checklist for production deployment

## FUTURE PHASES (Not implementing now, keep in plan)

### Phase 6: External Access (for later)

- Cloudflare Tunnel setup
- Tailscale mesh configuration
- ACME DNS-01 challenge

### Phase 7: Monitoring & Backups (for later)

- PostgreSQL backup automation
- Prometheus + Grafana
- Uptime monitoring

### Phase 8: Deployment Automation (for later)

- GitHub Actions
- Self-hosted runner
- Ansible playbooks

## Success Criteria (Immediate - Phases 1-5)

- [ ] `docker-compose up` runs production-ready config (no dev tools, no exposed
      ports)
- [ ] Developer can use `docker-compose.override.yaml` for dev mode (hot reload,
      ports)
- [ ] Production Dockerfiles build working containers without source mounts
- [ ] All certificates persist across restarts (no tmpfs)
- [ ] Config files organized and documented (ory/config, vault/config.hcl)
- [ ] Both .env.prod.example and .env.dev.example exist with clear differences
- [ ] README.md covers production, README.dev.md covers development
- [ ] Can run on Mac Mini with production settings

## Success Criteria (Future - Phases 6-8)

- [ ] All secrets managed in Vault (no .env in production)
- [ ] HTTPS via Let's Encrypt working
- [ ] Cloudflare Tunnel provides external access
- [ ] Admin interfaces only accessible via Tailscale
- [ ] Automated PostgreSQL backups to B2
- [ ] Monitoring dashboards operational
- [ ] Can deploy updates via GitHub Actions

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
