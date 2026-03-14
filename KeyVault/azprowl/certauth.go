package main

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/crypto/pkcs12"
)

// ExchangePFXForToken authenticates as an app registration using the OAuth 2.0
// client credentials flow with a signed JWT client assertion derived from a PFX
// (PKCS#12) certificate. Only RSA private keys are supported.
//
// Returns the access token, an optional refresh token (typically empty for
// client_credentials), expiry in seconds, and any error.
func ExchangePFXForToken(pfxData []byte, password, tenantID, clientID, scope string) (string, string, int, error) {
	// 1. Parse PFX — extract the private key and leaf certificate.
	privKey, cert, err := pkcs12.Decode(pfxData, password)
	if err != nil {
		return "", "", 0, fmt.Errorf("PFX decode failed: %w", err)
	}
	rsaKey, ok := privKey.(*rsa.PrivateKey)
	if !ok {
		return "", "", 0, fmt.Errorf("PFX private key is not RSA (got %T); only RSA keys are supported", privKey)
	}

	// 2. Build the signed JWT client assertion.
	assertion, err := buildClientAssertion(rsaKey, cert, tenantID, clientID)
	if err != nil {
		return "", "", 0, fmt.Errorf("building client assertion: %w", err)
	}

	// 3. POST to the Microsoft identity platform token endpoint.
	endpoint := fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", tenantID)
	form := url.Values{
		"client_id":             {clientID},
		"client_assertion_type": {"urn:ietf:params:oauth:client-assertion-type:jwt-bearer"},
		"client_assertion":      {assertion},
		"grant_type":            {"client_credentials"},
		"scope":                 {scope},
	}
	resp, err := http.Post(endpoint, "application/x-www-form-urlencoded", strings.NewReader(form.Encode())) //nolint:gosec
	if err != nil {
		return "", "", 0, fmt.Errorf("token request: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", 0, fmt.Errorf("reading token response: %w", err)
	}

	var tr TokenResponse
	if err := json.Unmarshal(raw, &tr); err != nil {
		return "", "", 0, fmt.Errorf("parse token response: %w", err)
	}
	if tr.Error != "" {
		return "", "", 0, fmt.Errorf("%s: %s", tr.Error, tr.ErrorDesc)
	}
	if tr.AccessToken == "" {
		return "", "", 0, fmt.Errorf("empty access token in response")
	}
	exp := tr.ExpiresIn
	if exp == 0 {
		exp = 3600
	}
	return tr.AccessToken, tr.RefreshToken, exp, nil
}

// buildClientAssertion creates a signed RS256 JWT for use as client_assertion
// in the OAuth 2.0 client credentials flow. The x5t header contains the SHA-1
// thumbprint of the certificate so Azure AD can locate the matching public key
// registered on the app registration.
func buildClientAssertion(privKey *rsa.PrivateKey, cert *x509.Certificate, tenantID, clientID string) (string, error) {
	// x5t = base64url( SHA-1( DER-encoded certificate ) )
	thumbSum := sha1.Sum(cert.Raw)
	x5t := base64.RawURLEncoding.EncodeToString(thumbSum[:])

	headerJSON, _ := json.Marshal(map[string]string{
		"alg": "RS256",
		"typ": "JWT",
		"x5t": x5t,
	})

	now := time.Now().Unix()
	jtiBytes := make([]byte, 16)
	if _, err := rand.Read(jtiBytes); err != nil {
		return "", fmt.Errorf("generating jti: %w", err)
	}
	jti := fmt.Sprintf("%x-%x-%x-%x-%x",
		jtiBytes[0:4], jtiBytes[4:6], jtiBytes[6:8], jtiBytes[8:10], jtiBytes[10:16])

	payloadJSON, _ := json.Marshal(map[string]interface{}{
		"aud": fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", tenantID),
		"exp": now + 600,
		"iss": clientID,
		"jti": jti,
		"nbf": now,
		"sub": clientID,
	})

	headerB64 := base64.RawURLEncoding.EncodeToString(headerJSON)
	payloadB64 := base64.RawURLEncoding.EncodeToString(payloadJSON)
	signingInput := headerB64 + "." + payloadB64

	// RS256: PKCS#1 v1.5 sign the SHA-256 hash of the signing input.
	digest := sha256.Sum256([]byte(signingInput))
	sig, err := rsa.SignPKCS1v15(rand.Reader, privKey, crypto.SHA256, digest[:])
	if err != nil {
		return "", fmt.Errorf("RSA signing: %w", err)
	}
	return signingInput + "." + base64.RawURLEncoding.EncodeToString(sig), nil
}
