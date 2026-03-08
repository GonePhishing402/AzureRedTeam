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
## User Enumeration
```powershell
# List all users
Get-MgUser -All

# Get specific user
Get-MgUser -UserId user@contoso.com

# Show useful properties
Get-MgUser -All -Property Id,DisplayName,UserPrincipalName,AccountEnabled |
Select-Object Id,DisplayName,UserPrincipalName,AccountEnabled

# Search users
Get-MgUser -Search '"displayName:admin"' -ConsistencyLevel eventual

# Get user's manager
Get-MgUserManager -UserId user@contoso.com

# User group memberships
Get-MgUserMemberOf -UserId user@contoso.com

# Authentication methods
Get-MgUserAuthenticationMethod -UserId user@contoso.com
```
## Group Enumeration
```powershell
# All groups
Get-MgGroup -All

# Microsoft 365 groups
Get-MgGroup -Filter "groupTypes/any(c:c eq 'Unified')"

# Security groups
Get-MgGroup -Filter "securityEnabled eq true"

# Group members
Get-MgGroupMember -GroupId "<GROUP_ID>"

# Group owners
Get-MgGroupOwner -GroupId "<GROUP_ID>"
```
## Application and Service Principal Enumeration
```powershell
# App registrations
Get-MgApplication -All

# Specific application
Get-MgApplication -ApplicationId "<APP_OBJECT_ID>"

# Useful properties
Get-MgApplication -All -Property Id,AppId,DisplayName |
Select-Object Id,AppId,DisplayName

# App secrets / certificates metadata
Get-MgApplication -ApplicationId "<APP_OBJECT_ID>" -Property PasswordCredentials,KeyCredentials

# Enterprise applications / service principals
Get-MgServicePrincipal -All

# Specific service principal
Get-MgServicePrincipal -ServicePrincipalId "<SP_OBJECT_ID>"

# Filter service principal by appId
Get-MgServicePrincipal -Filter "appId eq '<APP_ID>'"

# App role assignments
Get-MgServicePrincipalAppRoleAssignment -ServicePrincipalId "<SP_OBJECT_ID>"

# OAuth permission grants
Get-MgOauth2PermissionGrant -All
```
## Device Enumeration
```powershell
# List all devices
Get-MgDevice -All

# Enabled devices
Get-MgDevice -Filter "accountEnabled eq true"

# Device owners
Get-MgDeviceRegisteredOwner -DeviceId "<DEVICE_ID>"

# Device users
Get-MgDeviceRegisteredUser -DeviceId "<DEVICE_ID>"
```
## Role and Privilege Enumeration
```powershell
# Active directory roles
Get-MgDirectoryRole

# Members of role
Get-MgDirectoryRoleMember -DirectoryRoleId "<ROLE_ID>"

# Role templates
Get-MgDirectoryRoleTemplate

# Role assignments
Get-MgRoleManagementDirectoryRoleAssignment

# Role definitions
Get-MgRoleManagementDirectoryRoleDefinition
```
## Conditional Access and Policies
```powershell
# Conditional Access policies
Get-MgIdentityConditionalAccessPolicy

# Named locations
Get-MgIdentityConditionalAccessNamedLocation

# Authentication strength policies
Get-MgPolicyAuthenticationStrengthPolicy
```
## Mail Enumeration
```powershell
# List user messages
Get-MgUserMessage -UserId user@contoso.com

# Mail folders
Get-MgUserMailFolder -UserId user@contoso.com
```

## Calendar Enumeration
```powershell
# Calendar events
Get-MgUserEvent -UserId user@contoso.com

# User calendars
Get-MgUserCalendar -UserId user@contoso.com
```
## Microsoft Teams Enumeration
```powershell
# Teams
Get-MgTeam -All

# Channels in a team
Get-MgTeamChannel -TeamId "<TEAM_ID>"

# Teams a user has joined
Get-MgUserJoinedTeam -UserId user@contoso.com
```
## SharePoint Enumeration
```powershell
# Sites
Get-MgSite -All

# Lists in a site
Get-MgSiteList -SiteId "<SITE_ID>"
```
## OneDrive Enumeration
```powershell
# User drive
Get-MgUserDrive -UserId user@contoso.com

# Drive items
Get-MgUserDriveItem -UserId user@contoso.com -DriveId "<DRIVE_ID>"
```

## Export Results
```powershell
# Export users
Get-MgUser -All | Export-Csv .\users.csv -NoTypeInformation

# Export applications
Get-MgApplication -All | Export-Csv .\applications.csv -NoTypeInformation

# Export service principals
Get-MgServicePrincipal -All | Export-Csv .\serviceprincipals.csv -NoTypeInformation

# Export sign-in logs
Get-MgAuditLogSignIn -All | Export-Csv .\signinlogs.csv -NoTypeInformation
```


