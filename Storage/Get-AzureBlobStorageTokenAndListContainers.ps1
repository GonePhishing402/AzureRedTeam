###############################################################
# Script Name: Get-AzureBlobStorageTokenAndListContainers.ps1
#
# Description:
# This script uses a signed JWT client assertion to request an
# Azure Storage access token from the Microsoft identity platform.
# After obtaining the token, it uses the Azure Blob Storage REST
# API to list the blobs or containers available in a target
# storage account.
#
# The script:
# 1. Generates a signed JWT using the local New-SignedJWT function
# 2. Sends the JWT to Azure AD as a client assertion
# 3. Requests an access token for Azure Storage
# 4. Uses the access token to authenticate to Blob Storage
# 5. Retrieves and displays the storage listing via REST API
#
# This is useful for:
# - Token-based authentication testing for Azure Storage
# - Enumerating blob storage content through the REST API
# - Understanding app-only authentication with client assertions
#
# Requirements:
# - A working New-SignedJWT function
# - Valid Azure AD application client ID
# - Correct tenant ID
# - Proper app permissions to Azure Storage
###############################################################


###############################################################
# VARIABLES
###############################################################

# Azure AD tenant ID
$TenantID = ""

# Azure AD application (client) ID
$ClientID = ""

# Azure Storage resource scope
$StorageScope = "https://storage.azure.com/.default"

# Azure AD token endpoint
$TokenEndpoint = "https://login.microsoftonline.com/$TenantID/oauth2/v2.0/token"

# Target Azure Storage account name
$StorageAccountName = ""

# Blob Storage service endpoint used to list contents
$StorageListUrl = "https://$StorageAccountName.blob.core.windows.net/?comp=list"

# Storage API version
$StorageApiVersion = "2017-11-09"


###############################################################
# GENERATE SIGNED JWT CLIENT ASSERTION
###############################################################

# Generate a signed JWT that will be used as the client assertion
# in the OAuth 2.0 client credentials flow
$signedJWT = New-SignedJWT


###############################################################
# BUILD TOKEN REQUEST HEADERS
###############################################################

# Define headers for the token request to Azure AD
$TokenRequestHeaders = @{
    "Content-Type" = "application/x-www-form-urlencoded"
}


###############################################################
# REQUEST AZURE STORAGE ACCESS TOKEN
###############################################################

# Request an access token from Azure AD using:
# - client_id: the app registration ID
# - client_assertion: signed JWT proving app identity
# - scope: Azure Storage
# - grant_type: client_credentials
$TokenResponse = Invoke-RestMethod `
    -Uri $TokenEndpoint `
    -Method POST `
    -Headers $TokenRequestHeaders `
    -Body ([ordered]@{
        client_id             = $ClientID
        client_assertion      = $signedJWT
        client_assertion_type = "urn:ietf:params:oauth:client-assertion-type:jwt-bearer"
        scope                 = $StorageScope
        grant_type            = "client_credentials"
    })


###############################################################
# EXTRACT ACCESS TOKEN
###############################################################

# Store the Azure Storage access token from the response
$DataAnalyticsAppStorageToken = $TokenResponse.access_token

# Output the token if needed for validation or reuse
$DataAnalyticsAppStorageToken


###############################################################
# BUILD BLOB STORAGE REST API REQUEST
###############################################################

# Prepare the REST API request parameters used to query
# the Blob Storage endpoint with the bearer token
$StorageRequestParams = @{
    Uri    = $StorageListUrl
    Method = "GET"
    Headers = @{
        "Content-Type"    = "application/json"
        "Authorization"   = "Bearer $DataAnalyticsAppStorageToken"
        "x-ms-version"    = $StorageApiVersion
        "accept-encoding" = "gzip, deflate"
    }
}


###############################################################
# QUERY BLOB STORAGE
###############################################################

# Call the Blob Storage REST API and return the results
$StorageResult = Invoke-RestMethod @StorageRequestParams


###############################################################
# DISPLAY RESULTS
###############################################################

# Display the returned storage listing
$StorageResult
