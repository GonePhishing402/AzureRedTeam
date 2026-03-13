## Create Azure Management Token for Application
```powershell
$BlueTestingMgmtToken = New-AccessToken -clientCertificate $AppCertificate -tenantID $TenantId -appID $ClientId -scope 'https://management.azure.com/.default'
```

## Login with that Token to the Application
```powershell
Connect-AzAccount -AccessToken $BlueTestingMgmtToken -AccountId 9d1e53c7-28f6-4d1d-8d8a-1abeb4db2888
```

## Create a Key Vault Token for Data Plane Access with that Application - RBAC Needed for this to be successful
```powershell
$BlueTestingKeyVaultToken = New-AccessToken -clientCertificate $AppCertificate -tenantID $TenantId -appID $ClientId -scope 'https://vault.azure.net/.default'
```

## Login with Key Vault Data Plan Access for that Application
```powershell
Connect-AzAccount -AccessToken $BlueTestingMgmtToken -KeyVaultAccessToken $BlueTestingKeyVaultToken -AccountId 9d1e53c7-28f6-4d1d-8d8a-1abeb4db2888
```
  
