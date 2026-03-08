###############################################################
# Script Name: Get-AzureKeyVaultAccessTokenFromRefreshToken.ps1
#
# Description:
# This script exchanges an existing Azure AD OAuth refresh token
# for a new access token scoped for Azure Key Vault.
#
# The script sends a POST request to the Microsoft identity
# platform token endpoint and retrieves a new access token that
# can be used to authenticate to the Azure Key Vault REST API.
#
# This is useful for:
# - Automating Key Vault API authentication
# - Token research and security testing
# - Understanding OAuth token exchange workflows in Azure AD
#
# Requirements:
# - Valid Azure AD refresh token
# - Azure AD Application (Client ID)
# - Appropriate permissions for Key Vault access
###############################################################


###############################################################
# VARIABLES
###############################################################

# Azure AD Tenant ID
# Use "common" for multi-tenant authentication
$TenantID = "common"

# Azure AD Application (Client) ID
$ClientID = "<YOUR_CLIENT_ID>"

# OAuth scope for Azure Key Vault
# ".default" tells Azure AD to issue a token with all
# permissions granted to the application for Key Vault
$Scope = "https://vault.azure.net/.default"

# Refresh token previously obtained during authentication
$RefreshToken = "<YOUR_REFRESH_TOKEN>"

# Azure AD OAuth token endpoint
$TokenEndpoint = "https://login.microsoftonline.com/$TenantID/oauth2/v2.0/token"


###############################################################
# BUILD TOKEN REQUEST BODY
###############################################################

# Construct the body used to request a new access token
# using the OAuth refresh token grant
$Body = @{
    client_id     = $ClientID
    scope         = $Scope
    refresh_token = $RefreshToken
    grant_type    = "refresh_token"
}


###############################################################
# REQUEST NEW ACCESS TOKEN FROM AZURE AD
###############################################################

# Send the OAuth request to Azure AD
# Azure validates the refresh token and returns a new access token
$TokenResponse = Invoke-RestMethod `
    -Method POST `
    -Uri $TokenEndpoint `
    -ContentType "application/x-www-form-urlencoded" `
    -Body $Body


###############################################################
# STORE ACCESS TOKEN
###############################################################

# Extract the access token from the response
$KeyVaultAccessToken = $TokenResponse.access_token


###############################################################
# OUTPUT RESULTS
###############################################################

# Display the entire token response (includes expiration, scope, etc.)
$TokenResponse

# If only the raw access token is needed, output this variable
$KeyVaultAccessToken
