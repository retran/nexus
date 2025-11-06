- [ ] secrets in vault

- [ ] observability stack
- [ ] tooling stack (review)

# Nexus TO-DO

## Production (Mac Mini Deployment)

### Infrastructure & Networking

- [ ] Configure Cloudflare Tunnel for external access (no open router ports)
- [ ] Set up Tailscale mesh network for admin access
- [ ] Enable Let's Encrypt ACME in Traefik (add LETSENCRYPT_EMAIL to .env)
- [ ] Configure DNS records for nexus.retran.me and subdomains

### Vault Setup

- [ ] Run `vault operator init` and save unseal keys securely
- [ ] Configure auto-unseal (optional, for production restart automation)
- [ ] Rotate VAULT_ROOT_TOKEN from dev default
- [ ] Set up Vault backup strategy

### Monitoring & Observability

- [ ] Deploy Prometheus + Grafana stack
- [ ] Configure Uptime Kuma for service monitoring
- [ ] Set up PostgreSQL backup automation to Backblaze B2
- [ ] Configure log aggregation (Loki or similar)

### Security Hardening

- [ ] Generate production secrets (KRATOS_COOKIE_SECRET,
      OATHKEEPER_SHARED_SECRET, WEBHOOK_SECRET)
- [ ] Configure Google OAuth with production credentials
- [ ] Review and update all environment variables in .env
- [ ] Set up firewall rules (only Traefik ports exposed)

### Documentation

- [ ] Write README.md with production deployment guide
- [ ] Create docker-compose.override.yaml.example for local development
- [ ] Document backup/restore procedures
- [ ] Create runbook for common operational tasks

### CI/CD

- [ ] Set up GitHub Actions for automated builds
- [ ] Configure self-hosted runner on Mac Mini
- [ ] Implement automated testing pipeline
- [ ] Set up deployment automation (Ansible or similar)

## Future Enhancements

### Features

- [ ] Home Assistant integration for physical asset control
- [ ] Temporal workflows for business processes
- [ ] External system sync (Google Calendar, Todoist, Notion)
- [ ] Mobile app for family member access

### Architecture

- [ ] Implement actual API endpoints (replace hello world)
- [ ] Expand database schema beyond minimal setup
- [ ] Add caching layer with Redis
