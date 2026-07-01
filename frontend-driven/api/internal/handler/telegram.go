package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	db "github.com/user/auth-service-fd/internal/database"
	"github.com/user/auth-service-fd/internal/service"
	"github.com/user/auth-service-fd/internal/telegram"
	"github.com/user/auth-service-fd/internal/token"
)

type TelegramHandler struct {
	bot         *telegram.Bot
	userService *service.UserService
	queries     db.Querier
	jwtSecret   string
	cookieCfg   CookieConfig
}

func NewTelegramHandler(bot *telegram.Bot, userService *service.UserService, queries db.Querier, jwtSecret string, cookieCfg CookieConfig) *TelegramHandler {
	return &TelegramHandler{
		bot:         bot,
		userService: userService,
		queries:     queries,
		jwtSecret:   jwtSecret,
		cookieCfg:   cookieCfg,
	}
}

// Start mints a login token and returns the bot handle + deep link the user
// should open in Telegram.
func (h *TelegramHandler) Start(w http.ResponseWriter, r *http.Request) {
	loginToken, deepLink, err := h.bot.CreateLogin()
	if err != nil {
		http.Error(w, `{"error":"telegram login unavailable"}`, http.StatusServiceUnavailable)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"bot_username": h.bot.Username(),
		"deep_link":    deepLink,
		"login_token":  loginToken,
	})
}

// Verify checks the in-chat OTP for a login token and issues session tokens.
func (h *TelegramHandler) Verify(w http.ResponseWriter, r *http.Request) {
	var req struct {
		LoginToken string `json:"login_token"`
		Code       string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}
	if req.LoginToken == "" || req.Code == "" {
		http.Error(w, `{"error":"login_token and code are required"}`, http.StatusBadRequest)
		return
	}

	tgUser, err := h.bot.Verify(req.LoginToken, req.Code)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusUnauthorized)
		return
	}

	result, err := h.userService.FindOrCreateByTelegram(r.Context(), tgUser.ID, tgUser.DisplayName())
	if err != nil {
		log.Printf("user service error: %v", err)
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}

	// Issue tokens - same format as /auth/token
	accessToken, err := token.SignAccessToken(h.jwtSecret, result.UserID, result.Email)
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
	userUUID.Scan(result.UserID)

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

	// Refresh token in cookie
	setRefreshCookie(w, h.cookieCfg, rawRefresh)

	// Fetch user info
	uid := pgtype.UUID{}
	uid.Scan(result.UserID)

	user, _ := h.queries.GetUserByID(r.Context(), uid)
	identities, _ := h.queries.GetIdentitiesByUserID(r.Context(), uid)

	providers := make([]providerInfo, len(identities))
	for i, id := range identities {
		providers[i] = providerInfo{
			Provider: id.Provider,
			Email:    id.Email.String,
			Name:     id.Name.String,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(authenticateResponse{
		AccessToken: accessToken,
		ExpiresIn:   int(token.AccessTokenExpiry.Seconds()),
		User: userInfo{
			ID:        result.UserID,
			Email:     user.Email.String,
			Phone:     user.Phone.String,
			Name:      user.Name,
			AvatarURL: user.AvatarUrl,
			Providers: providers,
		},
	})
}
