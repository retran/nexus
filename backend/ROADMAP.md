# Backend IAM - Implementation Plan

## Phase 1: Token Management (Critical) 🔴

### 1.1 Refresh Token Flow

**Status**: Table exists, not implemented
**Priority**: Critical
**Effort**: 1-2 days

**Current Issues**:

- Таблица `refresh_tokens` существует, но не используется
- JWT tokens живут до expiry без возможности продления
- Нет endpoint для обновления токенов

**Implementation**:

- [ ] Implement `POST /api/auth/refresh` endpoint
- [ ] Generate refresh token on successful OAuth login
- [ ] Store refresh token in `refresh_tokens` table
- [ ] Validate refresh token and issue new JWT
- [ ] Rotate refresh tokens on use (security best practice)
- [ ] Set refresh token expiry (7-30 days)
- [ ] Return new JWT + refresh token pair

**Technical Details**:

- Short-lived JWT: 15 minutes
- Long-lived refresh token: 7 days
- Store refresh token hash in database (not plaintext)
- Use secure HTTP-only cookie for refresh token

---

### 1.2 Token Revocation

**Status**: Not implemented
**Priority**: Critical
**Effort**: 1 day

**Current Issues**:

- Logout только удаляет cookie, но JWT остается valid
- Compromised token нельзя отозвать до expiry
- Нет механизма инвалидации токенов

**Implementation**:

- [ ] Redis blacklist для revoked JWT tokens
- [ ] Add token JTI (JWT ID) to all tokens
- [ ] Store revoked JTI in Redis with TTL = token expiry
- [ ] Check blacklist in auth middleware
- [ ] Revoke all user tokens on password change/security event
- [ ] Add `POST /api/auth/revoke` endpoint

**Technical Details**:

- Redis key: `revoked:jwt:{jti}` with TTL
- Check on every authenticated request
- Minimal performance impact (Redis is fast)

---

## Phase 2: Session Management (Important) 🟡

### 2.1 Active Sessions Tracking

**Status**: Not implemented
**Priority**: Important
**Effort**: 1-2 days

**Current Issues**:

- Нет списка активных сессий пользователя
- Нельзя посмотреть, где пользователь залогинен
- Нет информации о device/browser/location

**Implementation**:

- [ ] Create `user_sessions` table (or use Redis)
- [ ] Track session on login: device, browser, IP, location
- [ ] Add `GET /api/me/sessions` endpoint
- [ ] Show active sessions in UI
- [ ] Auto-cleanup expired sessions

**Schema**:

```sql
CREATE TABLE user_sessions (
  id UUID PRIMARY KEY,
  user_id UUID NOT NULL REFERENCES users(id),
  refresh_token_id UUID REFERENCES refresh_tokens(id),
  device_type VARCHAR(50),
  browser VARCHAR(100),
  ip_address INET,
  location VARCHAR(200),
  last_activity_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ DEFAULT NOW()
);
```

---

### 2.2 Remote Session Revocation

**Status**: Not implemented
**Priority**: Important
**Effort**: 1 day

**Current Issues**:

- Нельзя выйти из аккаунта на другом устройстве
- Нет "logout from all devices" функции
- Украденную сессию нельзя удаленно закрыть

**Implementation**:

- [ ] Add `DELETE /api/me/sessions/{id}` endpoint
- [ ] Add `POST /api/me/sessions/revoke-all` endpoint
- [ ] Revoke associated refresh token
- [ ] Add to JWT blacklist if needed
- [ ] Send notification email on security events

---

## Phase 3: Security Hardening (Important) 🟡

### 3.1 Security Headers

**Status**: Not implemented
**Priority**: Important
**Effort**: 0.5 day

**Current Issues**:

- Нет security headers (CSP, HSTS, X-Frame-Options)
- Browser не защищен от XSS/clickjacking
- Missing CORS security configuration

**Implementation**:

- [ ] Add security headers middleware
- [ ] Content-Security-Policy
- [ ] Strict-Transport-Security (HSTS)
- [ ] X-Frame-Options: DENY
- [ ] X-Content-Type-Options: nosniff
- [ ] X-XSS-Protection: 1; mode=block
- [ ] Referrer-Policy: strict-origin-when-cross-origin

---

### 3.2 CSRF Protection

**Status**: Not implemented
**Priority**: Important
**Effort**: 1 day

**Current Issues**:

- Нет CSRF protection для state-changing operations
- Cookie-based auth vulnerable to CSRF
- POST/PUT/DELETE endpoints не защищены

**Implementation**:

- [ ] Generate CSRF token on login
- [ ] Store CSRF token in Redis (per session)
- [ ] Return CSRF token in response header
- [ ] Validate CSRF token on mutations
- [ ] Double Submit Cookie pattern или Synchronizer Token

---

## Phase 4: Observability (Important) 🟡

**Status**: Infrastructure Complete ✅ | Application Instrumentation Pending

### 4.1 Structured Logging + VictoriaLogs

**Status**: Infrastructure ready, awaiting application implementation
**Priority**: Important
**Effort**: 1.5 days

**Current Issues**:

- Logs не structured (plain text)
- Сложно парсить и анализировать
- Нет correlation IDs
- Нет централизованного хранилища логов
- Grep по Docker logs медленный для больших объемов

**Implementation**:

- [ ] Replace `log` with `zerolog` (structured JSON logs)
- [ ] Add request ID middleware (generate UUID per request)
- [ ] Add user ID to authenticated request logs
- [ ] Log levels: DEBUG, INFO, WARN, ERROR
- [x] Add VictoriaLogs to `docker-compose.dev.yaml`
- [x] Setup Promtail for log shipping (Docker → VictoriaLogs)
- [x] Configure log retention (30 days)

**Log Format**:

```json
{
  "level": "info",
  "time": "2025-10-23T12:00:00Z",
  "request_id": "550e8400-e29b-41d4-a716-446655440000",
  "user_id": "123e4567-e89b-12d3-a456-426614174000",
  "endpoint": "/api/auth/google/login",
  "method": "GET",
  "status": 200,
  "duration_ms": 145,
  "ip": "192.168.1.100",
  "message": "Request completed"
}
```

**Docker Compose Addition**:

```yaml
victorialogs:
  image: victoriametrics/victoria-logs:latest
  restart: always
  ports:
    - "9428:9428"
  volumes:
    - vl_data:/victoria-logs-data
  command:
    - "--storageDataPath=/victoria-logs-data"
    - "--retentionPeriod=30d"

promtail:
  image: grafana/promtail:latest
  restart: always
  volumes:
    - /var/lib/docker/containers:/var/lib/docker/containers:ro
    - /var/run/docker.sock:/var/run/docker.sock
    - ./monitoring/promtail.yml:/etc/promtail/config.yml
  command: -config.file=/etc/promtail/config.yml
  depends_on:
    - victorialogs
```

**Promtail Config** (`monitoring/promtail.yml`):

```yaml
server:
  http_listen_port: 9080
  grpc_listen_port: 0

positions:
  filename: /tmp/positions.yaml

clients:
  - url: http://victorialogs:9428/insert/jsonline

scrape_configs:
  - job_name: docker
    docker_sd_configs:
      - host: unix:///var/run/docker.sock
        refresh_interval: 5s
    relabel_configs:
      - source_labels: ['__meta_docker_container_name']
        regex: '/(.*)'
        target_label: 'container'
      - source_labels: ['__meta_docker_container_log_stream']
        target_label: 'stream'
```

**Query Examples**:

```bash
# All errors in last hour
curl 'http://localhost:9428/select/logsql/query' -d 'query={level="error"} | unpack_json'

# Auth failures
curl 'http://localhost:9428/select/logsql/query' -d 'query={container="gateway", endpoint=~"/api/auth/.*"} | unpack_json | filter status >= 400'

# Slow requests (>1s)
curl 'http://localhost:9428/select/logsql/query' -d 'query=duration_ms > 1000 | unpack_json'
```

---

### 4.2 Victoria Observability Stack (Metrics + Traces + Dashboards)

**Status**: Infrastructure Complete ✅ (Application Instrumentation Pending)
**Priority**: Important
**Effort**: 1.5 days

**Current Issues**:

- Нет метрик для мониторинга
- Нельзя отследить performance issues
- Нет visibility в auth events
- Нет distributed tracing между сервисами
- Нет единого dashboard для logs/metrics/traces

**Implementation**:

- [ ] Add `prometheus/client_golang` library for metrics
- [ ] Add OpenTelemetry SDK for tracing
- [ ] Create metrics middleware for HTTP requests
- [ ] Create tracing middleware (trace ID propagation)
- [ ] Add `/metrics` endpoint (Prometheus format)
- [x] Add VictoriaMetrics to `docker-compose.dev.yaml`
- [x] Add VictoriaLogs to `docker-compose.dev.yaml`
- [x] Add VictoriaTraces to `docker-compose.dev.yaml`
- [x] Add Promtail for log shipping
- [x] Add Grafana with auto-provisioned datasources
- [x] Add PostgreSQL Exporter for DB metrics
- [x] Add Redis Exporter for cache metrics
- [x] Configure Traefik metrics export
- [x] Configure Temporal metrics export
- [x] Create base dashboards (Overview, PostgreSQL, Redis, Traefik, Temporal)
- [x] Configure scraping from all infrastructure components
- [x] Write observability documentation (README, QUICKSTART, DNS setup)

**Metrics to Track**:

```go
// Authentication
auth_login_total{provider, status} // success, failed, rate_limited
auth_token_refresh_total{status}
auth_token_revoked_total
auth_session_revoked_total

// Performance
http_request_duration_seconds{endpoint, method, status}
http_requests_total{endpoint, method, status}
redis_operation_duration_seconds{operation}
graphql_query_duration_seconds{query}

// Rate Limiting
rate_limit_exceeded_total{endpoint}
rate_limit_requests_total{endpoint, status} // allowed, denied

// Health
postgres_connections_active
redis_connections_active
temporal_workflow_executions_total
```

**Tracing**:

```go
// OpenTelemetry instrumentation
// Trace request flow: Browser → Gateway → GraphQL → PostgreSQL
// Each span includes:
// - Service name (gateway, api-server)
// - Operation name (HTTP GET /api/users)
// - Duration, status, errors
// - Baggage (user_id, request_id)
```

**Docker Compose Addition**:

```yaml
victoriametrics:
  image: victoriametrics/victoria-metrics:latest
  restart: always
  ports:
    - "8428:8428"
  volumes:
    - vm_data:/victoria-metrics-data
  command:
    - "--storageDataPath=/victoria-metrics-data"
    - "--retentionPeriod=30d"
    - "--promscrape.config=/etc/prometheus/prometheus.yml"

victoriatraces:
  image: victoriametrics/victoria-traces:latest
  restart: always
  ports:
    - "4318:4318"   # OTLP HTTP
    - "14268:14268" # Jaeger HTTP
    - "9411:9411"   # Zipkin
  volumes:
    - vt_data:/victoria-traces-data
  command:
    - "--storageDataPath=/victoria-traces-data"
    - "--retentionPeriod=30d"

grafana:
  image: grafana/grafana:latest
  restart: always
  ports:
    - "3001:3000"
  environment:
    - GF_AUTH_ANONYMOUS_ENABLED=true
    - GF_AUTH_ANONYMOUS_ORG_ROLE=Admin
    - GF_INSTALL_PLUGINS=grafana-clickhouse-datasource
  volumes:
    - grafana_data:/var/lib/grafana
    - ./monitoring/grafana/provisioning:/etc/grafana/provisioning
  depends_on:
    - victoriametrics
    - victorialogs
    - victoriatraces
```

**Grafana Datasources** (`monitoring/grafana/provisioning/datasources/datasources.yml`):

```yaml
apiVersion: 1

datasources:
  - name: VictoriaMetrics
    type: prometheus
    access: proxy
    url: http://victoriametrics:8428
    isDefault: true

  - name: VictoriaLogs
    type: loki
    access: proxy
    url: http://victorialogs:9428

  - name: VictoriaTraces
    type: tempo
    access: proxy
    url: http://victoriatraces:4318
    jsonData:
      tracesToLogs:
        datasourceUid: victorialogs
        tags: ['trace_id']
      tracesToMetrics:
        datasourceUid: victoriametrics
        tags: [{key: 'service.name', value: 'service'}]
```

**Prometheus Scrape Config** (`monitoring/prometheus.yml`):

```yaml
scrape_configs:
  - job_name: 'gateway'
    static_configs:
      - targets: ['gateway:8080']
    metrics_path: '/metrics'
    scrape_interval: 15s

  - job_name: 'api-server'
    static_configs:
      - targets: ['api-server:8081']
    metrics_path: '/metrics'
    scrape_interval: 15s
```

**Grafana Dashboards**:

1. **Overview Dashboard**:
   - Request rate, error rate, latency (RED metrics)
   - Auth success/failure rate
   - Active users
   - Logs panel (errors only)

2. **Auth Dashboard**:
   - Login attempts (by provider)
   - Token refresh rate
   - Session duration
   - Failed login IPs
   - Traces for auth flow

3. **Performance Dashboard**:
   - Request latency histogram
   - Slow queries (>1s)
   - Redis operation latency
   - Database connection pool
   - Traces for slow requests

4. **Security Dashboard**:
   - Rate limit hits
   - Suspicious activities
   - Failed auth attempts by IP
   - Session revocations

**Victoria Stack Benefits**:

- ✅ Unified vendor (VictoriaMetrics company)
- ✅ Lightweight (perfect для Mac Mini)
- ✅ Prometheus/OpenTelemetry compatible
- ✅ Single Grafana UI для всего
- ✅ Correlation: Metrics → Logs → Traces
- ✅ Click на spike → see traces → see logs

---

## Phase 5: Account Management (Nice to Have) 🟢

### 5.1 Account Linking

**Status**: Not implemented
**Priority**: Nice to Have
**Effort**: 2 days

**Current Issues**:

- Нельзя связать несколько OAuth providers с одним user
- Один user = один identity provider
- Нельзя добавить Google после Apple login

**Implementation**:

- [ ] Allow multiple identities per user
- [ ] Add `POST /api/me/identities` endpoint to link new provider
- [ ] Merge accounts flow (detect existing email)
- [ ] UI for managing linked accounts
- [ ] Unlink identity (require at least one active)

**Database**:

- Table `user_identities` already supports multiple providers per user
- Just need to implement the linking flow

---

### 5.2 Apple OAuth Support

**Status**: Not implemented
**Priority**: Nice to Have
**Effort**: 1 day

**Implementation**:

- [ ] Register app in Apple Developer Portal
- [ ] Implement Apple OAuth flow (similar to Google)
- [ ] Handle Apple's unique user identifier
- [ ] Support Sign in with Apple button
- [ ] Handle Apple's privacy features (hide email)

---

## Phase 6: Production Hardening (Critical) 🔴

### 6.1 Backup & Recovery

**Status**: Not implemented
**Priority**: Critical
**Effort**: 1-2 days

**Current Issues**:

- PostgreSQL данные не бэкапятся (риск потери всех пользователей и audit logs)
- VictoriaMetrics/Logs данные не бэкапятся (потеря всей истории метрик)
- Redis данные не бэкапятся (потеря rate limits и будущих refresh tokens)
- Отказ диска = полная потеря данных
- Нет процесса восстановления (disaster recovery)

**Implementation**:

- [ ] Add Restic container to `docker-compose.dev.yaml`
- [ ] Configure Backblaze B2 as backup storage
- [ ] Create backup script for PostgreSQL (`pg_dump`)
- [ ] Add backup cron job (daily at 3 AM)
- [ ] Backup Docker volumes (postgres, victoriametrics, victorialogs)
- [ ] Implement retention policy (7 daily, 4 weekly, 12 monthly)
- [ ] Create restore script and documentation
- [ ] Test quarterly restore procedure
- [ ] Add monitoring for backup failures

**Backup Strategy**:

```yaml
# Docker Compose addition
backup:
  image: restic/restic:latest
  profiles:
    - tools
  environment:
    - RESTIC_REPOSITORY=b2:nexus-backups:/
    - RESTIC_PASSWORD=${RESTIC_PASSWORD}
    - B2_ACCOUNT_ID=${B2_ACCOUNT_ID}
    - B2_ACCOUNT_KEY=${B2_ACCOUNT_KEY}
  volumes:
    - postgres_dev_data:/data/postgres:ro
    - victoriametrics_data:/data/victoriametrics:ro
    - victorialogs_data:/data/victorialogs:ro
    - ./backup-scripts:/scripts
  command: /scripts/backup.sh
```

**Backup Script** (`backup-scripts/backup.sh`):

```bash
#!/bin/bash
set -e

# PostgreSQL dump
docker exec nexus-postgres-1 pg_dump -U ${POSTGRES_USER} ${POSTGRES_DB} > /tmp/postgres-backup.sql

# Initialize restic repo if needed
restic snapshots || restic init

# Create snapshot
restic backup \
  /data/postgres \
  /data/victoriametrics \
  /data/victorialogs \
  /tmp/postgres-backup.sql \
  --tag daily

# Prune old backups
restic forget --prune \
  --keep-daily 7 \
  --keep-weekly 4 \
  --keep-monthly 12
```

**Costs**:

- Backblaze B2: $5/TB/month storage + $10/TB egress
- Expected: ~5-10GB/month = $0.05/month storage
- Quarterly restore test: ~$0.10/year egress

---

### 6.2 Secrets Management

**Status**: Secrets in plaintext `.env`
**Priority**: Important
**Effort**: 0.5-1 day

**Current Issues**:

- `.env` файл содержит пароли в открытом виде
- PostgreSQL, Redis, JWT secrets доступны любому с доступом к файлу
- Нельзя безопасно хранить `.env` в Git
- Rotation секретов требует ручного редактирования

**Implementation**:

- [ ] Install Mozilla SOPS
- [ ] Generate PGP key or age key
- [ ] Store PGP key in macOS Keychain
- [ ] Encrypt `.env` file: `sops -e .env > .env.enc`
- [ ] Add `.env.enc` to Git, `.env` to `.gitignore`
- [ ] Create decrypt script for deployment
- [ ] Update deployment docs
- [ ] Rotate all secrets after implementation

**SOPS Setup**:

```bash
# Install SOPS
brew install sops age

# Generate age key
age-keygen -o ~/.config/sops/age/keys.txt

# Create .sops.yaml
cat > .sops.yaml <<EOF
creation_rules:
  - path_regex: \.env\.enc$
    age: age1ql3z7hjy54pw3hyww5ayyfg7zqgvc7w3j2elw8zmrj2kg5sfn9aqmcac8p
EOF

# Encrypt secrets
sops -e .env > .env.enc

# Decrypt on deployment
sops -d .env.enc > .env
docker-compose up -d
```

**Security Benefits**:

- ✅ Секреты зашифрованы в репозитории
- ✅ PGP/age key только на deployment машине
- ✅ История изменений секретов в Git
- ✅ Можно использовать разные ключи для dev/prod

---

### 6.3 Unified SSO (Ory Kratos + OAuth2)

**Status**: Infrastructure deployed, webhook integration pending
**Priority**: Important
**Effort**: 2-3 days

**Current Issues**:

- Gateway делает OAuth2 flow напрямую (нужно переделать)
- Grafana доступна с дефолтным паролем (admin/admin)
- VictoriaMetrics UI открыта без аутентификации
- VictoriaLogs UI открыта без аутентификации
- Traefik Dashboard открыт без аутентификации
- Temporal UI открыт без аутентификации
- Нет централизованного управления доступом для ВСЕХ сервисов
- Нет 2FA для критических сервисов
- Два разных логина (Nexus UI vs Grafana)

**Implementation**:

**Phase 1: Kratos Infrastructure** ✅ DONE

- [x] Add Ory Kratos to `docker-compose.dev.yaml`
- [x] Add Kratos Self-Service UI for login/registration pages
- [x] Add Ory Oathkeeper as forward auth middleware
- [x] Configure Kratos with PostgreSQL storage (reuse existing DB)
- [x] Configure Redis for session storage (reuse existing Redis)
- [x] Setup Google OAuth2 provider in Kratos config
- [x] Setup Apple OAuth2 provider in Kratos config
- [x] Configure Traefik forward auth middleware
- [x] Protect all services with Kratos SSO (Nexus UI, Grafana, Victoria stack, Traefik)
- [x] Create identity schema and OAuth mappers
- [x] Write complete documentation (README, QUICKSTART)

**Phase 2: Gateway Integration** ⏳ IN PROGRESS

- [ ] Implement webhook handler in Gateway (`/api/webhooks/kratos/registration`)
- [ ] Verify `X-Webhook-Secret` header
- [ ] Parse Kratos webhook payload (identity_id, email, provider, etc.)
- [ ] Create/update user in `users` table
- [ ] Create/update identity in `user_identities` table
- [ ] Implement header-based auth middleware
- [ ] Extract `X-User` and `X-User-Id` headers from Kratos forward auth
- [ ] Load user from database by ID
- [ ] Set user in request context

**Phase 3: Legacy OAuth Removal** ⏳ PENDING

- [ ] Remove OAuth2 logic from Gateway (`handlers/auth.go`)
- [ ] Remove Google OAuth environment variables (keeping for Kratos)
- [ ] Remove JWT token generation code
- [ ] Remove old auth middleware
- [ ] Update frontend to redirect to Kratos login page
- [ ] Remove JWT storage from frontend

**Phase 4: Testing & 2FA** ⏳ PENDING

- [ ] Test complete SSO flow (login → all services accessible)
- [ ] Test logout flow
- [ ] Test session expiration
- [ ] Enable TOTP 2FA in Kratos settings
- [ ] Document 2FA setup for users
- [ ] Test OAuth with real Google/Apple credentials

**Architecture**:

```text
┌────────────────────────────────────────────────────────────┐
│                   Internet (Public)                         │
│                                                             │
│  ┌──────────────────────────────────────────────────┐     │
│  │         Cloudflare Tunnel (Zero Trust)            │     │
│  │  - nexus.example.com → Nexus UI (BFF)            │     │
│  │  - auth.nexus.example.com → Authelia             │     │
│  └─────────────────────┬────────────────────────────┘     │
└────────────────────────┼────────────────────────────────────┘
                         │
┌────────────────────────┼────────────────────────────────────┐
│              Tailscale Mesh (Internal Only)                 │
│                        │                                     │
│  ┌─────────────────────▼──────────────────────────┐        │
│  │              Traefik (Reverse Proxy)            │        │
│  │                                                 │        │
│  │  ┌───────────────────────────────────────┐    │        │
│  │  │     Authelia (SSO Provider)            │    │        │
│  │  │  - Google OAuth2 upstream              │    │        │
│  │  │  - Apple OAuth2 upstream               │    │        │
│  │  │  - User storage: PostgreSQL            │    │        │
│  │  │  - Session storage: Redis              │    │        │
│  │  │  - 2FA (TOTP)                          │    │        │
│  │  └───────────────────────────────────────┘    │        │
│  │                     │                           │        │
│  │        Forward Auth Middleware                  │        │
│  └─────────────────────┼───────────────────────────┘        │
│                        │                                     │
│  ┌────────────────────────────────────────────────────┐    │
│  │                                                     │    │
│  │  ┌──────────────┐  ┌──────────┐  ┌──────────┐    │    │
│  │  │  Nexus UI    │  │ Grafana  │  │Victoria  │    │    │
│  │  │   (BFF)      │  │Protected │  │ Stack    │    │    │
│  │  │  ✅ Public   │  │🔒Internal│  │🔒Internal│    │    │
│  │  └──────┬───────┘  └──────────┘  └──────────┘    │    │
│  │         │                                          │    │
│  │    ┌────▼────────┐      ┌─────────────────┐      │    │
│  │    │ API Server  │      │    Traefik      │      │    │
│  │    │  (GraphQL)  │      │    Dashboard    │      │    │
│  │    │  🔒Internal │      │    🔒Internal   │      │    │
│  │    └─────────────┘      └─────────────────┘      │    │
│  │                                                    │    │
│  │    All 🔒Internal accessible ONLY via Tailscale   │    │
│  └────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────┘
```

**Authentication Flow**:

```text
1. User → nexus.example.com
2. Traefik checks Authelia session → Not authenticated
3. Redirect → auth.nexus.example.com/login
4. Authelia shows OAuth2 providers (Google, Apple)
5. User picks Google → Redirect to Google OAuth
6. Google callback → auth.nexus.example.com/callback
7. Authelia creates session in Redis
8. Authelia stores user in PostgreSQL
9. Redirect → nexus.example.com with session cookie
10. Gateway reads Remote-User header from Authelia
11. Gateway auto-creates/updates user in database
12. User sees Nexus UI - authenticated! ✅
```

**Docker Compose Addition**:

```yaml
authelia:
  image: authelia/authelia:latest
  restart: always
  volumes:
    - ./authelia:/config
  environment:
    - TZ=America/New_York
  labels:
    - "traefik.enable=true"
    - "traefik.http.routers.authelia.rule=Host(`auth.nexus.local`)"
    - "traefik.http.routers.authelia.entrypoints=web"
    - "traefik.http.middlewares.authelia.forwardauth.address=http://authelia:9091/api/verify?rd=https://auth.nexus.local"
    - "traefik.http.middlewares.authelia.forwardauth.trustForwardHeader=true"
    - "traefik.http.middlewares.authelia.forwardauth.authResponseHeaders=Remote-User,Remote-Groups,Remote-Name,Remote-Email"

# Update Grafana
grafana:
  labels:
    - "traefik.http.routers.grafana.middlewares=authelia@docker"

# Update VictoriaMetrics
victoriametrics:
  labels:
    - "traefik.http.routers.victoriametrics.middlewares=authelia@docker"
```

**Authelia Config** (`authelia/configuration.yml`):

```yaml
server:
  host: 0.0.0.0
  port: 9091

log:
  level: info

authentication_backend:
  file:
    path: /config/users_database.yml

access_control:
  default_policy: deny
  rules:
    - domain: "*.nexus.local"
      policy: two_factor

session:
  name: authelia_session
  domain: nexus.local
  expiration: 1h
  inactivity: 5m

storage:
  local:
    path: /config/db.sqlite3

notifier:
  filesystem:
    filename: /config/notification.txt
```

**Note**: Nexus UI (наше приложение) продолжает использовать собственный OAuth2 flow через Gateway. Authelia защищает только инфраструктурные сервисы.

---

## Summary

### Priorities

**Must Have (Critical)** 🔴

**IAM Core**:

1. Refresh Token Flow (1-2 days)
2. Token Revocation (1 day)

**Production Hardening**:

3. Backup & Recovery - Restic + B2 (1-2 days)
4. Secrets Management - SOPS (0.5-1 day)
5. SSO для Infrastructure - Authelia (1-2 days)

**Should Have (Important)** 🟡

6. Active Sessions Tracking (1-2 days)
7. Remote Session Revocation (1 day)
8. Security Headers (0.5 day)
9. CSRF Protection (1 day)
10. ~~Structured Logging + VictoriaLogs (1.5 days)~~ ✅ Infrastructure Ready
11. ~~VictoriaMetrics Integration (1 day)~~ ✅ Infrastructure Complete

**Nice to Have** 🟢

12. Account Linking (2 days)
13. Apple OAuth Support (1 day)

**Total Effort**: ~14.5-20 days for complete implementation
**Completed**: Victoria Observability Stack infrastructure (1.5 days equivalent)
**Remaining**: ~13-18.5 days

---

## Completed Work ✅

### Phase 4: Observability Infrastructure (Oct 23, 2025)

**Completed Components**:

- ✅ VictoriaMetrics (metrics storage, 90d retention)
- ✅ VictoriaLogs (log aggregation, 30d retention)
- ✅ VictoriaTraces (distributed tracing backend)
- ✅ Promtail (Docker log shipping)
- ✅ Grafana (unified dashboards)
- ✅ PostgreSQL Exporter (DB metrics)
- ✅ Redis Exporter (cache metrics)
- ✅ Traefik metrics export
- ✅ Temporal metrics export
- ✅ 5 Pre-configured Grafana dashboards
- ✅ Prometheus scrape configuration
- ✅ Complete observability documentation

**Dashboards Created**:

1. Nexus Overview - RED metrics (Rate, Errors, Duration)
2. Nexus PostgreSQL - Database health and performance
3. Nexus Redis - Cache metrics and connections
4. Nexus Traefik - Reverse proxy metrics
5. Nexus Temporal - Workflow execution metrics

**Access**:

- Grafana: <http://grafana.nexus.local> (admin/admin)
- VictoriaMetrics: <http://metrics.nexus.local>
- VictoriaLogs: <http://logs.nexus.local>

**Resource Usage**: ~310MB RAM (7x less than Prometheus/Loki/Jaeger stack)

**Next Steps for Phase 4**:

- Add zerolog structured logging to Go services
- Implement Prometheus metrics endpoints in Gateway/API/Worker
- Add OpenTelemetry tracing instrumentation

---

## Next Steps

### Recommended Implementation Order

**Phase 6: Production Hardening** (Critical - do FIRST) 🔴

1. **Backup & Recovery** (1-2 days)
   - Setup Restic + Backblaze B2
   - Daily automated backups
   - Test restore procedure
   - **Why first**: Защита от потери данных критична

2. **Secrets Management** (0.5-1 day)
   - Implement SOPS encryption
   - Rotate all secrets
   - **Why now**: Перед деплоем новых features

3. **SSO для Infrastructure** (1-2 days)
   - Setup Authelia
   - Protect Grafana/Victoria/Traefik
   - Enable 2FA
   - **Why now**: Текущие сервисы открыты без защиты

**Phase 1-3: IAM Core** (после hardening)

4. **Refresh Token Flow** (1-2 days)
5. **Token Revocation** (1 day)
6. **Session Management** (2-3 days)
7. **Security Headers + CSRF** (1.5 days)

**Phase 4: Application Instrumentation** (parallel to IAM)

8. **Structured Logging** (1 day) - zerolog в Gateway/API/Worker
9. **Application Metrics** (1 day) - Prometheus endpoints
10. **Distributed Tracing** (0.5 day) - OpenTelemetry

**Phase 5: Nice to Have**

11. **Account Linking** (2 days)
12. **Apple OAuth** (1 day)
   - Structured logging with zerolog
   - VictoriaLogs for log aggregation
   - VictoriaMetrics for metrics
   - Promtail for log shipping
6. Add **Nice-to-Have** features if time permits
