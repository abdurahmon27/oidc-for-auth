package auth

import (
	"context"
	"fmt"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type GoogleProvider struct {
	oauth2Cfg *oauth2.Config
	verifier  *oidc.IDTokenVerifier
}

func NewGoogleProvider(clientID, clientSecret, redirectURL string) (*GoogleProvider, error) {
	provider, err := oidc.NewProvider(context.Background(), "https://accounts.google.com")
	if err != nil {
		return nil, fmt.Errorf("google oidc provider: %w", err)
	}

	return &GoogleProvider{
		oauth2Cfg: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURL,
			Endpoint:     google.Endpoint,
			Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
		},
		verifier: provider.Verifier(&oidc.Config{ClientID: clientID}),
	}, nil
}

func (g *GoogleProvider) Name() string { return "google" }

func (g *GoogleProvider) AuthURL(state, codeVerifier string) string {
	return g.oauth2Cfg.AuthCodeURL(state,
		oauth2.SetAuthURLParam("code_challenge", oauth2.S256ChallengeFromVerifier(codeVerifier)),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
	)
}

func (g *GoogleProvider) Exchange(ctx context.Context, code, codeVerifier string) (*ProviderUser, error) {
	tok, err := g.oauth2Cfg.Exchange(ctx, code, oauth2.VerifierOption(codeVerifier))
	if err != nil {
		return nil, fmt.Errorf("google exchange: %w", err)
	}

	rawIDToken, ok := tok.Extra("id_token").(string)
	if !ok {
		return nil, fmt.Errorf("google: no id_token in response")
	}

	idToken, err := g.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return nil, fmt.Errorf("google: id_token verify: %w", err)
	}

	var claims struct {
		Sub     string `json:"sub"`
		Email   string `json:"email"`
		Name    string `json:"name"`
		Picture string `json:"picture"`
	}
	if err := idToken.Claims(&claims); err != nil {
		return nil, fmt.Errorf("google: claims: %w", err)
	}

	return &ProviderUser{
		ProviderID: claims.Sub,
		Email:      claims.Email,
		Name:       claims.Name,
		AvatarURL:  claims.Picture,
	}, nil
}
