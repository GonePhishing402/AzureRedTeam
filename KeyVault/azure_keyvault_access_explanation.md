# Why You Need a Key Vault Access Token in Azure (Even If You Can See the Vault)

Short answer: **Azure Key Vault separates management access from
secret/key access.**\
Even if you can *see or manage the Key Vault resource* through Azure
Resource Manager (ARM), you still need a **data‑plane token** to access
the **secrets, keys, or certificates stored inside the vault**.

------------------------------------------------------------------------

# Azure Key Vault Access: Management Plane vs Data Plane

Azure Key Vault uses two different APIs with separate authorization
models.

  ---------------------------------------------------------------------------------
  Plane          What it controls            API endpoint             Example
  -------------- --------------------------- ------------------------ -------------
  **Management   Manage the Key Vault        `management.azure.com`   Create vault,
  Plane**        resource                                             configure
                                                                      networking

  **Data Plane** Access secrets, keys,       `vault.azure.net`        Read secrets
                 certificates                                         or keys
  ---------------------------------------------------------------------------------

------------------------------------------------------------------------

# 1. Management Plane (ARM)

This is what most **Azure RBAC roles** control.

Example actions:

-   Create or delete a Key Vault
-   Configure networking
-   Configure RBAC policies
-   Configure access policies
-   Enable logging
-   Configure firewall rules

Example command:

``` powershell
Get-AzKeyVault
```

Example API:

    https://management.azure.com/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.KeyVault/vaults/{vault}

⚠️ This **does NOT allow you to read secrets stored inside the vault.**

------------------------------------------------------------------------

# 2. Data Plane (Key Vault Service)

The **Key Vault data plane** is where the **actual sensitive data
lives**.

Endpoints look like:

    https://<vault-name>.vault.azure.net

Example operations:

-   Read secrets
-   List secrets
-   Retrieve certificates
-   Encrypt/decrypt data
-   Sign keys

Example API:

    https://<vault-name>.vault.azure.net/secrets

------------------------------------------------------------------------

# Required Credentials for Data Plane Access

Access to Key Vault data requires **one of the following credentials**:

  Credential Type             Example
  --------------------------- ------------------------------------
  Azure AD OAuth token        `https://vault.azure.net/.default`
  Managed Identity            VM or service identity
  Key Vault access policy     Legacy permission model
  Azure RBAC Key Vault role   Modern authorization model

------------------------------------------------------------------------

# Example Scenario

Even if you can run:

``` powershell
Get-AzKeyVault
```

You still **cannot read secrets** unless you have data-plane
permissions.

This will fail without the correct permissions:

``` powershell
Get-AzKeyVaultSecret -VaultName "myvault"
```

------------------------------------------------------------------------

# Why the Key Vault OAuth Token Exists

Azure Key Vault supports **Azure AD authentication**.

You must request a token scoped for Key Vault:

    https://vault.azure.net/.default

Example:

``` powershell
$token = (Get-AzAccessToken -ResourceUrl "https://vault.azure.net").Token
```

Then call the Key Vault API:

    Authorization: Bearer <token>

------------------------------------------------------------------------

# Example Flow

### 1. Login to Azure

``` powershell
Connect-AzAccount
```

### 2. Request Key Vault token

``` powershell
Get-AzAccessToken -ResourceUrl https://vault.azure.net
```

### 3. Access vault secrets

    https://<vault-name>.vault.azure.net/secrets

------------------------------------------------------------------------

# Why Azure Separates These

Security.

Without this separation, **anyone with Reader on a subscription could
access all secrets stored in Key Vaults.**

Instead Azure separates permissions.

  Role                      Can Read Secrets
  ------------------------- ------------------
  Reader                    ❌ No
  Contributor               ❌ No
  Key Vault Reader          ❌ Metadata only
  Key Vault Secrets User    ✅ Yes
  Key Vault Administrator   ✅ Full control

------------------------------------------------------------------------

# Key Vault Data Roles

  Role                      Access
  ------------------------- --------------------
  Key Vault Secrets User    Read secrets
  Key Vault Crypto User     Use keys
  Key Vault Administrator   Full vault control

------------------------------------------------------------------------

# Quick Visual

    Azure Portal / ARM
            │
            │  (Management Token)
            ▼
    management.azure.com
            │
            │  Manage Key Vault resource
            ▼
    Key Vault Resource
            │
            │  (Vault Token Required)
            ▼
    vault.azure.net
            │
            ▼
    Secrets / Keys / Certificates

------------------------------------------------------------------------

# Why This Matters for Security / Red Teaming

Azure tokens are **resource specific**.

Examples:

  Token Scope            Access
  ---------------------- ---------------------------------
  management.azure.com   Manage Azure resources
  vault.azure.net        Access Key Vault secrets
  storage.azure.com      Access blob data
  graph.microsoft.com    Access Entra ID / Microsoft 365

A user may therefore have:

-   ✔ ARM access
-   ❌ Secret access

Or the reverse.

------------------------------------------------------------------------

# Example Attack Scenario

If an attacker obtains a **refresh token**, they could request tokens
for multiple services:

    https://management.azure.com
    https://vault.azure.net
    https://storage.azure.com
    https://graph.microsoft.com

Each token grants access to **different resources**.

------------------------------------------------------------------------

# Key Takeaway

Being able to **see or manage a Key Vault resource** does **not mean you
can access the secrets inside it**.

You must still have **data‑plane authorization**, such as:

-   Key Vault RBAC data roles
-   Key Vault access policies
-   OAuth tokens scoped to `vault.azure.net`
-   Managed identities with vault permissions
