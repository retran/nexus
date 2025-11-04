## Commit 1: Refactor Gateway S2S client to mTLS-only

Research:

Review mTLS client configuration in http.Client.

Identify all S2S JWT generation logic in gateway.

Files:

backend/internal/api/rest/server.go

backend/internal/api/rest/service_token_transport.go (DELETE)

backend/internal/auth/vault_token_client.go (DELETE)

backend/go.mod

backend/go.sum

Changes:

Modify createInternalAPIClient in server.go to use the same mTLS-only transport
as createGraphQLClient.

Remove newServiceTokenTransport и TokenClient из инициализации gateway.

Delete service_token_transport.go и vault_token_client.go (т.к. jwt_verifier.go
все еще нужен для OIDC).

Run go mod tidy.

Test: gateway service builds. internalAPIClient successfully connects to
internal-api using only mTLS.

Commit: refactor(gateway): step-1 - remove S2S JWT generation for internal-api
client

## Commit 2: Refactor internal-api to trust mTLS CN

Research:

Confirm mTLS CN validation in internal-api/main.go.

Identify JWT-consuming logic in internal-api.

Files:

backend/internal/internalapi/middleware/jwt.go (DELETE)

backend/internal/internalapi/handlers/admin.go

backend/internal/internalapi/main.go

Changes:

Delete internal-api/middleware/jwt.go and all its usages.

Remove JWTMiddleware from main.go.

Modify adminRoleHandler in admin.go to remove all dependencies on
TokenInfoFromContext.

Handler adminRoleHandler (и auditHandler) теперь полагается исключительно на
mTLSAuthMiddleware (в main.go), который проверяет CN (gateway.service.local или
oathkeeper.service.local).

Test: internal-api builds. adminRoleHandler correctly processes requests,
доверяя вызову от gateway.

Commit: refactor(internal-api): step-2 - remove S2S JWT validation, rely on mTLS
CN

## Commit 3: SSO for Traefik Dashboard

Research:

Review Traefik dashboard authentication options.

Review Oathkeeper remote_json authorizer.

Files:

ory/dev/access-rules.yml

docker-compose.dev.yaml

Changes:

В docker-compose.dev.yaml: убедиться, что traefik.http.routers.traefik не имеет
приоритета и будет перехвачен Oathkeeper. (Убрать priority=100, если есть, или
убедиться, что он ниже 50).

В access-rules.yml добавить новое правило:

id: "protected:traefik-dashboard"

match: url: "<http://traefik.nexus.local/><\*\*>"

upstream: url: "<http://traefik:8080>"

authenticators: [{ "handler": "cookie_session" }]

authorizer: handler: remote_json, config: { remote:
"<http://gateway:8080/api/internal/authorize>" } (требуем роль admin).

errors: redirect to <http://auth.nexus.local/login>.

Test: Доступ к <http://traefik.nexus.local> без логина перенаправляет на Kratos.
Доступ с ролью member запрещен. Доступ с ролью admin открывает дашборд.

Commit: feat(auth): step-3 - secure traefik dashboard with ory stack sso

## Commit 4: SSO Client Bootstrap Script

Research:

Kratos Admin API documentation for OIDC client management (/admin/clients).

Files:

ory/scripts/bootstrap_oidc_clients.sh (NEW)

Taskfile.yml

.env.dev.example

Changes:

Создать bootstrap_oidc_clients.sh. Скрипт должен:

Использовать curl (внутри docker compose exec kratos ...) для вызова
kratos:4434/admin/clients.

Идемпотентно (проверять GET перед POST/PUT) создавать/обновлять OIDC-клиентов
для temporal-ui и grafana-ui (на будущее).

Конфигурация (scopes, redirect_uris, client_name) должна быть захардкожена в
скрипте.

Добавить TEMPORAL_OIDC_CLIENT_ID и TEMPORAL_OIDC_CLIENT_SECRET в
.env.dev.example (можно захардкодить dev-temporal-id / dev-temporal-secret для
простоты).

Добавить task sso:clients в Taskfile.yml для вызова этого скрипта.

Test: Запуск task sso:clients создает OIDC-клиента в Kratos.

Commit: feat(auth): step-4 - add script to bootstrap sso clients in kratos

Commit 5: SSO for Temporal UI

Research:

Temporal UI OIDC configuration docs.

Files:

docker-compose.dev.yaml

.env.dev.example (уже обновлен)

Changes:

Добавить переменные окружения в сервис temporal:

TEMPORAL_UI_AUTH_ENABLED=true

TEMPORAL_UI_AUTH_AUTHORIZATION_PROVIDER=default

TEMPORAL_UI_OIDC_ISSUER_URL=<http://auth.nexus.local>

TEMPORAL_UI_OIDC_PROVIDER_NAME=Kratos

TEMPORAL_UI_OIDC_CLIENT_ID=${TEMPORAL_OIDC_CLIENT_ID}

TEMPORAL_UI_OIDC_CLIENT_SECRET=${TEMPORAL_OIDC_CLIENT_SECRET}

TEMPORAL_UI_OIDC_CALLBACK_URL=<http://localhost:8088/auth/sso/oidc/callback>

TEMPORAL_UI_OIDC_SCOPES=openid,profile,email

Test: Доступ к <http://localhost:8088> перенаправляет на Kratos OIDC. После
логина открывается Temporal UI.

Commit: feat(auth): step-5 - configure temporal ui for kratos oidc sso

## Commit 6: Vault PKI & AppRole for Temporal

Research:

Review Vault PKI and AppRole bootstrap scripts.

Files:

vault/scripts/init-pki.sh

vault/scripts/init-policies.sh

vault/scripts/bootstrap-dev.sh

.env.dev.example

Changes:

В init-pki.sh: Добавить temporal-role (аналогично worker-role,
CN=temporal.service.local).

В init-policies.sh: Добавить app-temporal (аналогично app-worker).

В bootstrap-dev.sh: Добавить temporal в services=(...).

В .env.dev.example: Добавить TEMPORAL_ROLE_ID и TEMPORAL_SECRET_ID.

Test: task vault:init успешно выполняется, создавая AppRole для temporal.

Commit: feat(vault): step-6 - add pki and approle for temporal service

## Commit 7: Secure Temporal gRPC API with mTLS

Research:

Temporal server TLS configuration (temporalio/auto-setup image env vars).

Files:

docker-compose.dev.yaml

Changes:

Добавить vault-agent-temporal (по аналогии с vault-agent-worker).

Добавить temporal-secrets (tmpfs volume).

В сервис temporal:

Удалить ports: - "7233:7233". (Оставить 8088:8088 для UI).

Добавить depends_on: vault-agent-temporal.

Добавить volumes: - temporal-secrets:/secrets:ro.

Добавить env vars:

TEMPORAL_TLS_CERT_PATH=/secrets/tls.crt

TEMPORAL_TLS_KEY_PATH=/secrets/tls.key

TEMPORAL_TLS_CA_PATH=/secrets/vault-ca.pem

TEMPORAL_TLS_REQUIRE_CLIENT_AUTH=true

TEMPORAL_TLS_ENABLE_HOST_VERIFICATION=false

TEMPORAL_TLS_CLIENT_CA_PATH=/secrets/vault-ca.pem

Test: Сервис temporal запускается, слушает gRPC по mTLS. worker не может
подключиться (пока).

Commit: feat(temporal): step-7 - secure temporal grpc api with mtls

## Commit 8: Update Temporal Clients for mTLS

Research:

Temporal Go SDK client.Options for TLS.

Go crypto/tls config loading.

Files:

backend/internal/mtls/client.go (NEW)

backend/worker/main.go

backend/internal-api/main.go

Changes:

Создать backend/internal/mtls/client.go: LoadClientTLSConfig(caPath, certPath,
keyPath) (загружает tls.Config из /secrets/\* файлов).

В worker/main.go: run() вызывает mtls.LoadClientTLSConfig() и передает
tls.Config в temporalclient.Dial(client.Options{ ConnectionOptions: { TLS:
tlsConfig } }).

В internal-api/main.go: newTemporalClient() делает то же самое.

Test: worker и internal-api успешно подключаются к temporal:7233 по mTLS.

Commit: refactor(clients): step-8 - update temporal clients to use mtls

## Commit 9: Secure Kratos Admin API with mTLS

Research:

Ory Kratos serve.admin.tls configuration.

Files:

ory/dev/kratos.yml

docker-compose.dev.yaml

Changes:

В kratos.yml:

Изменить serve.admin.base_url на <https://kratos:4434>.

Добавить serve.admin.tls:

enabled: true

key_path: /secrets/tls.key

cert_path: /secrets/tls.crt

client_ca_path: /secrets/vault-ca.pem

В docker-compose.dev.yaml: сервис kratos уже имеет vault-agent-kratos и
kratos-secrets, mTLS должен заработать "из коробки".

Test: kratos запускается, Admin API (:4434) слушает mTLS. gateway и internal-api
не могут подключиться (пока).

Commit: feat(kratos): step-9 - secure kratos admin api with mtls

## Commit 10: Update Kratos Admin API Clients for mTLS

Research:

Go http.Client mTLS transport configuration.

Files:

backend/internal/mtls/client.go (UPDATE)

backend/internal/api/rest/server.go

backend/internal/api/rest/handlers/me.go

backend/internal-api/main.go

backend/internalapi/handlers/admin.go

Changes:

В mtls/client.go: Добавить NewMTLSHttpClient() (возвращает *http.Client с
*http.Transport{ TLSClientConfig: tlsConfig }).

В internal-api/main.go:

initServices: Вызвать mtls.NewMTLSHttpClient().

Передать httpClient в handlers.NewAdminHandler().

В internalapi/handlers/admin.go:

Обновить NewAdminHandler для приема \*http.Client.

Использовать h.client.Do(reqHTTP) вместо &http.Client{}.

В backend/internal/api/rest/server.go:

New: Вызвать mtls.NewMTLSHttpClient() (или аналог, createMTLSHTTPClient).

Передать httpClient в handlers.NewMeHandlers().

В backend/internal/api/rest/handlers/me.go:

Обновить NewMeHandlers для приема \*http.Client.

В Logout использовать h.httpClient.Do(req) вместо &http.Client{}.

Test: Logout (через gateway) и смена ролей (через internal-api) успешно
выполняются, подключаясь к Kratos Admin API по mTLS.

Commit: refactor(clients): step-10 - update kratos admin clients to use mtls

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
