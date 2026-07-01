package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	db "github.com/user/auth-service-fd/internal/database"
	"github.com/user/auth-service-fd/internal/token"
)

type SessionHandler struct {
	queries   db.Querier
	jwtSecret string
	cookieCfg CookieConfig
}

func NewSessionHandler(queries db.Querier, jwtSecret string, cookieCfg CookieConfig) *SessionHandler {
	return &SessionHandler{
		queries:   queries,
		jwtSecret: jwtSecret,
		cookieCfg: cookieCfg,
	}
}

func (h *SessionHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("refresh_token")
	if err != nil {
		http.Error(w, `{"error":"no refresh token"}`, http.StatusUnauthorized)
		return
	}

	tokenHash := token.HashToken(cookie.Value)
	stored, err := h.queries.GetRefreshTokenByHash(r.Context(), tokenHash)
	if err != nil {
		http.Error(w, `{"error":"invalid refresh token"}`, http.StatusUnauthorized)
		return
	}

	// Reuse detection
	if stored.Revoked {
		log.Printf("refresh token reuse detected for family %v", stored.Family)
		h.queries.RevokeRefreshTokenFamily(r.Context(), stored.Family)
		clearRefreshCookie(w, h.cookieCfg)
		http.Error(w, `{"error":"token reuse detected"}`, http.StatusUnauthorized)
		return
	}

	if time.Now().After(stored.ExpiresAt.Time) {
		http.Error(w, `{"error":"refresh token expired"}`, http.StatusUnauthorized)
		return
	}

	// Revoke current token
	h.queries.RevokeRefreshToken(r.Context(), stored.ID)

	// Get user
	user, err := h.queries.GetUserByID(r.Context(), stored.UserID)
	if err != nil {
		http.Error(w, `{"error":"user not found"}`, http.StatusUnauthorized)
		return
	}

	// Issue new tokens
	accessToken, err := token.SignAccessToken(h.jwtSecret, uuidToString(user.ID), user.Email.String)
	if err != nil {
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}

	rawRefresh, hashRefresh, err := token.GenerateRefreshToken()
	if err != nil {
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}

	_, err = h.queries.CreateRefreshToken(r.Context(), db.CreateRefreshTokenParams{
		UserID:    stored.UserID,
		TokenHash: hashRefresh,
		Family:    stored.Family,
		ExpiresAt: pgtype.Timestamptz{Time: time.Now().Add(7 * 24 * time.Hour), Valid: true},
	})
	if err != nil {
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}

	// Set new refresh cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    rawRefresh,
		Path:     "/auth/refresh",
		Domain:   h.cookieCfg.Domain,
		MaxAge:   int((7 * 24 * time.Hour).Seconds()),
		HttpOnly: true,
		Secure:   h.cookieCfg.Secure,
		SameSite: http.SameSiteStrictMode,
	})

	// Fetch identities for response
	uid := user.ID
	identities, _ := h.queries.GetIdentitiesByUserID(r.Context(), uid)
	providers := make([]providerInfo, len(identities))
	for i, id := range identities {
		providers[i] = providerInfo{
			Provider: id.Provider,
			Email:    id.Email.String,
			Name:     id.Name.String,
		}
	}

	// Return access token in JSON body
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(authenticateResponse{
		AccessToken: accessToken,
		ExpiresIn:   int(token.AccessTokenExpiry.Seconds()),
		User: userInfo{
			ID:        uuidToString(user.ID),
			Email:     user.Email.String,
			Phone:     user.Phone.String,
			Name:      user.Name,
			AvatarURL: user.AvatarUrl,
			Providers: providers,
		},
	})
}

func (h *SessionHandler) Logout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("refresh_token")
	if err == nil {
		tokenHash := token.HashToken(cookie.Value)
		stored, err := h.queries.GetRefreshTokenByHash(r.Context(), tokenHash)
		if err == nil {
			h.queries.RevokeRefreshTokenFamily(r.Context(), stored.Family)
		}
	}

	clearRefreshCookie(w, h.cookieCfg)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "logged out"})
}

func clearRefreshCookie(w http.ResponseWriter, cfg CookieConfig) {
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
}

func uuidToString(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	b := u.Bytes
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
