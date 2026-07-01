// Minimal Telegram Bot API client. Uses Node 22's built-in fetch, so no HTTP
// client dependency is needed.

const TELEGRAM_API_BASE = "https://api.telegram.org";

export interface TgUser {
  id: number;
  is_bot: boolean;
  username?: string;
  first_name?: string;
  last_name?: string;
}

interface TelegramApiResponse<T> {
  ok: boolean;
  result?: T;
  description?: string;
}

/** Calls `https://api.telegram.org/bot<token>/<method>` and unwraps the result. */
export async function telegramApi<T>(
  token: string,
  method: string,
  params?: Record<string, unknown>
): Promise<T> {
  const res = await fetch(`${TELEGRAM_API_BASE}/bot${token}/${method}`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(params ?? {}),
  });
  const data = (await res.json()) as TelegramApiResponse<T>;
  if (!data.ok || data.result === undefined) {
    throw new Error(`telegram ${method} failed: ${data.description ?? res.status}`);
  }
  return data.result;
}

// Module-level memo: getMe only needs to succeed once per instance.
let cachedBotUsername: string | undefined;

/** Resolves (and caches) the bot's @username via getMe. Throws if the token is invalid. */
export async function getBotUsername(token: string): Promise<string> {
  if (cachedBotUsername) return cachedBotUsername;
  const me = await telegramApi<TgUser>(token, "getMe");
  if (!me.username) throw new Error("bot has no username");
  cachedBotUsername = me.username;
  return cachedBotUsername;
}

export async function sendTelegramMessage(token: string, chatId: number, text: string): Promise<void> {
  await telegramApi(token, "sendMessage", { chat_id: chatId, text });
}

// --- Telegram update payload (only the fields we use) ---

export interface TelegramUpdate {
  update_id: number;
  message?: {
    text?: string;
    chat: { id: number };
    from: TgUser;
  };
}
