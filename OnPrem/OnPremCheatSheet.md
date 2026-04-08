## Runas Hybrid User - User from Cloud has OnPrem Sync
```
runas /netonly /user:reservoirone.corp\hybriduser96 cmd
```
## Use InvisiShell to Bypass AV
```
C:\AzAD\Tools\InviShell\RunWithPathAsAdmin.bat
```
## Run PowerView to Enumerate On-Prem AD
```
C:\AzAD\Tools\PowerView.ps1
```
```
Get-DomainComputer -DomainController reservoirone-dc.reservoirone.corp -Domain reservoirone.corp
```
