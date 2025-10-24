# Nexus Observability Stack - Local DNS Setup

## Add to /etc/hosts

Add these entries to your `/etc/hosts` file:

```text
127.0.0.1 metrics.nexus.local
127.0.0.1 logs.nexus.local
127.0.0.1 traces.nexus.local
127.0.0.1 grafana.nexus.local
```

## Quick Command

Run this command to add all entries at once:

```bash
sudo bash -c 'cat >> /etc/hosts << EOF

# Nexus Observability Stack
127.0.0.1 metrics.nexus.local
127.0.0.1 logs.nexus.local
127.0.0.1 traces.nexus.local
127.0.0.1 grafana.nexus.local
EOF'
```

## Access Points

After starting the stack with `docker-compose -f docker-compose.dev.yaml up`:

- **Grafana Dashboard**: <http://grafana.nexus.local> (admin/admin)
  - Nexus Overview Dashboard
  - Nexus PostgreSQL Dashboard
  - Nexus Redis Dashboard
  - Nexus Traefik Dashboard
  - Nexus Temporal Dashboard
- **VictoriaMetrics UI**: <http://metrics.nexus.local>
- **VictoriaLogs UI**: <http://logs.nexus.local>
- **VictoriaTraces**: <http://traces.nexus.local>

## Direct Port Access

You can also access services directly without DNS:

- **Grafana**: <http://localhost:3001>
- **VictoriaMetrics**: <http://localhost:8428>
- **VictoriaLogs**: <http://localhost:9428>
- **OpenTelemetry HTTP**: <http://localhost:4318>
- **Jaeger HTTP**: <http://localhost:14268>
- **Zipkin**: <http://localhost:9411>
