# Container Apps

This folder is used to track resources, scripts, and cheatsheets related to Azure Container Apps.

---

## Overview

Azure Container Apps is a serverless container platform that enables running microservices and containerized applications without managing infrastructure. From a red team perspective, Container Apps can expose secrets, managed identity tokens, and provide command execution opportunities.

---

## Permissions (Microsoft.App Resource Provider)

### Container App Permissions

| Permission | Description |
|---|---|
| `microsoft.app/containerapps/read` | Get a Container App |
| `microsoft.app/containerapps/write` | Create or update a Container App |
| `microsoft.app/containerapps/delete` | Delete a Container App |
| `microsoft.app/containerapps/listsecrets/action` | List secrets of a container app |
| `microsoft.app/containerapps/getauthtoken/action` | Get auth token for Dev APIs (log stream, exec, port forward) |
| `microsoft.app/containerapps/authconfigs/read` | Get auth config of a container app |
| `microsoft.app/containerapps/authconfigs/write` | Create or update auth config of a container app |
| `microsoft.app/containerapps/authconfigs/delete` | Delete auth config of a container app |
| `microsoft.app/containerapps/stop/action` | Stop a Container App |
| `microsoft.app/containerapps/start/action` | Start a Container App |
| `microsoft.app/containerapps/revisions/read` | Get revision of a container app |
| `microsoft.app/containerapps/revisions/restart/action` | Restart a container app revision |
| `microsoft.app/containerapps/revisions/activate/action` | Activate a container app revision |
| `microsoft.app/containerapps/revisions/deactivate/action` | Deactivate a container app revision |
| `microsoft.app/containerapps/revisions/replicas/read` | Get replica of a container app revision |
| `microsoft.app/containerapps/sourcecontrols/read` | Get Container App Source Control Configuration |
| `microsoft.app/containerapps/sourcecontrols/write` | Create or Update Container App Source Control Configuration |
| `microsoft.app/containerapps/detectors/read` | Get detector of a container app |

### Container App Jobs Permissions

| Permission | Description |
|---|---|
| `microsoft.app/jobs/read` | Get a Container Apps Job |
| `microsoft.app/jobs/write` | Create or update a Container Apps Job |
| `microsoft.app/jobs/delete` | Delete a Container Apps Job |
| `microsoft.app/jobs/start/action` | Start a Container Apps Job |
| `microsoft.app/jobs/stop/action` | Stop multiple Container Apps Job executions |
| `microsoft.app/jobs/listsecrets/action` | List secrets of a container apps job |
| `microsoft.app/jobs/getauthtoken/action` | Get auth token for Dev APIs (log stream, exec, port forward) |
| `microsoft.app/jobs/execution/read` | Get a single execution from a Container Apps Job |
| `microsoft.app/jobs/executions/read` | Get a Container Apps Job's execution history |

### Built-in RBAC Roles

| Role | Description |
|---|---|
| **ContainerApp Contributor** | Manage Container Apps resources |
| **Reader** | View Container Apps resources (no secrets) |
| **Contributor** | Full management of Container Apps resources |
| **Owner** | Full management + role assignment |

> **Full permissions reference:** [Azure permissions for Compute - microsoft.app](https://learn.microsoft.com/azure/role-based-access-control/permissions/compute#microsoftapp)

---

## Command Execution on the Container App

```bash
az containerapp exec --name <container_name> --resource-group <resource_group_name>
```

Specify a particular revision or replica:

```bash
az containerapp exec --name <container_name> --resource-group <resource_group_name> --revision <revision_name>
az containerapp exec --name <container_name> --resource-group <resource_group_name> --replica <replica_name>
```

Run a specific command:

```bash
az containerapp exec --name <container_name> --resource-group <resource_group_name> --command <command>
```

---

## Useful Enumeration Commands

```bash
# List all container apps in a subscription
az containerapp list --output table

# List container apps in a specific resource group
az containerapp list --resource-group <resource_group_name> --output table

# Show details of a specific container app
az containerapp show --name <container_name> --resource-group <resource_group_name>

# List secrets
az containerapp secret list --name <container_name> --resource-group <resource_group_name>

# Show a specific secret value
az containerapp secret show --name <container_name> --resource-group <resource_group_name> --secret-name <secret_name>

# List revisions
az containerapp revision list --name <container_name> --resource-group <resource_group_name> --output table

# List replicas of a revision
az containerapp replica list --name <container_name> --resource-group <resource_group_name>

# View container app logs
az containerapp logs show --name <container_name> --resource-group <resource_group_name>
```

---

## Microsoft Documentation References

- [Azure Container Apps Overview](https://learn.microsoft.com/azure/container-apps/)
- [Microsoft.App Permissions (RBAC)](https://learn.microsoft.com/azure/role-based-access-control/permissions/compute#microsoftapp)
- [Secure Your Container Apps Deployment](https://learn.microsoft.com/azure/container-apps/secure-deployment)
- [Managed Identities in Container Apps](https://learn.microsoft.com/azure/container-apps/managed-identity)
- [Authentication and Authorization in Container Apps](https://learn.microsoft.com/azure/container-apps/authentication)
- [Container Apps Jobs](https://learn.microsoft.com/azure/container-apps/jobs)
- [Azure Security Baseline for Container Apps](https://learn.microsoft.com/security/benchmark/azure/baselines/azure-container-apps-security-baseline)

