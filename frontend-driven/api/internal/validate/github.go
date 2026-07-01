package validate

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type GitHubValidator struct {
	clientID     string
	clientSecret string
}

func NewGitHubValidator(clientID, clientSecret string) *GitHubValidator {
	return &GitHubValidator{
		clientID:     clientID,
		clientSecret: clientSecret,
	}
}

func (g *GitHubValidator) Name() string { return "github" }

func (g *GitHubValidator) Validate(code string) (*ProviderUser, error) {
	// Exchange code for access token
	data := url.Values{
		"client_id":     {g.clientID},
		"client_secret": {g.clientSecret},
		"code":          {code},
	}

	req, err := http.NewRequest("POST", "https://github.com/login/oauth/access_token", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("github: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("github: exchange code: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github exchange %d: %s", resp.StatusCode, body)
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		Error       string `json:"error"`
	}
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("github parse token: %w", err)
	}
	if tokenResp.Error != "" {
		return nil, fmt.Errorf("github: %s", tokenResp.Error)
	}

	// Fetch user profile
	user, err := g.fetchUser(tokenResp.AccessToken)
	if err != nil {
		return nil, err
	}

	if user.Email == "" {
		email, err := g.fetchPrimaryEmail(tokenResp.AccessToken)
		if err == nil {
			user.Email = email
		}
	}

	return user, nil
}

func (g *GitHubValidator) fetchUser(accessToken string) (*ProviderUser, error) {
	req, err := http.NewRequest("GET", "https://api.github.com/user", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
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

func (g *GitHubValidator) fetchPrimaryEmail(accessToken string) (string, error) {
	req, err := http.NewRequest("GET", "https://api.github.com/user/emails", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
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
