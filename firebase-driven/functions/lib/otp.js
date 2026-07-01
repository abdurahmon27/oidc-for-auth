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
exports.randomToken = randomToken;
exports.randomSalt = randomSalt;
exports.generateOtp = generateOtp;
exports.hashOtp = hashOtp;
exports.hashesEqual = hashesEqual;
// Small crypto helpers for the Telegram bot-OTP login flow. Kept separate
// from index.ts so the security-relevant primitives are easy to audit.
const crypto = __importStar(require("crypto"));
/** 32 random bytes, base64url-encoded. Used as the single-use login token. */
function randomToken() {
    return crypto.randomBytes(32).toString("base64url");
}
/** Random hex string used as the per-login OTP salt. */
function randomSalt() {
    return crypto.randomBytes(16).toString("hex");
}
/** 6-digit numeric OTP, zero-padded, drawn from a CSPRNG. */
function generateOtp() {
    return crypto.randomInt(0, 1_000_000).toString().padStart(6, "0");
}
/** sha256(salt + otp), hex-encoded. The OTP itself is never stored. */
function hashOtp(salt, otp) {
    return crypto.createHash("sha256").update(salt + otp).digest("hex");
}
/** Constant-time comparison of two hex-encoded hashes. */
function hashesEqual(a, b) {
    const bufA = Buffer.from(a, "hex");
    const bufB = Buffer.from(b, "hex");
    if (bufA.length !== bufB.length)
        return false;
    return crypto.timingSafeEqual(bufA, bufB);
}
//# sourceMappingURL=otp.js.map