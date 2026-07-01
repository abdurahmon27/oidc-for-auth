# Server-Driven OAuth — Architecture

**Module:** `backend/` (Go) + `frontend/` (React)
**Ports:** Backend `8080`, Frontend `5173`

The **backend owns the entire OAuth flow**. The browser only clicks a link; all redirects, PKCE, and code-for-token exchange happen server-side. The frontend never sees provider tokens or the app's JWT — everything is carried in **httpOnly cookies**.

---

## 1. Component Architecture

```mermaid
flowchart LR
    subgraph Browser["Frontend — React (5173)"]
        LP[LoginPage]
        DP[DashboardPage]
        API["api/auth.ts<br/>(credentials: include)"]
        RQ[TanStack Query · useAuth]
    end

    subgraph Backend["Backend — Go + Chi (8080)"]
        MW["Middleware<br/>CORS · RateLimit · Auth · CSRF"]
        OH[oauth.go<br/>login / callback]
        SH[session.go<br/>refresh / logout]
        TH[telegram.go<br/>send / verify]
        ME[me.go]
        AUTH["internal/auth/*<br/>provider impls + PKCE"]
        TOK["internal/token/*<br/>jwt · refresh · cookies"]
        SVC[service · FindOrCreate]
    end

    DB[(PostgreSQL<br/>users · identities · refresh_tokens)]
    IDP{{OAuth Providers<br/>Google · MS · FB · GitHub}}
    TG{{Telegram Gateway}}

    LP -->|"a href /api/auth/{provider}/login"| OH
    API -->|fetch, cookies| MW
    MW --> OH & SH & TH & ME
    OH <--> AUTH
    AUTH <-->|"redirect + code exchange"| IDP
    OH & SH & TH --> TOK
    TH <--> TG
    OH & TH --> SVC
    SVC --> DB
    TOK --> DB
    RQ --> API
```

**Backend packages (`backend/internal/`):**

| Package | Responsibility |
|---------|----------------|
| `auth/` | Per-provider OAuth impls + `pkce.go` (state + PKCE verifier) |
| `config/` | Env var loading (`config.go`) |
| `database/` | sqlc-generated queries + models, migrations |
| `handler/` | HTTP handlers: `oauth`, `session`, `telegram`, `me` |
| `middleware/` | `auth` (cookie JWT), `csrf`, rate limiting |
| `service/` | `FindOrCreateByProvider`, `FindOrCreateByPhone` |
| `telegram/` | OTP gateway client |
| `token/` | JWT + refresh issuance + cookie helpers |

**Libraries:** Chi v5 (routing) · `golang.org/x/oauth2` (exchange + PKCE) · `coreos/go-oidc/v3` (ID token verify) · `golang-jwt/jwt/v5` (state signing) · pgxpool.

---

## 2. OAuth Login Flow (server-driven, PKCE)

Applies to Google, Microsoft, Facebook, GitHub. The browser is redirected the whole way; the app JWT lands as an httpOnly cookie.

```mermaid
sequenceDiagram
    autonumber
    participant U as Browser
    participant B as Backend (Chi)
    participant P as OAuth Provider
    participant DB as PostgreSQL

    U->>B: GET /auth/{provider}/login
    Note over B: pkce.go — generate JWT-signed<br/>state (5 min) + PKCE verifier (S256)
    B-->>U: 302 to provider authorize URL<br/>Set-Cookie oauth_state, oauth_verifier (5 min, HttpOnly)
    U->>P: Authorize + consent
    P-->>U: 302 /auth/{provider}/callback?code&state
    U->>B: GET callback (code, state + cookies)
    Note over B: Verify cookie state == query state<br/>Verify state JWT signature<br/>Read verifier from cookie, clear oauth_* cookies
    B->>P: Exchange code + PKCE verifier → tokens
    P-->>B: id_token / access_token
    Note over B: Google/MS: verify id_token (OIDC)<br/>FB: Graph /me · GitHub: /user (+/user/emails)
    B->>DB: FindOrCreateByProvider (identity + user)
    B->>DB: Store refresh_token (SHA256 hash, family UUID)
    B-->>U: 302 to FRONTEND_URL<br/>Set-Cookie access_token (15m, Lax),<br/>refresh_token (7d, Strict, /auth/refresh),<br/>csrf_token (7d, readable)
    U->>B: GET /me (access_token cookie)
    B-->>U: user profile + linked providers
```

**Per-provider specifics:**

| Provider | Verification | Notes |
|----------|-------------|-------|
| **Google** | OIDC `id_token` verify | scopes `openid profile email` |
| **Microsoft** | OIDC via `login.microsoftonline.com/{tenant}/v2.0` | `MICROSOFT_TENANT` (default `common`) |
| **Facebook** | Graph API `/me?fields=id,name,email,picture` | no id_token |
| **GitHub** | REST `/user`, falls back to `/user/emails` for primary verified email | no id_token |

---

## 3. Telegram Phone Verification

```mermaid
sequenceDiagram
    autonumber
    participant U as Browser
    participant B as Backend
    participant TG as Telegram Gateway
    participant DB as PostgreSQL

    U->>B: POST /auth/telegram/send { phone_number }
    Note over B: validate ^\+[1-9]\d{6,14}$<br/>max 3 sends / phone / 10 min
    B->>TG: sendVerificationMessage (6-digit OTP)
    TG-->>U: SMS code
    U->>B: POST /auth/telegram/verify { phone_number, code }
    Note over B: SHA256 compare · 5-min expiry · max 5 attempts
    B->>DB: FindOrCreateByPhone (provider=telegram)
    B-->>U: Set-Cookie access + refresh + csrf (same as OAuth)
```

---

## 4. Session: Refresh & Logout

```mermaid
sequenceDiagram
    autonumber
    participant U as Browser
    participant B as Backend
    participant DB as PostgreSQL

    rect rgb(238,246,255)
    Note over U,DB: Refresh (rotation + reuse detection)
    U->>B: POST /auth/refresh (refresh_token cookie)
    B->>DB: lookup by SHA256(token)
    alt token already revoked → reuse!
        B->>DB: revoke entire family
        B-->>U: 401
    else valid
        B->>DB: revoke current, issue new pair (same family)
        B-->>U: 200 + new access/refresh cookies
    end
    end

    rect rgb(255,244,244)
    Note over U,DB: Logout (CSRF-protected)
    U->>B: POST /auth/logout (X-CSRF-Token header)
    B->>DB: revoke entire family
    B-->>U: clear access_token, refresh_token, csrf_token
    end
```

The frontend (`useAuth.ts`) retries `/auth/refresh` once on an auth failure, then re-queries `/me`.

---

## 5. Tokens & Cookies

| Token | Alg / Form | Expiry | Delivery | Storage |
|-------|-----------|--------|----------|---------|
| **Access** | JWT HS256 (`sub`, `email`, `iat`, `exp`) | 15 min | `access_token` cookie | not stored |
| **Refresh** | 32-byte hex random | 7 days | `refresh_token` cookie | **SHA256 hash** in DB + `family` UUID |
| **CSRF** | 16-byte hex | 7 days | `csrf_token` cookie | not stored |
| **OAuth state** | JWT HS256 | 5 min | `oauth_state` cookie | — |
| **PKCE verifier** | 32-byte base64url | 5 min | `oauth_verifier` cookie | — |

**Cookie attributes:**

| Cookie | HttpOnly | SameSite | Path | Secure |
|--------|----------|----------|------|--------|
| `access_token` | ✅ | Lax | `/` | config |
| `refresh_token` | ✅ | **Strict** | **`/auth/refresh`** | config |
| `csrf_token` | ❌ (JS reads it) | Lax | `/` | config |
| `oauth_state` / `oauth_verifier` | ✅ | Lax | `/` | config |

---

## 6. Data Model

```mermaid
erDiagram
    users ||--o{ identities : has
    users ||--o{ refresh_tokens : has

    users {
        uuid id PK
        text email UK
        text phone UK
        text name
        text avatar_url
        timestamptz created_at
        timestamptz updated_at
    }
    identities {
        uuid id PK
        uuid user_id FK
        text provider
        text provider_id
        text email
        text name
        _ UNIQUE "provider, provider_id"
    }
    refresh_tokens {
        uuid id PK
        uuid user_id FK
        text token_hash UK
        uuid family
        timestamptz expires_at
        bool revoked
    }
```

`service.FindOrCreateByProvider`: return existing identity → else **auto-link by email** to an existing user → else create user + identity. Accessed via sqlc-generated queries (`queries/*.sql`).

---

## 7. Frontend Notes

- React 19 + Vite 8 + React Router v7 + TanStack Query v5.
- Vite dev proxy sends `/api/*` → `http://localhost:8080` (`vite.config.ts`).
- Every request uses `credentials: 'include'` so cookies flow.
- Login is just an `<a href="/api/auth/{provider}/login">` (`OAuthButton.tsx`) — no JS SDKs.
- `useAuth` keys off `['me']`, `staleTime` 5 min; logout sends the `X-CSRF-Token` header read from the `csrf_token` cookie.

---

## 8. Key Env Vars (Backend)

```
DATABASE_URL · SERVER_PORT (8080) · FRONTEND_URL (5173)
COOKIE_DOMAIN · COOKIE_SECURE · JWT_SECRET (required)
GOOGLE_CLIENT_ID/SECRET · MICROSOFT_CLIENT_ID/SECRET/TENANT
FACEBOOK_CLIENT_ID/SECRET · GITHUB_CLIENT_ID/SECRET · TELEGRAM_API_TOKEN
```

---

## 9. Security Summary

PKCE S256 on all providers · JWT-signed 5-min state · refresh rotation with **family-based reuse detection** (cascade revoke) · httpOnly access+refresh cookies · SameSite=Strict + path-scoped refresh cookie · CSRF double-submit for mutations · rate limits (10 req/s IP; 3 Telegram sends / phone / 10 min) · short expiries · OIDC signature verification for Google/MS.
