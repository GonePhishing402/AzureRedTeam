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

- **[AuthenticatewithBackDoorSP.ps1](./Authentication/AuthenticatewithBackDoorSP.ps1)** — Authenticates to Microsoft Graph as a Service Principal by prompting for tenant ID, client ID, and client secret, then connecting via `Connect-MgGraph` using a `PSCredential` object. Useful for establishing persistent access using a backdoor service principal.
- **[AuthenticationCheatSheet.md](./Authentication/AuthenticationCheatSheet.md)** — Quick-reference commands for connecting to Microsoft Graph with an access token, authenticating to Azure Resource Manager (ARM), connecting with both ARM and Key Vault tokens simultaneously, and extracting Key Vault certificates to PFX files.
- **[Connect-AzureSPWithCertificate.ps1](./Authentication/Connect-AzureSPWithCertificate.ps1)** — PowerShell script to authenticate to Azure using a Service Principal with certificate-based authentication (`.pfx`). Useful for automation, CI/CD pipelines, and non-interactive login scenarios.
- **[ServicePrincipal.md](./Authentication/ServicePrincipal.md)** — Cheat sheet for creating a backdoor service principal: registering a new application, instantiating a service principal from it, and generating a client secret — all via Microsoft Graph PowerShell (`New-MgApplication`, `New-MgServicePrincipal`, `Add-MgApplicationPassword`).
- **[service_principal_walkthrough.md](./Authentication/service_principal_walkthrough.md)** — Step-by-step walkthrough covering the full service principal lifecycle: creating an app registration with `New-MgApplication`, instantiating a service principal, generating a client secret, building a `PSCredential` object, and authenticating to Azure via `Connect-AzAccount -ServicePrincipal`. Includes a consolidated full script for all steps.
- **[azure_app_vs_sp.md](./Authentication/azure_app_vs_sp.md)** — Reference explaining the distinction between Azure App Registrations (blueprints defining permissions and credentials) and Service Principals (per-tenant identity instances used to authenticate and receive RBAC roles). Covers the design rationale, an example creation workflow, and why RBAC is assigned to the Service Principal — not the Application.

---

### [Enumeration](./Enumeration/)

Comprehensive cheat sheets for enumerating Azure resources and Microsoft 365 / Entra ID objects.

- **[AzureARMEnumeration.md](./Enumeration/AzureARMEnumeration.md)** — Azure Resource Manager (Az module) enumeration cheat sheet covering subscriptions, resource groups, virtual machines, storage accounts, blob storage, Key Vaults, networking, NSGs, SQL servers, RBAC role assignments, managed identities, and result export.
- **[Get-AUUserMembers.ps1](./Enumeration/Get-AUUserMembers.ps1)** — Enumerates members of a Microsoft Entra ID Administrative Unit (AU) using Microsoft Graph PowerShell. Useful during red team operations to identify users within the scope of AU-delegated roles (e.g., Helpdesk Administrator, Password Administrator) to find privilege escalation or lateral movement paths.
- **[MicrosoftGraphEnumeration.md](./Enumeration/MicrosoftGraphEnumeration.md)** — Microsoft Graph PowerShell enumeration cheat sheet covering users, groups, applications, service principals, devices, directory roles, conditional access policies, mail, calendars, Teams, SharePoint, OneDrive, and result export.

---

### [KeyVault](./KeyVault/)

Scripts for obtaining Key Vault access tokens and auditing Key Vault RBAC permissions, plus the `azprowl` all-in-one Azure reconnaissance platform.

- **[Azure_KeyVault_RedTeam_Walkthrough_RCCE.md](./KeyVault/Azure_KeyVault_RedTeam_Walkthrough_RCCE.md)** — Red team walkthrough demonstrating the full Key Vault token abuse chain against an RCCE lab environment: authenticating to Azure with a captured ARM access token, enumerating Key Vaults, acquiring a separate Key Vault data-plane token scoped to `vault.azure.net`, re-authenticating with both tokens via `Connect-AzAccount`, and extracting plaintext secret values. Highlights the control-plane vs. data-plane token separation that requires operators to obtain a second resource-specific token for data access.
- **[Get-AzureKeyVaultAccessTokenFromRefreshToken.ps1](./KeyVault/Get-AzureKeyVaultAccessTokenFromRefreshToken.ps1)** — Exchanges an Azure AD OAuth refresh token for a new access token scoped to Azure Key Vault (`https://vault.azure.net/.default`) via the Microsoft identity platform token endpoint.
- **[Get-KeyVaultRBACPermissions.ps1](./KeyVault/Get-KeyVaultRBACPermissions.ps1)** — Queries the Azure Resource Manager (ARM) REST API to retrieve RBAC permission assignments configured on an Azure Key Vault. Uses the current Azure context to identify the vault and authenticates with a management bearer token.
- **[Invoke-KeyVaultRecon.ps1](./KeyVault/Invoke-KeyVaultRecon.ps1)** — All-in-one Azure Key Vault reconnaissance script. Exchanges an OAuth refresh token for both ARM and Key Vault-scoped access tokens, authenticates an Az PowerShell context, enumerates all accessible Key Vaults in the subscription (or a targeted vault), checks RBAC permissions per vault via the ARM REST API, retrieves all secret values via the Key Vault REST API, and exports certificates as both PFX (with private key) and PEM files. All results are written to screen and saved to disk under a configurable output path.
- **[SecretValue.ps1](./KeyVault/SecretValue.ps1)** — Retrieves a secret value from Azure Key Vault using the Key Vault REST API with a bearer token. Sends a GET request to the Key Vault secrets endpoint with a Key Vault-scoped JWT access token.
- **[azure_keyvault_access_explanation.md](./KeyVault/azure_keyvault_access_explanation.md)** — Deep-dive reference explaining why Azure Key Vault separates management-plane (ARM) access from data-plane access. Covers the two API endpoints, required credentials for data-plane access, Key Vault RBAC data roles, and a red-team perspective on requesting resource-specific tokens from a captured refresh token.
- **[azure_keyvault_redteam_walkthrough.md](./KeyVault/azure_keyvault_redteam_walkthrough.md)** — Step-by-step red team walkthrough for Azure Key Vault token theft: acquiring a Key Vault data-plane access token from a refresh token, authenticating with both ARM and Key Vault tokens, enumerating vaults and secrets, and extracting secret values via both the Key Vault REST API and native `Get-AzKeyVaultSecret` PowerShell.
- **[azprowl/](./KeyVault/azprowl/)** — All-in-one Azure red team reconnaissance platform (pure Go, single binary). Provides a browser-based web UI and headless CLI for Key Vault recon, Azure Blob Storage enumeration, Microsoft Graph enumeration (users, devices, applications, service principals, groups, OneDrive, mail, Teams, SharePoint), FOCI token exchange, device code phishing, and persistent token storage. See the [azprowl README](./KeyVault/azprowl/README.md) for full usage.

---

### [PowerShell](./PowerShell/)

General PowerShell reference material for operators building Azure and Microsoft 365 automation and attack workflows.

- **[powershell_cheatsheet.md](./PowerShell/powershell_cheatsheet.md)** — General-purpose PowerShell cheat sheet covering cmdlet syntax (Verb-Noun), variables, the pipeline, object manipulation, filtering, loops, conditionals, file operations, module management (`Install-Module`, `Import-Module`), REST API calls with `Invoke-RestMethod`, JSON handling, and red team / cloud use cases including Azure and Microsoft Graph automation.

---

### [Storage](./Storage/)

Scripts for acquiring Azure Storage tokens and auditing Storage Account RBAC permissions.

- **[Get-AzureBlobStorageTokenAndListContainers.ps1](./Storage/Get-AzureBlobStorageTokenAndListContainers.ps1)** — Generates a signed JWT client assertion, exchanges it for an Azure Storage access token via the OAuth 2.0 client credentials flow, and lists blob containers on a target storage account through the Blob Storage REST API.
- **[Get-AzureStorageAccessTokenFromRefreshToken.ps1](./Storage/Get-AzureStorageAccessTokenFromRefreshToken.ps1)** — Exchanges an Azure AD OAuth refresh token for a new access token scoped to Azure Storage (`https://storage.azure.com/.default`) via the Microsoft identity platform token endpoint.
- **[Get-StorageAccountRBACPermissions.ps1](./Storage/Get-StorageAccountRBACPermissions.ps1)** — Queries the Azure Resource Manager (ARM) REST API to retrieve RBAC permission assignments on an Azure Storage Account. Authenticates with a management bearer token and returns the permission details.
- **[Invoke-StorageRecon.ps1](./Storage/Invoke-StorageRecon.ps1)** — Full 6-phase Azure Storage reconnaissance script driven by an OAuth refresh token. Phases cover token acquisition (ARM + Storage bearer tokens), subscription discovery, storage account enumeration, RBAC permission checks per account, container enumeration via the ARM management plane (Storage Blob Data Reader not required), and blob enumeration via the data-plane Storage API. Results are saved as one text file per storage account under a configurable output path.
- **[StorageContainerEnum.ps1](./Storage/StorageContainerEnum.ps1)** — Lists blob containers on an Azure Storage Account using the Blob Storage REST API with a storage-scoped bearer token and the `x-ms-version` header.
- **[azure_storage_token_explanation.md](./Storage/azure_storage_token_explanation.md)** — Deep-dive reference explaining why Azure Storage separates management-plane (ARM) access from data-plane access. Covers storage credentials (access keys, SAS tokens, OAuth), Storage Blob data roles, and a red-team perspective on requesting resource-specific tokens from a captured refresh token.
- **[strgprowl/](./Storage/strgprowl/)** — Focused, standalone Azure Storage reconnaissance tool (pure Go, single binary) with a browser-based web UI and headless CLI mode. Mirrors the 6-phase StorageRecon flow from `azprowl` — token exchange, subscription discovery, storage account enumeration, RBAC checks, container listing, and blob enumeration via the ARM management plane and Storage data-plane — and adds a Blob Browser tab for interactively navigating accounts, containers, and blobs with inline view and download capabilities. See the [strgprowl README](./Storage/strgprowl/README.md) for full usage.

---

### [Tokens](./Tokens/)

Scripts and references for acquiring, manipulating, and abusing Azure access tokens — a core focus area for red team operations targeting Azure and Microsoft 365 environments.

- **[New-AccessToken.ps1](./Tokens/New-AccessToken.ps1)** — PowerShell function (sourced from Hunt & Hackett research) that generates a signed JWT client assertion using a client certificate and exchanges it for an Azure AD access token via the OAuth 2.0 client credentials flow. Supports configurable scope (defaults to Microsoft Graph).
- **[NewAccessTokenCheatSheet.md](./Tokens/NewAccessTokenCheatSheet.md)** — Cheat sheet demonstrating how to use the `New-AccessToken` function to create Azure Management and Key Vault tokens for an application, connect to Azure with those tokens, and access Key Vault data-plane resources.
- **[arm_vs_graph_token_breakdown.md](./Tokens/arm_vs_graph_token_breakdown.md)** — Deep-dive reference comparing ARM tokens (RBAC-based, `roles` claim, `management.azure.com` audience) vs. Microsoft Graph tokens (app permission-based, `scp` claim, `graph.microsoft.com` audience). Covers token structure differences, red team abuse paths for each token type, and a full attack chain from Graph token through ARM token to service-specific data access.
- **[graph_permissions_breakdown.md](./Tokens/graph_permissions_breakdown.md)** — Reference explaining how users obtain Microsoft Graph permissions indirectly through App Registrations and OAuth consent flows, detailing the `scp` (scope) claim, the difference between Graph's app-centric model and ARM's RBAC model, and red team targeting of high-value permissions such as `Mail.Read`, `Files.Read.All`, and `Directory.Read.All`.