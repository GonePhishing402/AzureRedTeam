# Use a Certificate for Authentication
## Create a Certificate from the Value
```powershell
$base64String = Get-Content -Path "C:\AzAD\Tools\christina_cert_b64.txt"
```
```powershell
$certBytes = [System.Convert]::FromBase64String($base64String)
```
```powershell
[System.IO.File]::WriteAllBytes("C:\AzAD\Tools\ChristinaCert.pfx", $certBytes)
```
