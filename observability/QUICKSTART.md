# Nexus Observability Stack - Quick Start

## ✅ Setup Checklist

### 1. DNS Configuration

```bash
sudo bash -c 'cat >> /etc/hosts << EOF

# Nexus Observability Stack
127.0.0.1 metrics.nexus.local
127.0.0.1 logs.nexus.local
127.0.0.1 traces.nexus.local
127.0.0.1 grafana.nexus.local
EOF'
```

### 2. Start the Stack

```bash
cd /Users/retran/workspace/nexus
docker-compose -f docker-compose.dev.yaml up -d
```

### 3. Verify Services

```bash
# Check all services are running
docker-compose -f docker-compose.dev.yaml ps

# Check VictoriaMetrics targets
curl http://metrics.nexus.local/targets

# Check Promtail is shipping logs
docker-compose logs promtail | grep "Successfully sent"
```

### 4. Access Grafana

1. Open: <http://grafana.nexus.local>
2. Login: admin/admin
3. Navigate to: Dashboards → Nexus folder
4. Open any dashboard to verify data flow

## 📊 Available Dashboards

- ✅ **Nexus Overview** - RED metrics (Rate, Errors, Duration)
- ✅ **Nexus PostgreSQL** - Database health and performance
- ✅ **Nexus Redis** - Cache metrics and connections
- ✅ **Nexus Traefik** - Reverse proxy and routing metrics
- ✅ **Nexus Temporal** - Workflow execution metrics

## 🔧 What's Configured

### Metrics Collection (VictoriaMetrics)

- ✅ PostgreSQL Exporter (port 9187)
- ✅ Redis Exporter (port 9121)
- ✅ Traefik Prometheus metrics (port 8080)
- ✅ Temporal Prometheus metrics (port 8000)
- ✅ VictoriaMetrics self-monitoring

### Log Aggregation (VictoriaLogs)

- ✅ Promtail auto-discovers Docker containers
- ✅ JSON log parsing enabled
- ✅ Labels extracted: service, container, project
- ✅ 30-day retention

### Tracing (VictoriaTraces)

- ✅ OpenTelemetry HTTP endpoint (4318)
- ✅ Jaeger HTTP endpoint (14268)
- ✅ Zipkin endpoint (9411)
- ✅ Traces stored in VictoriaMetrics

### Visualization (Grafana)

- ✅ 3 datasources auto-provisioned
- ✅ 5 dashboards auto-provisioned
- ✅ Trace-to-logs correlation
- ✅ Metrics-to-logs correlation

## 🧪 Testing

### Generate Some Traffic

```bash
# Health check requests
for i in {1..100}; do curl http://api.nexus.local/health; done

# OAuth endpoint (will hit rate limit)
for i in {1..10}; do curl http://api.nexus.local/api/auth/google; done
```

### View Metrics

```bash
# Check Traefik request count
curl -s 'http://metrics.nexus.local/api/v1/query?query=traefik_entrypoint_requests_total' | jq

# Check Redis connections
curl -s 'http://metrics.nexus.local/api/v1/query?query=redis_connected_clients' | jq

# Check PostgreSQL connections
curl -s 'http://metrics.nexus.local/api/v1/query?query=pg_stat_database_numbackends' | jq
```

### View Logs

```bash
# Query VictoriaLogs via API
curl -s 'http://logs.nexus.local/select/logsql/query' \
  -d 'query={service="gateway"}' | jq

# View in Grafana Explore
# 1. Open http://grafana.nexus.local
# 2. Go to Explore
# 3. Select VictoriaLogs datasource
# 4. Query: {service="gateway"} |= "error"
```

## 🎯 Next Steps

### Phase 4.1: Structured Logging (1.5 days)

1. Add zerolog to Go services
2. Configure JSON output
3. Add request ID to context
4. Update Promtail pipeline for structured logs

### Phase 4.2: Application Metrics (1.5 days)

1. Add Prometheus client to Gateway/API/Worker
2. Instrument HTTP handlers
3. Add business metrics (login rate, workflow executions)
4. Update dashboards with new metrics

See `backend/ROADMAP.md` Phase 4 for full details.

## 📚 Documentation

- Full guide: `observability/README.md`
- DNS setup: `observability/LOCAL_DNS_SETUP.md`
- Implementation plan: `backend/ROADMAP.md` (Phase 4)

## 🚨 Troubleshooting

### Metrics not showing

```bash
# Check exporter logs
docker-compose logs postgres-exporter
docker-compose logs redis-exporter

# Check VictoriaMetrics targets
open http://metrics.nexus.local/targets
```

### Logs not appearing

```bash
# Check Promtail logs
docker-compose logs promtail

# Verify VictoriaLogs is receiving data
curl http://logs.nexus.local/metrics | grep victorialogs_rows_ingested_total
```

### Grafana dashboards empty

```bash
# Test datasource connection
curl http://victoriametrics:8428/api/v1/query?query=up

# Restart Grafana
docker-compose restart grafana
```

## ✨ Success Criteria

You should see:

- ✅ All 5 dashboards loading in Grafana
- ✅ PostgreSQL connection count > 0
- ✅ Redis commands rate > 0
- ✅ Traefik requests flowing through
- ✅ Docker container logs in VictoriaLogs
- ✅ No errors in `docker-compose logs`

## 🎉 Congratulations

Your Nexus observability stack is now fully operational with:

- 📊 Metrics from all infrastructure components
- 📝 Centralized log aggregation
- 🔍 Distributed tracing ready
- 📈 5 production-ready dashboards
- 🔗 Trace-to-logs correlation

Total resource usage: **~310MB RAM** (vs 2GB for Prometheus/Loki/Jaeger stack)
