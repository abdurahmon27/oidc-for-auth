package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
)

type GitHubProvider struct {
	oauth2Cfg *oauth2.Config
}

func NewGitHubProvider(clientID, clientSecret, redirectURL string) *GitHubProvider {
	return &GitHubProvider{
		oauth2Cfg: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURL,
			Endpoint:     github.Endpoint,
			Scopes:       []string{"user:email"},
		},
	}
}

func (g *GitHubProvider) Name() string { return "github" }

func (g *GitHubProvider) AuthURL(state, codeVerifier string) string {
	return g.oauth2Cfg.AuthCodeURL(state,
		oauth2.SetAuthURLParam("code_challenge", oauth2.S256ChallengeFromVerifier(codeVerifier)),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
	)
}

func (g *GitHubProvider) Exchange(ctx context.Context, code, codeVerifier string) (*ProviderUser, error) {
	tok, err := g.oauth2Cfg.Exchange(ctx, code, oauth2.VerifierOption(codeVerifier))
	if err != nil {
		return nil, fmt.Errorf("github exchange: %w", err)
	}

	client := g.oauth2Cfg.Client(ctx, tok)

	user, err := g.fetchUser(client)
	if err != nil {
		return nil, err
	}

	if user.Email == "" {
		email, err := g.fetchPrimaryEmail(client)
		if err == nil {
			user.Email = email
		}
	}

	return user, nil
}

func (g *GitHubProvider) fetchUser(client *http.Client) (*ProviderUser, error) {
	resp, err := client.Get("https://api.github.com/user")
	if err != nil {
		return nil, fmt.Errorf("github user api: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github user api %d: %s", resp.StatusCode, body)
	}

	var profile struct {
		ID        int    `json:"id"`
		Login     string `json:"login"`
		Name      string `json:"name"`
		Email     string `json:"email"`
		AvatarURL string `json:"avatar_url"`
	}
	if err := json.Unmarshal(body, &profile); err != nil {
		return nil, fmt.Errorf("github parse user: %w", err)
	}

	name := profile.Name
	if name == "" {
		name = profile.Login
	}

	return &ProviderUser{
		ProviderID: strconv.Itoa(profile.ID),
		Email:      profile.Email,
		Name:       name,
		AvatarURL:  profile.AvatarURL,
	}, nil
}

func (g *GitHubProvider) fetchPrimaryEmail(client *http.Client) (string, error) {
	resp, err := client.Get("https://api.github.com/user/emails")
	if err != nil {
		return "", fmt.Errorf("github emails api: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("github emails api %d: %s", resp.StatusCode, body)
	}

	var emails []struct {
		Email    string `json:"email"`
		Primary  bool   `json:"primary"`
		Verified bool   `json:"verified"`
	}
	if err := json.Unmarshal(body, &emails); err != nil {
		return "", fmt.Errorf("github parse emails: %w", err)
	}

	for _, e := range emails {
		if e.Primary && e.Verified {
			return e.Email, nil
		}
	}
	return "", fmt.Errorf("no verified primary email found")
}
