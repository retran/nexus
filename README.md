# Nexus

**Private, self-hosted Integrated Operations Platform** — the central nervous
system for your household.

Nexus treats the family unit as a small-scale enterprise, unifying Business
Process Management (BPM) with Real-Time Asset Control through both digital
resource planning and physical smart home automation.

## Development Environment

### Prerequisites

Add the following entries to your `/etc/hosts` file:

```bash
127.0.0.1 nexus.local
127.0.0.1 api.nexus.local
127.0.0.1 graphql.nexus.local
127.0.0.1 auth.nexus.local
127.0.0.1 traefik.nexus.local
```

### Configuration

1. Copy the environment template:

```bash
cp .env.dev.example .env
```

1. Generate secrets for Ory Kratos (requires at least 32 characters):

```bash
echo "KRATOS_COOKIE_SECRET=$(openssl rand -hex 32)" >> .env
echo "KRATOS_CIPHER_SECRET=$(openssl rand -hex 32)" >> .env
echo "WEBHOOK_SECRET=$(openssl rand -hex 32)" >> .env
```

1. (Optional) Configure Google OAuth:
   - Create a project in
     [Google Cloud Console](https://console.cloud.google.com)
   - Enable the Google+ API
   - Create OAuth 2.0 credentials
   - Add authorized redirect URI:
     `http://auth.nexus.local/self-service/methods/oidc/callback/google`
   - Update `GOOGLE_CLIENT_ID` and `GOOGLE_CLIENT_SECRET` in `.env`

### Starting the Stack

```bash
# Start all services
docker-compose -f docker-compose.dev.yaml up -d

# Check status
docker-compose -f docker-compose.dev.yaml ps
```

### Service URLs

Once the stack is running, access the services at:

| Service               | URL                          | Description                                        |
| --------------------- | ---------------------------- | -------------------------------------------------- |
| **Frontend UI**       | <http://nexus.local>         | Main application interface                         |
| **REST API (BFF)**    | <http://api.nexus.local>     | Backend-for-Frontend gateway                       |
| **GraphQL API**       | <http://graphql.nexus.local> | GraphQL API endpoint                               |
| **Auth UI**           | <http://auth.nexus.local>    | Ory Kratos self-service UI                         |
| **Traefik Dashboard** | <http://traefik.nexus.local> | Reverse proxy dashboard                            |
| **Temporal UI**       | <http://localhost:8088>      | Workflow orchestration UI                          |
| **PostgreSQL**        | `localhost:5432`             | Database (user: `admin`, db: `nexus`, `kratos_db`) |
| **Redis**             | `localhost:6379`             | Cache (password in `.env`)                         |

### Internal Service Ports

These ports are exposed for debugging but not typically accessed directly:

- **Data API (GraphQL)**: `localhost:8081` - Direct database access API
- **Temporal (gRPC)**: `localhost:7233`
- **Delve Debugger**: `localhost:2345` (attached to data-api)

## License

Copyright 2025 Andrew Vasilyev Licensed under the Apache License, Version 2.0
