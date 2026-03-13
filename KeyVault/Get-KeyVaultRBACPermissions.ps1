$ApiVersion = "2022-04-01"
$AzureManagementEndpoint = "https://management.azure.com"
$SubscriptionID = (Get-AzContext).Subscription.Id
$KeyVaults = Get-AzKeyVault

foreach ($KeyVault in $KeyVaults) {
    $ResourceGroupName = $KeyVault.ResourceGroupName
    $KeyVaultName = $KeyVault.VaultName

    $URI = "$AzureManagementEndpoint/subscriptions/$SubscriptionID/resourceGroups/$ResourceGroupName/providers/Microsoft.KeyVault/vaults/$KeyVaultName/providers/Microsoft.Authorization/permissions?api-version=$ApiVersion"

    $RequestParams = @{
        Method = "GET"
        Uri = $URI
        Headers = @{
            Authorization = "Bearer $ARM"
        }
    }

    try {
        $Permissions = (Invoke-RestMethod @RequestParams).value

        if ($Permissions) {
            $Permissions | Select-Object @{Name="KeyVault";Expression={$KeyVaultName}},Actions,NotActions,DataActions,NotDataActions | Format-List
        }
        else {
            Write-Host "No permissions returned for $KeyVaultName" -ForegroundColor Yellow
        }
    }
    catch {
        Write-Host "Failed to retrieve permissions for $KeyVaultName : $($_.Exception.Message)" -ForegroundColor Red
    }
}
