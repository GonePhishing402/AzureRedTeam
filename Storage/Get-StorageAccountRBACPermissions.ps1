###############################################################
# Script Name: Get-StorageAccountRBACPermissions.ps1
#
# Description:
# This script queries the Azure Resource Manager (ARM) REST API
# to retrieve RBAC permission assignments configured on an
# Azure Storage Account.
#
# The script:
# 1. Defines the subscription, resource group, and storage account
# 2. Builds the Azure Management API permissions endpoint
# 3. Sends an authenticated REST request using a bearer token
# 4. Retrieves the RBAC permissions assigned to the storage account
# 5. Displays the returned permission information
#
# This is useful for:
# - Enumerating RBAC permissions on Azure Storage resources
# - Security auditing and investigation
# - Token-based Azure API interaction
#
# Requirements:
# - A valid Azure Management API access token
# - Permission to query the storage account resource
###############################################################


###############################################################
# VARIABLES (FILL THESE VALUES)
###############################################################

# Azure subscription ID containing the storage account
$SubscriptionID = ""

# Resource group where the storage account is located
$ResourceGroupName = ""

# Azure Storage account name
$StorageAccountName = ""

# Azure Management API access token
$AccessToken = ""

# Azure Management API base endpoint
$AzureManagementEndpoint = "https://management.azure.com"

# API version used for the permissions endpoint
$ApiVersion = "2022-04-01"


###############################################################
# BUILD AZURE MANAGEMENT API URI
###############################################################

# Construct the REST API endpoint used to retrieve
# authorization permissions for the storage account
$URI = "$AzureManagementEndpoint/subscriptions/$SubscriptionID/resourceGroups/$ResourceGroupName/providers/Microsoft.Storage/storageAccounts/$StorageAccountName/providers/Microsoft.Authorization/permissions?api-version=$ApiVersion"


###############################################################
# BUILD REST REQUEST PARAMETERS
###############################################################

# Prepare request parameters for the REST API call
$RequestParams = @{
    Method = "GET"
    Uri    = $URI
    Headers = @{
        # Azure Management API access token used for authentication
        Authorization = "Bearer $AccessToken"
    }
}


###############################################################
# SEND REST API REQUEST
###############################################################

# Invoke the REST API request to retrieve storage account permissions
$Permissions = (Invoke-RestMethod @RequestParams).value


###############################################################
# DISPLAY RESULTS
###############################################################

# Display all permission properties in a readable format
$Permissions | Format-List *
