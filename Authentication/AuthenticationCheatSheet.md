# Authentication CheatSheet
## Connect to MgGraph with Access Token
$GraphAccessToken variable would hold your JWT
```powershell
Connect-MgGraph -AccessToken ($GraphAccessToken | ConvertTo-SecureString -AsPlainText -Force)

```
## Connect to Azure Manager CLI with Access Token
$ARM will hold you JWT
```powershell
Connect-AzAccount -AccessToken $ARM -AccountId username -Tenant [TENANTID]
```

## Connect to Azure Account with Primary Access Token and Key Vault Access Token
$ARM will hold JWT for ARM and $KeyVaultAccessToken will hold JWT for the Key Vault
```powershell
Connect-AzAccount -AccessToken $ARM -KeyVaultAccessToken $KeyVaultToken -AccountId jaredgraff@domain.onmicrosoft.com
```

## Pull Certificate from Key Vault and Convert to PFX file
```powershell
#View Certificate in Vault
Get-AzKeyVaultCertificate -VaultName GISAppvault
```
```powershell
#Retreive secret from vault
$secret = Get-AzKeyVaultSecret -VaultName GISAppvault -Name GISAppCert -AsPlainText
```
```powershell
#Convert Secret from Base64
$secretByte = [Convert]::FromBase64String($secret)
```
```powershell
#Save Converted secret to pfx file
[System.IO.File]::WriteAllBytes("C:\Users\studentuser96\GISAppcert.pfx", $secretByte)

```
## Connect to MgGraph with Certificate
```powershell
$PfxPath = "/Path/to/Certificate"
$AppCertificate = Get-PfxCertificate -FilePath $PfxPath
```
```powershell
Connect-MgGraph -Certificate $AppCertificate -ClientId $ClientId -TenantId $TenantId
```
