# kv-recon

All-in-one Azure Key Vault reconnaissance CLI tool.  
Exchanges an OAuth refresh token for ARM and Key Vault access tokens, enumerates vaults, checks RBAC permissions, retrieves secrets, and exports certificates — all from a single compiled binary with no PowerShell, Az module, or C compiler dependency.

---

## Features

- CLI interface — all inputs passed as flags; no GUI or C compiler required
- Exchanges a refresh token for both an ARM token and a Key Vault token
- Discovers the active subscription automatically
- Enumerates all Key Vaults in the subscription (or targets a specific vault)
- Checks effective RBAC permissions on each vault via the ARM REST API
- Lists and retrieves all secret values with metadata (enabled state, expiry, content type)
- Enumerates certificates and exports each as:
  - **PFX** (PKCS#12, includes private key when stored in Key Vault)
  - **PEM** (public certificate)
  - Falls back to public-cert-only PEM if secrets endpoint access is denied
- Streams all output live to stdout
- Saves secrets to `{vault}-secrets.txt` and certificates to `.pfx` / `.pem` files
- Cross-platform: Windows, macOS, Linux (pure Go — no CGo, no external toolchain)

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

All output is written to stdout. Secrets and certificates are also saved to the output directory.

### Examples

```powershell
# Enumerate all vaults in a subscription
.\kv-recon.exe -refresh-token "0.AROA..." -client-id "<appId>" -tenant-id "<tenantId>"

# Target a specific vault and redirect output to a log file
.\kv-recon.exe -refresh-token "0.AROA..." -client-id "<appId>" -tenant-id "<tenantId>" `
    -vault-name "my-vault" -output "C:\Loot\KVRecon" | Tee-Object recon.log

# Show help
.\kv-recon.exe -h
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
  main.go       — CLI entry point: flag parsing, stdin validation, stdout log drain
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
