package validate

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

type FacebookValidator struct {
	clientID     string
	clientSecret string
}

func NewFacebookValidator(clientID, clientSecret string) *FacebookValidator {
	return &FacebookValidator{
		clientID:     clientID,
		clientSecret: clientSecret,
	}
}

func (f *FacebookValidator) Name() string { return "facebook" }

func (f *FacebookValidator) Validate(accessToken string) (*ProviderUser, error) {
	// Verify the token belongs to our app
	appToken := f.clientID + "|" + f.clientSecret
	debugURL := "https://graph.facebook.com/debug_token?" + url.Values{
		"input_token":  {accessToken},
		"access_token": {appToken},
	}.Encode()

	debugResp, err := http.Get(debugURL)
	if err != nil {
		return nil, fmt.Errorf("facebook debug_token: %w", err)
	}
	defer debugResp.Body.Close()

	debugBody, _ := io.ReadAll(debugResp.Body)
	if debugResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("facebook debug_token %d: %s", debugResp.StatusCode, debugBody)
	}

	var debugResult struct {
		Data struct {
			AppID   string `json:"app_id"`
			IsValid bool   `json:"is_valid"`
		} `json:"data"`
	}
	if err := json.Unmarshal(debugBody, &debugResult); err != nil {
		return nil, fmt.Errorf("facebook parse debug: %w", err)
	}
	if !debugResult.Data.IsValid || debugResult.Data.AppID != f.clientID {
		return nil, fmt.Errorf("facebook: token not valid for this app")
	}

	// Fetch user profile
	graphURL := "https://graph.facebook.com/me?" + url.Values{
		"fields":       {"id,name,email,picture.type(large)"},
		"access_token": {accessToken},
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
