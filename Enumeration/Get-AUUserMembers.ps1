###############################################################
# Script Name: Get-AUUserMembers.ps1
#
# Description:
# This script enumerates members of a Microsoft Entra ID
# Administrative Unit (AU) using Microsoft Graph PowerShell.
#
# From a red team or adversary simulation perspective, this
# enumeration is useful when a compromised user or service
# principal has a directory role (such as Helpdesk Administrator,
# User Administrator, or Password Administrator) scoped to an
# Administrative Unit rather than the entire tenant.
#
# Administrative Units allow organizations to delegate identity
# management permissions over a subset of users. If an attacker
# gains control of an identity with an AU-scoped role, they may
# be able to perform actions such as:
#
#   • Reset passwords for users within the Administrative Unit
#   • Unlock accounts or force password resets
#   • Modify authentication methods
#   • Potentially pivot into higher-privileged accounts that
#     exist inside the AU
#
# By enumerating AU membership, an attacker can identify which
# users fall within the scope of the delegated permissions and
# determine potential privilege escalation or lateral movement
# paths.
#
# This technique is commonly used during:
#
#   • Azure / Entra ID privilege escalation assessments
#   • Cloud identity attack path analysis
#   • Red team and adversary simulation exercises
#   • Post-compromise identity enumeration
#
# Requirements:
#   - Microsoft Graph PowerShell module
#   - Authenticated session via Connect-MgGraph
#   - Permissions to read Administrative Unit membership
#
###############################################################
# VARIABLES
$AdministrativeUnitId = "<ADMINISTRATIVE_UNIT_ID>"

# QUERY ADMINISTRATIVE UNIT MEMBERS
Get-MgDirectoryAdministrativeUnitMember -AdministrativeUnitId $AdministrativeUnitId -All |
Select-Object Id, @{
    Name = "UserPrincipalName"
    Expression = { $_.AdditionalProperties.userPrincipalName }
} | Format-List
