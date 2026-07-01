package handler

import (
	"encoding/json"
	"net/http"

	"github.com/jackc/pgx/v5/pgtype"

	db "github.com/user/auth-service-fd/internal/database"
	"github.com/user/auth-service-fd/internal/middleware"
)

type MeHandler struct {
	queries db.Querier
}

func NewMeHandler(queries db.Querier) *MeHandler {
	return &MeHandler{queries: queries}
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

	providers := make([]providerInfo, len(identities))
	for i, id := range identities {
		providers[i] = providerInfo{
			Provider: id.Provider,
			Email:    id.Email.String,
			Name:     id.Name.String,
		}
	}

	resp := userInfo{
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
