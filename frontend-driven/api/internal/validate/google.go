package validate

import (
	"context"
	"fmt"

	"github.com/coreos/go-oidc/v3/oidc"
)

type GoogleValidator struct {
	verifier *oidc.IDTokenVerifier
}

func NewGoogleValidator(clientID string) (*GoogleValidator, error) {
	provider, err := oidc.NewProvider(context.Background(), "https://accounts.google.com")
	if err != nil {
		return nil, fmt.Errorf("google oidc provider: %w", err)
	}
	return &GoogleValidator{
		verifier: provider.Verifier(&oidc.Config{ClientID: clientID}),
	}, nil
}

func (g *GoogleValidator) Name() string { return "google" }

func (g *GoogleValidator) Validate(idToken string) (*ProviderUser, error) {
	token, err := g.verifier.Verify(context.Background(), idToken)
	if err != nil {
		return nil, fmt.Errorf("google: verify id_token: %w", err)
	}

	var claims struct {
		Sub     string `json:"sub"`
		Email   string `json:"email"`
		Name    string `json:"name"`
		Picture string `json:"picture"`
	}
	if err := token.Claims(&claims); err != nil {
		return nil, fmt.Errorf("google: claims: %w", err)
	}

	return &ProviderUser{
		ProviderID: claims.Sub,
		Email:      claims.Email,
		Name:       claims.Name,
		AvatarURL:  claims.Picture,
	}, nil
}
