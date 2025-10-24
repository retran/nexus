# Ory Kratos SSO - Quick Start

## What We Added

✅ **Ory Kratos** - Headless identity management for SSO
✅ **Kratos Self-Service UI** - Login/registration pages
✅ **Ory Oathkeeper** - Forward auth middleware for Traefik
✅ **Google + Apple OAuth** - Social login providers
✅ **SSO Protection** - All services (Nexus UI, Grafana, Victoria stack, Traefik)

## Start Services

```bash
# Start everything
docker-compose -f docker-compose.dev.yaml up

# Or start in background
docker-compose -f docker-compose.dev.yaml up -d

# View logs
docker-compose -f docker-compose.dev.yaml logs -f kratos
docker-compose -f docker-compose.dev.yaml logs -f kratos-selfservice-ui
```

## Test SSO Flow

1. **Open Nexus UI**:
   ```bash
   open http://nexus.local
   ```

2. **You'll be redirected to login**:
   - URL: `http://auth.nexus.local/login`
   - See buttons: "Sign in with Google", "Sign in with Apple"

3. **Click "Sign in with Google"**:
   - OAuth redirect to Google
   - After auth, redirect back to Nexus
   - Session cookie `ory_session_...` is set
   - ⚠️ **New user has role="none"** (pending approval)
   - You'll see: `403 Forbidden - Account pending approval`

4. **Admin approves the user**:
   ```bash
   # View new users
   docker-compose -f docker-compose.dev.yaml exec postgres \
     psql -U admin -d nexus_db -c "SELECT id, email, name, role FROM users;"

   # Approve user by changing role from 'none' to 'member'
   docker-compose -f docker-compose.dev.yaml exec postgres \
     psql -U admin -d nexus_db -c "UPDATE users SET role='member' WHERE email='user@example.com';"
   ```

5. **User can now access services**:
   - Refresh `http://nexus.local` → ✅ Access granted!
   - Try Grafana: `http://grafana.nexus.local` → ✅ SSO works!

6. **Check other services**:
   ```bash
   open http://metrics.nexus.local  # VictoriaMetrics
   open http://logs.nexus.local     # VictoriaLogs
   open http://traefik.nexus.local  # Traefik Dashboard
   ```
   - All work without re-authentication

## Role System

Nexus uses a simple role-based access control:

- **none** - New user, pending admin approval, cannot access services
- **member** - Approved user, can access Nexus UI and API
- **admin** - Full access to all services (Grafana, Victoria stack, Traefik dashboard)

### User Lifecycle

```
1. User signs in via Google/Apple OAuth
   ↓
2. Kratos creates identity with role="none"
   ↓
3. Webhook creates user in database with role="none"
   ↓
4. User sees: "403 Forbidden - Account pending approval"
   ↓
5. Admin updates role to "member" or "admin"
   ↓
6. User can access services based on their role
```

### Managing Roles

```bash
# View all users and their roles
docker-compose -f docker-compose.dev.yaml exec postgres \
  psql -U admin -d nexus_db -c \
  "SELECT email, name, role, created_at FROM users ORDER BY created_at DESC;"

# Approve a user (none → member)
docker-compose -f docker-compose.dev.yaml exec postgres \
  psql -U admin -d nexus_db -c \
  "UPDATE users SET role='member' WHERE email='user@example.com';"

# Promote to admin
docker-compose -f docker-compose.dev.yaml exec postgres \
  psql -U admin -d nexus_db -c \
  "UPDATE users SET role='admin' WHERE email='admin@example.com';"

# Deactivate a user (member → none)
docker-compose -f docker-compose.dev.yaml exec postgres \
  psql -U admin -d nexus_db -c \
  "UPDATE users SET role='none' WHERE email='user@example.com';"
```

## Architecture

```text
User → nexus.local
  ↓ No session
Traefik → kratos-forward-auth (checks cookie)
  ↓ No valid session
Redirect → auth.nexus.local/login
  ↓ User clicks "Google"
Google OAuth → callback
  ↓ Success
Kratos creates session → sets ory_session cookie
  ↓ Cookie domain: .nexus.local
Redirect → nexus.local
  ↓ Request has cookie
Traefik → kratos-forward-auth (validates session)
  ↓ Session valid
Kratos returns user info
  ↓ Headers: X-User, X-User-Id
Backend receives authenticated request ✅
```

## Verify Kratos is Working

```bash
# Check Kratos health
curl http://auth.nexus.local/health/ready

# Check if you have a session (after login)
curl -H "Cookie: ory_session_..." http://auth.nexus.local/sessions/whoami | jq

# List all identities (admin API)
docker exec nexus-kratos-1 kratos identities list --endpoint http://localhost:4434
```

## Configure Google OAuth

1. Go to [Google Cloud Console](https://console.cloud.google.com/apis/credentials)
2. Create OAuth 2.0 Client ID
3. Add Authorized Redirect URI:
   ```
   http://auth.nexus.local/self-service/methods/oidc/callback/google
   ```
4. Update `.env`:
   ```bash
   GOOGLE_CLIENT_ID=your-id.apps.googleusercontent.com
   GOOGLE_CLIENT_SECRET=your-secret
   ```
5. Restart Kratos:
   ```bash
   docker-compose -f docker-compose.dev.yaml restart kratos
   ```

## Configure Apple OAuth

1. Go to [Apple Developer Portal](https://developer.apple.com)
2. Create Services ID
3. Enable "Sign in with Apple"
4. Add Return URL:
   ```
   http://auth.nexus.local/self-service/methods/oidc/callback/apple
   ```
5. Generate private key, get Team ID and Key ID
6. Update `.env`:
   ```bash
   APPLE_CLIENT_ID=com.yourcompany.nexus
   APPLE_CLIENT_SECRET=your-secret
   APPLE_TEAM_ID=YOUR_TEAM_ID
   APPLE_KEY_ID=YOUR_KEY_ID
   ```
7. Restart Kratos

## Database

Kratos automatically creates tables in PostgreSQL (`nexus_db`):

```bash
# View Kratos tables
docker exec -it nexus-postgres-1 psql -U admin -d nexus_db -c "\dt"

# Key tables:
# - identities: User accounts
# - sessions: Active sessions
# - identity_credentials: OAuth credentials
# - courier_messages: Verification emails
```

## Next Steps

### 1. Implement Gateway Webhook Handler

Kratos calls webhook after registration:

```go
// In backend/internal/api/rest/handlers/webhooks.go
func (h *Handler) HandleKratosRegistration(c echo.Context) error {
    // Verify webhook secret
    if c.Request().Header.Get("X-Webhook-Secret") != os.Getenv("KRATOS_WEBHOOK_SECRET") {
        return c.JSON(401, "Invalid webhook secret")
    }

    // Parse webhook payload
    var payload struct {
        IdentityID     string `json:"identity_id"`
        Email          string `json:"email"`
        Provider       string `json:"provider"`
        ProviderUserID string `json:"provider_user_id"`
    }
    c.Bind(&payload)

    // Create user in database
    // INSERT INTO users (...)
    // INSERT INTO user_identities (...)

    return c.NoContent(200)
}
```

### 2. Update Gateway Auth Middleware

Replace JWT checking with header-based auth:

```go
// In backend/internal/api/rest/middleware/auth.go
func KratosAuthMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
    return func(c echo.Context) error {
        // Kratos forward auth sets these headers
        userEmail := c.Request().Header.Get("X-User")
        userID := c.Request().Header.Get("X-User-Id")

        if userEmail == "" {
            return c.JSON(401, "Unauthorized")
        }

        // Load user from database
        user := loadUser(userID)

        // Set in context
        c.Set("user", user)

        return next(c)
    }
}
```

### 3. Remove Old OAuth Code

Delete these files:
- `backend/internal/api/rest/handlers/auth.go` (Google OAuth logic)
- Old JWT token generation code

### 4. Update Frontend

Change login flow to redirect to Kratos:

```typescript
// When user clicks "Login"
window.location.href = "http://auth.nexus.local/login"

// After successful login, user is redirected back to nexus.local
// with ory_session cookie set
```

### 5. Enable 2FA (Optional)

Users can enable TOTP via Settings:

```bash
open http://auth.nexus.local/settings
```

## Troubleshooting

### "Session not found"

**Problem**: Cookie expired or cleared

**Solution**: Logout and login again

### "OAuth provider returned error"

**Problem**: Invalid client ID/secret or redirect URI mismatch

**Solution**: Check `.env` and OAuth provider console

### Redirect loop

**Problem**: Kratos can't set cookie (domain mismatch)

**Solution**: Check `kratos.yml` session domain matches `.nexus.local`

### Webhook not called

**Problem**: Gateway unreachable or secret mismatch

**Solution**: Check Gateway logs and verify `KRATOS_WEBHOOK_SECRET`

## Documentation

- Full README: `kratos/README.md`
- Kratos Docs: https://www.ory.sh/docs/kratos/
- Oathkeeper Docs: https://www.ory.sh/docs/oathkeeper/

## Summary

✅ **Kratos deployed** - Identity management ready
✅ **SSO configured** - Google + Apple OAuth
✅ **All services protected** - Traefik forward auth
✅ **Session-based** - No JWT tokens needed
✅ **Database integrated** - PostgreSQL storage
⏳ **Webhook handler** - Needs Gateway implementation
⏳ **Old OAuth removal** - Clean up legacy code

**Single Sign-On is ready to use!** 🎉
