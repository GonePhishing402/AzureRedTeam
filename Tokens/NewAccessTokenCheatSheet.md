## Create Azure Management Token for Application
```powershell
$GISAppMgmtToken = New-AccessToken -clientCertificate
$clientCertificate -tenantID d6bd5a42-7c65-421c-ad23-a25a5d5fa57f -appID  2b7c28bd-def1-415a-b407-41627de6e8f1 -scope 'https://management.azure.com/.default'
```

## Login with that Token to the Application
```powershell
Connect-AzAccount -AccessToken $GISAppMgmtToken -AccountId 2b7c28bd-def1-415a-b407-41627de6e8f1
```

## Create a Key Vault Token for Data Plane Access with that Application - RBAC Needed for this to be successful
```powershell
$GISAppKeyVaultToken = New-AccessToken -clientCertificate $clientCertificate -tenantID d6bd5a42-7c65-421c-ad23-a25a5d5fa57f -appID 2b7c28bd-def1-415a-b407-41627de6e8f1 -scope 'https://vault.azure.net/.default'
```

## Login with Key Vault Data Plan Access for that Application
```powershell
Connect-AzAccount -AccessToken $GISAppMgmtToken -KeyVaultAccessToken $GISAppKeyVaultToken -AccountId 2b7c28bd-def1-415a-b407-41627de6e8f1
```
  
