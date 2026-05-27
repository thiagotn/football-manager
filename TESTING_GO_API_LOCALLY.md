# Testing Go API v2 Locally with Docker Compose

This guide explains how to set up a complete local testing environment for the Go API (v2) with the frontend and database, all containerized using the `docker-compose.go-dev.yml` file.

## Quick Start

### 1. Start the Stack

```bash
docker compose -f docker-compose.go-dev.yml up --build
```

This will:
- Build and start PostgreSQL on port 5433
- Run migrations automatically from `football-api/migrations/`
- Build and start the Go API (port 8080) with Air hot-reload
- Build and start the SvelteKit frontend (port 3000) with correct API URL (`/api/v2`)
- All services connect on an isolated network (`go-app-net`)

### 2. Access the Application

- **Frontend**: http://localhost:3000
- **Go API**: http://localhost:8080/api/v2
- **Health Check**: `curl http://localhost:8080/api/v2/health`

## Complete Testing Flow

### Register and Login

1. Go to http://localhost:3000
2. Click "Sign Up" or similar auth flow
3. Enter WhatsApp: `+5511999990000` (any format works)
4. OTP code: `123456` (bypass code, no Twilio needed)
5. Set password and name
6. Complete registration

### Promote to Admin (Optional)

After registering, create an admin user:

```bash
docker exec -it football-go-postgres \
  psql -U football -d football_dev \
  -c "UPDATE players SET role='admin' WHERE whatsapp='+5511999990000';"
```

Then log out, log back in — you'll have admin access.

### Test API Endpoints

Use the frontend normally, or curl directly:

```bash
# Login and get token
TOKEN=$(curl -s -X POST http://localhost:8080/api/v2/auth/login \
  -H "Content-Type: application/json" \
  -d '{"whatsapp":"+5511999990000","password":"yourpassword"}' \
  | jq -r '.access_token')

# Use token to call authenticated endpoints
curl -X GET http://localhost:8080/api/v2/auth/me \
  -H "Authorization: Bearer $TOKEN"
```

## Stopping and Cleanup

### Stop the Stack (preserves volumes)

```bash
docker compose -f docker-compose.go-dev.yml down
```

### Remove Everything (including database)

```bash
docker compose -f docker-compose.go-dev.yml down -v
```

## Viewing the Database

To visually inspect the database using Adminer:

```bash
docker compose -f docker-compose.go-dev.yml --profile tools up
```

Then access **http://localhost:8081** with:
- Server: `postgres`
- Username: `football`
- Password: `football`
- Database: `football_dev`

## Hot-Reload Development

### Go API

The Go API container uses [Air](https://github.com/air-verse/air) for hot-reload. Any changes to Go files in `football-api-go/` will:
1. Automatically recompile (watch `.go` files)
2. Restart the server
3. Logs visible in `docker compose logs api-go`

```bash
# Watch logs in real-time
docker compose -f docker-compose.go-dev.yml logs -f api-go
```

### Frontend

The SvelteKit frontend is built once at container start. To develop the frontend with hot-reload:

**Option A: Develop outside Docker**

```bash
cd football-frontend
npm run dev
# Frontend on http://localhost:5173
# Make sure VITE_API_URL environment variable is set:
#   export VITE_API_URL=http://localhost:8080/api/v2
```

**Option B: Rebuild inside Docker** (if you modify SvelteKit config)

```bash
docker compose -f docker-compose.go-dev.yml up --build frontend
```

## Debugging

### Check Service Status

```bash
docker compose -f docker-compose.go-dev.yml ps
```

### View Logs

```bash
# All services
docker compose -f docker-compose.go-dev.yml logs

# Specific service
docker compose -f docker-compose.go-dev.yml logs api-go
docker compose -f docker-compose.go-dev.yml logs frontend
docker compose -f docker-compose.go-dev.yml logs postgres
```

### Enter Container Shell

```bash
# Go API
docker exec -it football-go-api sh

# Frontend
docker exec -it football-go-frontend sh

# Database
docker exec -it football-go-postgres psql -U football -d football_dev
```

## Common Issues

### Port 8080 / 5433 / 3000 Already in Use

```bash
# Find what's using the port (macOS/Linux)
lsof -i :8080
lsof -i :5433
lsof -i :3000

# Kill the process or modify docker-compose.go-dev.yml ports section
```

### Migrations Failed

If the `migrate` service exits with an error, check logs:

```bash
docker compose -f docker-compose.go-dev.yml logs migrate
```

The most common issue is if the `football-api/migrations/` directory doesn't exist or has no SQL files. Ensure you're running this from the monorepo root.

### Frontend Shows Wrong API URL

Verify the build args were passed correctly:

```bash
docker inspect football-go-frontend | grep -A10 "Config"
```

Look for `VITE_API_URL: http://localhost:8080/api/v2` in the env section.

If incorrect, rebuild:

```bash
docker compose -f docker-compose.go-dev.yml build --no-cache frontend
docker compose -f docker-compose.go-dev.yml up frontend
```

### Database Errors

If you get "database does not exist" errors, check if migrations ran:

```bash
docker exec -it football-go-postgres \
  psql -U football -d football_dev -c "\dt"
```

Should show tables like `players`, `groups`, `matches`, etc. If empty, manually run:

```bash
docker compose -f docker-compose.go-dev.yml restart migrate
```

## Comparison: Python vs Go API Stack

| Feature | Python Stack (`docker-compose.yml`) | Go Stack (`docker-compose.go-dev.yml`) |
|---------|------|-----|
| **DB Port** | 5432 | 5433 |
| **DB Name** | `football` | `football_dev` |
| **API Port** | 8000 | 8080 |
| **API Path** | `/api/v1` | `/api/v2` |
| **Frontend Port** | 3000 | 3000 |
| **Hot-reload API** | ❌ (poetry install) | ✅ (Air) |
| **Auto-migrations** | ✅ | ⚠️ (via migrate service) |
| **Conflict** | NO — separate ports | YES — same frontend port, plan for port 3001 alternation |

**Note**: Both stacks try to use port 3000 for the frontend. If you want to run both simultaneously, modify one of them to use 3001 or 5173 instead.

## Next Steps

- Implement new features in Go API and test via frontend
- Run integration tests: `DATABASE_URL="postgres://football:football@localhost:5433/football_dev?sslmode=disable" go test ./tests/integration/...`
- Compare API v1 (Python) vs v2 (Go) responses for parity testing
