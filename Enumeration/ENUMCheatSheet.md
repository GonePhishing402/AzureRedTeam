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
# Identify Temporary Access Passwords (TAP)
## Check for Misonciguration in Authentication Policy
```powershell
(Get-MgPolicyAuthenticationMethodPolicy).AuthenticationMethodConfigurations
```
## Get more info on the temp password
```powershell
(Get-MgPolicyAuthenticationMethodPolicyAuthenticationMethodConfiguration -AuthenticationMethodConfigurationId TemporaryAccessPass).AdditionalProperties
```
## Check which groups can use TAP
```powershell
(Get-MgPolicyAuthenticationMethodPolicyAuthenticationMethodConfiguration -AuthenticationMethodConfigurationId TemporaryAccessPass).AdditionalProperties.includeTargets
```
## Create new TAP for user - Need Authentication Administrator Permission and must Assign to User who can accept TAP
```powershell
New-MgUserAuthenticationTemporaryAccessPassMethod -UserId explorationsyncuserX@oilcorporation.onmicrosoft.com -BodyParameter $propertiesJSON | fl
```
# Logic App Enumeration
## View logic app
```powershell
(Get-AzLogicApp -Name WellPlanningLogicApp).Definition
```
## View Trigger URL of Logic App
```powershell
Get-AzLogicAppTriggerCallbackUrl -TriggerName manual -Name WellPlanningLogicApp -ResourceGroupName Drilling
```
## Trigger a Logic App - Use Trigger URL from Previous Command
```powershell
Invoke-RestMethod -Method POST -UseBasicParsing -Uri 'https://prod-70.eastus.logic.azure.com:443/workflows/b8190f07ab9647ff99fe54887bae2962/triggers/manual/paths/invoke?api-version=2018-07-01-preview&sp=%2Ftriggers%2Fmanual%2Frun&sv=1.0&sig=ahq68505pOJPcuDLyB_SqhI1RGR_q9579HHUndempac'
```
