package main

import (
	"fmt"
	"path/filepath"
	"strings"
)

// RunRecon is the orchestrator called by the GUI in a background goroutine.
// All output is sent to logCh as human-readable lines.
func RunRecon(cfg ReconConfig, logCh chan<- string) {

	// ── Phase 1 — Token Acquisition ──────────────────────────────
	logBanner(logCh, "PHASE 1 — TOKEN ACQUISITION")

	logInfo(logCh, "Requesting ARM access token (management.azure.com)...")
	armToken, err := exchangeToken(
		cfg.TenantID, cfg.ClientID, cfg.RefreshToken,
		"https://management.azure.com/.default",
	)
	if err != nil {
		logFail(logCh, "ARM token failed: %v", err)
		return
	}
	logOK(logCh, "ARM token acquired. Preview: %s", tokenPreview(armToken))

	logInfo(logCh, "Requesting Key Vault access token (vault.azure.net)...")
	kvToken, err := exchangeToken(
		cfg.TenantID, cfg.ClientID, cfg.RefreshToken,
		"https://vault.azure.net/.default",
	)
	if err != nil {
		logFail(logCh, "Key Vault token failed: %v", err)
		return
	}
	logOK(logCh, "Key Vault token acquired. Preview: %s", tokenPreview(kvToken))

	// ── Phase 2 — Subscription Discovery ─────────────────────────
	logBanner(logCh, "PHASE 2 — SUBSCRIPTION DISCOVERY")

	sub, err := GetSubscription(armToken)
	if err != nil {
		logFail(logCh, "Subscription discovery failed: %v", err)
		return
	}
	logOK(logCh, "Subscription : %s (%s)", sub.DisplayName, sub.SubscriptionID)
	logOK(logCh, "Tenant       : %s", sub.TenantID)

	// ── Phase 3 — Vault Enumeration ───────────────────────────────
	logBanner(logCh, "PHASE 3 — VAULT ENUMERATION")

	logInfo(logCh, "Enumerating Key Vaults...")
	vaults, err := ListVaults(armToken, sub.SubscriptionID, cfg.VaultName)
	if err != nil {
		logFail(logCh, "Vault enumeration failed: %v", err)
		return
	}
	if len(vaults) == 0 {
		logWarn(logCh, "No Key Vaults found.")
		return
	}
	logOK(logCh, "Found %d vault(s):", len(vaults))
	for _, v := range vaults {
		logCh <- fmt.Sprintf("    Vault Name       : %s", v.Name)
		logCh <- fmt.Sprintf("    Resource Group   : %s", ResourceGroupFromID(v.ID))
		logCh <- fmt.Sprintf("    Location         : %s", v.Location)
		logCh <- fmt.Sprintf("    Vault URI        : %s", v.Properties.VaultURI)
		logCh <- fmt.Sprintf("    Soft Delete      : %s", BoolStr(v.Properties.EnableSoftDelete))
		logCh <- fmt.Sprintf("    Purge Protection : %s", BoolStr(v.Properties.EnablePurgeProtection))
		logCh <- ""
	}

	// Ensure output directory exists before writing any files.
	if err := EnsureDir(cfg.OutputPath); err != nil {
		logFail(logCh, "Cannot create output directory: %v", err)
		return
	}
	logOK(logCh, "Output directory: %s", cfg.OutputPath)

	// ── Per-Vault Operations ──────────────────────────────────────
	for _, vault := range vaults {
		vaultName     := vault.Name
		vaultURI      := strings.TrimRight(vault.Properties.VaultURI, "/")
		resourceGroup := ResourceGroupFromID(vault.ID)

		logCh <- ""
		logCh <- strings.Repeat("=", 63)
		logCh <- "  VAULT: " + vaultName
		logCh <- strings.Repeat("=", 63)

		// ── Phase 4 — RBAC ──────────────────────────────────────
		logBanner(logCh, "PHASE 4 — RBAC PERMISSIONS: "+vaultName)

		perms, err := GetRBACPermissions(armToken, sub.SubscriptionID, resourceGroup, vaultName)
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

		// ── Phase 5 — Secrets ────────────────────────────────────
		logBanner(logCh, "PHASE 5 — SECRETS: "+vaultName)

		secretNames, err := ListSecretNames(kvToken, vaultURI, cfg.KVAPIVersion)
		if err != nil {
			logFail(logCh, "Secret enumeration failed: %v", err)
		} else if len(secretNames) == 0 {
			logWarn(logCh, "No secrets found.")
		} else {
			logOK(logCh, "Found %d secret(s):", len(secretNames))

			var secretLines []string
			secretLines = append(secretLines,
				"# Key Vault : "+vaultName,
				"# Retrieved : "+nowString(),
				"",
			)

			for _, name := range secretNames {
				sr, err := GetSecret(kvToken, vaultURI, name, cfg.KVAPIVersion)
				if err != nil {
					logFail(logCh, "Could not retrieve '%s': %v", name, err)
					secretLines = append(secretLines,
						"Secret Name  : "+name,
						"Value        : [RETRIEVAL FAILED - "+err.Error()+"]",
						"",
					)
					continue
				}
				ct := sr.ContentType
				if ct == "" {
					ct = "N/A"
				}
				logCh <- fmt.Sprintf("    Secret Name  : %s", name)
				logCh <- fmt.Sprintf("    Value        : %s", sr.Value)
				logCh <- fmt.Sprintf("    Enabled      : %v", sr.Attributes.Enabled)
				logCh <- fmt.Sprintf("    Expires      : %s", FormatExpiry(sr.Attributes.Exp))
				logCh <- fmt.Sprintf("    Content Type : %s", ct)
				logCh <- ""

				secretLines = append(secretLines,
					"Secret Name  : "+name,
					"Value        : "+sr.Value,
					fmt.Sprintf("Enabled      : %v", sr.Attributes.Enabled),
					"Expires      : "+FormatExpiry(sr.Attributes.Exp),
					"Content Type : "+ct,
					"",
				)
			}

			outFile := filepath.Join(cfg.OutputPath, vaultName+"-secrets.txt")
			if err := WriteTextFile(outFile, strings.Join(secretLines, "\n")); err != nil {
				logFail(logCh, "Could not write secrets file: %v", err)
			} else {
				logOK(logCh, "Secrets saved to: %s", outFile)
			}
		}

		// ── Phase 6 — Certificates ───────────────────────────────
		logBanner(logCh, "PHASE 6 — CERTIFICATES: "+vaultName)

		certNames, err := ListCertNames(kvToken, vaultURI, cfg.KVAPIVersion)
		if err != nil {
			logFail(logCh, "Certificate enumeration failed: %v", err)
			continue
		}
		if len(certNames) == 0 {
			logWarn(logCh, "No certificates found.")
			continue
		}
		logOK(logCh, "Found %d certificate(s):", len(certNames))

		for _, certName := range certNames {
			meta, err := GetCert(kvToken, vaultURI, certName, cfg.KVAPIVersion)
			if err != nil {
				logFail(logCh, "Could not retrieve cert metadata '%s': %v", certName, err)
				continue
			}

			logCh <- fmt.Sprintf("    Certificate  : %s", certName)
			logCh <- fmt.Sprintf("    Thumbprint   : %s", meta.X5T)
			logCh <- fmt.Sprintf("    Enabled      : %v", meta.Attributes.Enabled)
			logCh <- fmt.Sprintf("    Valid From   : %s", FormatExpiry(meta.Attributes.Nbf))
			logCh <- fmt.Sprintf("    Expires      : %s", FormatExpiry(meta.Attributes.Exp))
			logCh <- fmt.Sprintf("    Content Type : %s", meta.Policy.SecretProperties.ContentType)

			// Try to export via secrets endpoint (includes private key when stored in KV).
			bundle, contentType, err := GetCertBundle(kvToken, vaultURI, certName, cfg.KVAPIVersion)
			if err != nil {
				logFail(logCh, "Private key export failed (secrets endpoint): %v", err)
				// Fallback: export public cert only from metadata .cer field (base64 DER).
				if meta.Cer != "" {
					cert, parseErr := ParseDERFromBase64(meta.Cer)
					if parseErr == nil {
						pemBytes := CertToPEM(cert.Raw)
						pemPath := filepath.Join(cfg.OutputPath,
							vaultName+"-"+certName+"-public-only.pem")
						if writeErr := WriteBinaryFile(pemPath, pemBytes); writeErr == nil {
							logWarn(logCh, "Public cert (no private key) saved: %s", pemPath)
						} else {
							logFail(logCh, "Public cert save failed: %v", writeErr)
						}
					}
				}
				logCh <- ""
				continue
			}

			pfxPath := filepath.Join(cfg.OutputPath, vaultName+"-"+certName+"-export.pfx")
			pemPath := filepath.Join(cfg.OutputPath, vaultName+"-"+certName+".pem")

			switch contentType {
			case "application/x-pkcs12", "":
				// Base64-encoded PFX — decode, save raw PFX, extract public cert as PEM.
				pfxBytes, leafCert, decodeErr := DecodePFXToLeafCert(bundle)
				if decodeErr != nil {
					logFail(logCh, "PFX decode failed: %v", decodeErr)
					// Still attempt to save the raw bytes if we got them back.
					if len(pfxBytes) > 0 {
						if writeErr := WriteBinaryFile(pfxPath, pfxBytes); writeErr == nil {
							logOK(logCh, "PFX saved (raw, cert parse failed): %s", pfxPath)
						}
					}
				} else {
					if writeErr := WriteBinaryFile(pfxPath, pfxBytes); writeErr != nil {
						logFail(logCh, "PFX save failed: %v", writeErr)
					} else {
						logOK(logCh, "PFX saved : %s", pfxPath)
					}
					if leafCert != nil {
						pemBytes := CertToPEM(leafCert.Raw)
						if writeErr := WriteBinaryFile(pemPath, pemBytes); writeErr != nil {
							logFail(logCh, "PEM save failed: %v", writeErr)
						} else {
							logOK(logCh, "PEM saved : %s", pemPath)
						}
					}
				}

			case "application/x-pem-file":
				// Already PEM-encoded bundle (cert + key in PEM format).
				if writeErr := WriteTextFile(pemPath, bundle); writeErr != nil {
					logFail(logCh, "PEM save failed: %v", writeErr)
				} else {
					logOK(logCh, "PEM saved : %s", pemPath)
				}

			default:
				// Unknown content type — save raw base64 for manual inspection.
				rawPath := filepath.Join(cfg.OutputPath, vaultName+"-"+certName+"-raw.b64")
				logWarn(logCh, "Unknown content type '%s' — saving raw base64.", contentType)
				if writeErr := WriteTextFile(rawPath, bundle); writeErr != nil {
					logFail(logCh, "Raw save failed: %v", writeErr)
				} else {
					logOK(logCh, "Raw saved : %s", rawPath)
				}
			}

			logCh <- ""
		}
	}

	// ── Done ─────────────────────────────────────────────────────
	logBanner(logCh, "RECON COMPLETE — Output: "+cfg.OutputPath)
}
