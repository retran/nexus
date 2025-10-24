# Nexus Observability Stack

Complete observability solution for Nexus using VictoriaMetrics ecosystem.

## Architecture

```text
┌─────────────────────────────────────────────────────────────────┐
│                       Application Layer                          │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐       │
│  │ Gateway  │  │API Server│  │  Worker  │  │   UI     │       │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘  └────┬─────┘       │
│       │             │              │             │              │
│       └─────────────┴──────────────┴─────────────┘              │
│                      │                                           │
└──────────────────────┼───────────────────────────────────────────┘
                       │
┌──────────────────────┼───────────────────────────────────────────┐
│              Infrastructure Layer                                │
│  ┌──────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐          │
│  │PostgreSQL│  │  Redis  │  │Temporal │  │ Traefik │          │
│  │          │  │         │  │         │  │         │          │
│  │  +───────┤  │  +──────┤  │         │  │         │          │
│  │Exporter │  │ Exporter│  │ Metrics │  │ Metrics │          │
│  └────┬─────┘  └────┬────┘  └────┬────┘  └────┬────┘          │
│       │             │            │            │                 │
│       └─────────────┴────────────┴────────────┘                 │
│                      │                                           │
└──────────────────────┼───────────────────────────────────────────┘
                       │
┌──────────────────────┼───────────────────────────────────────────┐
│              Observability Layer                                 │
│                      │                                           │
│       ┌──────────────┴──────────────┐                           │
│       │                              │                           │
│  ┌────▼─────────┐            ┌──────▼─────┐                    │
│  │VictoriaMetrics│            │  Promtail  │                    │
│  │  (Metrics)    │            │ (Log Ship) │                    │
│  └────┬──────────┘            └──────┬─────┘                    │
│       │                              │                           │
│       │                       ┌──────▼─────┐                    │
│       │                       │VictoriaLogs│                    │
│       │                       │   (Logs)   │                    │
│       │                       └──────┬─────┘                    │
│       │                              │                           │
│  ┌────▼──────────────────────────────▼─────┐                   │
│  │           Grafana                        │                   │
│  │  (Unified Visualization & Dashboards)    │                   │
│  └──────────────────────────────────────────┘                   │
│                                                                  │
└──────────────────────────────────────────────────────────────────┘
```

## Components

### VictoriaMetrics (Metrics Storage)

- **Purpose**: Prometheus-compatible time-series database for metrics
- **Port**: 8428
- **URL**: <http://metrics.nexus.local>
- **Retention**: 90 days
- **Features**:
  - 7x more resource-efficient than Prometheus
  - Prometheus scrape config support
  - PromQL query language
  - High compression ratio

### VictoriaLogs (Log Aggregation)

- **Purpose**: Log aggregation and storage
- **Port**: 9428
- **URL**: <http://logs.nexus.local>
- **Retention**: 30 days
- **Features**:
  - LogQL query language (Loki-compatible)
  - JSON log parsing
  - Label-based filtering
  - High ingestion performance

### VictoriaTraces (Distributed Tracing)

- **Purpose**: Distributed tracing backend
- **Ports**:
  - 4318 (OpenTelemetry HTTP)
  - 14268 (Jaeger HTTP)
  - 9411 (Zipkin)
- **URL**: <http://traces.nexus.local>
- **Features**:
  - Multiple protocol support
  - Stores traces in VictoriaMetrics
  - Service dependency graphs
  - Trace-to-logs correlation

### Promtail (Log Shipping)

- **Purpose**: Collect and ship Docker container logs to VictoriaLogs
- **Features**:
  - Auto-discovery of Docker containers
  - JSON log parsing
  - Label extraction
  - Pipeline stages for log processing

### Grafana (Visualization)

- **Purpose**: Unified observability dashboard
- **Port**: 3001
- **URL**: <http://grafana.nexus.local>
- **Credentials**: admin/admin (default)
- **Features**:
  - Pre-configured datasources (VictoriaMetrics, VictoriaLogs, VictoriaTraces)
  - 5 pre-built dashboards
  - Trace-to-logs correlation
  - Metrics-to-logs correlation

## Metrics Exporters

### PostgreSQL Exporter

- **Port**: 9187
- **Metrics**:
  - Active connections
  - Transaction rate (commits/rollbacks)
  - Cache hit ratio
  - Database size
  - Query performance

### Redis Exporter

- **Port**: 9121
- **Metrics**:
  - Commands rate
  - Connected clients
  - Memory usage
  - Cache hit rate
  - Key count
  - Eviction stats

### Traefik (Built-in Metrics)

- **Port**: 8080 (via `/metrics`)
- **Metrics**:
  - Request rate by entrypoint
  - Response codes distribution
  - Request duration (p95/p99)
  - Service health
  - Open connections

### Temporal (Built-in Metrics)

- **Port**: 8000 (via `/metrics`)
- **Metrics**:
  - Workflow execution rate
  - Workflow duration
  - Task queue depth
  - Active workflows
  - Worker slots

## Dashboards

All dashboards are auto-provisioned on Grafana startup.

### 1. Nexus Overview

- **UID**: `nexus-overview`
- **Panels**:
  - Request Rate (RED - Rate)
  - Error Rate (RED - Errors)
  - Request Duration p95 (RED - Duration)
  - Recent Errors (logs)

### 2. Nexus PostgreSQL

- **UID**: `nexus-postgres`
- **Panels**:
  - Active Connections
  - Transaction Rate (commits/rollbacks)
  - Cache Hit Ratio
  - Database Size

### 3. Nexus Redis

- **UID**: `nexus-redis`
- **Panels**:
  - Commands Rate
  - Connected Clients
  - Memory Usage
  - Cache Hit Rate
  - Total Keys

### 4. Nexus Traefik

- **UID**: `nexus-traefik`
- **Panels**:
  - Request Rate by Status Code
  - Request Duration (p95/p99)
  - Requests per Service
  - Open Connections
  - Healthy Backends

### 5. Nexus Temporal

- **UID**: `nexus-temporal`
- **Panels**:
  - Workflow Execution Rate
  - Workflow Duration (p95/p99)
  - Task Queue Depth
  - Active Workflows
  - Worker Slots Available

## Configuration Files

- `prometheus-config.yaml`: Prometheus scrape configuration for VictoriaMetrics
- `promtail-config.yaml`: Promtail configuration for log shipping
- `grafana/provisioning/datasources/victoria.yaml`: Grafana datasources
- `grafana/provisioning/dashboards/default.yaml`: Dashboard provider config
- `grafana/provisioning/dashboards/*.json`: Pre-built dashboards

## Usage

### Starting the Observability Stack

```bash
# Start all services
docker-compose -f docker-compose.dev.yaml up -d

# Start only observability services
docker-compose -f docker-compose.dev.yaml up -d victoriametrics victorialogs victoriatraces promtail grafana postgres-exporter redis-exporter
```

### Viewing Metrics

1. Open Grafana: <http://grafana.nexus.local>
2. Login with admin/admin
3. Navigate to Dashboards → Nexus folder
4. Select desired dashboard

### Querying Logs

**Via Grafana:**

1. Open Grafana → Explore
2. Select "VictoriaLogs" datasource
3. Use LogQL queries:

```logql
# All logs from gateway service
{service="gateway"}

# Error logs
{service="gateway"} |= "error"

# Logs with specific request ID
{service="gateway"} | json | request_id="abc-123"
```

**Via VictoriaLogs UI:**

1. Open <http://logs.nexus.local>
2. Use the query interface

### Querying Metrics

**Via Grafana:**

1. Open Grafana → Explore
2. Select "VictoriaMetrics" datasource
3. Use PromQL queries:

```promql
# Request rate
rate(nexus_http_requests_total[5m])

# P95 latency
histogram_quantile(0.95, rate(nexus_http_request_duration_seconds_bucket[5m]))

# PostgreSQL connections
pg_stat_database_numbackends{datname="nexus_db"}
```

**Via VictoriaMetrics UI:**

1. Open <http://metrics.nexus.local>
2. Use the VMUI query interface

## Adding Custom Metrics

To add metrics from your Go services:

1. Import Prometheus client:

   ```go
   import "github.com/prometheus/client_golang/prometheus"
   ```

2. Register metrics endpoint:

   ```go
   http.Handle("/metrics", promhttp.Handler())
   ```

3. Add scrape config to `prometheus-config.yaml`:

   ```yaml
   - job_name: 'my-service'
     static_configs:
       - targets: ['my-service:8080']
         labels:
           service: 'my-service'
   ```

4. Restart VictoriaMetrics:

   ```bash
   docker-compose -f docker-compose.dev.yaml restart victoriametrics
   ```

## Troubleshooting

### Metrics not appearing

1. Check VictoriaMetrics targets: <http://metrics.nexus.local/targets>
2. Verify exporter is running: `docker-compose ps`
3. Check logs: `docker-compose logs postgres-exporter`

### Logs not appearing

1. Check Promtail logs: `docker-compose logs promtail`
2. Verify VictoriaLogs is running: `docker-compose ps victorialogs`
3. Check container labels in `promtail-config.yaml`

### Grafana dashboards empty

1. Verify datasources: Grafana → Configuration → Data sources
2. Check datasource URLs are correct
3. Test datasource connection
4. Verify VictoriaMetrics/VictoriaLogs are receiving data

## Resource Usage

Typical resource consumption on Mac Mini M4:

- **VictoriaMetrics**: ~50MB RAM, minimal CPU
- **VictoriaLogs**: ~30MB RAM, minimal CPU
- **VictoriaTraces**: ~40MB RAM, minimal CPU
- **Promtail**: ~20MB RAM, minimal CPU
- **Grafana**: ~150MB RAM, minimal CPU
- **Postgres Exporter**: ~10MB RAM, minimal CPU
- **Redis Exporter**: ~10MB RAM, minimal CPU

**Total**: ~310MB RAM for complete observability stack

Compare to traditional stack (Prometheus + Loki + Jaeger + Grafana): ~2GB RAM

## Security Notes

- Default Grafana credentials are admin/admin - **change in production**
- VictoriaMetrics/VictoriaLogs have no authentication - use behind Traefik with
  auth
- Exporters expose sensitive database connection info - keep internal
- Consider enabling HTTPS in production with Traefik TLS

## References

- [VictoriaMetrics Documentation](https://docs.victoriametrics.com/)
- [VictoriaLogs Documentation](https://docs.victoriametrics.com/victorialogs/)
- [VictoriaTraces Documentation](https://docs.victoriametrics.com/victoriatraces/)
- [Grafana Documentation](https://grafana.com/docs/)
- [Prometheus Client Libraries](https://prometheus.io/docs/instrumenting/clientlibs/)
