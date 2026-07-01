package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/facebook"
)

type FacebookProvider struct {
	oauth2Cfg *oauth2.Config
}

func NewFacebookProvider(clientID, clientSecret, redirectURL string) *FacebookProvider {
	return &FacebookProvider{
		oauth2Cfg: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURL,
			Endpoint:     facebook.Endpoint,
			Scopes:       []string{"email", "public_profile"},
		},
	}
}

func (f *FacebookProvider) Name() string { return "facebook" }

func (f *FacebookProvider) AuthURL(state, codeVerifier string) string {
	return f.oauth2Cfg.AuthCodeURL(state,
		oauth2.SetAuthURLParam("code_challenge", oauth2.S256ChallengeFromVerifier(codeVerifier)),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
	)
}

func (f *FacebookProvider) Exchange(ctx context.Context, code, codeVerifier string) (*ProviderUser, error) {
	tok, err := f.oauth2Cfg.Exchange(ctx, code, oauth2.VerifierOption(codeVerifier))
	if err != nil {
		return nil, fmt.Errorf("facebook exchange: %w", err)
	}

	graphURL := "https://graph.facebook.com/me?" + url.Values{
		"fields":       {"id,name,email,picture.type(large)"},
		"access_token": {tok.AccessToken},
	}.Encode()

	resp, err := http.Get(graphURL)
	if err != nil {
		return nil, fmt.Errorf("facebook graph api: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("facebook graph api %d: %s", resp.StatusCode, body)
	}

	var profile struct {
		ID      string `json:"id"`
		Name    string `json:"name"`
		Email   string `json:"email"`
		Picture struct {
			Data struct {
				URL string `json:"url"`
			} `json:"data"`
		} `json:"picture"`
	}
	if err := json.Unmarshal(body, &profile); err != nil {
		return nil, fmt.Errorf("facebook parse profile: %w", err)
	}

	return &ProviderUser{
		ProviderID: profile.ID,
		Email:      profile.Email,
		Name:       profile.Name,
		AvatarURL:  profile.Picture.Data.URL,
	}, nil
}
