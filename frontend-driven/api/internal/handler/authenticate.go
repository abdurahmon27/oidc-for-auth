package handler

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	db "github.com/user/auth-service-fd/internal/database"
	"github.com/user/auth-service-fd/internal/service"
	"github.com/user/auth-service-fd/internal/token"
	"github.com/user/auth-service-fd/internal/validate"
)

type AuthenticateHandler struct {
	registry    *validate.Registry
	userService *service.UserService
	queries     db.Querier
	jwtSecret   string
	cookieCfg   CookieConfig
}

type CookieConfig struct {
	Domain string
	Secure bool
}

func NewAuthenticateHandler(registry *validate.Registry, userService *service.UserService, queries db.Querier, jwtSecret string, cookieCfg CookieConfig) *AuthenticateHandler {
	return &AuthenticateHandler{
		registry:    registry,
		userService: userService,
		queries:     queries,
		jwtSecret:   jwtSecret,
		cookieCfg:   cookieCfg,
	}
}

type authenticateRequest struct {
	Provider string `json:"provider"`
	Token    string `json:"token"`
}

type authenticateResponse struct {
	AccessToken string   `json:"access_token"`
	ExpiresIn   int      `json:"expires_in"`
	User        userInfo `json:"user"`
}

type userInfo struct {
	ID        string         `json:"id"`
	Email     string         `json:"email,omitempty"`
	Phone     string         `json:"phone,omitempty"`
	Name      string         `json:"name"`
	AvatarURL string         `json:"avatar_url,omitempty"`
	Providers []providerInfo `json:"providers"`
}

type providerInfo struct {
	Provider string `json:"provider"`
	Email    string `json:"email,omitempty"`
	Name     string `json:"name,omitempty"`
}

func (h *AuthenticateHandler) Authenticate(w http.ResponseWriter, r *http.Request) {
	var req authenticateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	if req.Provider == "" || req.Token == "" {
		http.Error(w, `{"error":"provider and token are required"}`, http.StatusBadRequest)
		return
	}

	validator, err := h.registry.Get(req.Provider)
	if err != nil {
		http.Error(w, `{"error":"unknown provider"}`, http.StatusBadRequest)
		return
	}

	providerUser, err := validator.Validate(req.Token)
	if err != nil {
		log.Printf("validation error (%s): %v", req.Provider, err)
		http.Error(w, `{"error":"authentication failed"}`, http.StatusUnauthorized)
		return
	}

	result, err := h.userService.FindOrCreateByProvider(
		r.Context(),
		req.Provider,
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

	h.issueTokensJSON(w, r, result.UserID, result.Email)
}

func (h *AuthenticateHandler) issueTokensJSON(w http.ResponseWriter, r *http.Request, userID, email string) {
	accessToken, err := token.SignAccessToken(h.jwtSecret, userID, email)
	if err != nil {
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}

	rawRefresh, hashRefresh, err := token.GenerateRefreshToken()
	if err != nil {
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}

	family := pgtype.UUID{}
	family.Scan(generateUUIDString())

	userUUID := pgtype.UUID{}
	userUUID.Scan(userID)

	_, err = h.queries.CreateRefreshToken(r.Context(), db.CreateRefreshTokenParams{
		UserID:    userUUID,
		TokenHash: hashRefresh,
		Family:    family,
		ExpiresAt: pgtype.Timestamptz{Time: time.Now().Add(7 * 24 * time.Hour), Valid: true},
	})
	if err != nil {
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}

	// Refresh token goes in httpOnly cookie
	setRefreshCookie(w, h.cookieCfg, rawRefresh)

	// Fetch user info for response
	uid := pgtype.UUID{}
	uid.Scan(userID)

	user, err := h.queries.GetUserByID(r.Context(), uid)
	if err != nil {
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}

	identities, _ := h.queries.GetIdentitiesByUserID(r.Context(), uid)

	providers := make([]providerInfo, len(identities))
	for i, id := range identities {
		providers[i] = providerInfo{
			Provider: id.Provider,
			Email:    id.Email.String,
			Name:     id.Name.String,
		}
	}

	resp := authenticateResponse{
		AccessToken: accessToken,
		ExpiresIn:   int(token.AccessTokenExpiry.Seconds()),
		User: userInfo{
			ID:        userID,
			Email:     user.Email.String,
			Phone:     user.Phone.String,
			Name:      user.Name,
			AvatarURL: user.AvatarUrl,
			Providers: providers,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func generateUUIDString() string {
	b := make([]byte, 16)
	rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
