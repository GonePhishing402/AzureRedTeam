# Container Apps
This folder is used to track resources, scripts, and cheatsheets related to Azure Container Apps.
## Permissions
- Microsoft.App/containerApps/*/read
- Microsoft.App/containerApps/*/action
- Microsoft.App/containerApps/listsecrets/action
- Microsoft.App/containerApps/authconfigs/read
- Microsoft.App/containerApps/getauthtoken/action
## Command Execution on the Container App
```
az containerapp exec --name [container_name] --resource-group [resource_group_name]
```

