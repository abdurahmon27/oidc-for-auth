"use strict";
var __createBinding = (this && this.__createBinding) || (Object.create ? (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    var desc = Object.getOwnPropertyDescriptor(m, k);
    if (!desc || ("get" in desc ? !m.__esModule : desc.writable || desc.configurable)) {
      desc = { enumerable: true, get: function() { return m[k]; } };
    }
    Object.defineProperty(o, k2, desc);
}) : (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    o[k2] = m[k];
}));
var __setModuleDefault = (this && this.__setModuleDefault) || (Object.create ? (function(o, v) {
    Object.defineProperty(o, "default", { enumerable: true, value: v });
}) : function(o, v) {
    o["default"] = v;
});
var __importStar = (this && this.__importStar) || (function () {
    var ownKeys = function(o) {
        ownKeys = Object.getOwnPropertyNames || function (o) {
            var ar = [];
            for (var k in o) if (Object.prototype.hasOwnProperty.call(o, k)) ar[ar.length] = k;
            return ar;
        };
        return ownKeys(o);
    };
    return function (mod) {
        if (mod && mod.__esModule) return mod;
        var result = {};
        if (mod != null) for (var k = ownKeys(mod), i = 0; i < k.length; i++) if (k[i] !== "default") __createBinding(result, mod, k[i]);
        __setModuleDefault(result, mod);
        return result;
    };
})();
Object.defineProperty(exports, "__esModule", { value: true });
exports.telegramVerify = exports.telegramWebhook = exports.telegramStart = void 0;
// Cloud Functions for the firebase-driven module.
//
// Re-implements the Telegram bot-OTP login flow from
// docs/adr/frontend-driven/0001-telegram-bot-otp-login.md as Firebase
// 2nd-gen HTTPS functions, minting a Firebase custom token instead of a
// bespoke JWT. See that ADR for the full security rationale; the properties
// it lists (single-use login token w/ 10-min TTL, salted+hashed OTP w/
// 5-min TTL and <=5 attempts compared in constant time, webhook secret,
// never logging the OTP or tokens) are preserved here.
//
// Config: TELEGRAM_BOT_TOKEN and WEBHOOK_SECRET are read from
// process.env. Firebase Functions v2 automatically loads a `.env` file
// placed next to this source (functions/.env, see .env.example) for local
// emulation, and picks up the same names from `firebase functions:secrets:set`
// / `firebase functions:config` at deploy time. Plain process.env keeps this
// demo readable without pulling in firebase-functions/params.
const admin = __importStar(require("firebase-admin"));
const https_1 = require("firebase-functions/v2/https");
const firebase_functions_1 = require("firebase-functions");
const telegram_1 = require("./telegram");
const otp_1 = require("./otp");
admin.initializeApp();
const db = admin.firestore();
const LOGIN_TOKEN_TTL_MS = 10 * 60 * 1000;
const OTP_TTL_MS = 5 * 60 * 1000;
const MAX_OTP_ATTEMPTS = 5;
const TELEGRAM_BOT_TOKEN = process.env.TELEGRAM_BOT_TOKEN ?? "";
const WEBHOOK_SECRET = process.env.WEBHOOK_SECRET ?? "";
const LOGINS_COLLECTION = "telegram_logins";
const USERS_COLLECTION = "users";
/** Thrown for expected verify failures; caught to produce a 400 { error }. */
class VerifyError extends Error {
}
// --- CORS helper ---
// Demo simplicity: allow all origins. A production deployment should
// restrict this to the web app's origin.
function applyCors(req, res) {
    res.set("Access-Control-Allow-Origin", "*");
    res.set("Access-Control-Allow-Methods", "POST, OPTIONS");
    res.set("Access-Control-Allow-Headers", "Content-Type, Authorization");
    if (req.method === "OPTIONS") {
        res.status(204).send("");
        return true;
    }
    return false;
}
// --- telegramStart ---
exports.telegramStart = (0, https_1.onRequest)(async (req, res) => {
    if (applyCors(req, res))
        return;
    if (req.method !== "POST") {
        res.status(405).json({ error: "method not allowed" });
        return;
    }
    let botUsername;
    try {
        botUsername = await (0, telegram_1.getBotUsername)(TELEGRAM_BOT_TOKEN);
    }
    catch (err) {
        firebase_functions_1.logger.error("telegramStart: getMe failed", err);
        res.status(503).json({ error: "Telegram login unavailable" });
        return;
    }
    const loginToken = (0, otp_1.randomToken)();
    const now = admin.firestore.Timestamp.now();
    const expiresAt = admin.firestore.Timestamp.fromMillis(now.toMillis() + LOGIN_TOKEN_TTL_MS);
    const doc = {
        status: "pending",
        createdAt: now,
        expiresAt,
    };
    await db.collection(LOGINS_COLLECTION).doc(loginToken).set(doc);
    res.status(200).json({
        bot_username: botUsername,
        deep_link: `https://t.me/${botUsername}?start=${loginToken}`,
        login_token: loginToken,
    });
});
// --- telegramWebhook ---
// Registered with Telegram as {base}/telegramWebhook/<WEBHOOK_SECRET>. The
// trailing path segment authenticates the caller as Telegram (nobody else
// knows WEBHOOK_SECRET), standing in for the long-poll loop's implicit trust
// in the reference Go implementation.
exports.telegramWebhook = (0, https_1.onRequest)(async (req, res) => {
    const segment = req.path.replace(/^\/+/, "");
    if (!WEBHOOK_SECRET || segment !== WEBHOOK_SECRET) {
        res.status(403).json({ error: "forbidden" });
        return;
    }
    if (req.method !== "POST") {
        res.status(405).json({ error: "method not allowed" });
        return;
    }
    // Always ack Telegram with 200 once the secret checks out, even if the
    // update is malformed, unrelated, or the login token doesn't match
    // anything pending -- Telegram retries non-2xx responses.
    try {
        await handleWebhookUpdate(req.body);
    }
    catch (err) {
        firebase_functions_1.logger.error("telegramWebhook: failed to process update", err);
    }
    res.status(200).send("ok");
});
async function handleWebhookUpdate(update) {
    const message = update?.message;
    const text = message?.text?.trim();
    if (!message || !text)
        return;
    const match = /^\/start\s+(\S+)$/.exec(text);
    if (!match)
        return;
    const loginToken = match[1];
    const loginRef = db.collection(LOGINS_COLLECTION).doc(loginToken);
    const snap = await loginRef.get();
    if (!snap.exists)
        return;
    const data = snap.data();
    const now = admin.firestore.Timestamp.now();
    if (data.status !== "pending" || data.expiresAt.toMillis() < now.toMillis()) {
        return;
    }
    const otp = (0, otp_1.generateOtp)();
    const salt = (0, otp_1.randomSalt)();
    const update_ = {
        status: "otp_sent",
        otpSalt: salt,
        otpHash: (0, otp_1.hashOtp)(salt, otp),
        otpExpiresAt: admin.firestore.Timestamp.fromMillis(now.toMillis() + OTP_TTL_MS),
        attempts: 0,
        tgUserId: message.from.id,
        tgFirstName: message.from.first_name,
        tgUsername: message.from.username,
    };
    await loginRef.update(update_);
    const text_ = `Your login code is: ${otp}\n\nIt expires in 5 minutes. If you didn't request this, ignore this message.`;
    await (0, telegram_1.sendTelegramMessage)(TELEGRAM_BOT_TOKEN, message.chat.id, text_);
}
// --- telegramVerify ---
exports.telegramVerify = (0, https_1.onRequest)(async (req, res) => {
    if (applyCors(req, res))
        return;
    if (req.method !== "POST") {
        res.status(405).json({ error: "method not allowed" });
        return;
    }
    const { login_token: loginToken, code } = (req.body ?? {});
    if (!loginToken || !code) {
        res.status(400).json({ error: "login_token and code are required" });
        return;
    }
    let resolved;
    try {
        resolved = await verifyOtp(loginToken, code);
    }
    catch (err) {
        if (err instanceof VerifyError) {
            res.status(400).json({ error: err.message });
            return;
        }
        firebase_functions_1.logger.error("telegramVerify: unexpected error", err);
        res.status(500).json({ error: "internal error" });
        return;
    }
    try {
        const customToken = await resolveUserAndMintToken(resolved);
        res.status(200).json({ custom_token: customToken });
    }
    catch (err) {
        firebase_functions_1.logger.error("telegramVerify: failed to resolve user", err);
        res.status(500).json({ error: "internal error" });
    }
});
/**
 * Validates the OTP against telegram_logins/{loginToken} inside a
 * transaction, incrementing attempts on a wrong guess and consuming
 * (deleting) the doc on success, expiry, or exhausted attempts. Never logs
 * the code or the stored hash.
 */
async function verifyOtp(loginToken, code) {
    const loginRef = db.collection(LOGINS_COLLECTION).doc(loginToken);
    return db.runTransaction(async (tx) => {
        const snap = await tx.get(loginRef);
        if (!snap.exists) {
            throw new VerifyError("login expired, please start again");
        }
        const data = snap.data();
        const now = admin.firestore.Timestamp.now().toMillis();
        if (data.expiresAt.toMillis() < now) {
            tx.delete(loginRef);
            throw new VerifyError("login expired, please start again");
        }
        if (data.status !== "otp_sent" || !data.otpHash || !data.otpSalt || !data.otpExpiresAt) {
            throw new VerifyError("no code sent yet; open the bot and press Start");
        }
        if (data.otpExpiresAt.toMillis() < now) {
            tx.delete(loginRef);
            throw new VerifyError("code expired, please start again");
        }
        const attempts = data.attempts ?? 0;
        if (attempts >= MAX_OTP_ATTEMPTS) {
            tx.delete(loginRef);
            throw new VerifyError("too many attempts, please start again");
        }
        const candidateHash = (0, otp_1.hashOtp)(data.otpSalt, code);
        if (!(0, otp_1.hashesEqual)(candidateHash, data.otpHash)) {
            tx.update(loginRef, { attempts: attempts + 1 });
            throw new VerifyError("invalid code");
        }
        tx.delete(loginRef);
        return {
            tgUserId: data.tgUserId,
            tgFirstName: data.tgFirstName,
            tgUsername: data.tgUsername,
        };
    });
}
/** Resolves (or creates) the Firebase Auth user for a Telegram identity and mints a custom token. */
async function resolveUserAndMintToken(resolved) {
    const uid = `telegram:${resolved.tgUserId}`;
    const displayName = resolved.tgFirstName || resolved.tgUsername || `telegram_${resolved.tgUserId}`;
    let userRecord;
    try {
        userRecord = await admin.auth().getUser(uid);
    }
    catch (err) {
        if (err.code === "auth/user-not-found") {
            userRecord = await admin.auth().createUser({ uid, displayName });
        }
        else {
            throw err;
        }
    }
    await db
        .collection(USERS_COLLECTION)
        .doc(uid)
        .set({
        name: userRecord.displayName ?? displayName,
        phone: null,
        providers: ["telegram"],
        telegram: { id: resolved.tgUserId, username: resolved.tgUsername ?? null },
        updatedAt: admin.firestore.FieldValue.serverTimestamp(),
    }, { merge: true });
    return admin.auth().createCustomToken(uid, { provider: "telegram" });
}
//# sourceMappingURL=index.js.map