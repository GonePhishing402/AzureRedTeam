package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	refreshToken := flag.String("refresh-token", "", "OAuth refresh token (required)")
	clientID     := flag.String("client-id", "", "Azure AD Application (Client) ID (required)")
	tenantID     := flag.String("tenant-id", "", "Azure AD Tenant ID (required)")
	vaultName    := flag.String("vault-name", "", "Target a specific vault (optional — blank enumerates all)")
	outputPath   := flag.String("output", "./KVReconOutput", "Directory for saved output")

	flag.Parse()

	missing := false
	if *refreshToken == "" {
		fmt.Fprintln(os.Stderr, "error: -refresh-token is required")
		missing = true
	}
	if *clientID == "" {
		fmt.Fprintln(os.Stderr, "error: -client-id is required")
		missing = true
	}
	if *tenantID == "" {
		fmt.Fprintln(os.Stderr, "error: -tenant-id is required")
		missing = true
	}
	if missing {
		fmt.Fprintln(os.Stderr, "")
		flag.Usage()
		os.Exit(1)
	}

	cfg := ReconConfig{
		RefreshToken: *refreshToken,
		ClientID:     *clientID,
		TenantID:     *tenantID,
		VaultName:    *vaultName,
		OutputPath:   *outputPath,
		KVAPIVersion: "7.4",
	}

	logCh := make(chan string, 512)
	go func() {
		RunRecon(cfg, logCh)
		close(logCh)
	}()

	for line := range logCh {
		fmt.Println(line)
	}
}

// ReconConfig holds all parameters for the recon run.
type ReconConfig struct {
	RefreshToken string
	ClientID     string
	TenantID     string
	VaultName    string
	OutputPath   string
	KVAPIVersion string
}
