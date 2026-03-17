###############################################################
# Script Name: Invoke-StorageRecon.ps1
#
# Description:
# Performs a full Azure Storage reconnaissance run using a
# refresh token. Mirrors the azprowl Go tool's StorageRecon
# flow across 6 phases:
#
#   Phase 1  — Token Acquisition (ARM + Storage bearer tokens)
#   Phase 2  — Subscription Discovery
#   Phase 3  — Storage Account Enumeration
#   Phase 4  — RBAC Permissions (per account)
#   Phase 5  — Container Enumeration via ARM management plane
#   Phase 6  — Blob Enumeration (per container)
#
# Containers are listed via the ARM management API so that
# Storage Blob Data Reader is NOT required — only management-
# plane Reader access is needed.  Blob listing uses the
# data-plane Storage bearer token.
#
# Output:
#   One text file per storage account saved under $OutputPath.
#   Format: {OutputPath}/{accountName}-storage.txt
#
# Requirements:
#   - Valid Azure AD / Microsoft Entra refresh token
#   - Azure AD Application (Client) ID
#   - Tenant ID (GUID or "common")
#   - PowerShell 5.1+ or PowerShell 7+
#
# Parameters:
#   -RefreshToken       (required) OAuth refresh token
#   -ClientID           (required) Azure AD Application client ID
#   -TenantID           (required) Azure AD Tenant ID or "common"
#   -StorageAccountName (optional) Limit recon to one account
#   -OutputPath         (optional) Output directory (default: ./StorageReconOutput)
###############################################################

[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [string]$RefreshToken,

    [Parameter(Mandatory = $true)]
    [string]$ClientID,

    [Parameter(Mandatory = $true)]
    [string]$TenantID,

    [Parameter(Mandatory = $false)]
    [string]$StorageAccountName = "",

    [Parameter(Mandatory = $false)]
    [string]$OutputPath = "./StorageReconOutput"
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

###############################################################
# LOG HELPERS
###############################################################

function Write-Banner {
    param([string]$Title)
    $line = "###############################################################"
    Write-Host ""
    Write-Host $line -ForegroundColor Cyan
    Write-Host "# $Title" -ForegroundColor Cyan
    Write-Host $line -ForegroundColor Cyan
}

function Write-OK   { param([string]$Msg) Write-Host "[+] $Msg" -ForegroundColor Green  }
function Write-Info { param([string]$Msg) Write-Host "[*] $Msg" -ForegroundColor White  }
function Write-Warn { param([string]$Msg) Write-Host "[!] $Msg" -ForegroundColor Yellow }
function Write-Fail { param([string]$Msg) Write-Host "[-] $Msg" -ForegroundColor Red    }

function Get-TokenPreview {
    param([string]$Token)
    if ($Token.Length -le 50) { return $Token }
    return $Token.Substring(0, 50) + "..."
}

function Get-NowString {
    return (Get-Date -Format "yyyy-MM-dd HH:mm:ss UTC")
}

###############################################################
# HELPER — TOKEN EXCHANGE
# Exchanges a refresh token for an access token scoped to
# the given resource. Returns the full response object so
# callers can also access refresh_token, expires_in, etc.
###############################################################

function Invoke-TokenExchange {
    param(
        [string]$TenantID,
        [string]$ClientID,
        [string]$RefreshToken,
        [string]$Scope
    )

    $endpoint = "https://login.microsoftonline.com/$TenantID/oauth2/v2.0/token"

    $body = @{
        client_id     = $ClientID
        scope         = $Scope
        refresh_token = $RefreshToken
        grant_type    = "refresh_token"
    }

    $response = Invoke-RestMethod `
        -Method POST `
        -Uri $endpoint `
        -ContentType "application/x-www-form-urlencoded" `
        -Body $body

    if (-not $response.access_token) {
        throw "Token exchange failed for scope '$Scope': no access_token in response."
    }

    return $response
}

###############################################################
# HELPER — ARM PAGINATED GET
# Follows .nextLink pagination and returns ALL .value items.
###############################################################

function Invoke-ARMGetPaged {
    param(
        [string]$Token,
        [string]$Uri
    )

    $allItems = @()
    $currentUri = $Uri

    while ($currentUri) {
        $response = Invoke-RestMethod `
            -Method GET `
            -Uri $currentUri `
            -Headers @{ Authorization = "Bearer $Token" }

        if ($response.value) {
            $allItems += $response.value
        }

        $currentUri = if ($response.PSObject.Properties['nextLink']) { $response.nextLink } else { $null }
    }

    return $allItems
}

###############################################################
# HELPER — EXTRACT RESOURCE GROUP FROM ARM RESOURCE ID
# e.g. /subscriptions/…/resourceGroups/my-rg/providers/…
#      → "my-rg"
###############################################################

function Get-ResourceGroupFromID {
    param([string]$ResourceID)

    $parts = $ResourceID -split "/"
    for ($i = 0; $i -lt $parts.Length - 1; $i++) {
        if ($parts[$i] -ieq "resourceGroups") {
            return $parts[$i + 1]
        }
    }
    return ""
}

###############################################################
# HELPER — BLOB DATA-PLANE GET (XML response)
# Uses the Storage bearer token and required x-ms-version
# header for the Azure Blob Storage REST API.
###############################################################

function Invoke-BlobGet {
    param(
        [string]$StorageToken,
        [string]$Uri
    )

    $response = Invoke-RestMethod `
        -Method GET `
        -Uri $Uri `
        -Headers @{
            Authorization = "Bearer $StorageToken"
            "x-ms-version" = "2020-04-08"
        }

    return $response
}

###############################################################
# PHASE 1 — TOKEN ACQUISITION
###############################################################

Write-Banner "PHASE 1 — TOKEN ACQUISITION"

Write-Info "Requesting ARM access token (management.azure.com)..."
try {
    $armResponse    = Invoke-TokenExchange -TenantID $TenantID -ClientID $ClientID `
                          -RefreshToken $RefreshToken `
                          -Scope "https://management.azure.com/.default"
    $armToken       = $armResponse.access_token
    Write-OK "ARM token acquired. Preview: $(Get-TokenPreview $armToken)"
}
catch {
    Write-Fail "ARM token failed: $_"
    exit 1
}

Write-Info "Requesting Storage access token (storage.azure.com)..."
try {
    $storageResponse = Invoke-TokenExchange -TenantID $TenantID -ClientID $ClientID `
                           -RefreshToken $RefreshToken `
                           -Scope "https://storage.azure.com/.default"
    $storageToken    = $storageResponse.access_token
    Write-OK "Storage token acquired. Preview: $(Get-TokenPreview $storageToken)"
}
catch {
    Write-Fail "Storage token failed: $_"
    exit 1
}

###############################################################
# PHASE 2 — SUBSCRIPTION DISCOVERY
###############################################################

Write-Banner "PHASE 2 — SUBSCRIPTION DISCOVERY"

try {
    $subUri   = "https://management.azure.com/subscriptions?api-version=2022-12-01"
    $subResponse = Invoke-RestMethod `
        -Method GET `
        -Uri $subUri `
        -Headers @{ Authorization = "Bearer $armToken" }

    # Use first enabled subscription; fall back to first in list
    $subscription = $subResponse.value | Where-Object { $_.state -ieq "Enabled" } | Select-Object -First 1
    if (-not $subscription) {
        $subscription = $subResponse.value | Select-Object -First 1
    }
    if (-not $subscription) {
        Write-Fail "No subscriptions found."
        exit 1
    }

    $subID       = $subscription.subscriptionId
    $subName     = $subscription.displayName
    $subTenantID = $subscription.tenantId

    Write-OK "Subscription : $subName ($subID)"
    Write-OK "Tenant       : $subTenantID"
}
catch {
    Write-Fail "Subscription discovery failed: $_"
    exit 1
}

###############################################################
# PHASE 3 — STORAGE ACCOUNT ENUMERATION
###############################################################

Write-Banner "PHASE 3 — STORAGE ACCOUNT ENUMERATION"

Write-Info "Enumerating Storage Accounts..."
try {
    $storageUri = "https://management.azure.com/subscriptions/$subID/providers/Microsoft.Storage/storageAccounts?api-version=2023-01-01"
    $accounts   = Invoke-ARMGetPaged -Token $armToken -Uri $storageUri
}
catch {
    Write-Fail "Storage account enumeration failed: $_"
    exit 1
}

# Filter to a specific account if requested
if ($StorageAccountName -ne "") {
    $accounts = $accounts | Where-Object { $_.name -ieq $StorageAccountName }
    if (-not $accounts) {
        Write-Fail "Storage account '$StorageAccountName' not found in subscription."
        exit 1
    }
}

if (-not $accounts -or $accounts.Count -eq 0) {
    Write-Warn "No Storage Accounts found."
    exit 0
}

Write-OK "Found $($accounts.Count) storage account(s):"
foreach ($acct in $accounts) {
    $blobEndpoint = $acct.properties.primaryEndpoints.blob
    if (-not $blobEndpoint) { $blobEndpoint = "https://$($acct.name).blob.core.windows.net/" }

    $publicAccess = if ($null -ne $acct.properties.allowBlobPublicAccess) { $acct.properties.allowBlobPublicAccess.ToString().ToLower() } else { "N/A" }
    $httpsOnly    = if ($null -ne $acct.properties.supportsHttpsTrafficOnly) { $acct.properties.supportsHttpsTrafficOnly.ToString().ToLower() } else { "N/A" }

    Write-Host "    Account Name      : $($acct.name)"
    Write-Host "    Resource Group    : $(Get-ResourceGroupFromID $acct.id)"
    Write-Host "    Location          : $($acct.location)"
    Write-Host "    Kind              : $($acct.kind)"
    Write-Host "    Blob Endpoint     : $blobEndpoint"
    Write-Host "    Public Blob Access: $publicAccess"
    Write-Host "    HTTPS Only        : $httpsOnly"
    Write-Host ""
}

# Ensure output directory exists
if (-not (Test-Path $OutputPath)) {
    New-Item -ItemType Directory -Path $OutputPath -Force | Out-Null
}
Write-OK "Output directory: $(Resolve-Path $OutputPath)"

###############################################################
# PER-ACCOUNT OPERATIONS
###############################################################

foreach ($account in $accounts) {
    $accountName   = $account.name
    $resourceGroup = Get-ResourceGroupFromID $account.id

    Write-Host ""
    Write-Host ("=" * 63) -ForegroundColor Cyan
    Write-Host "  ACCOUNT: $accountName" -ForegroundColor Cyan
    Write-Host ("=" * 63) -ForegroundColor Cyan

    $outputLines = @(
        "# Storage Account : $accountName",
        "# Retrieved       : $(Get-NowString)",
        ""
    )

    ##########################################################
    # PHASE 4 — RBAC PERMISSIONS
    ##########################################################

    Write-Banner "PHASE 4 — RBAC PERMISSIONS: $accountName"

    try {
        $rbacUri = "https://management.azure.com/subscriptions/$subID/resourceGroups/$resourceGroup" +
                   "/providers/Microsoft.Storage/storageAccounts/$accountName" +
                   "/providers/Microsoft.Authorization/permissions?api-version=2022-04-01"

        $rbacResponse = Invoke-RestMethod `
            -Method GET `
            -Uri $rbacUri `
            -Headers @{ Authorization = "Bearer $armToken" }

        $perms = $rbacResponse.value
        if (-not $perms -or $perms.Count -eq 0) {
            Write-Warn "No RBAC permissions returned."
        }
        else {
            foreach ($perm in $perms) {
                Write-Host "    Actions        :"
                foreach ($a in $perm.actions)        { Write-Host "      - $a" }
                Write-Host "    NotActions     :"
                foreach ($a in $perm.notActions)     { Write-Host "      - $a" }
                Write-Host "    DataActions    :"
                foreach ($a in $perm.dataActions)    { Write-Host "      - $a" }
                Write-Host "    NotDataActions :"
                foreach ($a in $perm.notDataActions) { Write-Host "      - $a" }
            }
        }
    }
    catch {
        Write-Fail "RBAC check failed: $_"
    }

    ##########################################################
    # PHASE 5 — CONTAINER ENUMERATION (ARM management plane)
    ##########################################################

    Write-Banner "PHASE 5 — CONTAINERS: $accountName"

    try {
        $containerUri = "https://management.azure.com/subscriptions/$subID/resourceGroups/$resourceGroup" +
                        "/providers/Microsoft.Storage/storageAccounts/$accountName" +
                        "/blobServices/default/containers?api-version=2023-01-01"

        $containers = Invoke-ARMGetPaged -Token $armToken -Uri $containerUri
    }
    catch {
        Write-Fail "Container enumeration failed: $_"
        continue
    }

    if (-not $containers -or $containers.Count -eq 0) {
        Write-Warn "No containers found (account is empty or access denied)."
        continue
    }

    Write-OK "Found $($containers.Count) container(s):"

    foreach ($container in $containers) {
        $cName        = $container.name
        $publicAccess = $container.properties.publicAccess
        if (-not $publicAccess) { $publicAccess = "None (private)" }
        $lastModified = $container.properties.lastModifiedTime

        Write-Host ("    Container    : {0,-40}  [access: {1}]" -f $cName, $publicAccess)

        $outputLines += @(
            "Container    : $cName",
            "Public Access: $publicAccess",
            "Last Modified: $lastModified"
        )

        ######################################################
        # PHASE 6 — BLOB ENUMERATION (data plane, XML)
        ######################################################

        Write-Info "  Listing blobs in '$cName'..."
        try {
            $baseUri    = "https://$accountName.blob.core.windows.net/$cName`?restype=container&comp=list"
            $allBlobs   = @()
            $marker     = ""

            do {
                $blobUri = $baseUri
                if ($marker -ne "") { $blobUri += "&marker=$marker" }

                $xmlResponse = Invoke-BlobGet -StorageToken $storageToken -Uri $blobUri

                # Collect blobs from this page
                if (-not $xmlResponse.PSObject.Properties['EnumerationResults']) {
                    throw "Unexpected response format — EnumerationResults not found."
                }
                if ($xmlResponse.EnumerationResults.Blobs.PSObject.Properties['Blob']) {
                    $allBlobs += $xmlResponse.EnumerationResults.Blobs.Blob
                }

                $marker = if ($xmlResponse.EnumerationResults.PSObject.Properties['NextMarker']) { $xmlResponse.EnumerationResults.NextMarker } else { $null }
            } while ($marker -ne "" -and $null -ne $marker)

            if ($allBlobs.Count -eq 0) {
                Write-Warn "  No blobs found in '$cName'."
                $outputLines += "  Blobs: [empty]"
            }
            else {
                Write-OK "  Found $($allBlobs.Count) blob(s) in '$cName':"
                $outputLines += "  Blobs ($($allBlobs.Count)):"

                foreach ($blob in $allBlobs) {
                    $bName         = $blob.Name
                    $bSize         = $blob.Properties."Content-Length"
                    $bContentType  = $blob.Properties."Content-Type"
                    $bLastModified = $blob.Properties."Last-Modified"

                    Write-Host ("    Blob : {0,-60}  [{1} bytes]  {2}" -f $bName, $bSize, $bContentType)
                    $outputLines += ("    - {0,-60}  {1} bytes  {2}  modified:{3}" -f $bName, $bSize, $bContentType, $bLastModified)
                }
            }
        }
        catch {
            Write-Fail "  Blob listing failed for '$cName': $_"
            $outputLines += "  Blobs: [listing failed: $_]"
        }

        $outputLines += ""
    }

    ##########################################################
    # WRITE PER-ACCOUNT OUTPUT FILE
    ##########################################################

    $outFile = Join-Path $OutputPath "$accountName-storage.txt"
    try {
        $outputLines | Set-Content -Path $outFile -Encoding UTF8
        Write-OK "Results saved to: $outFile"
    }
    catch {
        Write-Fail "Could not write output file: $_"
    }
}

###############################################################
# DONE
###############################################################

Write-Banner "STORAGE RECON COMPLETE — Output: $OutputPath"
