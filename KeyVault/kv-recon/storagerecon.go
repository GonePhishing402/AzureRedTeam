package main

import (
	"fmt"
	"path/filepath"
	"strings"
)

// StorageReconConfig holds all parameters for a Storage Recon run.
type StorageReconConfig struct {
	RefreshToken       string
	ClientID           string
	TenantID           string
	StorageAccountName string // optional — blank enumerates all
	OutputPath         string
	ListBlobs          bool // when true, enumerate blobs inside each container
}

// RunStorageRecon is the main orchestrator for Azure Storage enumeration.
// All output is written to logCh as human-readable lines (same format as KV recon).
func RunStorageRecon(cfg StorageReconConfig, logCh chan<- string) {

	// ── Phase 1 — Token Acquisition ──────────────────────────────
	logBanner(logCh, "PHASE 1 — TOKEN ACQUISITION")

	logInfo(logCh, "Requesting ARM access token (management.azure.com)...")
	armToken, _, _, err := exchangeToken(
		cfg.TenantID, cfg.ClientID, cfg.RefreshToken,
		"https://management.azure.com/.default",
	)
	if err != nil {
		logFail(logCh, "ARM token failed: %v", err)
		return
	}
	logOK(logCh, "ARM token acquired. Preview: %s", tokenPreview(armToken))

	logInfo(logCh, "Requesting Storage access token (storage.azure.com)...")
	storageToken, _, _, err := exchangeToken(
		cfg.TenantID, cfg.ClientID, cfg.RefreshToken,
		"https://storage.azure.com/.default",
	)
	if err != nil {
		logFail(logCh, "Storage token failed: %v", err)
		return
	}
	logOK(logCh, "Storage token acquired. Preview: %s", tokenPreview(storageToken))

	// ── Phase 2 — Subscription Discovery ─────────────────────────
	logBanner(logCh, "PHASE 2 — SUBSCRIPTION DISCOVERY")

	sub, err := GetSubscription(armToken)
	if err != nil {
		logFail(logCh, "Subscription discovery failed: %v", err)
		return
	}
	logOK(logCh, "Subscription : %s (%s)", sub.DisplayName, sub.SubscriptionID)
	logOK(logCh, "Tenant       : %s", sub.TenantID)

	// ── Phase 3 — Storage Account Enumeration ────────────────────
	logBanner(logCh, "PHASE 3 — STORAGE ACCOUNT ENUMERATION")

	logInfo(logCh, "Enumerating Storage Accounts...")
	accounts, err := ListStorageAccounts(armToken, sub.SubscriptionID, cfg.StorageAccountName)
	if err != nil {
		logFail(logCh, "Storage account enumeration failed: %v", err)
		return
	}
	if len(accounts) == 0 {
		logWarn(logCh, "No Storage Accounts found.")
		return
	}
	logOK(logCh, "Found %d storage account(s):", len(accounts))
	for _, a := range accounts {
		blobEndpoint := a.Properties.PrimaryEndpoints.Blob
		if blobEndpoint == "" {
			blobEndpoint = fmt.Sprintf("https://%s.blob.core.windows.net/", a.Name)
		}
		logCh <- fmt.Sprintf("    Account Name      : %s", a.Name)
		logCh <- fmt.Sprintf("    Resource Group    : %s", ResourceGroupFromID(a.ID))
		logCh <- fmt.Sprintf("    Location          : %s", a.Location)
		logCh <- fmt.Sprintf("    Kind              : %s", a.Kind)
		logCh <- fmt.Sprintf("    Blob Endpoint     : %s", blobEndpoint)
		logCh <- fmt.Sprintf("    Public Blob Access: %s", BoolStr(a.Properties.AllowBlobPublicAccess))
		logCh <- fmt.Sprintf("    HTTPS Only        : %s", BoolStr(a.Properties.HTTPSOnly))
		logCh <- ""
	}

	// Ensure output directory exists.
	if err := EnsureDir(cfg.OutputPath); err != nil {
		logFail(logCh, "Cannot create output directory: %v", err)
		return
	}
	logOK(logCh, "Output directory: %s", cfg.OutputPath)

	// ── Per-Account Operations ────────────────────────────────────
	for _, account := range accounts {
		accountName   := account.Name
		resourceGroup := ResourceGroupFromID(account.ID)

		logCh <- ""
		logCh <- strings.Repeat("=", 63)
		logCh <- "  ACCOUNT: " + accountName
		logCh <- strings.Repeat("=", 63)

		// ── Phase 4 — RBAC ──────────────────────────────────────
		logBanner(logCh, "PHASE 4 — RBAC PERMISSIONS: "+accountName)

		perms, err := GetStorageRBACPermissions(armToken, sub.SubscriptionID, resourceGroup, accountName)
		if err != nil {
			logFail(logCh, "RBAC check failed: %v", err)
		} else if len(perms) == 0 {
			logWarn(logCh, "No RBAC permissions returned.")
		} else {
			for _, p := range perms {
				logCh <- "    Actions        :"
				for _, a := range p.Actions {
					logCh <- "      - " + a
				}
				logCh <- "    NotActions     :"
				for _, a := range p.NotActions {
					logCh <- "      - " + a
				}
				logCh <- "    DataActions    :"
				for _, a := range p.DataActions {
					logCh <- "      - " + a
				}
				logCh <- "    NotDataActions :"
				for _, a := range p.NotDataActions {
					logCh <- "      - " + a
				}
			}
		}

		// ── Phase 5 — Container Enumeration ──────────────────────
		logBanner(logCh, "PHASE 5 — CONTAINERS: "+accountName)

		containers, err := ListContainersViaARM(armToken, sub.SubscriptionID, resourceGroup, accountName)
		if err != nil {
			logFail(logCh, "Container enumeration failed: %v", err)
			continue
		}
		if len(containers) == 0 {
			logWarn(logCh, "No containers found (account is empty or access denied).")
			continue
		}
		logOK(logCh, "Found %d container(s):", len(containers))

		var outputLines []string
		outputLines = append(outputLines,
			"# Storage Account : "+accountName,
			"# Retrieved       : "+nowString(),
			"",
		)

		for _, c := range containers {
			publicAccess := c.Properties.PublicAccess
			if publicAccess == "" {
				publicAccess = "None (private)"
			}
			logCh <- fmt.Sprintf("    Container    : %-40s  [access: %s]", c.Name, publicAccess)
			outputLines = append(outputLines,
				"Container    : "+c.Name,
				"Public Access: "+publicAccess,
				"Last Modified: "+c.Properties.LastModifiedTime,
			)

			if !cfg.ListBlobs {
				outputLines = append(outputLines, "")
				continue
			}

			// ── Phase 6 — Blob Enumeration ───────────────────────
			logInfo(logCh, "  Listing blobs in '%s'...", c.Name)
			blobs, err := ListBlobs(storageToken, accountName, c.Name)
			if err != nil {
				logFail(logCh, "  Blob listing failed for '%s': %v", c.Name, err)
				outputLines = append(outputLines, "  Blobs: [listing failed: "+err.Error()+"]", "")
				continue
			}
			if len(blobs) == 0 {
				logWarn(logCh, "  No blobs found in '%s'.", c.Name)
				outputLines = append(outputLines, "  Blobs: [empty]", "")
				continue
			}
			logOK(logCh, "  Found %d blob(s) in '%s':", len(blobs), c.Name)
			outputLines = append(outputLines, fmt.Sprintf("  Blobs (%d):", len(blobs)))
			for _, b := range blobs {
				logCh <- fmt.Sprintf("    Blob : %-60s  [%d bytes]  %s",
					b.Name, b.ContentLength, b.ContentType)
				outputLines = append(outputLines,
					fmt.Sprintf("    - %-60s  %d bytes  %s  modified:%s",
						b.Name, b.ContentLength, b.ContentType, b.LastModified),
				)
			}
			outputLines = append(outputLines, "")
		}

		outFile := filepath.Join(cfg.OutputPath, accountName+"-storage.txt")
		if err := WriteTextFile(outFile, strings.Join(outputLines, "\n")); err != nil {
			logFail(logCh, "Could not write output file: %v", err)
		} else {
			logOK(logCh, "Results saved to: %s", outFile)
		}
	}

	logBanner(logCh, "STORAGE RECON COMPLETE — Output: "+cfg.OutputPath)
}
