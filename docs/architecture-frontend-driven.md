# Frontend-Driven OAuth — Architecture

**Module:** `frontend-driven/web/` (React) + `frontend-driven/api/` (Go)
**Ports:** API `8080`, Web `5173`

The **frontend drives OAuth** using provider JS SDKs. The backend does **not** redirect — it only **validates** whatever token/code the browser obtained and then **issues its own JWT**. The app's access token lives **in memory** (Bearer header); only the refresh token is a cookie.

---

## 1. Component Architecture

```mermaid
flowchart LR
    subgraph Web["Web — React (5173)"]
        LP[LoginPage]
        PR["providers/*<br/>google · microsoft<br/>facebook · github"]
        REG[registry.ts<br/>conditional by env]
        AUTHAPI["api/auth.ts<br/>in-memory access token"]
        HOOKS["hooks<br/>useAuth · useProviderLogin<br/>useGitHubCallback"]
    end

    subgraph API["API — Go (8080)"]
        MW["Middleware<br/>CORS · RateLimit · Bearer Auth"]
        AH["handler/authenticate.go<br/>POST /auth/token"]
        SH["handler/session.go<br/>refresh / logout"]
        VAL["validate/*<br/>per-provider validators"]
        TOK["token/jwt.go"]
    end

    DB[(PostgreSQL<br/>users · identities · refresh_tokens)]
    SDK{{Provider SDKs<br/>Google GSI · MSAL · FB JS}}
    GH{{GitHub OAuth<br/>redirect + code}}

    LP --> PR
    REG --> PR
    PR <--> SDK
    PR -.redirect.-> GH
    PR --> HOOKS --> AUTHAPI
    AUTHAPI -->|"Bearer / cookie"| MW
    MW --> AH & SH
    AH --> VAL
    VAL <-->|verify token / exchange code| SDK
    VAL <-->|code→token, /user| GH
    AH & SH --> TOK
    AH --> DB
    TOK --> DB
```

**Web providers register conditionally** (`registry.ts` + `config/providers.ts`): a provider only appears if its `VITE_*_CLIENT_ID` env var is set.

**Libraries:** `@azure/msal-browser` (Microsoft) · Google GSI · Facebook JS SDK · React Query. Backend uses `coreos/go-oidc/v3` for OIDC validation.

---

## 2. OAuth Flow — SDK providers (Google / Microsoft / Facebook)

The SDK returns a token in the browser; the browser POSTs it to the backend for validation.

```mermaid
sequenceDiagram
    autonumber
    participant U as Browser (React)
    participant S as Provider SDK
    participant B as API (8080)
    participant DB as PostgreSQL

    U->>S: init SDK + login (popup/prompt)
    S-->>U: ID token (Google/MS)<br/>or access token (Facebook)
    U->>B: POST /auth/token { provider, token }
    Note over B: validate/registry → provider validator
    alt Google / Microsoft
        B->>S: OIDC verify id_token signature + claims
    else Facebook
        B->>S: Debug-Token API + Graph /me
    end
    B->>DB: FindOrCreate user + identity (provider, provider_id)
    B->>DB: store refresh_token (SHA256, family UUID)
    B-->>U: 200 { access_token, expires_in: 900, user }<br/>Set-Cookie refresh_token (7d, HttpOnly, Strict)
    Note over U: access token kept in-memory only
    U->>B: GET /me (Authorization: Bearer <jwt>)
    B-->>U: user profile
```

| Provider | Browser obtains | Backend validates via |
|----------|-----------------|-----------------------|
| **Google** | ID token (GSI) | `go-oidc` verify |
| **Microsoft** | ID token (MSAL `loginPopup`, scopes `openid profile email`) | OIDC `login.microsoftonline.com/{tenant}/v2.0` |
| **Facebook** | access token (FB.login) | Debug-Token API + Graph `/me` |

---

## 3. OAuth Flow — GitHub (Authorization Code)

GitHub has no pure-JS SDK, so it uses a real redirect + **server-side code exchange** (the client secret never reaches the browser).

```mermaid
sequenceDiagram
    autonumber
    participant U as Browser (React)
    participant GH as GitHub
    participant B as API (8080)
    participant DB as PostgreSQL

    Note over U: store state UUID in sessionStorage
    U->>GH: redirect authorize?client_id&redirect_uri&scope=user:email&state
    GH-->>U: redirect /auth/github/callback?code&state
    Note over U: useGitHubCallback — validate state == sessionStorage
    U->>B: POST /auth/token { provider: github, token: code }
    B->>GH: POST /login/oauth/access_token (client_id + secret + code)
    GH-->>B: access token
    B->>GH: GET /user (+ /user/emails for primary verified)
    B->>DB: FindOrCreate user + identity
    B->>DB: store refresh_token
    B-->>U: 200 { access_token, expires_in, user } + refresh cookie
```

---

## 4. Telegram Phone Verification

```mermaid
sequenceDiagram
    autonumber
    participant U as Browser
    participant B as API (8080)
    participant DB as PostgreSQL

    U->>B: POST /auth/telegram/send { phone_number }
    B-->>U: code sent (SMS via gateway)
    U->>B: POST /auth/telegram/verify { phone_number, code }
    B->>DB: FindOrCreateByPhone (provider=telegram)
    B-->>U: 200 { access_token (JWT in body), user } + refresh cookie
```

Unlike the SDK providers, the JWT is returned **in the response body** (same as `/auth/token`), not via redirect.

---

## 5. Session: Refresh & Logout

```mermaid
sequenceDiagram
    autonumber
    participant U as Browser
    participant B as API (8080)
    participant DB as PostgreSQL

    rect rgb(238,246,255)
    Note over U,DB: Silent refresh — timer fires 60s before expiry
    U->>B: POST /auth/refresh (refresh_token cookie)
    B->>DB: lookup SHA256(token)
    alt already revoked → reuse
        B->>DB: revoke entire family
        B-->>U: 401 (force re-login)
    else valid
        B->>DB: revoke old, issue new pair (same family)
        B-->>U: 200 { access_token } (body) + new refresh cookie
    end
    end

    rect rgb(255,244,244)
    U->>B: POST /auth/logout (refresh cookie)
    B->>DB: revoke entire family
    B-->>U: clear refresh cookie · frontend drops in-memory token
    end
```

`useAuth.ts`: schedules a refresh 60s before `expires_in`, and attempts a silent refresh on mount so a reload restores the session from the refresh cookie.

---

## 6. Tokens & Storage — the key contrast

| Token | Form | Expiry | Where it lives |
|-------|------|--------|----------------|
| **Access** | JWT HS256 (`sub`, `email`, `iat`, `exp`) | 15 min (`expires_in: 900`) | **In-memory JS variable** — sent as `Authorization: Bearer` |
| **Refresh** | 32-byte random | 7 days | `refresh_token` **httpOnly cookie**, `SameSite=Strict`, `Path=/`; **SHA256-hashed** in DB with a `family` UUID |

> No access-token cookie and no CSRF cookie here (server-driven has both). The access token being in-memory means a page reload relies on the silent refresh to re-issue it.

---

## 7. Data Model

```mermaid
erDiagram
    users ||--o{ identities : has
    users ||--o{ refresh_tokens : has

    users {
        uuid id PK
        text email
        text phone
        text name
        text avatar_url
    }
    identities {
        uuid id PK
        uuid user_id FK
        text provider
        text provider_id
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

---

## 8. Endpoints (API, port 8080)

| Method | Path | Auth | Purpose |
|--------|------|------|---------|
| POST | `/auth/token` | none | Validate provider token/code → issue JWT + refresh cookie |
| POST | `/auth/refresh` | refresh cookie | Rotate; return new JWT in body |
| POST | `/auth/logout` | refresh cookie | Revoke family, clear cookie |
| POST | `/auth/telegram/send` | none | Send SMS code |
| POST | `/auth/telegram/verify` | none | Verify → JWT in body |
| GET | `/me` | Bearer | Current user |
| GET | `/health` | none | Health check |

`middleware/auth.go` verifies the `Authorization: Bearer <jwt>` (HS256 signature + expiry) and injects `UserID`/`UserEmail` into context. `main.go` registers validators conditionally by config; CORS is scoped to the web origin; rate limit 10 req/s per IP.

---

## 9. Key Env Vars

**Web (`VITE_*`):** `GOOGLE_CLIENT_ID`, `MICROSOFT_CLIENT_ID`, `MICROSOFT_TENANT` (default `common`), `FACEBOOK_APP_ID`, `GITHUB_CLIENT_ID`, `GITHUB_REDIRECT_URI` (default `/auth/github/callback`).

**API:** `GOOGLE_CLIENT_ID/SECRET`, `MICROSOFT_CLIENT_ID/SECRET/TENANT`, `GITHUB_CLIENT_ID/SECRET`, `JWT_SECRET`, `COOKIE_DOMAIN`, `COOKIE_SECURE`, `FRONTEND_URL`.

---

## 10. Server-Driven vs Frontend-Driven — at a glance

| | Server-Driven | Frontend-Driven |
|--|---------------|-----------------|
| Who runs OAuth | Backend redirects | Browser SDKs |
| Provider token seen by | Backend only | Browser, then sent to backend |
| App access token | httpOnly cookie | **in-memory** Bearer |
| Refresh token | httpOnly cookie (Strict, `/`) | same |
| CSRF token | yes (double-submit) | not needed (Bearer, not cookie-auth) |
| GitHub | code exchange server-side | code exchange server-side (frontend redirects) |
| Ports | 8080 / 5173 | 8080 / 5173 |

Shared by both: HS256 JWT (15 min), 7-day rotated refresh tokens with **family-based reuse detection**, SHA256-hashed refresh storage, same `users`/`identities`/`refresh_tokens` schema, and OIDC signature verification for Google/Microsoft.
