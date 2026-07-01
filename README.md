# Multi-Provider OAuth Auth Service

A self-contained authentication service with a Go backend and React frontend. Supports OAuth login via Google, Facebook, Microsoft, and GitHub, plus Telegram phone verification.

Three modules are provided:

1. **Server-Driven** (`server-driven/`) — OAuth redirects and token exchange happen server-side. Frontend uses httpOnly cookies.
2. **Frontend-Driven** (`frontend-driven/`) — React frontend drives OAuth via provider JS SDKs. Backend only validates tokens and issues JWTs. Access tokens are in-memory (Bearer auth).
3. **Firebase-Driven** (`firebase-driven/`) — Firebase-native OAuth. Fully frontend: the React app calls Firebase Auth directly (no custom backend, no Postgres); the only server-side code is a handful of Cloud Functions implementing Telegram login.

`server-driven/` and `frontend-driven/` run on the **same ports** — API on `8080`, web on `5173` — so you launch one stack at a time (see Quick Start). `firebase-driven/` only needs the web port (`5173`); Firebase itself is the backend.

```
.
├── server-driven/          # Part 1 — server-side OAuth, cookie sessions
│   ├── backend/            #   Go + Chi + PostgreSQL
│   └── frontend/           #   React + Vite (httpOnly cookies)
├── frontend-driven/        # Part 2 — SDK-driven OAuth, Bearer JWTs
│   ├── api/                #   Go — validates provider tokens, issues JWTs
│   └── web/                #   React + Vite (provider JS SDKs)
├── firebase-driven/        # Part 3 — Firebase-native OAuth, no custom backend
│   ├── web/                #   React + Vite (Firebase Auth SDK, no Postgres)
│   └── functions/          #   Cloud Functions — Telegram bot OTP → Firebase custom token
├── docs/                   # architecture notes + ADRs (per module)
└── docker-compose.yml      # profiles: server-driven | frontend-driven | firebase-driven
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

### Firebase-Driven (Part 3)
- **Web** (`firebase-driven/web/`): React frontend calling Firebase Auth directly (`signInWithPopup`) — no custom backend for OAuth
- **Functions** (`firebase-driven/functions/`): Cloud Functions implementing Telegram login only (`telegramStart`, `telegramWebhook`, `telegramVerify`), since Firebase has no native Telegram provider
- **Auth**: Firebase Auth owns OAuth end-to-end (Google/Microsoft/Facebook/GitHub via popup); account linking via `linkWithCredential`
- **Tokens**: Firebase-managed ID token (~1h, auto-refreshed) + refresh token, stored by the SDK in IndexedDB — no app JWT, no `/refresh` or `/logout` endpoints, no Postgres
- **Data**: Firebase Auth users + Firestore `users/{uid}` profile mirror + Firestore `telegram_logins` for the OTP handshake

See [`docs/architecture-firebase-driven.md`](docs/architecture-firebase-driven.md) for diagrams and full details.

## Quick Start

### Prerequisites

- Docker & Docker Compose
- Go 1.25+
- Node.js 22+

### Run with Docker Compose

The `server-driven` and `frontend-driven` stacks share the same ports, so they (plus
`firebase-driven`) run via mutually exclusive Compose profiles — pick one:

```bash
# Server-Driven (Part 1)
cp server-driven/.env.example server-driven/.env
# Edit server-driven/.env with your OAuth provider credentials
docker compose --profile server-driven up

# Frontend-Driven (Part 2)
cp frontend-driven/api/.env.example frontend-driven/api/.env
cp frontend-driven/web/.env.example frontend-driven/web/.env
docker compose --profile frontend-driven up

# Firebase-Driven (Part 3) — web only; Firebase itself is the backend
cp firebase-driven/web/.env.example firebase-driven/web/.env
# Edit firebase-driven/web/.env with your Firebase project's web config
docker compose --profile firebase-driven up
```

`server-driven` and `frontend-driven` each expose:

- Web (React): http://localhost:5173
- API (Go): http://localhost:8080
- PostgreSQL: localhost:5432

`firebase-driven` only starts a `fb-web` container on http://localhost:5173 — there's no
Postgres and no local API container, because Firebase is the backend. Cloud Functions are
**not** run via Compose; they're deployed straight to your Firebase project (see below).

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

**Firebase-Driven Web:**
```bash
cp firebase-driven/web/.env.example firebase-driven/web/.env
# Fill in VITE_FIREBASE_* with your Firebase project's web config
cd firebase-driven/web
npm install
npm run dev
```
Runs on http://localhost:5173. There is no local API to start — the app talks to Firebase
directly.

**Firebase-Driven Functions (Telegram login):**
```bash
cd firebase-driven/functions
# Set TELEGRAM_BOT_TOKEN and WEBHOOK_SECRET (e.g. via `firebase functions:config:set`
# or a `.env` per the Firebase CLI's supported config method)
firebase deploy --only functions,firestore:rules
```
Then register the webhook so Telegram knows where to deliver `/start <login_token>`:
```bash
curl "https://api.telegram.org/bot<TELEGRAM_BOT_TOKEN>/setWebhook" \
  -d "url=https://<region>-<project-id>.cloudfunctions.net/telegramWebhook/<WEBHOOK_SECRET>"
```
This module targets a real Firebase project — there is no local emulator setup here.

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

### Firebase (for `firebase-driven/`)
1. Create a project at https://console.firebase.google.com
2. In **Authentication → Sign-in method**, enable **Google**, **Microsoft**, **Facebook**,
   and **GitHub** as sign-in providers.
3. Each provider's client id/secret is configured **inside the Firebase console**, on that
   provider's row in Sign-in method (not in app env vars):
   - **Google**: enabled with no extra config needed (Firebase provisions its own OAuth
     client), or bring your own from https://console.cloud.google.com/apis/credentials.
   - **Microsoft**: application (client) ID + client secret from
     https://portal.azure.com/#blade/Microsoft_AAD_RegisteredApps, entered on the
     Microsoft provider row. Set `VITE_MICROSOFT_TENANT` in the web app to match.
   - **Facebook**: App ID + App Secret from https://developers.facebook.com, entered on
     the Facebook provider row.
   - **GitHub**: Client ID + Client Secret from https://github.com/settings/developers,
     entered on the GitHub provider row. OAuth callback URL is Firebase's own
     `authDomain` handler — copy it from the console, no app-side callback route needed.
4. In **Authentication → Settings → Authorized domains**, add `localhost` (needed for
   `signInWithPopup` during local dev).
5. Copy the project's **Web app** config (`apiKey`, `authDomain`, `projectId`,
   `storageBucket`, `messagingSenderId`, `appId`) into `firebase-driven/web/.env` as
   `VITE_FIREBASE_*`. These values are **public by design** — they identify the project to
   the client SDK, the same way a database hostname would; they are not secrets, so it's
   fine for them to end up in a shipped browser bundle. Real secrets (bot token, webhook
   secret, and each OAuth provider's client secret above) never go in web env — the OAuth
   secrets stay in the Firebase console, and `TELEGRAM_BOT_TOKEN` / `WEBHOOK_SECRET` stay
   in the Cloud Functions environment.
6. For Telegram, get a bot token from [@BotFather](https://t.me/BotFather) and set
   `TELEGRAM_BOT_TOKEN` and a random `WEBHOOK_SECRET` in the Functions environment — see
   [`docs/adr/firebase-driven/0001-telegram-custom-token-login.md`](docs/adr/firebase-driven/0001-telegram-custom-token-login.md).

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

### Firebase-Driven (Cloud Functions)

OAuth (Google/Microsoft/Facebook/GitHub) has no endpoints — the browser calls Firebase
Auth directly via the SDK. Only Telegram needs custom server code:

| Function | Auth | Description |
|----------|------|-------------|
| `telegramStart` | No | Mint login token, return `{ bot_username, deep_link, login_token }` |
| `telegramWebhook/<WEBHOOK_SECRET>` | Secret path segment | Telegram delivers `/start <login_token>`; generates + sends OTP |
| `telegramVerify` | No | Verify `{ login_token, code }`, return `{ custom_token }` |

## Tests

```bash
cd server-driven/backend
go test ./tests/...
```
