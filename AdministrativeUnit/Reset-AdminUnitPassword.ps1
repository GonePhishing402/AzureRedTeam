###############################################################
# Script Name: Reset-AdminUnitPassword.ps1
#
# Description:
# Resets the password for the user StorageMapper96 using
# Microsoft Graph PowerShell.
#
# Requirements:
# - Authenticated session via Connect-MgGraph
# - Appropriate privileges (Helpdesk Administrator, User Admin,
#   or Global Admin depending on the target account)
###############################################################

# Define new password profile
$passwordProfile = @{
    forceChangePasswordNextSignIn = $false
    password = "NewUserSecret@PassX"
}

# Reset the password for StorageMapper96
Update-MgUser `
    -UserId "User@domain.onmicrosoft.com" `
    -PasswordProfile $passwordProfile

Write-Host "Password reset successfully for Admin Unit"
