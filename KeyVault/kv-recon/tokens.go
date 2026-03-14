package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// TokenResponse is the OAuth 2.0 token endpoint response.
type TokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope"`
	Error       string `json:"error"`
	ErrorDesc   string `json:"error_description"`
}

// exchangeToken exchanges a refresh token for an access token scoped to the
// given resource. tenantID may be a GUID or "common".
func exchangeToken(tenantID, clientID, refreshToken, scope string) (string, error) {
	endpoint := fmt.Sprintf(
		"https://login.microsoftonline.com/%s/oauth2/v2.0/token", tenantID)

	body := url.Values{
		"client_id":     {clientID},
		"scope":         {scope},
		"refresh_token": {refreshToken},
		"grant_type":    {"refresh_token"},
	}

	resp, err := http.Post( //nolint:gosec // intentional token exchange
		endpoint,
		"application/x-www-form-urlencoded",
		strings.NewReader(body.Encode()),
	)
	if err != nil {
		return "", fmt.Errorf("token request failed: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading token response: %w", err)
	}

	var tr TokenResponse
	if err := json.Unmarshal(raw, &tr); err != nil {
		return "", fmt.Errorf("parsing token response: %w", err)
	}
	if tr.Error != "" {
		return "", fmt.Errorf("token error %s: %s", tr.Error, tr.ErrorDesc)
	}
	if tr.AccessToken == "" {
		return "", fmt.Errorf("empty access token in response")
	}
	return tr.AccessToken, nil
}

// tokenPreview returns the first 50 characters of a token for safe display.
func tokenPreview(token string) string {
	if len(token) <= 50 {
		return token
	}
	return token[:50] + "..."
}
