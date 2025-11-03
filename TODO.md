# Nexus IAM Implementation - Atomic Commits (Vault-Only)

## Overview

Complete IAM, Logging & Observability implementation as series of **42
independent, atomic commits**.

**Key Principles**:

- Each step = 1 commit = 1 deployable unit
- All commits buildable and testable independently
- Can revert any commit without breaking others
- Commit format: `feat(iam): step-XX - description`
- Automation-first approach with `go-task`
- Single Sign-On (SSO) for all admin tools

**Timeline**: 7-9 weeks (5-6 commits/week)

**Automation Commits** (01a, 01b):

- Vault bootstrap automation (AppRole, Transit, KV)
- Shared KV secrets rotation

**SSO Integration** (02a):

- Kratos OIDC for Vault UI access

**Simplified Architecture**:

- ✅ **Vault** as the single source of truth for all secrets.
- ✅ **Vault Transit Engine** for JWT signing (replaces `token-service`).
- ✅ **Vault KV Engine** for static secrets (e.g., webhook secrets).
- ✅ **Vault AppRole** for service-to-service authentication against Vault.
- ✅ Fewer moving parts, easier maintenance.

## ⚠️ IMPORTANT: Before Each Commit

**ALWAYS check for latest versions and documentation:**

1. **Docker Images**: Check Docker Hub for latest stable tags

- HashiCorp Vault:
  [https://hub.docker.com/\_/vault](https://hub.docker.com/_/vault)
- PostgreSQL:
  [https://hub.docker.com/\_/postgres](https://hub.docker.com/_/postgres)
- Others as needed

1. **Go Dependencies**: Check latest versions before `go get`

- golang-jwt/jwt:
  [https://github.com/golang-jwt/jwt](https://github.com/golang-jwt/jwt)
- HashiCorp Vault API:
  [https://github.com/hashicorp/vault/tree/main/api](https://github.com/hashicorp/vault/tree/main/api)
- Other libraries: check GitHub releases

1. **Official Documentation**: Review current docs (not cached)

- HashiCorp Vault: <https://developer.hashicorp.com/vault/docs>
- Ory Kratos: <https://www.ory.sh/docs/kratos>
- Ory Oathkeeper: <https://www.ory.sh/docs/oathkeeper>
- JWT Best Practices: <https://datatracker.ietf.org/doc/html/rfc8725>
- Go slog: <https://pkg.go.dev/log/slog>
- VictoriaMetrics: <https://docs.victoriametrics.com/>
- Grafana: <https://grafana.com/docs/grafana/latest/>

1. **Breaking Changes**: Check CHANGELOG/migration guides for major version
   updates

**Why?** Using outdated versions can cause:

- Security vulnerabilities
- Incompatibility issues
- Missing features
- Deprecated API usage

---

## Commit 01 ✅: Add HashiCorp Vault service

**Research**:

- Check latest Vault Docker image version:
  [https://hub.docker.com/\_/vault/tags](https://hub.docker.com/_/vault/tags)
- Review official docs:
  [https://developer.hashicorp.com/vault/docs](https://developer.hashicorp.com/vault/docs)
- Check Vault dev mode vs production mode setup
- Review storage backend options (file, PostgreSQL, Consul)

**Files**: `docker-compose.dev.yaml`, `vault/config.hcl`

**Changes**: Add `vault` service (port 8200) with file storage backend

**Test**: `docker-compose up vault` starts successfully, vault status shows
unsealed

**Commit**:
`feat(iam): step-01 - add HashiCorp Vault secrets management service`

---

## Commit 01a ✅: Add Vault bootstrap automation

**Research**:

- Review `go-task` best practices:
  [https://taskfile.dev/usage/](https://taskfile.dev/usage/)
- Check Vault auto-unseal options:
  [https://developer.hashicorp.com/vault/docs/concepts/seal](https://developer.hashicorp.com/vault/docs/concepts/seal)
- Review Vault init and unseal automation for dev mode
- Check Vault KV secrets engine v2

**Files**:

- `Taskfile.yml` (root)
- `vault/Taskfile.yml`
- `vault/scripts/bootstrap-dev.sh`
- `vault/scripts/init-policies.sh`
- `docker-compose.dev.yaml`

**Changes**:

- Add `task vault:init` command
- Script auto-unseals Vault in dev mode
- Create KV secrets engine v2
- Generate AppRole credentials for each service
- Setup policies for least-privilege access (KV read, Transit sign)

**Test**: `task vault:init` sets up Vault completely, services can authenticate

**Commit**: `feat(iam): step-01a - automate Vault bootstrap and AppRole setup`

---

## Commit 01b ✅: Add KV secrets rotation automation

**Research**:

- Review secret rotation best practices
- Check Vault KV v2 secret versioning API
- Review zero-downtime rotation patterns

**Files**:

- `vault/scripts/rotate-shared-secrets.sh`
- `Taskfile.yml`

**Changes**:

- Script generates new shared secrets (e.g., Kratos webhook secret)
- Updates secrets in Vault KV engine (creates new version)
- Triggers graceful service restarts (e.g.,
  `docker-compose restart internal-api`)
- Logs rotation in audit trail

**Commands**:

- `task secrets:rotate` - Rotate all shared KV secrets
- `task secrets:verify` - Verify all secrets are valid

**Test**: `task secrets:rotate` rotates secrets without downtime

**Commit**:
`feat(iam): step-01b - add shared KV secrets rotation automation for Vault`

---

## Commit 02 ✅: Add Vault secrets seeding script

**Research**:

- Review Vault KV v2 API:
  [https://developer.hashicorp.com/vault/api-docs/secret/kv/kv-v2](https://developer.hashicorp.com/vault/api-docs/secret/kv/kv-v2)
- Check Vault CLI usage patterns
- Review secret versioning and metadata

**Files**: `vault/scripts/seed-dev-secrets.sh`, `.env.example`

**Changes**: Script auto-populates dev secrets (database passwords, API keys,
etc.)

**Test**: Script runs without errors, secrets readable by services

**Commit**: `feat(iam): step-02 - add Vault dev secrets seeding script`

---

## Commit 02a ✅: Configure Vault OIDC auth with Kratos

**Research**:

- Review Vault OIDC auth method:
  [https://developer.hashicorp.com/vault/docs/auth/jwt](https://developer.hashicorp.com/vault/docs/auth/jwt)
- Check Kratos as OIDC provider:
  [https://www.ory.sh/docs/kratos/social-signin/overview](https://www.ory.sh/docs/kratos/social-signin/overview)
- Review OIDC role mapping in Vault

**Files**:

- `vault/config.hcl`
- `vault/scripts/init-oidc.sh`
- `kratos/dev/kratos.yml`

**Changes**:

- Enable OIDC auth method in Vault
- Configure Kratos as OIDC provider
- Map Kratos roles to Vault policies
- Auto-provision users on first SSO login

**Benefits**:

- ✅ Single sign-on to Vault UI via Kratos
- ✅ Centralized user management in Kratos
- ✅ Role-based Vault policies from Kratos traits
- ✅ No separate Vault user management

**Test**: Can login to Vault UI using Nexus credentials via Kratos SSO

**Commit**: `feat(iam): step-02a - configure Vault OIDC auth via Kratos`

---

## Commit 03 ✅: Add Vault Go client library

**Research**:

- Check official Vault Go API:
  [https://github.com/hashicorp/vault/tree/main/api](https://github.com/hashicorp/vault/tree/main/api)
- Review AppRole authentication flow
- Check KV v2 secrets engine API usage

**Files**:

- `backend/internal/secrets/client.go`
- `backend/internal/secrets/client_test.go`
- `backend/go.mod`

**Changes**: Shared library for reading KV secrets via AppRole auth

**Test**: Unit tests pass, can authenticate and read secrets from Vault

**Commit**: `feat(iam): step-03 - add Vault Go client library with AppRole auth`

---

## Commit 04 ✅: Configure Vault Transit engine for JWT signing

**Research**:

- Review Vault Transit Secrets Engine:
  [https://developer.hashicorp.com/vault/docs/secrets/transit](https://developer.hashicorp.com/vault/docs/secrets/transit)
- Check Transit sign/verify operations:
  [https://developer.hashicorp.com/vault/api-docs/secret/transit](https://developer.hashicorp.com/vault/api-docs/secret/transit)
- Review RSA-4096 for JWT signing
- Check Vault policy requirements for Transit

**Files**:

- `vault/scripts/init-transit.sh` (as part of `vault:init`)
- `vault/scripts/init-policies.sh` (updated)
- `Taskfile.yml`

**Changes**:

- Enable Transit engine with RSA-4096 key (`service-jwt-key`)
- Update AppRole policies to allow signing
- Automatic key rotation support built-in via Vault
- Add `task vault:rotate-transit-key`

**Benefits**:

- ✅ No separate Token Service needed
- ✅ Vault handles key management and rotation
- ✅ Cryptographic operations audited by Vault
- ✅ Built-in key versioning and rotation

**Test**: `task vault:init` enables Transit, can sign data via API

**Commit**: `feat(iam): step-04 - configure Vault Transit for JWT signing`

---

## Commit 05 ✅: Create JWT verifier library with Vault

**Research**:

- Review JWT validation best practices (exp, aud, signature checks)
- Check Vault Transit verify operation:
  [https://developer.hashicorp.com/vault/api-docs/secret/transit\#verify-signed-data](https://developer.hashicorp.com/vault/api-docs/secret/transit#verify-signed-data)
- Review local JWK caching pattern from Vault Transit public keys

**Files**:

- `backend/internal/auth/jwt_verifier.go`
- `backend/internal/auth/jwt_verifier_test.go`

**Changes**:

- Shared lib for verifying service JWT tokens
- **Hot Path**: Verifies locally using a cached JWKSet from Vault.
- **Cold Path**: On unknown `kid`, re-fetches JWKSet from Vault Transit.
- This avoids a network call to Vault for _every_ verification.

**Test**: Verifies valid tokens, rejects invalid, handles key rotation

**Commit**:
`feat(iam): step-05 - create JWT verifier library with Vault Transit`

---

## Commit 06 ✅: Create Vault token client library

**Research**:

- Review Vault AppRole login:
  [https://developer.hashicorp.com/vault/api-docs/auth/approle](https://developer.hashicorp.com/vault/api-docs/auth/approle)
- Check Vault token renewal strategies
- Review Vault Transit sign operation to create a JWT

**Files**:

- `backend/internal/auth/vault_token_client.go`
- `backend/internal/auth/vault_token_client_test.go`

**Changes**:

- Shared lib for requesting JWT tokens
- 1. Authenticates to Vault using AppRole (from `secrets` lib)
- 1. Builds JWT claims (sub, aud, exp)
- 1. Sends claims to Vault Transit `sign` endpoint
- 1. Returns the signed JWT

**Test**: Can obtain and refresh tokens, tokens verified by `jwt_verifier`

**Commit**: `feat(iam): step-06 - create Vault token client library`

---

## Commit 07 ✅: Add role to Kratos schema

**Research**:

- Review Ory Kratos identity schema docs:
  [https://www.ory.sh/docs/kratos/manage-identities/customize-identity-schema](https://www.ory.sh/docs/kratos/manage-identities/customize-identity-schema)
- Check JSON Schema enum validation examples
- Review Kratos v1.3+ schema changes

**Files**: `kratos/dev/identity.schema.json`

**Changes**: Add `traits.role` enum (none, member, admin)

**Test**: Can create identity with role in Kratos UI

**Commit**: `feat(iam): step-07 - add role field to Kratos identity schema`

---

## Commit 08 ✅: Create migration to remove role column

**Research**:

- Review go-migrate best practices:
  [https://github.com/golang-migrate/migrate](https://github.com/golang-migrate/migrate)
- Check PostgreSQL `DROP COLUMN` syntax for v16+
- Review safe migration patterns (rollback support)

**Files**: `postgres/migrations/20251103000000_remove_role_column.sql`

**Changes**: Drop `role` column and `user_role` enum from users table

**Test**: Migration runs and can rollback

**Commit**: `feat(iam): step-08 - create migration to remove role from users`

---

## Commit 09 ✅: Remove UserRole from GraphQL schema

**Research**:

- Review `gqlgen` latest version:
  [https://github.com/99designs/gqlgen](https://github.com/99designs/gqlgen)
- Check `gqlgen` code generation commands
- Review GraphQL schema best practices for breaking changes

**Files**:

- `backend/internal/api/graphql/schema.graphql`
- `backend/internal/api/graphql/generated.go` (regenerated)

**Changes**: Delete `UserRole` enum and related mutations

**Test**: GraphQL queries work without role field

**Commit**: `feat(iam): step-09 - remove UserRole enum from GraphQL schema`

---

## Commit 10 ✅: Update sqlc queries to remove role

**Research**:

- Check `sqlc` latest version:
  [https://github.com/sqlc-dev/sqlc](https://github.com/sqlc-dev/sqlc)
- Review `sqlc` v1.27+ configuration and generation
- Check `sqlc.yaml` syntax changes

**Files**:

- `backend/internal/repository/postgres/queries/users.sql`
- `backend/internal/repository/postgres/users.sql.go` (regenerated)

**Changes**: Remove role from all SQL queries

**Test**: All queries compile

**Commit**: `feat(iam): step-10 - update sqlc queries to remove role`

---

## Commit 11 ✅: Add X-User-Role to Oathkeeper mutator

**Research**:

- Review Ory Oathkeeper mutators:
  [https://www.ory.sh/docs/oathkeeper/pipeline/mutator](https://www.ory.sh/docs/oathkeeper/pipeline/mutator)
- Check header mutator template syntax and available variables
- Review Oathkeeper v0.40+ configuration changes

**Files**: `kratos/dev/oathkeeper.yml`

**Changes**: Pass role from Kratos session to Gateway via `X-User-Role` header

**Test**: `X-User-Role` appears in Gateway requests

**Commit**: `feat(iam): step-11 - add X-User-Role header to Oathkeeper`

---

## Commit 12 ✅: Create Internal API skeleton

**Files**:

- `backend/cmd/internal-api/main.go`
- `backend/Dockerfile.internal-api.dev`
- `backend/.air.internal-api.toml`
- `docker-compose.dev.yaml`

**Changes**: Empty service with health endpoint

**Test**: Health endpoint responds

**Commit**: `feat(iam): step-12 - create Internal API skeleton`

---

## Commit 13 ✅: Migrate Kratos webhook to Internal API

**Files**:

- `backend/cmd/internal-api/main.go`
- `backend/internal/webhooks/kratos/handler.go`

**Changes**: Move registration webhook from (old) `webhooks` service

**Test**: Webhook works, creates users

**Commit**: `feat(iam): step-13 - migrate Kratos webhook to Internal API`

---

## Commit 14 ✅: Add JWT middleware to Internal API

**Files**:

- `backend/cmd/internal-api/main.go`
- `backend/internal/api/internal/middleware/jwt.go`

**Changes**: Protect internal endpoints with JWT verification (using
`jwt_verifier`)

**Test**: Requests without JWT fail, with JWT succeed

**Commit**: `feat(iam): step-14 - add JWT verification middleware`

---

## Commit 15 ✅: Add role management endpoint

**Research**:

- Review Kratos Admin API:
  [https://www.ory.sh/docs/kratos/reference/api](https://www.ory.sh/docs/kratos/reference/api)
- Check `PATCH /admin/identities/{id}` endpoint documentation
- Review identity traits update patterns

**Files**:

- `backend/cmd/internal-api/main.go`
- `backend/internal/api/internal/handlers/admin.go`

**Changes**: `POST /admin/users/{id}/role` updates Kratos traits

**Test**: Can change role via API

**Commit**: `feat(iam): step-15 - add role management endpoint`

---

## Commit 16 ✅: Update Kratos webhook URL

**Files**: `kratos/dev/kratos.yml`

**Changes**: Point registration webhook to `internal-api:8083`

**Test**: Registration still creates users

**Commit**: `feat(iam): step-16 - update Kratos webhook URL`

---

## Commit 17 ✅: Remove old webhooks service

**Files**:

- DELETE `backend/cmd/webhooks/`
- DELETE `backend/Dockerfile.webhooks.dev`
- DELETE `backend/.air.webhooks.toml`
- `docker-compose.dev.yaml`

**Changes**: Delete obsolete service completely

**Test**: System works without webhooks service

**Commit**: `feat(iam): step-17 - remove obsolete webhooks service`

---

## Commit 18 ✅: Update Gateway AuthMiddleware

**Files**: `backend/internal/api/rest/middleware/auth.go`

**Changes**: Verify Oathkeeper JWT, read `X-User-*` headers safely

**Test**: Gateway auth still works

**Commit**: `feat(iam): step-18 - verify Oathkeeper JWT in Gateway`

---

## Commit 19 ✅: Add token client to Gateway

**Files**: `backend/cmd/gateway/main.go`

**Changes**: Request JWT (using `vault_token_client`) before calling Data API

**Test**: Data API receives JWT from Gateway

**Commit**: `feat(iam): step-19 - add JWT token client to Gateway`

---

## Commit 20 ✅: Add JWT middleware to Data API

**Files**: `backend/cmd/data-api/main.go`

**Changes**: Protect GraphQL with JWT verification (using `jwt_verifier`)

**Test**: Data API rejects requests without JWT

**Commit**: `feat(iam): step-20 - add JWT verification to Data API`

---

## Commit 21 ✅: Delete old JWT user auth code

**Files**:

- `backend/internal/api/rest/middleware/auth.go`
- `backend/go.mod`

**Changes**: Remove unused JWT validation for user tokens

**Test**: All tests still pass

**Commit**: `feat(iam): step-21 - remove old JWT user authentication`

---

## Commit 22 ✅: Refactor audit to call Internal API

**Files**: `backend/internal/api/rest/services/temporal_audit.go`

**Changes**: Call Internal API instead of Temporal directly

**Test**: Audit events still logged

**Commit**: `feat(iam): step-22 - refactor audit to use Internal API`

---

## Commit 23 ✅: Remove Temporal client from Gateway

**Files**:

- `backend/cmd/gateway/main.go`
- `backend/go.mod`

**Changes**: Gateway no longer imports Temporal

**Test**: Gateway builds and runs

**Commit**: `feat(iam): step-23 - remove Temporal dependency from Gateway`

---

## Commit 24 ✅: Add audit handler to Internal API

**Files**:

- `backend/cmd/internal-api/main.go`
- `backend/internal/api/internal/handlers/audit.go`

**Changes**: Receive audit events, trigger Temporal workflows

**Test**: Audit events processed via Temporal

**Commit**: `feat(iam): step-24 - add audit workflow handler`

---

## Commit 25: Add Kratos login webhook

**Research**:

- Review Kratos webhooks:
  [https://www.ory.sh/docs/kratos/hooks/configure-hooks](https://www.ory.sh/docs/kratos/hooks/configure-hooks)
- Check available webhook events (e.g., `after.login`)
- Review webhook payload structure and security

**Files**:

- `backend/cmd/internal-api/main.go`
- `kratos/dev/kratos.yml`

**Changes**: Log user login events

**Test**: Login events appear in audit log

**Commit**: `feat(iam): step-25 - add Kratos login webhook`

---

## Commit 26: Update logout to revoke session

**Files**: `backend/internal/api/rest/handlers/me.go`

**Changes**: Call Kratos `DELETE /sessions` API

**Test**: Logout invalidates Kratos session

**Commit**: `feat(iam): step-26 - revoke Kratos session on logout`

---

## Commit 27: Add logout webhook

**Files**:

- `backend/cmd/internal-api/main.go`
- `kratos/dev/kratos.yml`

**Changes**: Log user logout events

**Test**: Logout events appear in audit log

**Commit**: `feat(iam): step-27 - add logout webhook for audit`

---

## Commit 28: Add RBAC authorizer endpoint

**Files**:

- `backend/internal/api/rest/handlers/authorizer.go`
- `backend/cmd/gateway/main.go`

**Changes**: `/api/internal/authorize` for Oathkeeper `remote_json`

**Test**: Returns 200 for admin, 403 for non-admin

**Commit**: `feat(iam): step-28 - add RBAC authorizer endpoint`

---

## Commit 29: Add admin access rules

**Research**:

- Review Ory Oathkeeper authorizers:
  [https://www.ory.sh/docs/oathkeeper/pipeline/authn](https://www.ory.sh/docs/oathkeeper/pipeline/authn)
- Check `remote_json` authorizer configuration examples
- Review access rules matching patterns and priority

**Files**: `kratos/dev/access-rules.yml`

**Changes**: Protect admin endpoints with role check

**Test**: Admin operations require admin role

**Commit**: `feat(iam): step-29 - add admin RBAC rules to Oathkeeper`

---

## Commit 30: Add dev-mode Traefik labels

**Files**: `docker-compose.dev.yaml`

**Changes**: Direct access to internal services in dev

**Test**: Can access services directly in dev

**Commit**: `feat(iam): step-30 - add dev-mode direct access labels`

---

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

---

## Service Architecture

### Final Service Map

```text
PUBLIC (через Traefik + Oathkeeper):
├── api.nexus.local → gateway:8080
├── nexus.local → ui:3000
├── auth.nexus.local → kratos-selfservice-ui:3000
└── grafana.nexus.local → grafana:3001 (SSO via Kratos)

INTERNAL (Docker network only):
├── data-api:8081 (GraphQL)
├── internal-api:8083 (Webhooks + System Events)
├── vault:8200 (Secrets + JWT issuer via Transit)
├── worker (Temporal)
└── Observability Stack:
    ├── victoriametrics:8428 (Metrics TSDB)
    ├── victorialogs:9428 (Logs)
    ├── victoriatraces:4318 (Traces)
    └── promtail (Log shipper)

DEV-ONLY (debug access):
├── graphql-dev.nexus.local → data-api:8081
├── internal-dev.nexus.local → internal-api:8083
└── vault-dev.nexus.local → vault:8200
```

### Service-to-Service Auth Matrix

| Connection              | Auth Method        | Token / Secret Details                    |
| :---------------------- | :----------------- | :---------------------------------------- |
| Oathkeeper → Gateway    | JWT                | `sub=oathkeeper`, `aud=gateway`           |
| Gateway → Data API      | JWT                | `sub=gateway`, `aud=data-api`             |
| Gateway → Internal API  | JWT                | `sub=gateway`, `aud=internal-api`         |
| Internal API → Data API | JWT                | `sub=internal-api`, `aud=data-api`        |
| Worker → Internal API   | JWT                | `sub=worker`, `aud=internal-api`          |
| Kratos → Internal API   | `X-Webhook-Secret` | (Secret stored in Vault KV)               |
| Services → Vault        | AppRole            | (Used to auth for KV read & Transit sign) |

---

## Implementation Schedule

### Phase 1: Infrastructure (Commits 1-6 + 01a, 01b, 02a, Week 1-2)

- Vault setup + Go client integration
- **Automation**: Vault bootstrap (AppRole, Transit, KV), KV secrets rotation
- **SSO**: Kratos OIDC for Vault UI access
- **Vault Transit**: JWT signing via Transit engine
- JWT verifier + token client libraries

### Phase 2: Role Management (Commits 7-11, Week 2-3)

- Kratos schema with role field
- Remove role from PostgreSQL
- Oathkeeper role header mutation

### Phase 3: Internal API (Commits 12-17, Week 3-4)

- Internal API skeleton
- Migrate webhooks
- JWT protection
- Remove old webhooks service

### Phase 4: Gateway Integration (Commits 18-23, Week 4-5)

- Gateway JWT verification
- Data API protection
- Remove old auth code
- Audit refactoring

### Phase 5: Audit & RBAC (Commits 24-29, Week 5-6)

- Login/logout webhooks
- Session management
- RBAC authorizer
- Oathkeeper access rules

### Phase 6: Dev Experience (Commit 30, Week 6)

- Direct access labels for debugging

### Phase 7: Structured Logging (Commits 31-35, Week 7)

- Add `slog` structured logging library
- Integrate `slog` in internal-api
- Integrate `slog` in data-api
- Integrate `slog` in gateway
- Integrate `slog` in worker

### Phase 8: Observability Stack (Commits 36-42, Week 8-9)

- VictoriaMetrics core services (TSDB, Logs, Traces)
- Promtail with JSON log parsing
- Metric exporters (Postgres, Redis)
- Prometheus scraping configuration
- Grafana service with SSO
- Auto-provision datasources and dashboards
- Observability documentation

---

## Security Checklist

- ✅ JWT service-to-service auth (RS256)
- ✅ Short-lived tokens (5 min TTL)
- ✅ Audience (`aud`) validation (service → service)
- ✅ **Vault Transit** for centralized key management/rotation
- ✅ **Secrets in Vault** (not `.env` files)
- ✅ No service trusts headers without JWT
- ✅ RBAC at Oathkeeper level (edge)
- ✅ Audit trail for all IAM events (Kratos + Vault)
- ✅ **Kratos**: Source of Truth for _user_ identities/roles
- ✅ **Vault**: Source of Truth for _service_ identities/secrets
- ✅ **Structured Logging**: All services use `slog` with JSON output
- ✅ **Observability**: Metrics, logs, and traces in VictoriaMetrics stack
- ✅ **Grafana SSO**: Unified access to observability with Kratos auth

---

## Automation with go-task

### Key Management Commands

**Vault Transit Keys**:

```bash
task vault:rotate-transit-key          # Rotate Transit signing key
task vault:list-transit-keys           # Show all key versions
task vault:revert-transit-key VERSION  # Revert to previous key version
```

**Secrets Management**:

```bash
task secrets:rotate                    # Rotate all shared KV secrets
task secrets:verify                    # Verify all secrets are valid
task vault:snapshot-save               # Backup current Vault data
```

**Vault Bootstrap**:

```bash
task vault:init                        # Full Vault setup (unseal + configure)
task vault:seed-dev-secrets            # Populate KV secrets
```

### Rotation Strategy

**Shared Secrets (KV)** (Monthly or on compromise):

1. Generate new secrets with `task secrets:rotate`
2. Update KV secret in Vault (creates new version)
3. Graceful service restarts (zero downtime)
4. Verify with `task secrets:verify`

**Transit Keys** (Quarterly or on compromise):

1. Rotate Transit key with `task vault:rotate-transit-key`
2. Vault automatically maintains old key versions for verification
3. `jwt_verifier` fetches new JWKSet on `kid` mismatch
4. Old versions auto-expire after configured period

**AppRole Credentials** (On compromise only):

1. Revoke compromised AppRole `SecretID` in Vault
2. Generate new `SecretID`
3. Update service configuration (e.g., in `docker-compose.override.yml`)
4. Restart affected service

### Zero-Downtime Rotation

All rotation scripts follow the pattern:

- Generate new credentials
- Activate both old + new (grace period)
- Services gradually adopt new credentials
- Old credentials expire automatically
- Rollback available via Vault versioning

---

## Single Sign-On (SSO) Strategy

### Current SSO Integrations

**HashiCorp Vault** (Commit 02a):

- Protocol: OIDC via Kratos
- Benefit: Unified admin access to secrets management
- Auto-provision: Users created on first login
- Policies: Mapped from Kratos role trait

### Future SSO Integrations

**Temporal UI**:

- Protocol: OIDC/SAML (Temporal Cloud or self-hosted with auth)
- Status: Planned for future commit
- Benefit: Workflow management with centralized auth

**Grafana/Monitoring**:

- Protocol: Generic OAuth2 via Kratos
- Status: Planned for observability phase
- Benefit: Unified access to metrics and dashboards

**Home Assistant**:

- Protocol: Generic OAuth2 (via custom integration)
- Status: Planned for IoT phase
- Benefit: Physical asset control with Nexus credentials

### SSO Architecture

```text
┌──────────────┐
│ Kratos       │ ← Single Source of Truth
│ (OIDC Server)│    - User identities
└──────┬───────┘    - Roles & traits
       │            - Session management
       │
       ├─────→ Vault (Secrets UI)
       ├─────→ Temporal (Workflow UI)
       ├─────→ Grafana (Monitoring)
       └─────→ Home Assistant (IoT Control)
```

### Benefits

- ✅ **One set of credentials** for all admin tools
- ✅ **Centralized user management** in Kratos
- ✅ **Role-based access** propagates to all services
- ✅ **Single logout** terminates all sessions
- ✅ **Audit trail** for all authentications
- ✅ **Consistent UX** across admin panels

---

## Post-Implementation

After all 42 commits:

### IAM (Commits 1-30)

- [ ] Update `README.md` with new architecture
- [ ] Add sequence diagrams for AuthN/AuthZ
- [ ] Document JWT flow (AppRole → Transit → Sign → Verify)
- [ ] Security audit
- [ ] Load test Vault Transit signing operations
- [ ] Document rotation procedures (Runbooks)
- [ ] Test emergency key rollback (`vault:revert-transit-key`)
- [ ] Test SSO login flow for Vault UI
- [ ] Document SSO onboarding for new admins
- [ ] Test Vault AppRole authentication flow

### Logging (Commits 31-35)

- [ ] Verify all services output structured JSON logs
- [ ] Test log parsing in VictoriaLogs
- [ ] Verify all required fields present (`level`, `service`, `error`)
- [ ] Test context propagation across service calls

### Observability (Commits 36-42)

- [ ] Verify Promtail correctly parses JSON logs
- [ ] Test all 5 Grafana dashboards load and display data
- [ ] Verify metrics from all exporters are scraped
- [ ] Test Grafana SSO login via Kratos
- [ ] Verify LogQL queries work correctly
- [ ] Test alert rules (if configured)
- [ ] Document common troubleshooting queries

---

## References

### IAM & Security

- **HashiCorp Vault**: <https://developer.hashicorp.com/vault/docs>
- **Vault Go API**: <https://github.com/hashicorp/vault/tree/main/api>
- **Ory Kratos**: <https://www.ory.sh/kratos/docs/>
- **Ory Oathkeeper**: <https://www.ory.sh/oathkeeper/docs/>
- **Zero Trust**: <https://www.nist.gov/publications/zero-trust-architecture>
- **JWT Best Practices**: <https://datatracker.ietf.org/doc/html/rfc8725>

### Logging

- **Go slog**: <https://pkg.go.dev/log/slog>
- **Structured Logging Best Practices**:
  <https://betterstack.com/community/guides/logging/go/logging-with-slog/>

### Observability

- **VictoriaMetrics**: <https://docs.victoriametrics.com/>
- **VictoriaLogs**: <https://docs.victoriametrics.com/victorialogs/>
- **Promtail**: <https://grafana.com/docs/loki/latest/send-data/promtail/>
- **Grafana**: <https://grafana.com/docs/grafana/latest/>
- **LogQL**: <https://grafana.com/docs/loki/latest/query/>
- **Postgres Exporter**:
  <https://github.com/prometheus-community/postgres_exporter>
- **Redis Exporter**: <https://github.com/oliver006/redis_exporter>
