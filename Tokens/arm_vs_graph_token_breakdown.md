# ARM Token vs Graph Token (User Perspective -- Javier)

## Key Concept

Azure uses **two different authorization models**:

-   **Microsoft Graph → App-based permissions (scp)**
-   **Azure Resource Manager (ARM) → RBAC roles (roles)**

These result in **different token structures and capabilities**

------------------------------------------------------------------------

## How Javier Gets a Graph Token

1.  Javier signs into an application (Azure Portal, GraphSpy, Teams,
    etc.)
2.  The application has **Graph API permissions configured**
3.  Javier consents (or admin consent is pre-approved)
4.  Azure issues a token containing:

-   `aud = https://graph.microsoft.com`
-   `scp = permissions defined by the app`

### Example

"scp": "User.Read Mail.Read"

------------------------------------------------------------------------

## How Javier Gets an ARM Token

1.  Javier authenticates to Azure
2.  Azure evaluates **RBAC role assignments**
3.  Based on scope (subscription, resource group, resource)
4.  Azure issues a token containing:

-   `aud = https://management.azure.com`
-   `roles = RBAC roles assigned to Javier`

### Example

"roles": \["Reader", "Contributor"\]

------------------------------------------------------------------------

## Core Difference

  Component           Graph Token        ARM Token
  ------------------- ------------------ -----------------------
  Permission Source   App Registration   Azure RBAC
  Token Field         scp                roles
  Model               App-centric        User/resource-centric
  Scope Type          API permissions    Resource hierarchy
  Example Resource    Graph API          Azure infrastructure

------------------------------------------------------------------------

## What the Token Represents

### Graph Token

User (Javier) + Application + API Permissions = Token with `scp`

------------------------------------------------------------------------

### ARM Token

User (Javier) + RBAC Role Assignments + Scope (subscription/resource) =
Token with `roles`

------------------------------------------------------------------------

## Why This Matters

-   Graph token → Access to **data (mail, files, directory)**
-   ARM token → Access to **infrastructure (VMs, storage, Key Vaults)**

They are **not interchangeable**

------------------------------------------------------------------------

## Red Team Perspective

### Graph Token

-   Used for:
    -   Email access
    -   File access (OneDrive/SharePoint)
    -   Directory enumeration
-   Abuse paths:
    -   OAuth consent phishing
    -   App registration abuse

------------------------------------------------------------------------

### ARM Token

-   Used for:
    -   Enumerating Azure resources
    -   Modifying infrastructure
    -   Discovering Key Vaults, storage accounts
-   Abuse paths:
    -   Privilege escalation via RBAC
    -   Resource manipulation

------------------------------------------------------------------------

## Attack Chain Example

Graph Token → Identity Recon ↓ ARM Token → Resource Discovery ↓ Service
Token (Key Vault / Storage) → Data Access

------------------------------------------------------------------------

## Key Takeaways

-   Graph = **App permissions (scp)**
-   ARM = **RBAC roles (roles)**
-   Same user → different tokens → different capabilities
-   Always check:
    -   `aud`
    -   `scp` vs `roles`
