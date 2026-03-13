#Make sure you JWT for Key Vault is stored in the variable and replace the URI with the Key Vault URI - URI is formatted like this: https://<vault-name>.vault.azure.net/secrets/<secret-name>/<version>
Invoke-RestMethod `
  -Uri "https://kvbluerangesecrets.vault.azure.net/secrets/AnotherSecret?api-version=7.4" `
  -Headers @{Authorization = "Bearer $KeyVaultAccessToken"}
