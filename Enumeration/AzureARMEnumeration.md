# Azure ARM PowerShell (Az Module) Enumeration Cheat Sheet

> **Note:** The **Az PowerShell module** is used to enumerate and manage **Azure Resource Manager (ARM)** resources such as subscriptions, resource groups, virtual machines, storage accounts, Key Vaults, networking, and RBAC permissions.  
> For **Microsoft 365 / Entra ID objects** like users, groups, applications, and sign-in logs, use the **Microsoft Graph PowerShell module** instead.

---

## Install and Import

```powershell
# Install Az module
Install-Module Az -Scope CurrentUser

# Import Az module
Import-Module Az
```
## Connect to Azure
```powershell
# Interactive login
Connect-AzAccount

# Device code login
Connect-AzAccount -UseDeviceAuthentication

# Login with specific tenant
Connect-AzAccount -Tenant "<TENANT_ID>"

# Service principal login (client secret)
Connect-AzAccount -ServicePrincipal `
-Tenant "<TENANT_ID>" `
-Credential (Get-Credential)

# Service principal login (certificate)
Connect-AzAccount `
-ServicePrincipal `
-Tenant "<TENANT_ID>" `
-ApplicationId "<APP_ID>" `
-CertificateThumbprint "<CERT_THUMBPRINT>"

# View current login context
Get-AzContext

# List all available contexts
Get-AzContext -ListAvailable

# Logout
Disconnect-AzAccount
```
## Context and Tenant Enumeration
```powershell
# Current context information
Get-AzContext | Format-List *

# List tenants accessible to account
Get-AzTenant

# List subscriptions
Get-AzSubscription

# Set active subscription
Set-AzContext -SubscriptionId "<SUBSCRIPTION_ID>"
```
## Resource Enumeration
```powershell
# List all resources in subscription
Get-AzResource

# Filter resources by type
Get-AzResource | Select Name,ResourceType,Location

# Get specific resource
Get-AzResource -Name "<RESOURCE_NAME>"

# List resources by resource group
Get-AzResource -ResourceGroupName "<RESOURCE_GROUP>"
```
## Resource Group Enumeration
```powershell
# List resource groups
Get-AzResourceGroup

# Specific resource group
Get-AzResourceGroup -Name "<RESOURCE_GROUP>"

# Show resources in resource group
Get-AzResource | Where ResourceGroupName -eq "<RESOURCE_GROUP>"
```
## Virtual Machine Enumeration
```powershell
# List VMs
Get-AzVM

# Detailed VM info
Get-AzVM | Format-List *

# VM status
Get-AzVM -Status

# Specific VM
Get-AzVM -Name "<VM_NAME>" -ResourceGroupName "<RESOURCE_GROUP>"
```
## Storage Account Enumeration
```powershell
# List storage accounts
Get-AzStorageAccount

# Specific storage account
Get-AzStorageAccount -Name "<STORAGE_ACCOUNT>"

# Storage account keys
Get-AzStorageAccountKey `
-ResourceGroupName "<RESOURCE_GROUP>" `
-Name "<STORAGE_ACCOUNT>"
```
## Blob Storage Enumeration
```powershell
# Create storage context
$ctx = (Get-AzStorageAccount -Name "<STORAGE_ACCOUNT>" `
-ResourceGroupName "<RESOURCE_GROUP>").Context

# List containers
Get-AzStorageContainer -Context $ctx

# List blobs
Get-AzStorageBlob -Container "<CONTAINER_NAME>" -Context $ctx
```
## Key Vault Enumeration
```powershell
# List Key Vaults
Get-AzKeyVault

# Specific vault
Get-AzKeyVault -VaultName "<VAULT_NAME>"

# List secrets
Get-AzKeyVaultSecret -VaultName "<VAULT_NAME>"

# List keys
Get-AzKeyVaultKey -VaultName "<VAULT_NAME>"

# List certificates
Get-AzKeyVaultCertificate -VaultName "<VAULT_NAME>"
```
## Networking Enumeration
```powershell
# Virtual networks
Get-AzVirtualNetwork

# Subnets
Get-AzVirtualNetworkSubnetConfig

# Network interfaces
Get-AzNetworkInterface

# Public IP addresses
Get-AzPublicIpAddress

# Network security groups
Get-AzNetworkSecurityGroup

# NSG rules
Get-AzNetworkSecurityGroup | Select -ExpandProperty SecurityRules
```
## SQL Enumeration
```powershell
# SQL servers
Get-AzSqlServer

# SQL databases
Get-AzSqlDatabase
```
## RBAC Enumeration
```powershell
# List role assignments
Get-AzRoleAssignment

# Role assignments for resource group
Get-AzRoleAssignment -ResourceGroupName "<RESOURCE_GROUP>"

# Role assignments for specific user
Get-AzRoleAssignment -SignInName "<USER_EMAIL>"

# Role definitions
Get-AzRoleDefinition
```
## Managed Identity Enumeration
```powershell
# List managed identities
Get-AzUserAssignedIdentity

# Managed identity role assignments
Get-AzRoleAssignment | Where-Object {$_.ObjectType -eq "ServicePrincipal"}
```
## Export Results
```powershell
# Export resources
Get-AzResource | Export-Csv .\resources.csv -NoTypeInformation

# Export VMs
Get-AzVM | Export-Csv .\vms.csv -NoTypeInformation

# Export storage accounts
Get-AzStorageAccount | Export-Csv .\storage.csv -NoTypeInformation

# Export RBAC assignments
Get-AzRoleAssignment | Export-Csv .\rbac.csv -NoTypeInformation
```
## Helpful Discovery Commands
```powershell
# List all Az commands
Get-Command -Module Az*

# Search commands
Get-Command *AzStorage*

# Show module versions
Get-Module Az* -ListAvailable
```
