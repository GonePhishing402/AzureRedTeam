###############################################################
# Script Name: Get-KeyVaultRBACPermissions.ps1
#
# Description:
# This script queries Azure Resource Manager (ARM) to retrieve
# RBAC permission assignments configured for an Azure Key Vault.
#
# The script:
# 1. Retrieves the current Key Vault in the active Azure context
# 2. Identifies the subscription and resource group
# 3. Builds the ARM REST API request for Key Vault permissions
# 4. Uses an Azure management access token to authenticate
# 5. Returns the RBAC permissions assigned to the Key Vault
#
# This is useful for:
# - Enumerating Key Vault access permissions
# - Security auditing and investigation
# - Token-based API access testing
#
# Requirements:
# - Logged into Azure or valid Azure context
# - Azure management API access token ($GISAppMgmtToken)
# - Az PowerShell module installed
###############################################################


###############################################################
# VARIABLES
###############################################################

# Azure Management API version for permissions endpoint
$ApiVersion = "2022-04-01"

# Azure Management API base URL
$AzureManagementEndpoint = "https://management.azure.com"


###############################################################
# GET KEY VAULT INFORMATION FROM CURRENT AZURE CONTEXT
###############################################################

# Retrieve Key Vault object from the current Azure session
$KeyVault = Get-AzKeyVault

# Get the subscription ID currently in context
$SubscriptionID = (Get-AzSubscription).Id

# Extract resource group name where the Key Vault resides
$ResourceGroupName = $KeyVault.ResourceGroupName

# Extract the Key Vault name
$KeyVaultName = $KeyVault.VaultName


###############################################################
# BUILD AZURE MANAGEMENT REST API URI
###############################################################

# Construct the ARM REST API endpoint used to query
# authorization permissions assigned to the Key Vault
$URI = "$AzureManagementEndpoint/subscriptions/$SubscriptionID/resourceGroups/$ResourceGroupName/providers/Microsoft.KeyVault/vaults/$KeyVaultName/providers/Microsoft.Authorization/permissions?api-version=$ApiVersion"


###############################################################
# BUILD REST REQUEST PARAMETERS
###############################################################

# Prepare the REST request with authentication header
$RequestParams = @{
    Method = "GET"
    Uri    = $URI
    Headers = @{
        # Azure Management access token required for ARM API
        Authorization = "Bearer $GISAppMgmtToken"
    }
}


###############################################################
# SEND REST API REQUEST
###############################################################

# Call the Azure Management API to retrieve permissions
$Permissions = (Invoke-RestMethod @RequestParams).value


###############################################################
# DISPLAY PERMISSION RESULTS
###############################################################

# Display all returned permission properties
$Permissions | Format-List *
