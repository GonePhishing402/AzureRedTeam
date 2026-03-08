###############################################################
# Script Name: Connect-AzureWithServicePrincipalCertificate.ps1
#
# Description:
# This script authenticates to Azure using a Service Principal
# and certificate-based authentication.
#
# The script:
# 1. Defines the Azure AD application (client) ID
# 2. Defines the Azure AD tenant ID
# 3. Specifies the path to the certificate (.pfx)
# 4. Uses Connect-AzAccount to authenticate as the service principal
#
# This authentication method is commonly used for:
# - Automation scripts
# - CI/CD pipelines
# - Infrastructure management tools
# - Secure non-interactive authentication
#
# Requirements:
# - Az PowerShell module installed
# - Valid Service Principal with certificate credentials
# - Access to the certificate (.pfx file)
###############################################################


###############################################################
# VARIABLES (FILL THESE VALUES)
###############################################################

# Azure AD Application (Client) ID
$ApplicationID = ""

# Azure AD Tenant ID
$TenantID = ""

# Path to the service principal certificate (.pfx file)
$CertificatePath = ""


###############################################################
# AUTHENTICATE TO AZURE
###############################################################

# Connect to Azure using certificate-based authentication
# for the specified service principal
Connect-AzAccount `
    -ServicePrincipal `
    -Tenant $TenantID `
    -ApplicationId $ApplicationID `
    -CertificatePath $CertificatePath
