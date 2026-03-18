# Azure App Registration vs Service Principal

## 🧠 Short Answer
You don’t *really* create a service principal *from* an app —  
you create a **service principal as the identity instance of that app in a tenant**.

---

## 🏗️ Mental Model

| Object | What it is | Purpose |
|------|--------|--------|
| **Application (App Registration)** | Blueprint / definition | Defines permissions, auth methods |
| **Service Principal** | Identity in a tenant | Used to authenticate + access resources |

---

## 🎯 Why You Need Both

### 🔹 Application = “what the app is”
- Client ID
- API permissions (Graph, etc.)
- Redirect URIs
- Credentials (secret/cert)

### 🔹 Service Principal = “who is acting”
- Gets RBAC roles (Contributor, Reader, etc.)
- Shows up in logs
- Used by `Connect-AzAccount`
- Exists per tenant

---

## 🔥 Critical Point
**RBAC is assigned to the Service Principal — NOT the Application**

---

## 📦 Example Workflow

### Step 1: Create app
```powershell
$app = New-MgApplication -DisplayName "MySP"
```

### Step 2: Create service principal
```powershell
$sp = New-MgServicePrincipal -AppId $app.AppId
```

### Step 3: Assign RBAC
```powershell
New-AzRoleAssignment -ObjectId $sp.Id -Role Contributor -Scope /subscriptions/xxx
```

### Step 4: Authenticate
```powershell
Connect-AzAccount -ServicePrincipal ...
```

---

## 🧩 Why Microsoft Designed It This Way

### 1. Multi-tenant support
- One app → many tenants
- Each tenant gets its own service principal

### 2. Separation of concerns
- App = configuration
- SP = identity + access

### 3. Security boundary
- Permissions can differ per tenant

---

## ⚡ Shortcut (Azure CLI)
```powershell
az ad sp create-for-rbac
```

This creates:
- Application
- Service Principal
- RBAC assignment

---

## 🧠 TL;DR
- App = definition  
- Service Principal = identity  
- RBAC attaches to SP  
- You need both to authenticate and access Azure
