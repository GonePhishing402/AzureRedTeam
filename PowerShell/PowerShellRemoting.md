## Check if PowerShell Remoting is enabled on a system - WinRM
```powershell
Test-WSMan -ComputerName 3.208.47.144 -Verbose
```
## Use Creds to Connect to Remote Powershell - WinRM
```powershell
$password = ConvertTo-SecureString 'ASxuwjmlzgbfspykt4859' -AsPlainText -Force
```
```powershell
$creds = New-Object System.Management.Automation.PSCredential('ASbiulag4854', $password)
```
```powershell
$ec2instance = New-PSSession -ComputerName 3.208.47.144 -Credential $creds
```
```powershell
Enter-PSSession $ec2instance
```
