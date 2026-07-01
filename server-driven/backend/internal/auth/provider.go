package auth

import "context"

type ProviderUser struct {
	ProviderID string
	Email      string
	Name       string
	AvatarURL  string
}

type Provider interface {
	Name() string
	AuthURL(state, codeVerifier string) string
	Exchange(ctx context.Context, code, codeVerifier string) (*ProviderUser, error)
}
