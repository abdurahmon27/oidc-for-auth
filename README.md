# Multi-Provider OAuth Auth Service

A self-contained authentication service with a Go backend and React frontend. Supports OAuth login via Google, Facebook, Microsoft, and GitHub, plus Telegram phone verification.

Two modules are provided:

1. **Server-Driven** (`server-driven/`) — OAuth redirects and token exchange happen server-side. Frontend uses httpOnly cookies.
2. **Frontend-Driven** (`frontend-driven/`) — React frontend drives OAuth via provider JS SDKs. Backend only validates tokens and issues JWTs. Access tokens are in-memory (Bearer auth).

Both modules run on the **same ports** — API on `8080`, web on `5173` — so you launch one stack at a time (see Quick Start).

```
.
├── server-driven/          # Part 1 — server-side OAuth, cookie sessions
│   ├── backend/            #   Go + Chi + PostgreSQL
│   └── frontend/           #   React + Vite (httpOnly cookies)
├── frontend-driven/        # Part 2 — SDK-driven OAuth, Bearer JWTs
│   ├── api/                #   Go — validates provider tokens, issues JWTs
│   └── web/                #   React + Vite (provider JS SDKs)
├── docs/                   # architecture notes + ADRs (per module)
└── docker-compose.yml      # profiles: server-driven | frontend-driven
```

## Architecture

### Server-Driven (Part 1)
- **Backend** (`server-driven/backend/`): Go + Chi router + PostgreSQL (via sqlc)
- **Frontend** (`server-driven/frontend/`): React + TypeScript + Vite + TanStack Query
- **Auth**: Server-side OAuth redirects, httpOnly cookie tokens
- **Tokens**: Short-lived JWT access tokens (15min) + rotated refresh tokens (7 days) with reuse detection

### Frontend-Driven (Part 2)
- **API** (`frontend-driven/api/`): Minimal Go backend — validates provider tokens, issues JWTs
- **Web** (`frontend-driven/web/`): React frontend with provider JS SDKs (Google GSI, MSAL.js, Facebook JS SDK)
- **Auth**: Frontend obtains tokens from provider SDKs, sends to backend for validation
- **Tokens**: Access JWT in-memory (Bearer header), refresh token in httpOnly cookie

## Quick Start

### Prerequisites

- Docker & Docker Compose
- Go 1.25+
- Node.js 22+

### Run with Docker Compose

The two stacks share the same ports, so they run via mutually exclusive Compose
profiles — pick one:

```bash
# Server-Driven (Part 1)
cp server-driven/.env.example server-driven/.env
# Edit server-driven/.env with your OAuth provider credentials
docker compose --profile server-driven up

# Frontend-Driven (Part 2)
cp frontend-driven/api/.env.example frontend-driven/api/.env
cp frontend-driven/web/.env.example frontend-driven/web/.env
docker compose --profile frontend-driven up
```

Either stack exposes:

- Web (React): http://localhost:5173
- API (Go): http://localhost:8080
- PostgreSQL: localhost:5432

### Local Development

**Server-Driven Backend:**
```bash
cd server-driven/backend
cp ../.env.example ../.env
# Start PostgreSQL (via docker-compose or locally)
go run ./cmd/server
```

**Server-Driven Frontend:**
```bash
cd server-driven/frontend
npm install
npm run dev
```

**Frontend-Driven API:**
```bash
cd frontend-driven/api
cp .env.example .env
go run ./cmd/server
```

**Frontend-Driven Web:**
```bash
cd frontend-driven/web
cp .env.example .env
npm install
npm run dev
```

## Provider Setup

### Google
1. Create credentials at https://console.cloud.google.com/apis/credentials
2. Server-driven redirect URI: `http://localhost:8080/auth/google/callback`
3. Frontend-driven: Add `http://localhost:5173` to authorized JavaScript origins
4. Set `GOOGLE_CLIENT_ID` and `GOOGLE_CLIENT_SECRET`

### Microsoft
1. Register app at https://portal.azure.com/#blade/Microsoft_AAD_RegisteredApps
2. Server-driven redirect URI: `http://localhost:8080/auth/microsoft/callback`
3. Frontend-driven: Add `http://localhost:5173` as redirect URI (SPA platform)
4. Set `MICROSOFT_CLIENT_ID`, `MICROSOFT_CLIENT_SECRET`, `MICROSOFT_TENANT`

### Facebook
1. Create app at https://developers.facebook.com
2. Server-driven redirect URI: `http://localhost:8080/auth/facebook/callback`
3. Frontend-driven: Add `localhost` to app domains
4. Set `FACEBOOK_CLIENT_ID` and `FACEBOOK_CLIENT_SECRET`

### GitHub
1. Create OAuth app at https://github.com/settings/developers
2. Server-driven callback URL: `http://localhost:8080/auth/github/callback`
3. Frontend-driven callback URL: `http://localhost:5173/auth/github/callback`
4. Set `GITHUB_CLIENT_ID` and `GITHUB_CLIENT_SECRET`

### Telegram
1. Get API token from https://core.telegram.org/gateway
2. Set `TELEGRAM_API_TOKEN`

## API Endpoints

### Server-Driven Backend (port 8080)

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/auth/{provider}/login` | No | Redirects to OAuth provider |
| GET | `/auth/{provider}/callback` | No | OAuth callback handler |
| POST | `/auth/telegram/send` | No | Send verification code |
| POST | `/auth/telegram/verify` | No | Verify code, issue tokens |
| POST | `/auth/refresh` | Cookie | Rotate refresh token |
| POST | `/auth/logout` | Cookie | Revoke tokens, clear cookies |
| GET | `/me` | Cookie | Get current user info |
| GET | `/health` | No | Health check |

### Frontend-Driven API (port 8080)

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/auth/token` | No | Validate provider token, issue JWT |
| POST | `/auth/refresh` | Cookie | Rotate refresh token, return new JWT in body |
| POST | `/auth/logout` | Cookie | Revoke tokens, clear cookie |
| POST | `/auth/telegram/send` | No | Send verification code |
| POST | `/auth/telegram/verify` | No | Verify code, return JWT in body |
| GET | `/me` | Bearer | Get current user info |
| GET | `/health` | No | Health check |

## Tests

```bash
cd server-driven/backend
go test ./tests/...
```
