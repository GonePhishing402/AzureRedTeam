package main

import (
	"encoding/base64"
	"encoding/json"
	"strings"
)

// jwtClaims holds the fields we care about for auto-labelling tokens.
type jwtClaims struct {
	UPN   string `json:"upn"`
	AppID string `json:"appid"`
	OID   string `json:"oid"`
	Name  string `json:"name"`
}

// parseJWTLabel decodes the payload of a JWT (no signature validation) and
// returns a human-readable label derived from its claims.
// Priority: upn → name → appid → first 8 chars of oid → "unknown".
func parseJWTLabel(token string) string {
	parts := strings.Split(token, ".")
	if len(parts) < 2 {
		return "unknown"
	}

	// JWT payload uses base64url without padding.
	payload := parts[1]
	// Add padding if needed.
	switch len(payload) % 4 {
	case 2:
		payload += "=="
	case 3:
		payload += "="
	}

	raw, err := base64.URLEncoding.DecodeString(payload)
	if err != nil {
		return "unknown"
	}

	var c jwtClaims
	if err := json.Unmarshal(raw, &c); err != nil {
		return "unknown"
	}

	switch {
	case c.UPN != "":
		return c.UPN
	case c.Name != "":
		return c.Name
	case c.AppID != "":
		return "app:" + c.AppID
	case len(c.OID) >= 8:
		return "oid:" + c.OID[:8]
	default:
		return "unknown"
	}
}
