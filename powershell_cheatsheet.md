# 🟦 PowerShell Cheat Sheet

## 🔹 What is PowerShell?
- Task automation + scripting language
- Built on .NET (outputs **objects**, not text)
- Cross-platform (Windows, Linux, macOS)

---

## 🔹 Cmdlet Basics
- Format: **Verb-Noun**
- Examples:
  - Get-Process
  - Get-Service
  - Set-ExecutionPolicy

### Discover Commands
```powershell
Get-Command
Get-Command *process*
```

---

## 🔹 Getting Help
```powershell
Get-Help Get-Process
Get-Help Get-Process -Examples
Update-Help
```

---

## 🔹 Variables
```powershell
$name = "Jared"
$number = 10
$array = @(1,2,3)
```

---

## 🔹 Pipeline (|)
- Pass objects between commands

```powershell
Get-Process | Where-Object {$_.CPU -gt 100}
```

---

## 🔹 Objects & Properties
- PowerShell outputs objects with properties

```powershell
Get-Process | Get-Member
Get-Process | Select-Object Name, Id
```

---

## 🔹 Filtering
```powershell
Get-Service | Where-Object {$_.Status -eq "Running"}
```

---

## 🔹 Loops
```powershell
foreach ($item in $array) {
    Write-Output $item
}
```

---

## 🔹 Conditionals
```powershell
if ($x -eq 1) {
    "True"
} else {
    "False"
}
```

---

## 🔹 Working with Files
```powershell
Get-ChildItem
Copy-Item file.txt C:\temp\
Remove-Item file.txt
```

---

## 🔹 Scripts
- Save as `.ps1`

```powershell
.\script.ps1
```

### Execution Policy
```powershell
Set-ExecutionPolicy RemoteSigned
```

---

## 🔹 Modules
```powershell
Install-Module Az
Import-Module Az
Get-Module -ListAvailable
```

---

## 🔹 Invoke-RestMethod (APIs)
```powershell
Invoke-RestMethod -Uri "https://api.example.com"
```

### With Headers (Token Auth)
```powershell
$headers = @{
    Authorization = "Bearer <token>"
}

Invoke-RestMethod -Uri "https://graph.microsoft.com/v1.0/me" -Headers $headers
```

---

## 🔹 JSON Handling
```powershell
$data = Invoke-RestMethod -Uri "https://api.example.com"
$data.property

# Convert to JSON
$data | ConvertTo-Json -Depth 5
```

---

## 🔹 Property Access (Dot Notation)
```powershell
$TokenResponse.access_token
```

- `$TokenResponse` = object  
- `.access_token` = property  

---

## 🔹 Common Operators
- `-eq` → equal  
- `-ne` → not equal  
- `-gt` → greater than  
- `-lt` → less than  
- `-like` → wildcard matching  

---

## 🔹 Useful Commands
```powershell
Get-Alias
Get-History
Clear-Host
```

---

## 🔹 Formatting Output
```powershell
Get-Process | Format-Table
Get-Process | Format-List
```

---

## 🔹 Pro Tips
- Use **Tab** for auto-complete  
- Use `Get-Member` to explore objects  
- Use `|` to chain commands  
- Prefer objects over text parsing  

---

## 🔹 One-Liner Example
```powershell
Get-Process | Where-Object {$_.CPU -gt 100} | Select Name, CPU
```

---

## 🔹 Red Team / Cloud Use Cases
- Azure & M365 automation (`Az`, `MgGraph`)
- Token-based API access
- Enumeration & data collection
- Scripted attack workflows
