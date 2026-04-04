# Cheat Sheet to find different roles and paths
## Find Entra Roles to SP - Help Desk Admin
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
