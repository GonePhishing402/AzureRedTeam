# Defend Azure Web Apps — Diagnostic Settings & Investigation

A guide for configuring Azure App Service diagnostic logging to a Log Analytics Workspace and investigating suspicious activity using KQL.

---

## 1. Enable Diagnostic Logging

### Azure Portal

1. Navigate to your **App Service** in the [Azure Portal](https://portal.azure.com)
2. Select **Monitoring** > **App Service logs**
3. Enable the following:
   - **Application logging** — set to **File System** (or **Blob** for long-term retention)
   - **Web server logging** — set to **File System** or **Storage**
   - **Detailed error messages** — **On**
   - **Failed request tracing** — **On**
4. Select **Save**

### Azure CLI

```bash
# Create a Log Analytics Workspace (if one doesn't exist)
az monitor log-analytics workspace create \
  --resource-group <resource-group> \
  --workspace-name <workspace-name>

# Get the resource IDs
resourceID=$(az webapp show -g <resource-group> -n <app-name> --query id --output tsv)
workspaceID=$(az monitor log-analytics workspace show -g <resource-group> --workspace-name <workspace-name> --query id --output tsv)

# Create diagnostic setting to send all logs to Log Analytics
az monitor diagnostic-settings create \
  --name "AppServiceDiagnostics" \
  --resource "$resourceID" \
  --workspace "$workspaceID" \
  --logs '[
    {"category": "AppServiceHTTPLogs", "enabled": true},
    {"category": "AppServiceConsoleLogs", "enabled": true},
    {"category": "AppServiceAppLogs", "enabled": true},
    {"category": "AppServiceAuditLogs", "enabled": true},
    {"category": "AppServiceIPSecAuditLogs", "enabled": true},
    {"category": "AppServicePlatformLogs", "enabled": true},
    {"category": "AppServiceFileAuditLogs", "enabled": true},
    {"category": "AppServiceAntivirusScanAuditLogs", "enabled": true}
  ]'
```

### PowerShell

```powershell
# Create diagnostic setting
$resourceId = (Get-AzWebApp -ResourceGroupName <resource-group> -Name <app-name>).Id
$workspaceId = (Get-AzOperationalInsightsWorkspace -ResourceGroupName <resource-group> -Name <workspace-name>).ResourceId

Set-AzDiagnosticSetting -ResourceId $resourceId -WorkspaceId $workspaceId -Enabled $true -Name "AppServiceDiagnostics"
```

---

## 2. Log Analytics Tables for App Service

Once diagnostic settings are configured, logs flow into the following tables in your Log Analytics Workspace:

| Table | Description | Availability |
|---|---|---|
| `AppServiceHTTPLogs` | Web server HTTP request logs (method, URI, status code, client IP, user agent) | Windows, Linux, Containers |
| `AppServiceConsoleLogs` | Standard output and standard error from the application | Java SE, Tomcat, Linux, Containers |
| `AppServiceAppLogs` | Application-level logs (traces, errors, warnings) | ASP.NET, Java SE, Tomcat |
| `AppServiceAuditLogs` | Login activity via FTP and Kudu | Windows, Linux, Containers |
| `AppServiceFileAuditLogs` | File changes made to site content (**Premium tier and above**) | Windows, Windows Containers |
| `AppServiceIPSecAuditLogs` | Requests evaluated against IP restriction rules | Windows, Linux, Containers |
| `AppServicePlatformLogs` | Container operation logs (start, stop, pull) | Linux, Containers |
| `AppServiceAntivirusScanAuditLogs` | Antivirus scan logs via Microsoft Defender for Cloud (**Premium tier**) | Windows, Linux, Containers |
| `AppServiceAuthenticationLogs` | Authentication and authorization events | Windows, Linux, Containers |
| `AzureActivity` | Subscription-level control plane operations (create, delete, restart, config changes) | All |

---

## 3. Investigation Queries (KQL)

### Detect Suspicious HTTP Requests

```kusto
// Requests returning 500+ errors — may indicate exploitation attempts
AppServiceHTTPLogs
| where ScStatus >= 500
| project TimeGenerated, CsMethod, CsUriStem, ScStatus, CIp, CsHost, UserAgent
| order by TimeGenerated desc
```

### Identify Command Injection Attempts

```kusto
// Look for common OS command injection patterns in request URIs
AppServiceHTTPLogs
| where CsUriStem has_any ("$(", "%24(", "`", "%60", ";", "%3B", "&&", "%26%26", "|", "%7C")
   or CsUriStem has_any ("whoami", "/etc/passwd", "cmd.exe", "powershell", "curl", "wget")
| project TimeGenerated, CsMethod, CsUriStem, ScStatus, CIp, UserAgent
| order by TimeGenerated desc
```

### Monitor Console Output for Suspicious Activity

```kusto
// Look for evidence of RCE in console logs
AppServiceConsoleLogs
| where ResultDescription has_any ("uid=", "root", "id=", "whoami", "IDENTITY_ENDPOINT", "MSI_SECRET", "access_token")
| project TimeGenerated, Host, ResultDescription
| order by TimeGenerated desc
```

### Correlate HTTP 500 Errors with Console Logs

```kusto
// Join HTTP errors with console output to find exploitation artifacts
let myHttp = AppServiceHTTPLogs
| where ScStatus == 500
| project TimeGen=substring(TimeGenerated, 0, 19), CsUriStem, ScStatus, CIp;

let myConsole = AppServiceConsoleLogs
| project TimeGen=substring(TimeGenerated, 0, 19), ResultDescription;

myHttp
| join myConsole on TimeGen
| project TimeGen, CsUriStem, ScStatus, CIp, ResultDescription
```

### Detect FTP/Kudu Login Activity

```kusto
// Monitor for unauthorized access via FTP or Kudu (SCM)
AppServiceAuditLogs
| project TimeGenerated, User, UserAddress, Protocol, ResourceName
| order by TimeGenerated desc
```

### Detect File Changes (Premium tier)

```kusto
// Monitor file modifications — may indicate webshell deployment
AppServiceFileAuditLogs
| where OperationName == "Write" or OperationName == "Create"
| project TimeGenerated, OperationName, Path, Process, UserDisplayName
| order by TimeGenerated desc
```

### Detect Requests Blocked by IP Restrictions

```kusto
// Check IP security rule evaluations
AppServiceIPSecAuditLogs
| where Result == "Denied"
| project TimeGenerated, CIp, CsHost, Details, Result
| summarize Count=count() by CIp, Details
| order by Count desc
```

### Monitor Managed Identity Token Requests (via Console Logs)

```kusto
// Look for token requests to the managed identity endpoint — may indicate token theft
AppServiceConsoleLogs
| where ResultDescription has_any ("IDENTITY_ENDPOINT", "169.254.169.254", "metadata/identity", "oauth2/token", "vault.azure.net", "management.azure.com", "graph.microsoft.com")
| project TimeGenerated, Host, ResultDescription
| order by TimeGenerated desc
```

### Monitor App Configuration Changes (Activity Log)

```kusto
// Track control plane operations on the web app
AzureActivity
| where ResourceProvider == "Microsoft.Web"
| where OperationNameValue has_any ("Microsoft.Web/sites/write", "Microsoft.Web/sites/delete", "Microsoft.Web/sites/config/write", "Microsoft.Web/sites/publishxml/action")
| project TimeGenerated, OperationNameValue, Caller, CallerIpAddress, ActivityStatusValue
| order by TimeGenerated desc
```

### Summary: Log Severity Over Time

```kusto
// Bar chart of app log severities over time
AppServiceAppLogs
| summarize count() by CustomLevel, bin(TimeGenerated, 1h), _ResourceId
| render barchart
```

---

## 4. Alert Rules to Configure

| Alert | Table | Condition |
|---|---|---|
| High rate of 500 errors | `AppServiceHTTPLogs` | `ScStatus >= 500` count > threshold in 5 min |
| Command injection patterns in URIs | `AppServiceHTTPLogs` | URI contains `$(`, `` ` ``, `&&`, `|` |
| FTP/Kudu login from unknown IP | `AppServiceAuditLogs` | `UserAddress` not in allowed list |
| File modifications outside deployments | `AppServiceFileAuditLogs` | `Write` or `Create` operations outside deploy windows |
| Managed identity token scraping | `AppServiceConsoleLogs` | Output contains `IDENTITY_ENDPOINT`, `oauth2/token` |
| IP restriction denials spike | `AppServiceIPSecAuditLogs` | `Result == "Denied"` count > threshold |

---

## 5. Microsoft Documentation References

- [Enable Diagnostic Logging for Apps in Azure App Service](https://learn.microsoft.com/azure/app-service/troubleshoot-diagnostic-logs)
- [Monitor Azure App Service](https://learn.microsoft.com/azure/app-service/monitor-app-service)
- [App Service Monitoring Data Reference (Log Tables & Metrics)](https://learn.microsoft.com/azure/app-service/monitor-app-service-reference)
- [Tutorial: Troubleshoot an App Service App with Azure Monitor](https://learn.microsoft.com/azure/app-service/tutorial-troubleshoot-monitor)
- [Secure Your Azure App Service Deployment](https://learn.microsoft.com/azure/app-service/overview-security)
- [Send Logs to Azure Monitor (Diagnostic Settings)](https://learn.microsoft.com/azure/app-service/troubleshoot-diagnostic-logs#send-logs-to-azure-monitor)
- [Managed Identities for App Service](https://learn.microsoft.com/azure/app-service/overview-managed-identity)
- [KQL Query Samples for AppServiceAppLogs](https://learn.microsoft.com/azure/azure-monitor/reference/queries/appserviceapplogs)
- [Create Diagnostic Settings in Azure Monitor](https://learn.microsoft.com/azure/azure-monitor/essentials/diagnostic-settings)
