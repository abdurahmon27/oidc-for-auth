package token

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"time"
)

type CookieConfig struct {
	Domain string
	Secure bool
}

func SetAccessTokenCookie(w http.ResponseWriter, cfg CookieConfig, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "access_token",
		Value:    token,
		Path:     "/",
		Domain:   cfg.Domain,
		MaxAge:   int(AccessTokenExpiry.Seconds()),
		HttpOnly: true,
		Secure:   cfg.Secure,
		SameSite: http.SameSiteLaxMode,
	})
}

func SetRefreshTokenCookie(w http.ResponseWriter, cfg CookieConfig, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    token,
		Path:     "/auth/refresh",
		Domain:   cfg.Domain,
		MaxAge:   int((7 * 24 * time.Hour).Seconds()),
		HttpOnly: true,
		Secure:   cfg.Secure,
		SameSite: http.SameSiteStrictMode,
	})
}

func SetCSRFCookie(w http.ResponseWriter, cfg CookieConfig) string {
	b := make([]byte, 16)
	rand.Read(b)
	csrfToken := hex.EncodeToString(b)
	http.SetCookie(w, &http.Cookie{
		Name:     "csrf_token",
		Value:    csrfToken,
		Path:     "/",
		Domain:   cfg.Domain,
		MaxAge:   int((7 * 24 * time.Hour).Seconds()),
		HttpOnly: false,
		Secure:   cfg.Secure,
		SameSite: http.SameSiteLaxMode,
	})
	return csrfToken
}

func ClearAuthCookies(w http.ResponseWriter, cfg CookieConfig) {
	http.SetCookie(w, &http.Cookie{
		Name:     "access_token",
		Value:    "",
		Path:     "/",
		Domain:   cfg.Domain,
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   cfg.Secure,
		SameSite: http.SameSiteLaxMode,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Path:     "/auth/refresh",
		Domain:   cfg.Domain,
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   cfg.Secure,
		SameSite: http.SameSiteStrictMode,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "csrf_token",
		Value:    "",
		Path:     "/",
		Domain:   cfg.Domain,
		MaxAge:   -1,
		HttpOnly: false,
		Secure:   cfg.Secure,
		SameSite: http.SameSiteLaxMode,
	})
}
