# ================================
# Authenticate to Microsoft Graph as Service Principal
# ================================

# Prompt for required values
$tenantId     = Read-Host "Enter Tenant ID"
$clientId     = Read-Host "Enter Client ID (AppId)"
$clientSecret = Read-Host "Enter Client Secret"

# Convert secret to secure string
$secureSecret = ConvertTo-SecureString $clientSecret -AsPlainText -Force

# Create credential object
$cred = New-Object System.Management.Automation.PSCredential($clientId, $secureSecret)

# Connect to Microsoft Graph
Connect-MgGraph -TenantId $tenantId -ClientSecretCredential $cred

# Verify connection
Get-MgContext
