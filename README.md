# AzureRedTeam

A collection of Azure Red Team notes, cheat sheets, and PowerShell scripts for enumerating and interacting with Azure and Microsoft 365 environments.

---

## 📂 Repository Structure

### [Authentication](./Authentication/)

Guides and scripts for authenticating to Azure and Microsoft services using access tokens, certificates, and service principals.

- **[AuthenticationCheatSheet.md](./Authentication/AuthenticationCheatSheet.md)** — Quick-reference commands for connecting to Microsoft Graph with an access token, authenticating to Azure Resource Manager (ARM), connecting with both ARM and Key Vault tokens simultaneously, and extracting Key Vault certificates to PFX files.
- **[Connect-AzureSPWithCertificate.ps1](./Authentication/Connect-AzureSPWithCertificate.ps1)** — PowerShell script to authenticate to Azure using a Service Principal with certificate-based authentication (`.pfx`). Useful for automation, CI/CD pipelines, and non-interactive login scenarios.

---

### [Enumeration](./Enumeration/)

Comprehensive cheat sheets for enumerating Azure resources and Microsoft 365 / Entra ID objects.

- **[AzureARMEnumeration.md](./Enumeration/AzureARMEnumeration.md)** — Azure Resource Manager (Az module) enumeration cheat sheet covering subscriptions, resource groups, virtual machines, storage accounts, blob storage, Key Vaults, networking, NSGs, SQL servers, RBAC role assignments, managed identities, and result export.
- **[MicrosoftGraphEnumeration.md](./Enumeration/MicrosoftGraphEnumeration.md)** — Microsoft Graph PowerShell enumeration cheat sheet covering users, groups, applications, service principals, devices, directory roles, conditional access policies, mail, calendars, Teams, SharePoint, OneDrive, and result export.

---

### [KeyVault](./KeyVault/)

Scripts for obtaining Key Vault access tokens and auditing Key Vault RBAC permissions.

- **[Get-AzureKeyVaultAccessTokenFromRefreshToken.ps1](./KeyVault/Get-AzureKeyVaultAccessTokenFromRefreshToken.ps1)** — Exchanges an Azure AD OAuth refresh token for a new access token scoped to Azure Key Vault (`https://vault.azure.net/.default`) via the Microsoft identity platform token endpoint.
- **[Get-KeyVaultRBACPermissions.ps1](./KeyVault/Get-KeyVaultRBACPermissions.ps1)** — Queries the Azure Resource Manager (ARM) REST API to retrieve RBAC permission assignments configured on an Azure Key Vault. Uses the current Azure context to identify the vault and authenticates with a management bearer token.

---

### [Storage](./Storage/)

Scripts for acquiring Azure Storage tokens and auditing Storage Account RBAC permissions.

- **[Get-AzureBlobStorageTokenAndListContainers.ps1](./Storage/Get-AzureBlobStorageTokenAndListContainers.ps1)** — Generates a signed JWT client assertion, exchanges it for an Azure Storage access token via the OAuth 2.0 client credentials flow, and lists blob containers on a target storage account through the Blob Storage REST API.
- **[Get-StorageAccountRBACPermissions.ps1](./Storage/Get-StorageAccountRBACPermissions.ps1)** — Queries the Azure Resource Manager (ARM) REST API to retrieve RBAC permission assignments on an Azure Storage Account. Authenticates with a management bearer token and returns the permission details.