package main

import (
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"golang.org/x/crypto/pkcs12"
)

// ── Secrets ───────────────────────────────────────────────────────────────────

// SecretItem is a single entry from the secrets list endpoint.
type SecretItem struct {
	ID string `json:"id"`
}

type secretListPage struct {
	Value    []SecretItem `json:"value"`
	NextLink string       `json:"nextLink"`
}

// SecretAttributes holds enabled/expiry metadata for a secret.
type SecretAttributes struct {
	Enabled bool  `json:"enabled"`
	Exp     int64 `json:"exp"`
}

// SecretResponse is the response from the secret GET endpoint.
type SecretResponse struct {
	Value       string           `json:"value"`
	ContentType string           `json:"contentType"`
	Attributes  SecretAttributes `json:"attributes"`
}

// ListSecretNames returns the names of all secrets in the vault (handles pagination).
func ListSecretNames(kvToken, vaultURI, apiVersion string) ([]string, error) {
	uri := fmt.Sprintf("%s/secrets?api-version=%s", vaultURI, apiVersion)
	var names []string
	for uri != "" {
		var page secretListPage
		if err := kvGet(kvToken, uri, &page); err != nil {
			return nil, err
		}
		for _, item := range page.Value {
			parts := strings.Split(strings.TrimRight(item.ID, "/"), "/")
			names = append(names, parts[len(parts)-1])
		}
		uri = page.NextLink
	}
	return names, nil
}

// GetSecret retrieves a single secret's value and metadata.
func GetSecret(kvToken, vaultURI, name, apiVersion string) (SecretResponse, error) {
	uri := fmt.Sprintf("%s/secrets/%s?api-version=%s", vaultURI, name, apiVersion)
	var sr SecretResponse
	err := kvGet(kvToken, uri, &sr)
	return sr, err
}

// FormatExpiry converts a Unix epoch to a date string, or "No expiry" if zero.
func FormatExpiry(exp int64) string {
	if exp == 0 {
		return "No expiry"
	}
	return time.Unix(exp, 0).UTC().Format("2006-01-02")
}

// ── Certificates ─────────────────────────────────────────────────────────────

// CertItem is a single entry from the certificates list endpoint.
type CertItem struct {
	ID string `json:"id"`
}

type certListPage struct {
	Value    []CertItem `json:"value"`
	NextLink string     `json:"nextLink"`
}

// CertAttributes holds enabled/validity metadata for a certificate.
type CertAttributes struct {
	Enabled bool  `json:"enabled"`
	Nbf     int64 `json:"nbf"`
	Exp     int64 `json:"exp"`
}

// SecretProperties holds the content type from a certificate policy.
type SecretProperties struct {
	ContentType string `json:"contentType"`
}

// CertPolicy wraps the secret properties in a certificate policy.
type CertPolicy struct {
	SecretProperties SecretProperties `json:"secretProperties"`
}

// CertResponse is the response from the certificate GET endpoint.
type CertResponse struct {
	ID         string         `json:"id"`
	X5T        string         `json:"x5t"`
	Cer        string         `json:"cer"` // base64 DER of public cert
	Attributes CertAttributes `json:"attributes"`
	Policy     CertPolicy     `json:"policy"`
}

// ListCertNames returns the names of all certificates in the vault (handles pagination).
func ListCertNames(kvToken, vaultURI, apiVersion string) ([]string, error) {
	uri := fmt.Sprintf("%s/certificates?api-version=%s", vaultURI, apiVersion)
	var names []string
	for uri != "" {
		var page certListPage
		if err := kvGet(kvToken, uri, &page); err != nil {
			return nil, err
		}
		for _, item := range page.Value {
			parts := strings.Split(strings.TrimRight(item.ID, "/"), "/")
			names = append(names, parts[len(parts)-1])
		}
		uri = page.NextLink
	}
	return names, nil
}

// GetCert retrieves certificate metadata.
func GetCert(kvToken, vaultURI, name, apiVersion string) (CertResponse, error) {
	uri := fmt.Sprintf("%s/certificates/%s?api-version=%s", vaultURI, name, apiVersion)
	var cr CertResponse
	err := kvGet(kvToken, uri, &cr)
	return cr, err
}

// GetCertBundle retrieves the certificate + private key bundle via the secrets
// endpoint. Returns (base64bundle, contentType, error).
// Key Vault stores cert+key as a secret at the same name as the certificate.
// Requires Key Vault Secrets User permission or higher.
func GetCertBundle(kvToken, vaultURI, name, apiVersion string) (string, string, error) {
	uri := fmt.Sprintf("%s/secrets/%s?api-version=%s", vaultURI, name, apiVersion)
	var sr SecretResponse
	if err := kvGet(kvToken, uri, &sr); err != nil {
		return "", "", err
	}
	return sr.Value, sr.ContentType, nil
}

// DecodePFXToLeafCert decodes a base64-encoded PFX and returns the raw PFX
// bytes and the leaf x509 certificate.
func DecodePFXToLeafCert(b64pfx string) (pfxBytes []byte, cert *x509.Certificate, err error) {
	pfxBytes, err = base64.StdEncoding.DecodeString(b64pfx)
	if err != nil {
		return nil, nil, fmt.Errorf("base64 decode: %w", err)
	}

	_, cert, err = pkcs12.Decode(pfxBytes, "")
	if err != nil {
		// Return the raw bytes even on parse failure so caller can still save the PFX.
		return pfxBytes, nil, fmt.Errorf("pkcs12 decode: %w", err)
	}
	return pfxBytes, cert, nil
}

// CertToPEM wraps DER-encoded certificate bytes in a PEM CERTIFICATE block.
func CertToPEM(derBytes []byte) []byte {
	return pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: derBytes,
	})
}

// ParseDERFromBase64 decodes a raw base64 DER certificate (e.g. from CertResponse.Cer).
func ParseDERFromBase64(b64 string) (*x509.Certificate, error) {
	der, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return nil, fmt.Errorf("base64 decode: %w", err)
	}
	return x509.ParseCertificate(der)
}

// ── HTTP helper ───────────────────────────────────────────────────────────────

func kvGet(token, uri string, out interface{}) error {
	req, err := http.NewRequest(http.MethodGet, uri, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}
	return json.Unmarshal(body, out)
}
