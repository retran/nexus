**Конечная цель:**

1. **Никаких S2S JWT.** `Gateway`, `data-api`, `internal-api` и `worker`
   общаются друг с другом _только_ по `https`.
2. **`Vault Agent` (Sidecar):** Каждый сервис (`gateway`, `data-api` и т.д.)
   получает свой TLS-сертификат (с CN `gateway.service.local`) от `Vault PKI`
   через `vault-agent`.
3. **`Gateway` (BFF):** Доверяет `Oathkeeper` не по S2S JWT, а потому что
   `Oathkeeper` предъявляет свой mTLS-сертификат
   (`cn=oathkeeper.service.local`).
4. **`internal-api` (Webhooks):** Kratos _по-прежнему_ использует
   `X-Webhook-Secret`. `Oathkeeper` выступает "переводчиком": он принимает
   `X-Webhook-Secret`, проверяет его, а затем вызывает `internal-api` от своего
   имени, используя свой `mTLS` сертификат.

Это устраняет _все_ JWT для S2S-связей и _весь_ "Network-Level Trust", заменяя
его криптографическим `mTLS`.

Вот **новый план** (начиная с `Commit 31`), который заменит всю вашу S2S
JWT-логику (`id_token`, `vault_token_client`, `jwks_verifier` и кастомный
плагин).

---

## 📅 План перехода на mTLS (Vault PKI Engine)

---

### Фаза 1: 🔐 Настройка Vault PKI (Центр Сертификации)

Commit 28 (НОВЫЙ): Добавить Ory Keto в docker-compose Research:

Изучить oryd/keto <https://www.ory.sh/docs/keto/getting-started/install-docker>.

Keto требует собственных миграций, как и Kratos.

Files:

docker-compose.dev.yaml

kratos/dev/keto.yml (новый)

.env.dev.example (добавить KETO_DSN)

postgres/init-kratos-db.sh (обновить, чтобы создавал и keto_db)

Changes:

docker-compose.dev.yaml:

Добавить сервис keto-migrate (keto migrate sql -e --yes).

Добавить сервис keto (keto serve -c ...).

keto должен зависеть от keto-migrate.

keto.yml: Создать конфиг, указывающий на KETO_DSN.

Test: task up запускает keto и keto-migrate без ошибок.

Commit: feat(iam): step-28 - add Ory Keto service skeleton

Commit 29 (НОВЫЙ): Cоздать и загрузить политики Keto Research:

Изучить синтаксис Ory Keto Policies
<https://www.ory.sh/docs/keto/concepts/policies>.

Изучить ory create policy CLI.

Files:

kratos/dev/policies/admin.json (новый)

Taskfile.yml (обновлен)

vault/Taskfile.yml (добавить keto:seed)

Changes:

admin.json: Создать базовую политику (Ory ACP):

JSON

{ "id": "policy-admin-access", "subjects": ["<role:admin>"], "actions":
["create", "read", "update", "delete"], "resources": ["api:admin:<.*>"],
"effect": "allow" } Taskfile.yml: Добавить task keto:seed.

Скрипт keto:seed должен вызывать docker-compose exec keto ory create policy ....

Test: task keto:seed успешно загружает политику.

#### Commit 31 (НОВЫЙ): Настроить Vault PKI Engine

**Research**:

- `vault secrets enable pki`
- `vault pki root generate internal`
- `vault pki int generate internal`

**Files**:

- `vault/scripts/init-pki.sh` (новый)
- `vault/scripts/bootstrap-dev.sh` (обновлен)

**Changes**:

1. **`init-pki.sh`**:
   - `vault secrets enable -path=pki_int pki` (Промежуточный CA)
   - `vault pki tune -max-lease-ttl=43800h pki_int` (TTL для CA)
   - `vault write pki_int/intermediate/generate/internal common_name="Nexus Dev Intermediate CA"`
2. **`bootstrap-dev.sh`**:
   - Добавить вызов `bash scripts/init-pki.sh`.

**Test**: `task vault:init` создает Intermediate CA в `Vault`.

**Commit**: `feat(iam): step-31 - configure Vault PKI engine for mTLS`

---

#### Commit 32 (НОВЫЙ): Создать PKI-роли и политики для сервисов

**Research**:

- `vault write pki_int/roles/...` (для выдачи сертов)
- Обновить политики AppRole (для запроса сертов)

**Files**:

- `vault/scripts/init-pki.sh` (обновлен)
- `vault/scripts/init-policies.sh` (обновлен)

**Changes**:

1. **`init-pki.sh`**:
   - **Создать PKI-роли:**
     - `vault write pki_int/roles/oathkeeper-role allowed_domains="oathkeeper.service.local" ttl="1h"`
     - `vault write pki_int/roles/gateway-role allowed_domains="gateway.service.local" ttl="1h"`
     - `vault write pki_int/roles/data-api-role allowed_domains="data-api.service.local" ttl="1h"`
     - `vault write pki_int/roles/internal-api-role allowed_domains="internal-api.service.local" ttl="1h"`
     - `vault write pki_int/roles/worker-role allowed_domains="worker.service.local" ttl="1h"`
2. **`init-policies.sh`**:
   - **Удалить** права на `transit/sign/*`.
   - **Добавить** права на PKI:
     - Политика `app-gateway` должна включать:
       `path "pki_int/issue/gateway-role" { capabilities = ["update"] }`
     - Политика `app-data-api` должна включать:
       `path "pki_int/issue/data-api-role" { capabilities = ["update"] }`
     - ...и так далее для всех сервисов.

**Test**: `task vault:init` создает PKI-роли.

**Commit**: `feat(iam): step-32 - define PKI roles and policies for mTLS`

---

### Фаза 2: 🚗 Внедрение Vault Agent Sidecar (для всех)

#### Commit 33 (НОВЫЙ): Создать конфиг Vault Agent для mTLS

**Files**:

- `vault/agent/template.hcl` (новый)
- `vault/agent/cert.tpl` (новый)
- `vault/agent/key.tpl` (новый)
- `vault/agent/ca.tpl` (новый)
- `vault/agent/webhook.tpl` (новый, для `kratos-agent`)

**Changes**:

1. **`template.hcl`**: Общий HCL-конфиг для `vault-agent`:

   ```hcl
   vault { address = "http://vault:8200" }
   auto_auth { method "approle" { config = { ... } } }

   // Рендерит CA (один раз)
   template {
     source = "/config/ca.tpl"
     destination = "/secrets/vault-ca.pem"
     command = "touch /secrets/.ca-ready"
   }

   // Рендерит серт (и ротирует)
   template {
     source = "/config/cert.tpl"
     destination = "/secrets/tls.crt"
     command = "touch /secrets/.cert-ready"
   }
   // ... (и для .key)
   ```

2. **`ca.tpl`**:
   `{{- with secret "pki_int/ca/pem" -}}{{ .Data.certificate }}{{- end -}}`
3. **`cert.tpl`**:
   `{{- $role := env "PKI_ROLE" -}} ... {{- with secret (printf "pki_int/issue/%s" $role) "common_name=..." -}}{{ .Data.certificate }}{{- end -}}`
   (аналогично для `key.tpl`).
4. **`webhook.tpl`** (для `kratos-agent`):
   `{{- with secret "kv/data/shared/webhook" -}}{{ .Data.data.current }}{{- end -}}`

**Commit**:
`feat(iam): step-33 - create Vault Agent templates for mTLS & webhooks`

---

#### Commit 34 (НОВЫЙ): Внедрить Sidecar-контейнеры

**Files**:

- `docker-compose.dev.yaml`
- `.env.dev.example`
- `vault/scripts/bootstrap-dev.sh` (обновлен)

**Changes**:

1. **`bootstrap-dev.sh`**: Обновить скрипт, чтобы он _читал_ `role_id/secret_id`
   из `vault/.dev/approles/*.json` и _записывал_ их в `.env` (например,
   `GATEWAY_ROLE_ID=...`). `docker-compose` будет читать их из `.env`.
2. **`docker-compose.dev.yaml`**:
   - Создать 6 `tmpfs` volumes: `gateway-secrets`, `data-api-secrets` и т.д.
   - Для **каждого** сервиса (`gateway`, `data-api`, `internal-api`, `worker`,
     `oathkeeper`, `kratos`):
     - **Добавить `vault-agent-*` sidecar** (`image: hashicorp/vault`).
     - `command: vault agent -config=/config/template.hcl`
     - `volumes`: `./vault/agent:/config:ro`, `[service]-secrets:/secrets`.
     - `environment`: `PKI_ROLE=[service]-role`,
       `COMMON_NAME=[service].service.local`, `VAULT_ROLE_ID=...` (из `.env`).
     - **Обновить основной сервис**:
       - `hostname: [service].service.local`
       - `volumes`: `[service]-secrets:/secrets:ro`
       - `depends_on`: `vault-agent-[service]`.
       - `command`: Добавить `wait-for`-скрипт (ожидание
         `/secrets/.cert-ready`).

**Commit**: `feat(iam): step-34 - implement mTLS Sidecars for all services`

---

### Фаза 3: 🔌 Переключение Сервисов на mTLS

#### Commit 35 (НОВЫЙ): Перевести Go-сервисы на `ListenAndServeTLS`

**Files**:

- `backend/cmd/gateway/main.go`
- `backend/cmd/data-api/main.go`
- `backend/cmd/internal-api/main.go`
- `backend/cmd/worker/main.go`

**Changes**:

- **Все `main.go`**:
  - Заменить `http.ListenAndServe` на
    `http.ListenAndServeTLS("/secrets/tls.crt", "/secrets/tls.key")`.
  - Настроить `tls.Config` сервера, чтобы он _требовал_ клиентские сертификаты:
    - `ClientAuth: tls.RequireAndVerifyClientCert`
    - `ClientCAs`: Загрузить `/secrets/vault-ca.pem`.
- **Все S2S-клиенты** (в `gateway`, `worker`):
  - `http.Client` должен использовать `http.Transport` с `tls.Config`, который
    загружает _свой_ клиентский сертификат (`/secrets/tls.crt`) и CA.
- **Удалить** всю S2S JWT-логику: `vault_token_client.go`,
  `service_token_transport.go`, `jwt_verifier.go`.

**Commit**: `feat(iam): step-35 - enable mTLS server/client in all Go services`

---

#### Commit 36 (НОВЫЙ): Переписать AuthN Middleware на mTLS (CN Check)

**Files**:

- `backend/internal/api/rest/middleware/auth.go` (`gateway`)
- `backend/cmd/data-api/main.go` (добавить `mTLSAuthMiddleware`)
- `backend/cmd/internal-api/main.go` (обновить `mTLSAuthMiddleware`)
- `backend/internal/auth/jwks_verifier.go` (УДАЛИТЬ)

**Changes**:

1. **Удалить `jwks_verifier.go`**.
2. **`gateway` (`auth.go`):**
   - `AuthMiddleware` теперь проверяет
     `r.TLS.PeerCertificates[0].Subject.CommonName`.
   - Если `cn != "oathkeeper.service.local"`, то `403 Forbidden`.
   - Если `cn == "oathkeeper.service.local"`, то читаем `X-User-ID` /
     `X-User-Role` (Trusted Subsystem).
3. **`data-api` (`main.go`):**
   - Добавить `mTLSAuthMiddleware`: `cn == "gateway.service.local"` ИЛИ
     `cn == "worker.service.local"` -\> `allow`.
4. **`internal-api` (`main.go`):**
   - Добавить `mTLSAuthMiddleware`:
     - `cn == "gateway.service.local"` (для `/admin/*`) -\> `allow`.
     - _Пока_ оставить `WebhookAuthMiddleware` (для `X-Webhook-Secret`), он
       будет заменен в `Commit 37`.

**Commit**:
`refactor(iam): step-36 - switch AuthN middleware from JWT to mTLS CN validation`

---

#### Commit 37 (НОВЫЙ): Переключить Oathkeeper и Kratos Webhooks

**Files**:

- `kratos/dev/kratos.yml`
- `kratos/dev/oathkeeper.yml`
- `kratos/dev/access-rules.yml`
- `backend/cmd/internal-api/main.go`

**Changes**:

1. **`kratos.yml`**:
   - `url:` вебхука:
     `http://oathkeeper:4455/api/v1/internal/webhooks/kratos/registration`
   - `auth:` (вебхук): `type: api_key`, `value: file:///secrets/webhook.key`
     (этот файл рендерит `vault-agent-kratos`).
2. **`oathkeeper.yml`**:
   - `upstream:` (для `gateway`): настроить на mTLS
     (`httpsS://gateway.service.local:8080`, используя
     `tls.client_certificate_path: /secrets/tls.crt` от
     `vault-agent-oathkeeper`).
3. **`access-rules.yml`**:
   - **Правило "Admin" (`protected:api:admin`):**
     - `authenticator: cookie_session`
     - `authorizer: keto_engine_acp_ory` (настроен на `keto:4466`)
     - `mutators: [header]`
     - `upstream`: `https://gateway.service.local:8080` (уже настроен в
       `oathkeeper.yml`)
   - **Правило "Kratos Webhook" (НОВОЕ):**
     - `id: webhook-kratos`
     - `match`: `host: <любой, т.к. Kratos идет по имени сервиса>`,
       `path: /api/v1/internal/webhooks/kratos/<*>`
     - `authenticator: api_key` (проверяет `X-Webhook-Secret`, который Kratos
       прочитал из `/secrets/webhook.key`)
     - `authorizer: allow`
     - `mutators: noop`
     - `upstream`: `httpshttps://internal-api.service.local:8083` (Oathkeeper
       использует свой mTLS-серт для вызова).
4. **`internal-api` (`main.go`):**
   - **Удалить** `WebhookAuthMiddleware` (проверку `X-Webhook-Secret`).
   - **Обновить `mTLSAuthMiddleware`:**
     - `cn == "gateway.service.local"` -\> `allow`.
     - `cn == "oathkeeper.service.local"` (для `/webhooks/*`) -\> `allow`.

**Commit**:
`feat(iam): step-37 - reroute Kratos webhooks via mTLS-enabled Oathkeeper`

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
