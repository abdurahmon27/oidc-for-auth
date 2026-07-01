package service

import (
	"context"
	"fmt"
	"strconv"

	"github.com/jackc/pgx/v5/pgtype"

	db "github.com/user/auth-service/internal/database"
)

type UserService struct {
	queries db.Querier
}

func NewUserService(queries db.Querier) *UserService {
	return &UserService{queries: queries}
}

type AuthResult struct {
	UserID string
	Email  string
}

func (s *UserService) FindOrCreateByProvider(ctx context.Context, provider, providerID, email, name, avatarURL string) (*AuthResult, error) {
	// 1. Check if this provider identity already exists
	identity, err := s.queries.GetIdentityByProviderID(ctx, db.GetIdentityByProviderIDParams{
		Provider:   provider,
		ProviderID: providerID,
	})
	if err == nil {
		// Returning user — fetch and return
		user, err := s.queries.GetUserByID(ctx, identity.UserID)
		if err != nil {
			return nil, fmt.Errorf("get user: %w", err)
		}
		return &AuthResult{
			UserID: uuidToString(user.ID),
			Email:  user.Email.String,
		}, nil
	}

	// 2. Try to find existing user by email (auto-link)
	if email != "" {
		existingUser, err := s.queries.GetUserByEmail(ctx, pgtype.Text{String: email, Valid: true})
		if err == nil {
			// Link new identity to existing user
			_, err := s.queries.CreateIdentity(ctx, db.CreateIdentityParams{
				UserID:     existingUser.ID,
				Provider:   provider,
				ProviderID: providerID,
				Email:      pgtype.Text{String: email, Valid: email != ""},
				Name:       pgtype.Text{String: name, Valid: name != ""},
			})
			if err != nil {
				return nil, fmt.Errorf("link identity: %w", err)
			}

			// Update user info if better data available
			if (existingUser.Name == "" && name != "") || (existingUser.AvatarUrl == "" && avatarURL != "") {
				updateName := name
				if existingUser.Name != "" {
					updateName = existingUser.Name
				}
				updateAvatar := avatarURL
				if existingUser.AvatarUrl != "" {
					updateAvatar = existingUser.AvatarUrl
				}
				s.queries.UpdateUser(ctx, db.UpdateUserParams{
					ID:        existingUser.ID,
					Name:      updateName,
					AvatarUrl: updateAvatar,
				})
			}

			return &AuthResult{
				UserID: uuidToString(existingUser.ID),
				Email:  existingUser.Email.String,
			}, nil
		}
	}

	// 3. Create new user + identity
	user, err := s.queries.CreateUser(ctx, db.CreateUserParams{
		Email:     pgtype.Text{String: email, Valid: email != ""},
		Name:      name,
		AvatarUrl: avatarURL,
	})
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	_, err = s.queries.CreateIdentity(ctx, db.CreateIdentityParams{
		UserID:     user.ID,
		Provider:   provider,
		ProviderID: providerID,
		Email:      pgtype.Text{String: email, Valid: email != ""},
		Name:       pgtype.Text{String: name, Valid: name != ""},
	})
	if err != nil {
		return nil, fmt.Errorf("create identity: %w", err)
	}

	return &AuthResult{
		UserID: uuidToString(user.ID),
		Email:  user.Email.String,
	}, nil
}

// FindOrCreateByTelegram resolves a Telegram account (keyed by the numeric
// Telegram user id) to a local user, creating one on first login.
func (s *UserService) FindOrCreateByTelegram(ctx context.Context, telegramID int64, name string) (*AuthResult, error) {
	providerID := strconv.FormatInt(telegramID, 10)

	identity, err := s.queries.GetIdentityByProviderID(ctx, db.GetIdentityByProviderIDParams{
		Provider:   "telegram",
		ProviderID: providerID,
	})
	if err == nil {
		user, err := s.queries.GetUserByID(ctx, identity.UserID)
		if err != nil {
			return nil, fmt.Errorf("get user: %w", err)
		}
		return &AuthResult{
			UserID: uuidToString(user.ID),
			Email:  user.Email.String,
		}, nil
	}

	// Create new user
	user, err := s.queries.CreateUser(ctx, db.CreateUserParams{
		Name: name,
	})
	if err != nil {
		return nil, fmt.Errorf("create user by telegram: %w", err)
	}

	// Create telegram identity
	_, err = s.queries.CreateIdentity(ctx, db.CreateIdentityParams{
		UserID:     user.ID,
		Provider:   "telegram",
		ProviderID: providerID,
		Name:       pgtype.Text{String: name, Valid: name != ""},
	})
	if err != nil {
		return nil, fmt.Errorf("create telegram identity: %w", err)
	}

	return &AuthResult{
		UserID: uuidToString(user.ID),
		Email:  "",
	}, nil
}

func uuidToString(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	b := u.Bytes
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
