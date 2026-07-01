package validate

import (
	"context"
	"fmt"

	"github.com/coreos/go-oidc/v3/oidc"
)

type MicrosoftValidator struct {
	verifier *oidc.IDTokenVerifier
}

func NewMicrosoftValidator(clientID, tenant string) (*MicrosoftValidator, error) {
	issuerURL := fmt.Sprintf("https://login.microsoftonline.com/%s/v2.0", tenant)
	provider, err := oidc.NewProvider(context.Background(), issuerURL)
	if err != nil {
		return nil, fmt.Errorf("microsoft oidc provider: %w", err)
	}
	return &MicrosoftValidator{
		verifier: provider.Verifier(&oidc.Config{ClientID: clientID}),
	}, nil
}

func (m *MicrosoftValidator) Name() string { return "microsoft" }

func (m *MicrosoftValidator) Validate(idToken string) (*ProviderUser, error) {
	token, err := m.verifier.Verify(context.Background(), idToken)
	if err != nil {
		return nil, fmt.Errorf("microsoft: verify id_token: %w", err)
	}

	var claims struct {
		Sub   string `json:"sub"`
		Email string `json:"email"`
		Name  string `json:"name"`
	}
	if err := token.Claims(&claims); err != nil {
		return nil, fmt.Errorf("microsoft: claims: %w", err)
	}

	return &ProviderUser{
		ProviderID: claims.Sub,
		Email:      claims.Email,
		Name:       claims.Name,
	}, nil
}
