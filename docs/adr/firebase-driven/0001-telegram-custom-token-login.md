# ADR 0001: Telegram login via Cloud Functions + Firebase custom token

- Status: Accepted
- Date: 2026-07-01
- Applies to: `firebase-driven/web` + `firebase-driven/functions`. The
  `server-driven/` and `frontend-driven/` modules carry their own copies of
  this decision for their Go backends; this module implements the same
  bot-OTP idea but on Firebase primitives.

## Context

Firebase Authentication ships built-in providers for Google, Microsoft,
Facebook, GitHub, phone (SMS), email link, and a handful of others — but it
has **no native Telegram provider**. To keep Telegram login as part of this
module (for parity with `server-driven/` and `frontend-driven/`), we need
custom server-side code, since Firebase's client SDK alone cannot talk to the
Telegram Bot API (the bot token must never reach the browser) or mint a
session the rest of the app recognizes.

Unlike the Go modules, this module has no long-running process to host a
`getUpdates` long-polling loop — Cloud Functions are request-driven, not
long-lived daemons. It does, however, get a stable, public HTTPS URL for
free on every deploy, which the Go modules' local-dev setups deliberately
avoided.

## Decision

Implement Telegram login as three **Cloud Functions** running the same
bot-delivered-OTP idea as the Go modules, but ending in a **Firebase custom
token** instead of an app-issued JWT:

1. `telegramStart` mints a single-use, random **login token**, writes a
   pending-login document to Firestore `telegram_logins/{login_token}`
   (10-minute TTL), and returns `{ bot_username, deep_link, login_token }`
   where `deep_link = https://t.me/<bot>?start=<login_token>`.
2. `web` shows the deep link; the user opens it and presses **Start**, which
   makes Telegram deliver `/start <login_token>` to the bot via a webhook —
   **`telegramWebhook/<WEBHOOK_SECRET>`**, an HTTPS Cloud Function registered
   with Telegram's `setWebhook` API. The secret path segment authenticates
   the request as genuinely coming from Telegram's servers.
3. `telegramWebhook` matches the login token to its pending-login doc,
   generates a 6-digit **OTP**, stores only its salted SHA-256 hash (5-minute
   TTL, max 5 attempts) back onto the same document, and `sendMessage`s the
   OTP into the user's Telegram chat.
4. The user enters the OTP in `web`, which calls **`telegramVerify({
   login_token, code })`**. The function constant-time-compares the OTP
   against the stored hash, and on success uses the **Admin SDK** to
   find-or-create a Firebase user with `uid = telegram:<tg_id>` and call
   `createCustomToken(uid)`.
5. `telegramVerify` returns `{ custom_token }`. `web` calls
   `signInWithCustomToken(custom_token)`, which hands the client a normal
   Firebase ID token + refresh token — from that point on Telegram users are
   indistinguishable from OAuth users to the rest of the app.

## Why this design

- **No native alternative exists**: Firebase Auth simply has no Telegram
  provider, so *some* custom code is unavoidable if Telegram login is to stay
  in scope for this module.
- **Webhook over long-polling**: a deployed Cloud Function has a real public
  HTTPS URL by construction, so the long-polling workaround the Go modules
  use to avoid exposing a URL locally isn't needed here — a webhook is
  simpler, doesn't hold a process open, and scales the same way the rest of
  the Functions do.
- **Firebase custom token, not an app JWT**: minting our own JWT would mean
  re-implementing session issuance, refresh, and verification that Firebase
  Auth already provides for every other provider in this module. A custom
  token is the SDK-blessed way to hand the client a Firebase-recognized
  identity from server-side code — it converts a one-off server-verified fact
  ("this Telegram user proved control of their chat") into a session using
  the exact same client-side primitives (`onAuthStateChanged`, `getIdToken`)
  as Google/Microsoft/Facebook/GitHub.
- **Two-channel binding** (same as the Go modules): the login token travels
  in the URL/deep link, but the OTP is only ever shown inside the user's
  Telegram chat. Completing login requires control of *both* the browser
  session (holds the login token) and the Telegram account (sees the OTP).

## Security properties

- Login token: random, single-use, short TTL (10 min). Consumed on successful
  verify.
- OTP: 6 digits, stored only as a salted SHA-256 hash, 5-minute TTL, max 5
  verify attempts, then invalidated. Compared in constant time.
- Never logs the OTP or tokens.
- `WEBHOOK_SECRET` is an unguessable path segment; Telegram is only told the
  full `telegramWebhook/<WEBHOOK_SECRET>` URL via `setWebhook`, so requests to
  the plain function path (without the secret) are rejected.
- `TELEGRAM_BOT_TOKEN` and `WEBHOOK_SECRET` live only in the Cloud Functions
  environment and are never sent to the browser.
- Firebase custom tokens are single-use and short-lived by design (minted
  just-in-time, immediately exchanged via `signInWithCustomToken`); the
  function does not hand out anything longer-lived than that.

## Consequences

- Telegram login only works once the functions are **deployed** (`firebase
  deploy --only functions`) and the webhook is **registered** with Telegram
  (`setWebhook` pointed at the deployed `telegramWebhook/<WEBHOOK_SECRET>`
  URL) — there is no local-dev-only path the way the Go modules'
  long-polling loop provides. This module targets a real Firebase project,
  not an emulator, so this is an accepted trade-off of the module's scope.
- Pending logins and OTP hashes live in **Firestore**, not in memory, so
  unlike the Go modules' in-memory store they survive a Cloud Functions cold
  start, redeploy, or concurrent invocations across multiple function
  instances — which is actually a robustness improvement over the Go
  modules' single-process, single-poller constraint, and comes for free from
  using a managed document store instead of a process-local map.
- Firestore security rules must deny client-side read/write access to
  `telegram_logins` — only the Admin SDK (running inside the functions)
  should ever touch that collection.
- The bot must have its webhook set before Telegram login works; if
  `setWebhook` was never called (or points at the wrong URL), `telegramStart`
  still succeeds but the user never receives an OTP. There is no equivalent
  of the Go modules' graceful "`getMe` fails → disable Telegram" fallback
  here, since there's no long-running process to run that check at startup.

## Alternatives considered

- **Firebase Phone Auth (SMS)** — rejected: it would replace Telegram
  entirely with a different second factor, and this module is explicitly
  meant to keep Telegram parity with `server-driven/` and `frontend-driven/`,
  not substitute a different provider for it.
- **Drop Telegram from this module** — rejected: it's the one flow that
  meaningfully exercises "what do you do when the identity provider has no
  Firebase-native support," which is worth keeping for comparison against the
  other two modules' Go implementations of the same idea.
