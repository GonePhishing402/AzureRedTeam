# How Users Get Microsoft Graph Permissions (Clear Breakdown)

## Key Concept

Users do **NOT directly have Microsoft Graph permissions**.

Instead:

-   **Applications (App Registrations)** have Graph permissions
-   Users authenticate **through an application**
-   The token combines:
    -   User identity
    -   Application permissions

------------------------------------------------------------------------

## How It Works (Step-by-Step)

1.  A user (Javier) signs into an application

    -   Azure Portal
    -   Teams
    -   GraphSpy
    -   Custom app

2.  That application has **Microsoft Graph API permissions configured**

3.  The user authenticates

4.  Azure issues a token that includes:

    -   User identity
    -   Application permissions (`scp`)

------------------------------------------------------------------------

## What Creates the `scp` Claim?

The `scp` (scope) is derived from:

-   App Registration API permissions
-   User or Admin consent
-   Authentication flow used

Formula:

User + App + Consent = Token with `scp`

------------------------------------------------------------------------

## Example

If an app has:

-   User.Read
-   Mail.Read

Then the token will contain:

"scp": "User.Read Mail.Read"

------------------------------------------------------------------------

## Why This Is Confusing

Azure has two different models:

### Azure RBAC (ARM)

-   Permissions assigned directly to users
-   Uses `roles` in tokens

### Microsoft Graph

-   Permissions assigned to applications
-   Uses `scp` in tokens

------------------------------------------------------------------------

## Key Difference

  System   Permissions Source   Token Field
  -------- -------------------- -------------
  Graph    App Registration     scp
  ARM      RBAC Roles           roles

------------------------------------------------------------------------

## Red Team Perspective

-   Control the app → control the permissions
-   Abuse OAuth consent flows
-   Use phishing (device code, consent phishing)
-   Target high-value permissions:
    -   Mail.Read
    -   Files.Read.All
    -   Directory.Read.All

------------------------------------------------------------------------

## Key Takeaways

-   Users alone do NOT have Graph permissions
-   Applications define Graph access
-   Tokens combine:
    -   User identity
    -   App permissions
-   Always check:
    -   aud (target resource)
    -   scp (permissions)
