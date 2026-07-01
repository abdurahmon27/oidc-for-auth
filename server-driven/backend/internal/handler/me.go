package handler

import (
	"encoding/json"
	"net/http"

	"github.com/jackc/pgx/v5/pgtype"

	db "github.com/user/auth-service/internal/database"
	"github.com/user/auth-service/internal/middleware"
)

type MeHandler struct {
	queries db.Querier
}

func NewMeHandler(queries db.Querier) *MeHandler {
	return &MeHandler{queries: queries}
}

type MeResponse struct {
	ID        string         `json:"id"`
	Email     string         `json:"email,omitempty"`
	Phone     string         `json:"phone,omitempty"`
	Name      string         `json:"name"`
	AvatarURL string         `json:"avatar_url,omitempty"`
	Providers []ProviderInfo `json:"providers"`
}

type ProviderInfo struct {
	Provider string `json:"provider"`
	Email    string `json:"email,omitempty"`
	Name     string `json:"name,omitempty"`
}

func (h *MeHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	uid := pgtype.UUID{}
	if err := uid.Scan(userID); err != nil {
		http.Error(w, `{"error":"invalid user id"}`, http.StatusBadRequest)
		return
	}

	user, err := h.queries.GetUserByID(r.Context(), uid)
	if err != nil {
		http.Error(w, `{"error":"user not found"}`, http.StatusNotFound)
		return
	}

	identities, err := h.queries.GetIdentitiesByUserID(r.Context(), uid)
	if err != nil {
		identities = []db.Identity{}
	}

	providers := make([]ProviderInfo, len(identities))
	for i, id := range identities {
		providers[i] = ProviderInfo{
			Provider: id.Provider,
			Email:    id.Email.String,
			Name:     id.Name.String,
		}
	}

	resp := MeResponse{
		ID:        uuidToString(user.ID),
		Email:     user.Email.String,
		Phone:     user.Phone.String,
		Name:      user.Name,
		AvatarURL: user.AvatarUrl,
		Providers: providers,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
