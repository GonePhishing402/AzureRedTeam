###############################################################
# Script Name: Get-AzureStorageAccessTokenFromRefreshToken.ps1
#
# Description:
# This script exchanges an existing Azure AD / Microsoft Entra
# OAuth refresh token for a new access token scoped for
# Azure Storage.
#
# The script sends a POST request to the Microsoft identity
# platform token endpoint and retrieves a new access token
# that can be used to authenticate to Azure Storage REST APIs.
#
# This is useful for:
# - Automating Azure Storage API authentication
# - Token research and security testing
# - Understanding OAuth token exchange workflows in Azure AD
#
# Requirements:
# - Valid Azure AD refresh token
# - Azure AD Application (Client ID)
# - Appropriate Azure Storage permissions
###############################################################


###############################################################
# VARIABLES
###############################################################

# Azure AD Tenant ID
# Use "common" for multi-tenant authentication
$TenantID = "common"

# Azure AD Application (Client) ID
$ClientID = "<YOUR_CLIENT_ID>"

# OAuth scope for Azure Storage
# ".default" tells Azure AD to issue a token with all
# permissions granted to the application for Azure Storage
$Scope = "https://storage.azure.com/.default"

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
$StorageAccessToken = $TokenResponse.access_token


###############################################################
# OUTPUT RESULTS
###############################################################

# Display the entire token response (includes expiration, scope, etc.)
$TokenResponse

# If only the raw access token is needed, output this variable
$StorageAccessToken
