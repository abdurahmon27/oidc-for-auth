package handler

import (
	"crypto/rand"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/user/auth-service/internal/auth"
	db "github.com/user/auth-service/internal/database"
	"github.com/user/auth-service/internal/service"
	"github.com/user/auth-service/internal/token"
)

type OAuthHandler struct {
	registry    *auth.Registry
	userService *service.UserService
	queries     db.Querier
	jwtSecret   string
	cookieCfg   token.CookieConfig
	frontendURL string
}

func NewOAuthHandler(registry *auth.Registry, userService *service.UserService, queries db.Querier, jwtSecret string, cookieCfg token.CookieConfig, frontendURL string) *OAuthHandler {
	return &OAuthHandler{
		registry:    registry,
		userService: userService,
		queries:     queries,
		jwtSecret:   jwtSecret,
		cookieCfg:   cookieCfg,
		frontendURL: frontendURL,
	}
}

func (h *OAuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	providerName := chi.URLParam(r, "provider")
	provider, err := h.registry.Get(providerName)
	if err != nil {
		http.Error(w, `{"error":"unknown provider"}`, http.StatusBadRequest)
		return
	}

	state, err := auth.GenerateState(h.jwtSecret)
	if err != nil {
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}

	codeVerifier, err := auth.GenerateCodeVerifier()
	if err != nil {
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}

	// Store state and code_verifier in short-lived cookies
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    state,
		Path:     "/",
		MaxAge:   300,
		HttpOnly: true,
		Secure:   h.cookieCfg.Secure,
		SameSite: http.SameSiteLaxMode,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_verifier",
		Value:    codeVerifier,
		Path:     "/",
		MaxAge:   300,
		HttpOnly: true,
		Secure:   h.cookieCfg.Secure,
		SameSite: http.SameSiteLaxMode,
	})

	authURL := provider.AuthURL(state, codeVerifier)
	http.Redirect(w, r, authURL, http.StatusTemporaryRedirect)
}

func (h *OAuthHandler) Callback(w http.ResponseWriter, r *http.Request) {
	providerName := chi.URLParam(r, "provider")
	provider, err := h.registry.Get(providerName)
	if err != nil {
		http.Error(w, `{"error":"unknown provider"}`, http.StatusBadRequest)
		return
	}

	// Surface an error handed back by the provider instead of masking it as a
	// generic "missing code" — this is where reasons like consent_required or a
	// redirect-URI/platform mismatch show up.
	if providerErr := r.URL.Query().Get("error"); providerErr != "" {
		log.Printf("oauth callback error (%s): %s: %s", providerName,
			providerErr, r.URL.Query().Get("error_description"))
		http.Error(w, `{"error":"authentication failed"}`, http.StatusUnauthorized)
		return
	}

	// Get code and state from query
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	if code == "" || state == "" {
		http.Error(w, `{"error":"missing code or state"}`, http.StatusBadRequest)
		return
	}

	// Verify state
	stateCookie, err := r.Cookie("oauth_state")
	if err != nil || stateCookie.Value != state {
		http.Error(w, `{"error":"invalid state"}`, http.StatusBadRequest)
		return
	}
	if err := auth.VerifyState(h.jwtSecret, state); err != nil {
		http.Error(w, `{"error":"invalid state"}`, http.StatusBadRequest)
		return
	}

	// Get code verifier
	verifierCookie, err := r.Cookie("oauth_verifier")
	if err != nil {
		http.Error(w, `{"error":"missing code verifier"}`, http.StatusBadRequest)
		return
	}

	// Clear OAuth cookies
	clearCookie(w, "oauth_state", h.cookieCfg.Secure)
	clearCookie(w, "oauth_verifier", h.cookieCfg.Secure)

	// Exchange code for user info
	providerUser, err := provider.Exchange(r.Context(), code, verifierCookie.Value)
	if err != nil {
		log.Printf("oauth exchange error (%s): %v", providerName, err)
		http.Error(w, `{"error":"authentication failed"}`, http.StatusUnauthorized)
		return
	}

	// Find or create user
	result, err := h.userService.FindOrCreateByProvider(
		r.Context(),
		providerName,
		providerUser.ProviderID,
		providerUser.Email,
		providerUser.Name,
		providerUser.AvatarURL,
	)
	if err != nil {
		log.Printf("user service error: %v", err)
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}

	// Issue tokens
	if err := h.issueTokens(w, r, result.UserID, result.Email); err != nil {
		log.Printf("issue tokens error: %v", err)
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, h.frontendURL+"/dashboard", http.StatusTemporaryRedirect)
}

func (h *OAuthHandler) issueTokens(w http.ResponseWriter, r *http.Request, userID, email string) error {
	accessToken, err := token.SignAccessToken(h.jwtSecret, userID, email)
	if err != nil {
		return err
	}

	rawRefresh, hashRefresh, err := token.GenerateRefreshToken()
	if err != nil {
		return err
	}

	family := pgtype.UUID{}
	family.Scan(newUUID())

	userUUID := pgtype.UUID{}
	userUUID.Scan(userID)

	_, err = h.queries.CreateRefreshToken(r.Context(), db.CreateRefreshTokenParams{
		UserID:    userUUID,
		TokenHash: hashRefresh,
		Family:    family,
		ExpiresAt: pgtype.Timestamptz{Time: time.Now().Add(7 * 24 * time.Hour), Valid: true},
	})
	if err != nil {
		return err
	}

	token.SetAccessTokenCookie(w, h.cookieCfg, accessToken)
	token.SetRefreshTokenCookie(w, h.cookieCfg, rawRefresh)
	token.SetCSRFCookie(w, h.cookieCfg)

	return nil
}

func clearCookie(w http.ResponseWriter, name string, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   secure,
	})
}

func newUUID() string {
	b := make([]byte, 16)
	rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
