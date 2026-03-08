
# Why You Need a Storage Access Token in Azure (Even If You Can Read the Storage Account)

Short answer: **because Azure Storage uses a separate data-plane authorization model.**  
Even if you can *see* or *read metadata about* a storage account in Azure (**management plane**), you still need a **data-plane token** to actually access the blobs/files/tables inside it.

---

# Azure Storage Access: Management Plane vs Data Plane

Azure separates access into **two different APIs**.

| Plane | What it controls | API endpoint | Example |
|------|------------------|--------------|---------|
| **Management Plane** | Managing the storage account resource | `management.azure.com` | Create/delete storage account |
| **Data Plane** | Accessing the actual data inside storage | `*.blob.core.windows.net` | Read blobs |

---

# 1. Management Plane (ARM)

This is what **Azure RBAC roles like `Reader`** typically give you access to.

Example actions:

- View storage account
- See configuration
- View networking settings
- View resource metadata
- Modify settings (if higher role)

Example command:

```powershell
Get-AzStorageAccount
```

Example API:

```
https://management.azure.com/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Storage/storageAccounts/{name}
```

⚠️ This **does NOT give access to the blob data itself.**

---

# 2. Data Plane (Storage API)

This is the **actual storage service**.

Endpoints look like:

```
https://<storageaccount>.blob.core.windows.net
```

Example actions:

- List containers
- Download blobs
- Upload blobs
- Delete blobs
- Read files

These require **one of the following credentials**.

| Credential Type | Example |
|----------------|--------|
| Storage Access Key | Account key |
| SAS Token | Shared Access Signature |
| Azure AD OAuth token | `https://storage.azure.com/.default` |
| Managed Identity token | Same scope |

---

# Example

Even if you can run this:

```powershell
Get-AzStorageAccount
```

You still **cannot do this** without a data-plane credential:

```powershell
Invoke-RestMethod https://storageaccount.blob.core.windows.net/container/file
```

Unless you have:

- Storage key
- SAS token
- **Storage OAuth token**

---

# Why the Storage OAuth Token Exists

Azure Storage supports **Azure AD authentication** for data access.

You request a token for the resource:

```
https://storage.azure.com/.default
```

Example:

```powershell
$token = (Get-AzAccessToken -ResourceUrl "https://storage.azure.com").Token
```

Then call the storage API using the token:

```
Authorization: Bearer <token>
```

---

# Example Flow

### 1. Login to Azure

```powershell
Connect-AzAccount
```

### 2. Get a storage access token

```powershell
Get-AzAccessToken -ResourceUrl https://storage.azure.com
```

### 3. Call the storage API

```
https://storageaccount.blob.core.windows.net
```

---

# Why Azure Separates These

**Security.**

Without this separation, **anyone with `Reader` on the subscription could download all storage data.**

Instead, Azure uses **data-plane roles**.

| Role | Can Access Blob Data |
|-----|----------------------|
Reader | ❌ No |
Contributor | ❌ No |
Storage Blob Data Reader | ✅ Yes |
Storage Blob Data Contributor | ✅ Yes |

---

# Example Storage Data Roles

| Role | Access |
|-----|--------|
Storage Blob Data Reader | Read blobs |
Storage Blob Data Contributor | Read/write blobs |
Storage Blob Data Owner | Full control |

---

# Quick Visual

```
Azure Portal / ARM
        │
        │  (Management Token)
        ▼
management.azure.com
        │
        │  Manage storage account
        ▼
Storage Account Resource
        │
        │  (Storage Token Required)
        ▼
blob.core.windows.net
        │
        ▼
Actual Blob Data
```

---

# Why This Matters for Security / Red Teaming

Azure tokens are **resource-specific**.

Examples:

| Token Scope | Access |
|-------------|--------|
management.azure.com | Manage Azure resources |
storage.azure.com | Access blob data |
vault.azure.net | Access Key Vault secrets |
graph.microsoft.com | Access Entra ID / M365 |

A user may therefore have:

- ✔ **ARM access**
- ❌ **Storage data access**

Or the reverse.

---

# Example Attack Scenario

If an attacker obtains a **refresh token**, they can request tokens for multiple services:

```
https://management.azure.com
https://storage.azure.com
https://vault.azure.net
https://graph.microsoft.com
```

Each token grants access to **different services**.

---

# Key Takeaway

Being able to **read a storage account resource** does **not mean you can read the data inside it**.

You still need **data-plane authorization**, which typically requires:

- Storage RBAC roles
- Storage keys
- SAS tokens
- Azure AD OAuth tokens scoped to `storage.azure.com`
