package auth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

const ElyByAuthURL = "https://authserver.ely.by/auth/"

type ElyByProfile struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type ElyBySession struct {
	AccessToken     string       `json:"accessToken"`
	SelectedProfile ElyByProfile `json:"selectedProfile"`
}

type ElyByClient struct {
	HTTPClient *http.Client
	BaseURL    string
}

func (c ElyByClient) Authenticate(username, password string) (ElyBySession, error) {
	return c.request("authenticate", map[string]any{
		"agent":       map[string]string{"name": "Minecraft", "version": "1"},
		"username":    username,
		"password":    password,
		"requestUser": true,
	})
}

func (c ElyByClient) Refresh(accessToken string) (ElyBySession, error) {
	return c.request("refresh", map[string]any{"accessToken": accessToken, "requestUser": true})
}

func (c ElyByClient) Invalidate(accessToken string) error {
	_, err := c.request("invalidate", map[string]any{"accessToken": accessToken})
	return err
}

func (c ElyByClient) request(path string, payload any) (ElyBySession, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return ElyBySession{}, err
	}
	baseURL := c.BaseURL
	if baseURL == "" {
		baseURL = ElyByAuthURL
	}
	client := c.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	response, err := client.Post(baseURL+path, "application/json", bytes.NewReader(body))
	if err != nil {
		return ElyBySession{}, fmt.Errorf("ely.by request: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return ElyBySession{}, fmt.Errorf("ely.by authentication failed: %s", response.Status)
	}
	if path == "invalidate" {
		return ElyBySession{}, nil
	}
	var session ElyBySession
	if err := json.NewDecoder(response.Body).Decode(&session); err != nil {
		return ElyBySession{}, fmt.Errorf("decode ely.by session: %w", err)
	}
	if session.AccessToken == "" || session.SelectedProfile.ID == "" || session.SelectedProfile.Name == "" {
		return ElyBySession{}, fmt.Errorf("ely.by returned an incomplete session")
	}
	return session, nil
}
