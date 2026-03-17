# strgprowl

Azure Storage reconnaissance tool with a browser-based web UI and headless CLI.  
Exchanges an OAuth refresh token for ARM and Storage access tokens to enumerate storage accounts, containers, and blobs — all from a single compiled binary with no PowerShell, Az module, or C compiler dependency.

---

## Features

### Web UI
- **Dark-themed single-page app** — run `strgprowl.exe` with no flags to open `http://127.0.0.1:8080`
- **Persistent credentials** — Tenant ID, Client ID, and Refresh Token are saved to `localStorage` and auto-filled on every page reload
- **Tokens tab** — displays the raw JWT and decoded JSON payload for both the ARM token and the Storage token after any Connect or Recon operation; individual copy buttons for each

### Recon tab
- Exchanges a refresh token for ARM + Storage tokens (`storage.azure.com/.default`)
- Discovers the active subscription automatically
- Enumerates all Storage Accounts in the subscription (or targets a specific account)
- Checks effective RBAC permissions per account via ARM
- Lists blob containers per account with public access level (ARM management plane — Storage Blob Data Reader is **not** required)
- Enumerates all blobs within each container (name, size, content type, last modified)
- Streams log output live to the browser; saves one `.txt` file per account to the configured output directory

### Blob Browser tab
- Connect once with credentials to get a persistent session
- Navigate accounts → containers → blobs in a sortable, filterable table UI
- **View** any blob inline in a modal (text, JSON pretty-printed, images rendered)
- **Download** any blob directly via the browser
- Breadcrumb navigation to jump back up the hierarchy

---

## Requirements

Install Go 1.20 or later from [go.dev/dl](https://go.dev/dl/) or via winget:

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
cd Storage\strgprowl

# Build for Windows
go build -o strgprowl.exe .
```

### Cross-compile

```powershell
# Linux
$env:GOOS = "linux"; go build -o strgprowl .

# macOS
$env:GOOS = "darwin"; go build -o strgprowl .
```

---

## Usage

### Web UI mode (default)

Run with no credential flags to launch the browser-based interface:

```powershell
.\strgprowl.exe
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
.\strgprowl.exe -addr 0.0.0.0 -port 9000

# Start server without auto-opening browser
.\strgprowl.exe -no-open
```

---

### Headless CLI mode

Provide credential flags to skip the web UI and run storage recon directly to stdout:

```
strgprowl -refresh-token <token> -client-id <id> -tenant-id <id> [flags]
```

| Flag | Required | Default | Description |
|---|---|---|---|
| `-refresh-token` | Yes | — | OAuth refresh token |
| `-client-id` | Yes | — | Azure AD Application (Client) ID |
| `-tenant-id` | Yes | — | Azure AD Tenant ID (GUID) |
| `-account` | No | _(all)_ | Target a specific storage account; omit to enumerate all |
| `-output` | No | `./StorageReconOutput` | Directory for saved output files |
| `-blobs` | No | `true` | Enumerate blobs inside each container |

```powershell
# Enumerate all storage accounts in a subscription
.\strgprowl.exe -refresh-token "0.AROA..." -client-id "<appId>" -tenant-id "<tenantId>"

# Target one account, skip blob enumeration, save to custom path
.\strgprowl.exe -refresh-token "0.AROA..." -client-id "<appId>" -tenant-id "<tenantId>" `
    -account "mystorageaccount" -blobs=false -output "C:\Loot\Storage"

# Tee output to a log file
.\strgprowl.exe -refresh-token "0.AROA..." -client-id "<appId>" -tenant-id "<tenantId>" `
    | Tee-Object recon.log

# Show help
.\strgprowl.exe -h
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
cd Storage\strgprowl
go build -o strgprowl.exe .
```

### Step 3 — Obtain a refresh token

You need a valid OAuth refresh token for the target tenant. See `Authentication/` for methods.  
The token must have access to `management.azure.com` and `storage.azure.com`.

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
# If using a known first-party client ID (e.g. Azure CLI):
$clientId = "04b07795-8ddb-461a-bbee-02f9e1bf7b46"   # Azure CLI public client

# Tenant ID from an existing session
(Get-AzContext).Tenant.Id
```

### Step 5 — Run recon (Web UI)

```powershell
.\strgprowl.exe
```

1. Open `http://127.0.0.1:8080` (auto-opened on Windows)
2. Fill in **Tenant ID**, **Client ID**, and **Refresh Token** — they are saved automatically
3. Click **▶ Run Recon** on the Recon tab
4. Watch the live log stream; output files are saved to `./StorageReconOutput/`
5. Switch to the **Blob Browser** tab to interactively navigate and download files
6. Switch to the **Tokens** tab to inspect and copy the raw or decoded ARM / Storage JWTs

### Step 6 — Run recon (Headless)

```powershell
.\strgprowl.exe `
    -refresh-token $refreshToken `
    -client-id "04b07795-8ddb-461a-bbee-02f9e1bf7b46" `
    -tenant-id $tenantId
```

---

## Output

Each storage account produces a text file at `{OutputPath}/{accountName}-storage.txt` containing:

- Account metadata header
- RBAC permissions (actions / notActions / dataActions / notDataActions)
- Each container with public access level and last-modified timestamp
- Each blob with name, size, content type, and last-modified timestamp

---

## Required Permissions

| Permission | Purpose |
|---|---|
| ARM **Reader** on the subscription | Enumerate storage accounts, containers (via ARM management plane), and RBAC permissions |
| Any role that grants `Microsoft.Storage/storageAccounts/read` | List accounts |
| **Storage Blob Data Reader** _(or equivalent)_ | Enumerate and download blob contents (data-plane) |

> Containers are listed via the ARM management API (`/blobServices/default/containers`) rather than the data-plane API, so Storage Blob Data Reader is **not** required for container enumeration — only for blob listing and content access.

---

## Architecture

```
strgprowl/
├── main.go           — Entry point; flag parsing; selects web UI or headless mode
├── server.go         — HTTP server, session store, all API route handlers
├── storagerecon.go   — 6-phase orchestrator (tokens → sub → accounts → RBAC → containers → blobs)
├── storage.go        — Storage data-plane helpers (list containers/blobs, stream content)
├── arm.go            — ARM helpers (list storage accounts, subscription lookup, paginated GET)
├── tokens.go         — OAuth refresh token exchange
├── util.go           — Shared utilities (JSON helpers, resource group parser)
└── ui/
    └── index.html    — Embedded single-page web UI
```

---

## Relationship to azprowl

`strgprowl` is a focused, standalone version of the Storage Recon module from `azprowl` (see `KeyVault/azprowl/`).  
It uses the same 6-phase reconnaissance flow and identical API calls, but removes the Key Vault, Graph, and device-code phishing modules to produce a smaller, single-purpose binary. The Blob Browser and Tokens tab are extensions not present in the original azprowl Storage tab.
