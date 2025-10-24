# Ory Kratos SSO Configuration

## Overview

Ory Kratos provides unified Single Sign-On (SSO) for all Nexus services using Google and Apple as OAuth2 identity providers.

## Architecture

```text
┌──────────────────────────────────────────────────────────────┐
│                    User's Browser                             │
│  Cookie: ory_session_... (.nexus.local domain)                │
└────────────┬─────────────────────────────────────────────────┘
             │
             │ First Login
             ▼
┌────────────────────────────────────────────────────────────┐
│                 auth.nexus.local                            │
│           (Kratos Self-Service UI)                          │
│  - Login page with "Sign in with Google/Apple" buttons    │
│  - Registration, Settings, Verification flows              │
└────────────┬───────────────────────────────────────────────┘
             │
             ▼
┌────────────────────────────────────────────────────────────┐
│                   Ory Kratos                                │
│  - Identity Management (PostgreSQL storage)                │
│  - Session Management (Cookie-based)                       │
│  - OAuth2 Integration (Google, Apple)                      │
│  - 2FA/TOTP Support                                        │
└────────────┬───────────────────────────────────────────────┘
             │
             │ Session Cookie established
             │
             ▼
┌────────────────────────────────────────────────────────────┐
│               Subsequent Requests                           │
│                                                             │
│  User → nexus.local ──────┐                                │
│  User → grafana.nexus.local ──┐                            │
│  User → metrics.nexus.local ──┼─→ Traefik                  │
│                               │                             │
│                               ▼                             │
│                   kratos-forward-auth middleware            │
│                   (checks ory_session cookie)               │
│                               │                             │
│                               ├─→ Kratos /sessions/whoami   │
│                               │   (validates session)        │
│                               │                             │
│                               ▼                             │
│                       ✅ Authenticated                       │
│                       Sets headers:                         │
│                       X-User: user@email.com                │
│                       X-User-Id: uuid                       │
│                               │                             │
│                               ▼                             │
│                     Protected Service                       │
│                     (Nexus UI, Grafana, etc.)              │
└─────────────────────────────────────────────────────────────┘
```

## Components

### 1. Kratos (Identity Server)

- **Public API** (port 4433): Self-service flows, session management
- **Admin API** (port 4434): Internal identity management
- **Storage**: PostgreSQL (reuses existing `nexus_db`)
- **Session Storage**: Cookie-based (stored in PostgreSQL)

### 2. Kratos Self-Service UI

- **URL**: `http://auth.nexus.local`
- **Flows**: Login, Registration, Settings, Verification, 2FA setup
- **OAuth Buttons**: "Sign in with Google", "Sign in with Apple"

### 3. Kratos Forward Auth Middleware

- **Function**: Validates Kratos sessions for Traefik
- **Endpoint**: Checks `/sessions/whoami` on Kratos API
- **Headers**: Passes user identity to backend services

## Configuration Files

### kratos.yml

Main Kratos configuration:
- Database connection (PostgreSQL)
- OAuth2 providers (Google, Apple)
- Self-service flow URLs
- Session configuration
- Security settings

### identity.schema.json

Defines user identity structure:
```json
{
  "traits": {
    "email": "user@example.com",
    "name": {
      "first": "John",
      "last": "Doe"
    },
    "picture": "https://..."
  },
  "metadata_public": {
    "provider": "google",
    "provider_user_id": "..."
  }
}
```

### oidc.google.jsonnet / oidc.apple.jsonnet

Maps OAuth claims to Kratos identity traits.

### webhook.jsonnet

Called after registration to sync user to Gateway's PostgreSQL database.

## User Flow

### First-Time Login (New User)

1. User visits `http://nexus.local`
2. No session cookie → Traefik redirects to `http://auth.nexus.local/login`
3. User clicks "Sign in with Google"
4. Google OAuth flow → callback to Kratos
5. Kratos creates identity in PostgreSQL
6. Kratos calls webhook → Gateway creates user in `users` table
7. Kratos sets `ory_session_...` cookie (domain: `.nexus.local`)
8. Redirect back to `http://nexus.local`
9. ✅ User is authenticated!

### Subsequent Access (Existing Session)

1. User visits `http://grafana.nexus.local`
2. Traefik intercepts request
3. `kratos-forward-auth` middleware checks `ory_session_...` cookie
4. Kratos validates session via `/sessions/whoami`
5. Middleware sets headers: `X-User`, `X-User-Id`
6. Traefik forwards request to Grafana with user headers
7. ✅ User sees Grafana without re-authentication!

### SSO Across All Services

Same cookie works for:
- `nexus.local` (Nexus UI)
- `grafana.nexus.local` (Grafana)
- `metrics.nexus.local` (VictoriaMetrics)
- `logs.nexus.local` (VictoriaLogs)
- `traefik.nexus.local` (Traefik Dashboard)
- `temporal.nexus.local` (Temporal UI)

**Single Sign-On achieved!** 🎉

## OAuth Provider Setup

### Google OAuth

1. Go to [Google Cloud Console](https://console.cloud.google.com/apis/credentials)
2. Create OAuth 2.0 Client ID
3. Add Authorized Redirect URI:
   ```
   http://auth.nexus.local/self-service/methods/oidc/callback/google
   ```
4. Copy Client ID and Client Secret to `.env`

### Apple OAuth

1. Go to [Apple Developer Portal](https://developer.apple.com)
2. Create Services ID
3. Enable "Sign in with Apple"
4. Add Return URL:
   ```
   http://auth.nexus.local/self-service/methods/oidc/callback/apple
   ```
5. Generate private key, get Team ID and Key ID
6. Add to `.env`

## Environment Variables

Required in `.env`:

```bash
# Kratos Session & Encryption
SESSION_SECRET=your-64-char-random-string
STORAGE_ENCRYPTION_KEY=your-64-char-random-string

# Webhook Authentication
KRATOS_WEBHOOK_SECRET=your-random-webhook-secret

# Google OAuth
GOOGLE_CLIENT_ID=your-client-id.apps.googleusercontent.com
GOOGLE_CLIENT_SECRET=your-client-secret

# Apple OAuth
APPLE_CLIENT_ID=com.yourcompany.nexus
APPLE_CLIENT_SECRET=your-apple-client-secret
APPLE_TEAM_ID=YOUR_TEAM_ID
APPLE_KEY_ID=YOUR_KEY_ID
```

## Database Schema

Kratos automatically creates these tables in PostgreSQL:

```sql
-- identities: User accounts
-- identity_credentials: OAuth credentials
-- identity_credential_identifiers: Email/provider lookups
-- sessions: Active user sessions
-- session_devices: Device tracking
-- courier_messages: Verification emails (if enabled)
```

Your Gateway continues to use existing `users` table, synced via webhook.

## Webhook Integration

When a user registers via OAuth, Kratos calls Gateway's webhook to sync user data:

### Webhook Endpoint

```
POST http://gateway:8080/api/webhooks/kratos/registration
X-Webhook-Secret: <KRATOS_WEBHOOK_SECRET>

Payload:
{
  "identity_id": "uuid",
  "email": "user@example.com",
  "name": {"first": "John", "last": "Doe"},
  "picture": "https://...",
  "provider": "google",
  "provider_user_id": "..."
}
```

### Webhook Handler Behavior

1. **Validates webhook secret** - Rejects unauthorized requests
2. **Checks if user exists** by `kratos_identity_id`
3. **If exists**: Updates `name` and `picture` only
4. **If new**: Creates user with `role="none"` (pending admin approval)
5. **Returns response**:
   ```json
   {
     "status": "created",
     "user_id": "uuid",
     "message": "User created successfully with role=none (pending admin approval)"
   }
   ```

### User Approval Flow

New users have `role="none"` and cannot access services:

```bash
# Admin views pending users
docker-compose -f docker-compose.dev.yaml exec postgres \
  psql -U admin -d nexus_db -c \
  "SELECT id, email, name, role, created_at FROM users WHERE role='none';"

# Approve user (change role to member)
docker-compose -f docker-compose.dev.yaml exec postgres \
  psql -U admin -d nexus_db -c \
  "UPDATE users SET role='member' WHERE email='user@example.com';"
```

After approval, user can access Nexus services on next request.

## Testing

```bash
# Start services
docker-compose -f docker-compose.dev.yaml up

# Check Kratos health
curl http://auth.nexus.local/health/ready

# Open login page
open http://auth.nexus.local/login

# Test session validation
curl -H "Cookie: ory_session_..." http://auth.nexus.local/sessions/whoami
```

## Traefik Integration

Services protected by Kratos forward auth:

```yaml
labels:
  - "traefik.http.routers.myservice.middlewares=kratos-auth@docker"
```

The middleware automatically:
1. Checks for valid `ory_session_...` cookie
2. Redirects to login if no session
3. Validates session with Kratos
4. Sets `X-User` and `X-User-Id` headers
5. Allows request through

## Security Features

- ✅ OAuth2 only (no passwords to leak)
- ✅ Session cookies (HttpOnly, Secure, SameSite=Lax)
- ✅ TOTP 2FA support
- ✅ Session expiration (24h default)
- ✅ Device tracking
- ✅ CSRF protection built-in
- ✅ PostgreSQL audit trail

## Admin Operations

### View Active Sessions

```sql
SELECT
  i.id,
  i.traits->>'email' as email,
  s.id as session_id,
  s.issued_at,
  s.expires_at,
  sd.ip_address,
  sd.user_agent
FROM identities i
JOIN sessions s ON s.identity_id = i.id
JOIN session_devices sd ON sd.session_id = s.id
WHERE s.active = true;
```

### Revoke User Session

```bash
# Via Kratos Admin API
curl -X DELETE http://kratos:4434/admin/identities/<identity-id>/sessions/<session-id>
```

### List All Identities

```bash
curl http://kratos:4434/admin/identities | jq
```

## Migration from Current OAuth

The Gateway's existing OAuth code (`handlers/auth.go`) will be replaced with:
1. Webhook handler for Kratos registration
2. Middleware to read `X-User` and `X-User-Id` headers
3. User auto-creation/update logic

Old JWT-based auth → New session cookie-based auth (via Kratos)

## Troubleshooting

### "Session not found"

**Cause**: Cookie expired or cleared

**Solution**: Redirect user to login page

### "OIDC provider returned error"

**Cause**: Invalid OAuth credentials or redirect URI mismatch

**Solution**: Check `.env` and OAuth provider console settings

### Webhook not called

**Cause**: Gateway not reachable or webhook secret mismatch

**Solution**: Check `docker-compose logs gateway` and verify `KRATOS_WEBHOOK_SECRET`

## Next Steps

1. ✅ Kratos infrastructure deployed
2. ⏳ Implement Gateway webhook handler
3. ⏳ Implement Gateway header-based auth middleware
4. ⏳ Remove old OAuth code from Gateway
5. ⏳ Test SSO flow across all services
6. ⏳ Enable 2FA/TOTP
7. ⏳ Production deployment with real OAuth credentials
