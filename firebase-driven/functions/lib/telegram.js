"use strict";
// Minimal Telegram Bot API client. Uses Node 22's built-in fetch, so no HTTP
// client dependency is needed.
Object.defineProperty(exports, "__esModule", { value: true });
exports.telegramApi = telegramApi;
exports.getBotUsername = getBotUsername;
exports.sendTelegramMessage = sendTelegramMessage;
const TELEGRAM_API_BASE = "https://api.telegram.org";
/** Calls `https://api.telegram.org/bot<token>/<method>` and unwraps the result. */
async function telegramApi(token, method, params) {
    const res = await fetch(`${TELEGRAM_API_BASE}/bot${token}/${method}`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(params ?? {}),
    });
    const data = (await res.json());
    if (!data.ok || data.result === undefined) {
        throw new Error(`telegram ${method} failed: ${data.description ?? res.status}`);
    }
    return data.result;
}
// Module-level memo: getMe only needs to succeed once per instance.
let cachedBotUsername;
/** Resolves (and caches) the bot's @username via getMe. Throws if the token is invalid. */
async function getBotUsername(token) {
    if (cachedBotUsername)
        return cachedBotUsername;
    const me = await telegramApi(token, "getMe");
    if (!me.username)
        throw new Error("bot has no username");
    cachedBotUsername = me.username;
    return cachedBotUsername;
}
async function sendTelegramMessage(token, chatId, text) {
    await telegramApi(token, "sendMessage", { chat_id: chatId, text });
}
//# sourceMappingURL=telegram.js.map