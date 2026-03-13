# AzureRedTeam

A collection of Azure Red Team notes, cheat sheets, and PowerShell scripts for enumerating and interacting with Azure and Microsoft 365 environments.

---

## 📂 Repository Structure

### [AdministrativeUnit](./AdministrativeUnit/)

Scripts for interacting with Microsoft Entra ID Administrative Units — scoped organizational containers that delegate administrative rights over a subset of users or groups.

- **[Reset-AdminUnitPassword.ps1](./AdministrativeUnit/Reset-AdminUnitPassword.ps1)** — Resets the password for a user scoped within a Microsoft Entra ID Administrative Unit using Microsoft Graph PowerShell (`Update-MgUser`). Requires Helpdesk Administrator, User Administrator, or Global Administrator privileges scoped to the target AU.

---

### [Authentication](./Authentication/)

Guides and scripts for authenticating to Azure and Microsoft services using access tokens, certificates, and service principals.

- **[AuthenticationCheatSheet.md](./Authentication/AuthenticationCheatSheet.md)** — Quick-reference commands for connecting to Microsoft Graph with an access token, authenticating to Azure Resource Manager (ARM), connecting with both ARM and Key Vault tokens simultaneously, and extracting Key Vault certificates to PFX files.
- **[Connect-AzureSPWithCertificate.ps1](./Authentication/Connect-AzureSPWithCertificate.ps1)** — PowerShell script to authenticate to Azure using a Service Principal with certificate-based authentication (`.pfx`). Useful for automation, CI/CD pipelines, and non-interactive login scenarios.

---

### [Enumeration](./Enumeration/)

Comprehensive cheat sheets for enumerating Azure resources and Microsoft 365 / Entra ID objects.

- **[AzureARMEnumeration.md](./Enumeration/AzureARMEnumeration.md)** — Azure Resource Manager (Az module) enumeration cheat sheet covering subscriptions, resource groups, virtual machines, storage accounts, blob storage, Key Vaults, networking, NSGs, SQL servers, RBAC role assignments, managed identities, and result export.
- **[Get-AUUserMembers.ps1](./Enumeration/Get-AUUserMembers.ps1)** — Enumerates members of a Microsoft Entra ID Administrative Unit (AU) using Microsoft Graph PowerShell. Useful during red team operations to identify users within the scope of AU-delegated roles (e.g., Helpdesk Administrator, Password Administrator) to find privilege escalation or lateral movement paths.
- **[MicrosoftGraphEnumeration.md](./Enumeration/MicrosoftGraphEnumeration.md)** — Microsoft Graph PowerShell enumeration cheat sheet covering users, groups, applications, service principals, devices, directory roles, conditional access policies, mail, calendars, Teams, SharePoint, OneDrive, and result export.

---

### [KeyVault](./KeyVault/)

Scripts for obtaining Key Vault access tokens and auditing Key Vault RBAC permissions.

- **[Get-AzureKeyVaultAccessTokenFromRefreshToken.ps1](./KeyVault/Get-AzureKeyVaultAccessTokenFromRefreshToken.ps1)** — Exchanges an Azure AD OAuth refresh token for a new access token scoped to Azure Key Vault (`https://vault.azure.net/.default`) via the Microsoft identity platform token endpoint.
- **[Get-KeyVaultRBACPermissions.ps1](./KeyVault/Get-KeyVaultRBACPermissions.ps1)** — Queries the Azure Resource Manager (ARM) REST API to retrieve RBAC permission assignments configured on an Azure Key Vault. Uses the current Azure context to identify the vault and authenticates with a management bearer token.
- **[SecretValue.ps1](./KeyVault/SecretValue.ps1)** — Retrieves a secret value from Azure Key Vault using the Key Vault REST API with a bearer token. Sends a GET request to the Key Vault secrets endpoint with a Key Vault-scoped JWT access token.
- **[azure_keyvault_access_explanation.md](./KeyVault/azure_keyvault_access_explanation.md)** — Deep-dive reference explaining why Azure Key Vault separates management-plane (ARM) access from data-plane access. Covers the two API endpoints, required credentials for data-plane access, Key Vault RBAC data roles, and a red-team perspective on requesting resource-specific tokens from a captured refresh token.

---

### [Storage](./Storage/)

Scripts for acquiring Azure Storage tokens and auditing Storage Account RBAC permissions.

- **[Get-AzureBlobStorageTokenAndListContainers.ps1](./Storage/Get-AzureBlobStorageTokenAndListContainers.ps1)** — Generates a signed JWT client assertion, exchanges it for an Azure Storage access token via the OAuth 2.0 client credentials flow, and lists blob containers on a target storage account through the Blob Storage REST API.
- **[Get-AzureStorageAccessTokenFromRefreshToken.ps1](./Storage/Get-AzureStorageAccessTokenFromRefreshToken.ps1)** — Exchanges an Azure AD OAuth refresh token for a new access token scoped to Azure Storage (`https://storage.azure.com/.default`) via the Microsoft identity platform token endpoint.
- **[Get-StorageAccountRBACPermissions.ps1](./Storage/Get-StorageAccountRBACPermissions.ps1)** — Queries the Azure Resource Manager (ARM) REST API to retrieve RBAC permission assignments on an Azure Storage Account. Authenticates with a management bearer token and returns the permission details.
- **[StorageContainerEnum.ps1](./Storage/StorageContainerEnum.ps1)** — Lists blob containers on an Azure Storage Account using the Blob Storage REST API with a storage-scoped bearer token and the `x-ms-version` header.
- **[azure_storage_token_explanation.md](./Storage/azure_storage_token_explanation.md)** — Deep-dive reference explaining why Azure Storage separates management-plane (ARM) access from data-plane access. Covers storage credentials (access keys, SAS tokens, OAuth), Storage Blob data roles, and a red-team perspective on requesting resource-specific tokens from a captured refresh token.

---

### [Tokens](./Tokens/)

Scripts and references for acquiring, manipulating, and abusing Azure access tokens — a core focus area for red team operations targeting Azure and Microsoft 365 environments.

- **[New-AccessToken.ps1](./Tokens/New-AccessToken.ps1)** — PowerShell function (sourced from Hunt & Hackett research) that generates a signed JWT client assertion using a client certificate and exchanges it for an Azure AD access token via the OAuth 2.0 client credentials flow. Supports configurable scope (defaults to Microsoft Graph).
- **[NewAccessTokenCheatSheet.md](./Tokens/NewAccessTokenCheatSheet.md)** — Cheat sheet demonstrating how to use the `New-AccessToken` function to create Azure Management and Key Vault tokens for an application, connect to Azure with those tokens, and access Key Vault data-plane resources.