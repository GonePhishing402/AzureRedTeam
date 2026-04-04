# Find Entra Roles to SP - Help Desk Admin
## Enumerate Entra Roles
```powershell
Get-MgRoleManagementDirectoryRoleAssignment -Filter "principalId eq 'cddece96-16c9-4c93-9c9d-97c96f98be1d'" | ForEach-Object {
    $roleDef = Get-MgRoleManagementDirectoryRoleDefinition -UnifiedRoleDefinitionId $_.RoleDefinitionId
    [PSCustomObject]@{
        RoleDisplayName = $roleDef.DisplayName
        RoleId = $roleDef.Id
        DirectoryScopeId = $_.DirectoryScopeId
    }
} | Select-Object RoleDisplayName, RoleId, DirectoryScopeId | fl
```
## Find API permissions for App Role Assignments and Resolve App Role Field
```powershell
Get-MgServicePrincipalAppRoleAssignment -ServicePrincipalId cddece96-16c9-4c93-9c9d-97c96f98be1d | fl
```
## Resolve App Role Field
```powershell
(Get-MgServicePrincipal -Filter "appId eq '00000003-0000-0000-c000-000000000000'").AppRoles | ? {$_.id -eq '246dd0d5-5bd0-4def-940b-0421030a5b68' } | fl
```
# Identify Temp Pass
## Check for Misonciguration in Authentication Policy
```powershell
(Get-MgPolicyAuthenticationMethodPolicy).AuthenticationMethodConfigurations
```
