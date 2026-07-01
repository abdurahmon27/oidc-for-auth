// Small crypto helpers for the Telegram bot-OTP login flow. Kept separate
// from index.ts so the security-relevant primitives are easy to audit.
import * as crypto from "crypto";

/** 32 random bytes, base64url-encoded. Used as the single-use login token. */
export function randomToken(): string {
  return crypto.randomBytes(32).toString("base64url");
}

/** Random hex string used as the per-login OTP salt. */
export function randomSalt(): string {
  return crypto.randomBytes(16).toString("hex");
}

/** 6-digit numeric OTP, zero-padded, drawn from a CSPRNG. */
export function generateOtp(): string {
  return crypto.randomInt(0, 1_000_000).toString().padStart(6, "0");
}

/** sha256(salt + otp), hex-encoded. The OTP itself is never stored. */
export function hashOtp(salt: string, otp: string): string {
  return crypto.createHash("sha256").update(salt + otp).digest("hex");
}

/** Constant-time comparison of two hex-encoded hashes. */
export function hashesEqual(a: string, b: string): boolean {
  const bufA = Buffer.from(a, "hex");
  const bufB = Buffer.from(b, "hex");
  if (bufA.length !== bufB.length) return false;
  return crypto.timingSafeEqual(bufA, bufB);
}
