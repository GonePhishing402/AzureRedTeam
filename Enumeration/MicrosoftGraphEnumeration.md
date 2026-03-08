# Microsoft Graph PowerShell Enumeration Cheat Sheet

> **Note:** Microsoft Graph PowerShell is best for enumerating **Microsoft Entra ID / Azure AD, Microsoft 365, users, groups, apps, service principals, devices, roles, mail, Teams, SharePoint, and OneDrive** objects. For **Azure resource enumeration** like VMs, storage accounts, subscriptions, and Key Vault resource inventory, use the **Az PowerShell** module instead.

---

## Install and Import

```powershell
# Install the full SDK
Install-Module Microsoft.Graph -Scope CurrentUser

# Import the main module
Import-Module Microsoft.Graph

```

## Connect to Microsoft Graph
```powershell
# Delegated login
Connect-MgGraph -Scopes "User.Read.All","Group.Read.All","Application.Read.All","Directory.Read.All"

# Device code login
Connect-MgGraph -Scopes "User.Read.All","Group.Read.All","Application.Read.All","Directory.Read.All" -UseDeviceCode

# App-only authentication with certificate
Connect-MgGraph -ClientId "<APP_ID>" -TenantId "<TENANT_ID>" -CertificateThumbprint "<THUMBPRINT>"

# View current session context
Get-MgContext

# Disconnect session
Disconnect-MgGraph
```
## Check you Graph Session
```powershell
# Show authentication context
Get-MgContext | Format-List *

# Show environment information
Get-MgEnvironment

# View all Graph commands available
Get-Command -Module Microsoft.Graph*
```

## Helpful Discovery Commands
```powershell
# Find which cmdlet maps to a Graph endpoint
Find-MgGraphCommand -Uri "/users"
Find-MgGraphCommand -Command Get-MgUser

# View required permissions for a command
Find-MgGraphCommand -Command Get-MgApplication | Select-Object -ExpandProperty Permissions

# Search Graph permissions
Find-MgGraphPermission "Application.Read.All"
```
## Tenant/Organization Enumeration
```powershell
# Tenant information
Get-MgOrganization

# Domains in tenant
Get-MgDomain

# Directory roles
Get-MgDirectoryRole
Get-MgDirectoryRoleTemplate

# Administrative units
Get-MgDirectoryAdministrativeUnit
```


