# Service Principal Creation & Authentication Walkthrough

## Overview
This walkthrough demonstrates how to:
- Create an application registration
- Create a service principal
- Generate a client secret
- Authenticate to Azure using the service principal

> Use only in authorized environments.

---

## 1. Create Application Registration

```powershell
$app = New-MgApplication -DisplayName "BackDoorSPDemo"
```

---

## 2. Create Service Principal

```powershell
$sp = New-MgServicePrincipal -AppId $app.AppId
```

---

## 3. Add Client Secret

```powershell
$secret = Add-MgApplicationPassword -ApplicationId $app.Id
$secret.SecretText
```

> Save the secret securely — it is only shown once.

---

## 4. Create Credential Object

Replace CLIENT_ID and SECRET with actual values:

```powershell
$cred = New-Object PSCredential(
    "APP_ID",
    (ConvertTo-SecureString "SECRET" -AsPlainText -Force)
)
```

---

## 5. Authenticate to Azure

```powershell
Connect-AzAccount -ServicePrincipal -Credential $cred -TenantId "146ea19f-5e54-4e83-b037-e06518439cef"
```

---

## 6. Verify Connection

```powershell
Get-AzContext
```

---

## Key Notes
- CLIENT_ID = AppId from the application
- SECRET = value returned from SecretText
- Tenant ID = your Entra tenant

---

## Full Script

```powershell
$app = New-MgApplication -DisplayName "BackDoorSPDemo"
$sp  = New-MgServicePrincipal -AppId $app.AppId
$secret = Add-MgApplicationPassword -ApplicationId $app.Id
$secret.SecretText

$cred = New-Object PSCredential(
    "CLIENT_ID",
    (ConvertTo-SecureString "SECRET" -AsPlainText -Force)
)

Connect-AzAccount -ServicePrincipal -Credential $cred -TenantId "146ea19f-5e54-4e83-b037-e06518439cef"
Get-AzContext
```
