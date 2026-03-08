# Authentication CheatSheet
## Connect to MgGraph with Access Token
$GraphAccessToken variable would hold your JWT
```
Connect-MgGraph -AccessToken ($GraphAccessToken | ConvertTo-SecureString -AsPlainText -Force)

```
## Connect to Azure Manager CLI with Access Token
$ARM will hold you JWT
```
Connect-AzAccount -AccessToken $ARM -AccountId JaredGraff@domain.onmicrosoft.com -Tenant d6bd5a42-7c65-421c-ad23-a25a5d5fa57f
```

## Connect to Azure Account with Primary Access Token and Key Vault Access Token
$ARM will hold JWT for ARM and $KeyVaultAccessToken will hold JWT for the Key Vault
```
Connect-AzAccount -AccessToken $ARM -KeyVaultAccessToken $KeyVaultToken -AccountId jaredgraff@domain.onmicrosoft.com
```

## Pull Certificate from Key Vault and Convert to PFX file
```
#View Certificate in Vault
Get-AzKeyVaultCertificate -VaultName GISAppvault

#Retreive secret from vault
$secret = Get-AzKeyVaultSecret -VaultName GISAppvault -Name GISAppCert -AsPlainText

#Convert Secret from Base64
$secretByte = [Convert]::FromBase64String($secret)

#Save Converted secret to pfx file
[System.IO.File]::WriteAllBytes("C:\Users\studentuser96\GISAppcert.pfx", $secretByte)

```
