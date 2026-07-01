package auth

import (
	"context"
	"fmt"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/microsoft"
)

type MicrosoftProvider struct {
	oauth2Cfg *oauth2.Config
	verifier  *oidc.IDTokenVerifier
}

func NewMicrosoftProvider(clientID, clientSecret, tenant, redirectURL string) (*MicrosoftProvider, error) {
	ctx := context.Background()
	issuerURL := fmt.Sprintf("https://login.microsoftonline.com/%s/v2.0", tenant)

	// Multi-tenant endpoints (common/organizations/consumers) advertise a
	// templated issuer ("…/{tenantid}/v2.0") that never matches the discovery
	// URL, and each user's token carries their own tenant's issuer. Tell go-oidc
	// to accept the templated issuer during discovery and skip the per-tenant
	// issuer check at verification time; single-tenant stays strict.
	multiTenant := tenant == "common" || tenant == "organizations" || tenant == "consumers"
	if multiTenant {
		ctx = oidc.InsecureIssuerURLContext(ctx, "https://login.microsoftonline.com/{tenantid}/v2.0")
	}

	provider, err := oidc.NewProvider(ctx, issuerURL)
	if err != nil {
		return nil, fmt.Errorf("microsoft oidc provider: %w", err)
	}

	return &MicrosoftProvider{
		oauth2Cfg: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURL,
			Endpoint:     microsoft.AzureADEndpoint(tenant),
			Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
		},
		verifier: provider.Verifier(&oidc.Config{ClientID: clientID, SkipIssuerCheck: multiTenant}),
	}, nil
}

func (m *MicrosoftProvider) Name() string { return "microsoft" }

func (m *MicrosoftProvider) AuthURL(state, codeVerifier string) string {
	return m.oauth2Cfg.AuthCodeURL(state,
		oauth2.SetAuthURLParam("code_challenge", oauth2.S256ChallengeFromVerifier(codeVerifier)),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
	)
}

func (m *MicrosoftProvider) Exchange(ctx context.Context, code, codeVerifier string) (*ProviderUser, error) {
	tok, err := m.oauth2Cfg.Exchange(ctx, code, oauth2.VerifierOption(codeVerifier))
	if err != nil {
		return nil, fmt.Errorf("microsoft exchange: %w", err)
	}

	rawIDToken, ok := tok.Extra("id_token").(string)
	if !ok {
		return nil, fmt.Errorf("microsoft: no id_token in response")
	}

	idToken, err := m.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return nil, fmt.Errorf("microsoft: id_token verify: %w", err)
	}

	var claims struct {
		Sub   string `json:"sub"`
		Email string `json:"email"`
		Name  string `json:"name"`
	}
	if err := idToken.Claims(&claims); err != nil {
		return nil, fmt.Errorf("microsoft: claims: %w", err)
	}

	return &ProviderUser{
		ProviderID: claims.Sub,
		Email:      claims.Email,
		Name:       claims.Name,
	}, nil
}
