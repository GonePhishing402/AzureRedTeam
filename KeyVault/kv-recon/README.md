# kv-recon

All-in-one Azure red team reconnaissance platform with a browser-based web UI and headless CLI.  
Exchanges an OAuth refresh token for ARM, Key Vault, Storage, and Graph access tokens to enumerate and extract data from Azure and Microsoft 365 environments — all from a single compiled binary with no PowerShell, Az module, or C compiler dependency.

---

## Features

### Web UI
- **Dark-themed single-page app** — run `kv-recon.exe` with no flags to open `http://127.0.0.1:8080`
- **Persistent token store** — save and manage refresh tokens across sessions (backed by bbolt, stored at `~/.kv-recon.db`); all credential forms support one-click auto-fill from stored tokens
- **JWT Inspector** — paste any JWT to decode the header and claims inline
- **Device Code Phishing** — initiate an OAuth device code flow and display the user code; captures the resulting refresh token automatically and saves it to the token store

### Key Vault Recon
- Exchanges a refresh token for ARM + Key Vault tokens
- Discovers the active subscription automatically
- Enumerates all Key Vaults (or targets a specific vault)
- Checks effective RBAC permissions per vault via ARM
- Lists and retrieves all secret values with metadata
- Enumerates certificates and exports each as PFX (with private key) and PEM
- Streams output live to the browser log panel; saves results to disk

### Storage Recon
- Exchanges a refresh token for ARM + Storage tokens (`storage.azure.com/.default`)
- Enumerates all Storage Accounts in the subscription (or targets a specific account)
- Checks effective RBAC permissions per account via ARM
- Lists blob containers per account with public access level
- Optionally enumerates all blobs within each container (name, size, content type)
- Saves results per account to `{account}-storage.txt`

### Microsoft Graph Enumeration
- Token-driven enumeration against Microsoft Graph (any stored or manually entered access token)
- Enumerates: **Users**, **Devices**, **Applications**, **Service Principals**, **Groups**, **OneDrive** (with file download proxy), **Mail**, **Teams**, **SharePoint sites**
- Results rendered in sortable tables with live client-side filtering

### FOCI Token Exchange
- Exchanges a refresh token for a new token against any resource using any client ID
- Built-in presets for common first-party client IDs (Azure CLI, Teams, Office, etc.)
- Built-in scope presets for common resources (ARM, Graph, Storage, Key Vault, etc.)
- Saves results to the token store

### Headless CLI
- Pass `-refresh-token`, `-client-id`, and `-tenant-id` for scripted Key Vault recon to stdout

---

## Requirements

### Go

Install Go 1.22 or later from [go.dev/dl](https://go.dev/dl/) or via winget:

```powershell
winget install GoLang.Go
```

Verify:

```powershell
go version
```

---

## Installation

```powershell
# Navigate to the tool directory
cd KeyVault\kv-recon

# Download dependencies and generate go.sum
go mod tidy

# Build for Windows
go build -o kv-recon.exe .
```

### Cross-compile

Because the tool is pure Go (no CGo), cross-compilation requires no extra steps:

```powershell
# Linux
$env:GOOS = "linux"; go build -o kv-recon .

# macOS
$env:GOOS = "darwin"; go build -o kv-recon .
```

---

## Usage

### Web UI mode (default)

Run with no credential flags to launch the browser-based interface:

```powershell
.\kv-recon.exe
# Opens http://127.0.0.1:8080 automatically
```

**Web UI flags:**

| Flag | Default | Description |
|---|---|---|
| `-addr` | `127.0.0.1` | Address to listen on |
| `-port` | `8080` | Port to listen on |
| `-no-open` | — | Do not auto-open the browser |

```powershell
# Listen on all interfaces (e.g. from a remote jump host)
.\kv-recon.exe -addr 0.0.0.0 -port 9000

# Start server without auto-opening browser
.\kv-recon.exe -no-open
```

---

### Headless CLI mode

Provide credential flags to skip the web UI and run Key Vault recon directly to stdout:

```
kv-recon -refresh-token <token> -client-id <id> -tenant-id <id> [flags]
```

| Flag | Required | Description |
|---|---|---|
| `-refresh-token` | Yes | OAuth refresh token |
| `-client-id` | Yes | Azure AD Application (Client) ID |
| `-tenant-id` | Yes | Azure AD Tenant ID (GUID) |
| `-vault-name` | No | Target a specific vault; omit to enumerate all |
| `-output` | No | Directory for saved output (default: `./KVReconOutput`) |

```powershell
# Enumerate all vaults in a subscription
.\kv-recon.exe -refresh-token "0.AROA..." -client-id "<appId>" -tenant-id "<tenantId>"

# Target a specific vault and tee output to a log file
.\kv-recon.exe -refresh-token "0.AROA..." -client-id "<appId>" -tenant-id "<tenantId>" `
    -vault-name "my-vault" -output "C:\Loot\KVRecon" | Tee-Object recon.log

# Show help
.\kv-recon.exe -h
```

---

## Step-by-Step Cheatsheet

### Step 1 — Install Go (one-time)

```powershell
winget install GoLang.Go
# Restart your terminal after install so `go` is on PATH
go version
```

### Step 2 — Build the binary (one-time)

```powershell
cd KeyVault\kv-recon
go mod tidy
go build -o kv-recon.exe .
```

### Step 3 — Obtain a refresh token

You need a valid OAuth refresh token for the target tenant. See `Authentication/` for methods.  
The token must have access to `management.azure.com` and `vault.azure.net`.

```powershell
# Example: extract a refresh token from an existing Az PowerShell session
$ctx   = Get-AzContext
$token = [Microsoft.Azure.Commands.Common.Authentication.AzureSession]::Instance `
             .AuthenticationFactory.Authenticate(
                 $ctx.Account, $ctx.Environment, $ctx.Tenant.Id, $null,
                 [Microsoft.Azure.Commands.Common.Authentication.ShowDialog]::Never,
                 $null,
                 "https://management.azure.com/"
             )
$token.RefreshToken
```

### Step 4 — Identify your Client ID and Tenant ID

```powershell
# Client ID — the Application (client) ID of the service principal / app registration
# Tenant ID  — the GUID of the Azure AD tenant

# If you are using a known first-party client ID (e.g., Azure CLI):
$clientId = "04b07795-8ddb-461a-bbee-02f9e1bf7b46"   # Azure CLI public client

# Tenant ID from an existing session
(Get-AzContext).Tenant.Id
```

### Step 5 — Run the recon

**Option A — Web UI (recommended)**

```powershell
.\kv-recon.exe
# Browser opens automatically at http://127.0.0.1:8080
# Navigate using the left sidebar:
#   Tokens        → Save and manage refresh tokens
#   Device Code   → OAuth device code phishing
#   FOCI Exchange → Token exchange with custom client ID / scope
#   KV Recon      → Key Vault enumeration and secret/cert export
#   Storage Recon → Azure Blob Storage enumeration
#   Graph         → Microsoft Graph enumeration (users, devices, apps, etc.)
```

**Option B — Headless CLI**

```powershell
# Enumerate ALL vaults in the subscription
.\kv-recon.exe `
  -refresh-token "<refresh_token>" `
  -client-id     "<client_id>" `
  -tenant-id     "<tenant_id>"

# Target a SPECIFIC vault and save output to a custom directory
.\kv-recon.exe `
  -refresh-token "<refresh_token>" `
  -client-id     "<client_id>" `
  -tenant-id     "<tenant_id>" `
  -vault-name    "<vault_name>" `
  -output        "C:\Loot\KVRecon"

# Tee output to both stdout and a log file
.\kv-recon.exe `
  -refresh-token "<refresh_token>" `
  -client-id     "<client_id>" `
  -tenant-id     "<tenant_id>" | Tee-Object -FilePath recon.log
```

### Step 6 — Review output

```powershell
# List saved KV files
Get-ChildItem .\KVReconOutput -Recurse

# View secrets for a specific vault
Get-Content .\KVReconOutput\<vault-name>-secrets.txt

# List exported certificates
Get-ChildItem .\KVReconOutput -Filter *.pfx
Get-ChildItem .\KVReconOutput -Filter *.pem

# List saved Storage files
Get-ChildItem .\StorageReconOutput -Recurse

# View results for a specific storage account
Get-Content .\StorageReconOutput\<account-name>-storage.txt
```

---

## What Gets Saved

### Key Vault Recon

| File | Content |
|---|---|
| `{vault}-secrets.txt` | All secret names, values, and metadata |
| `{vault}-{cert}-export.pfx` | Certificate + private key bundle (PKCS#12) |
| `{vault}-{cert}.pem` | Public certificate in PEM format |
| `{vault}-{cert}-public-only.pem` | Public cert only (if private key export was denied) |
| `{vault}-{cert}-raw.b64` | Raw base64 bundle (unknown content type fallback) |

### Storage Recon

| File | Content |
|---|---|
| `{account}-storage.txt` | RBAC permissions, container list, and (if enabled) blob inventory |

---

## How It Works

### Key Vault Recon Phases

**Phase 1 — Token Acquisition**  
Exchanges the refresh token for `management.azure.com/.default` (ARM) and `vault.azure.net/.default` (Key Vault) tokens.

**Phase 2 — Subscription Discovery**  
Calls `GET /subscriptions` with the ARM token to identify the active subscription and tenant.

**Phase 3 — Vault Enumeration**  
Calls `GET /subscriptions/{sub}/providers/Microsoft.KeyVault/vaults` (paginated). Applies optional vault name filter.

**Phase 4 — RBAC Permissions**  
For each vault, calls the ARM authorization permissions endpoint to display Actions, NotActions, DataActions, and NotDataActions.

**Phase 5 — Secret Retrieval**  
Lists all secrets (paginated), retrieves each value, and saves results to `{vault}-secrets.txt`.

**Phase 6 — Certificate Export**  
Lists certificates, retrieves metadata, and exports full bundles (PFX + PEM) via the Key Vault secrets endpoint.

### Storage Recon Phases

**Phase 1 — Token Acquisition**  
Exchanges the refresh token for `management.azure.com/.default` (ARM) and `storage.azure.com/.default` (Storage) tokens.

**Phase 2 — Subscription Discovery**  
Reuses ARM token to identify the active subscription.

**Phase 3 — Storage Account Enumeration**  
Calls `GET /subscriptions/{sub}/providers/Microsoft.Storage/storageAccounts` (paginated). Applies optional account name filter. Logs account name, resource group, location, kind, blob endpoint, public network access, and HTTPS-only status.

**Phase 4 — RBAC Permissions**  
For each account, calls the ARM authorization permissions endpoint.

**Phase 5 — Container Enumeration**  
Lists blob containers per account via the Blob Storage REST API (`x-ms-version: 2020-04-08`). Logs container name and public access level.

**Phase 6 — Blob Enumeration** *(optional — enable "Enumerate blobs" checkbox)*  
Lists all blobs within each container. Logs blob name, size, and content type.

---

## Required Permissions

| Operation | Minimum Role |
|---|---|
| List vaults / storage accounts | `Reader` on the subscription |
| Check RBAC permissions | `Reader` on the resource |
| Read KV secrets | `Key Vault Secrets User` (data plane) |
| Export certificate with private key | `Key Vault Secrets User` (data plane) |
| List blob containers | `Storage Blob Data Reader` (data plane) |
| List blobs | `Storage Blob Data Reader` (data plane) |

---

## Project Structure

```
kv-recon/
  main.go          — Entry point: detects web UI vs headless CLI mode
  server.go        — HTTP server, routes, SSE log streaming, browser auto-open
  ui/index.html    — Embedded single-page web UI (dark theme)
  recon.go         — KV recon orchestrator (all phases)
  storagerecon.go  — Storage recon orchestrator (all phases)
  tokens.go        — OAuth refresh token exchange
  arm.go           — Subscription discovery, vault/storage enumeration, RBAC
  keyvault.go      — Secret and certificate retrieval, PFX/PEM export
  storage.go       — Azure Blob Storage REST API (containers, blobs)
  graph.go         — Microsoft Graph API enumeration
  store.go         — Persistent token store (bbolt)
  jwt.go           — JWT decode helpers
  output.go        — File I/O helpers, log channel helpers
  util.go          — Shared utilities
  go.mod           — Module definition and dependencies
```

---

## Dependencies

| Package | Purpose |
|---|---|
| [`golang.org/x/crypto/pkcs12`](https://pkg.go.dev/golang.org/x/crypto/pkcs12) | PKCS#12 (PFX) decode — not in Go stdlib |
| [`go.etcd.io/bbolt`](https://pkg.go.dev/go.etcd.io/bbolt) | Embedded key-value store for persistent token storage |

---

## Requirements

### Go

Install Go 1.22 or later from [go.dev/dl](https://go.dev/dl/) or via winget:

```powershell
winget install GoLang.Go
```

Verify:

```powershell
go version
```

---

## Installation

```powershell
# Navigate to the tool directory
cd KeyVault\kv-recon

# Download dependencies and generate go.sum
go mod tidy

# Build for Windows
go build -o kv-recon.exe .
```

### Cross-compile

Because the tool is pure Go (no CGo), cross-compilation requires no extra steps:

```powershell
# Linux
$env:GOOS = "linux"; go build -o kv-recon .

# macOS
$env:GOOS = "darwin"; go build -o kv-recon .
```

---

## Usage

### Web UI mode (default)

Run with no credential flags to launch the browser-based interface:

```powershell
.\kv-recon.exe
# Opens http://127.0.0.1:8080 automatically
```

Fill in the form and click **▶ Run Recon**. Output streams live in the log panel below the form, color-coded by severity. Secrets and certificates are saved to the configured output directory.

**Web UI flags:**

| Flag | Default | Description |
|---|---|---|
| `-addr` | `127.0.0.1` | Address to listen on |
| `-port` | `8080` | Port to listen on |
| `-no-open` | — | Do not auto-open the browser |

```powershell
# Listen on all interfaces (e.g. from a remote jump host)
.\kv-recon.exe -addr 0.0.0.0 -port 9000

# Start server without auto-opening browser
.\kv-recon.exe -no-open
```

---

### Headless CLI mode

Provide credential flags to skip the web UI and run directly to stdout:

```
kv-recon -refresh-token <token> -client-id <id> -tenant-id <id> [flags]
```

| Flag | Required | Description |
|---|---|---|
| `-refresh-token` | Yes | OAuth refresh token |
| `-client-id` | Yes | Azure AD Application (Client) ID |
| `-tenant-id` | Yes | Azure AD Tenant ID (GUID) |
| `-vault-name` | No | Target a specific vault; omit to enumerate all |
| `-output` | No | Directory for saved output (default: `./KVReconOutput`) |

```powershell
# Enumerate all vaults in a subscription
.\kv-recon.exe -refresh-token "0.AROA..." -client-id "<appId>" -tenant-id "<tenantId>"

# Target a specific vault and tee output to a log file
.\kv-recon.exe -refresh-token "0.AROA..." -client-id "<appId>" -tenant-id "<tenantId>" `
    -vault-name "my-vault" -output "C:\Loot\KVRecon" | Tee-Object recon.log

# Show help
.\kv-recon.exe -h
```

---

## Step-by-Step Cheatsheet

### Step 1 — Install Go (one-time)

```powershell
winget install GoLang.Go
# Restart your terminal after install so `go` is on PATH
go version
```

### Step 2 — Build the binary (one-time)

```powershell
cd KeyVault\kv-recon
go mod tidy
go build -o kv-recon.exe .
```

### Step 3 — Obtain a refresh token

You need a valid OAuth refresh token for the target tenant. See `Authentication/` for methods.  
The token must have access to `management.azure.com` and `vault.azure.net`.

```powershell
# Example: extract a refresh token from an existing Az PowerShell session
$ctx   = Get-AzContext
$token = [Microsoft.Azure.Commands.Common.Authentication.AzureSession]::Instance `
             .AuthenticationFactory.Authenticate(
                 $ctx.Account, $ctx.Environment, $ctx.Tenant.Id, $null,
                 [Microsoft.Azure.Commands.Common.Authentication.ShowDialog]::Never,
                 $null,
                 "https://management.azure.com/"
             )
$token.RefreshToken
```

### Step 4 — Identify your Client ID and Tenant ID

```powershell
# Client ID — the Application (client) ID of the service principal / app registration
# Tenant ID  — the GUID of the Azure AD tenant

# If you are using a known first-party client ID (e.g., Azure CLI):
$clientId = "04b07795-8ddb-461a-bbee-02f9e1bf7b46"   # Azure CLI public client

# Tenant ID from an existing session
(Get-AzContext).Tenant.Id
```

### Step 5 — Run the recon

**Option A — Web UI (recommended)**

```powershell
.\kv-recon.exe
# Browser opens automatically at http://127.0.0.1:8080
# Fill in the form and click ▶ Run Recon
```

**Option B — Headless CLI**

```powershell
# Enumerate ALL vaults in the subscription
.\kv-recon.exe `
  -refresh-token "<refresh_token>" `
  -client-id     "<client_id>" `
  -tenant-id     "<tenant_id>"

# Target a SPECIFIC vault and save output to a custom directory
.\kv-recon.exe `
  -refresh-token "<refresh_token>" `
  -client-id     "<client_id>" `
  -tenant-id     "<tenant_id>" `
  -vault-name    "<vault_name>" `
  -output        "C:\Loot\KVRecon"

# Tee output to both stdout and a log file
.\kv-recon.exe `
  -refresh-token "<refresh_token>" `
  -client-id     "<client_id>" `
  -tenant-id     "<tenant_id>" | Tee-Object -FilePath recon.log
```

### Step 6 — Review output

```powershell
# List saved files
Get-ChildItem .\KVReconOutput -Recurse

# View secrets for a specific vault
Get-Content .\KVReconOutput\<vault-name>-secrets.txt

# List exported certificates
Get-ChildItem .\KVReconOutput -Filter *.pfx
Get-ChildItem .\KVReconOutput -Filter *.pem
```

---

## What Gets Saved

All output files are written to the configured **Output Path** directory.

| File | Content |
|---|---|
| `{vault}-secrets.txt` | All secret names, values, and metadata |
| `{vault}-{cert}-export.pfx` | Certificate + private key bundle (PKCS#12) |
| `{vault}-{cert}.pem` | Public certificate in PEM format |
| `{vault}-{cert}-public-only.pem` | Public cert only (if private key export was denied) |
| `{vault}-{cert}-raw.b64` | Raw base64 bundle (unknown content type fallback) |

---

## How It Works

### Phase 1 — Token Acquisition
Exchanges the provided refresh token twice using the OAuth 2.0 refresh token grant:
- `https://management.azure.com/.default` → ARM access token
- `https://vault.azure.net/.default` → Key Vault access token

### Phase 2 — Subscription Discovery
Calls `GET /subscriptions` with the ARM token to identify the active subscription and tenant.

### Phase 3 — Vault Enumeration
Calls `GET /subscriptions/{sub}/providers/Microsoft.KeyVault/vaults` to list all accessible vaults. Handles `nextLink` pagination. If a vault name filter is specified, only that vault is returned.

### Phase 4 — RBAC Permissions
For each vault, calls the ARM authorization permissions endpoint:
```
GET /subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.KeyVault/vaults/{name}
    /providers/Microsoft.Authorization/permissions
```
Displays Actions, NotActions, DataActions (effective data-plane permissions), and NotDataActions.

### Phase 5 — Secret Retrieval
Lists all secrets via `GET /secrets` (paginated), then retrieves each value via `GET /secrets/{name}`. Displays name, value, enabled state, expiry, and content type. Saves all results to `{vault}-secrets.txt`.

### Phase 6 — Certificate Export
Lists all certificates via `GET /certificates` (paginated), retrieves metadata via `GET /certificates/{name}`, and exports the full bundle via `GET /secrets/{name}` (Key Vault stores certs + private keys as secrets at the same name). Handles both PKCS#12 and PEM content types.

---

## Required Permissions

| Operation | Minimum Role |
|---|---|
| List vaults | `Reader` on the subscription |
| Check RBAC permissions | `Reader` on the vault resource |
| Read secrets | `Key Vault Secrets User` (data plane) |
| Export certificate with private key | `Key Vault Secrets User` (data plane) |
| Read certificate metadata only | `Key Vault Reader` (data plane) |

---

## Project Structure

```
kv-recon/
  main.go       — Entry point: detects web UI vs headless CLI mode
  server.go     — HTTP server, SSE log streaming, browser auto-open
  ui/index.html — Embedded single-page web UI (dark theme)
  recon.go      — Orchestrator: runs all phases in order
  tokens.go     — OAuth refresh token exchange
  arm.go        — Subscription discovery, vault enumeration, RBAC
  keyvault.go   — Secret and certificate retrieval, PFX/PEM export
  output.go     — File I/O helpers, log channel helpers
  util.go       — Shared utilities
  go.mod        — Module definition and dependencies
```

---

## Dependencies

| Package | Purpose |
|---|---|
| [`golang.org/x/crypto/pkcs12`](https://pkg.go.dev/golang.org/x/crypto/pkcs12) | PKCS#12 (PFX) decode — not in Go stdlib |
