# ADR 0001: Telegram login via bot-delivered OTP (frontend-driven module)

- Status: Accepted
- Date: 2026-07-01
- Applies to: `frontend-driven/api` + `frontend-driven/web`. The root
  `task-01` project carries the same decision in its own copy of this ADR;
  both modules implement the identical flow.

## Context

The original Telegram login used Telegram's **Gateway API**
(`gatewayapi.telegram.org/sendVerificationMessage`) to send a verification
code to a user-supplied phone number. That endpoint requires a paid Gateway
API access token. The configured `TELEGRAM_API_TOKEN` was an ordinary
**BotFather bot token**, so every send failed with `ACCESS_TOKEN_INVALID`
(surfaced to the client as a misleading HTTP 429).

We want Telegram login to work locally with only a free bot token and no
Docker, no public webhook URL, and no paid service.

## Decision

Replace the phone + Gateway API flow with a **bot-delivered OTP** flow driven
by a standard bot token and the Telegram **Bot API**
(`api.telegram.org/bot<token>/...`):

1. On startup the API calls `getMe` to learn the bot's `@username` and starts
   a background **`getUpdates` long-polling** loop (works behind NAT, no
   public URL needed).
2. `POST /auth/telegram/start` mints a single-use, random **login token**,
   records a pending login, and returns `{ bot_username, deep_link,
   login_token }` where `deep_link = https://t.me/<bot>?start=<login_token>`.
3. `web` shows the bot username and the deep link. The user opens it and
   presses **Start**, which makes Telegram deliver `/start <login_token>` to
   the bot.
4. The poller matches the login token to the pending login, generates a
   6-digit **OTP**, stores its salted hash, and `sendMessage`s the OTP into
   that user's Telegram chat.
5. The user clicks **"Enter OTP"** in `web` (revealing the input), copies the
   code from Telegram, and submits it.
6. `POST /auth/telegram/verify` with `{ login_token, code }` checks the OTP,
   resolves the Telegram user to an account (`FindOrCreateByTelegram`), issues
   the normal session tokens, and consumes the login token.

Identity is now the **Telegram numeric user id** (`provider = "telegram"`,
`provider_id = <telegram user id>`), not a phone number. No DB migration is
required — the existing `identities(provider, provider_id)` table already
models this.

## Why this design

- **Free & local**: only a BotFather token is needed; long-polling avoids the
  public-URL requirement of webhooks and the paid Gateway API.
- **Two-channel binding**: the login token travels in the URL, but the OTP is
  only ever shown inside the user's Telegram chat. Completing login requires
  control of *both* the browser session (holds the login token) and the
  Telegram account (sees the OTP), which defeats an attacker who only has one.

## Security properties

- Login token: 32 bytes from `crypto/rand`, base64url, single-use, short TTL
  (10 min). Consumed on successful verify.
- OTP: 6 digits from `crypto/rand`, stored only as a SHA-256 hash, 5-minute
  TTL, max 5 verify attempts, then invalidated. Compared in constant time.
- Never logs the OTP or tokens.
- If `getMe` fails (missing/invalid token), Telegram login is disabled
  gracefully: `/auth/telegram/start` returns 503 and the rest of the service
  keeps running.
- Existing global rate-limiting middleware covers the new endpoints.

## Consequences

- Requires a valid bot token from BotFather; the bot must be able to receive
  `/start` (it can, by default).
- The API process must run the long-polling goroutine; only one poller may
  consume `getUpdates` for a given bot at a time.
- Pending logins and OTPs are held in memory, so they do not survive a restart
  and do not work across multiple API replicas without a shared store. This is
  acceptable for the current single-instance local/dev deployment; a shared
  store (e.g. Redis/Postgres) would be the follow-up for horizontal scaling.
- The `web` app's `vite.config.ts` proxies `/api` to the API, so the browser
  talks to the bot flow through the same origin.
- The old `/auth/telegram/send` endpoint and phone-number input are removed.

## Alternatives considered

- **Fix the Gateway API token** — rejected: paid service, still needs a real
  Gateway account and phone delivery.
- **Telegram Login Widget / webhook** — rejected for local dev: needs a public
  HTTPS callback URL and domain configuration.
