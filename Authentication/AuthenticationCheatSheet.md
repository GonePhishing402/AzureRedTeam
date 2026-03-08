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
