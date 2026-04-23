# Attacking Azure Container Apps

## Contents
- [Enumerating Container Secrets](#enumerating-container-secrets)  
- [Connecting to Container with Passwords Found](#connecting-to-container-with-passwords-found)

---

## Enumerating Container Secrets

Once you have an access token:

- Extract the **Subscription ID** from the token

### List available resources
- Enumerate Azure resources
- Identify **Container Apps**

### Identify target container
- Store the container resource ID:
```powershell
$Id = "<container-resource-id>"
```

### Enumerate secrets
- Query the container for stored secrets using the resource ID

---

## Connecting to Container with Passwords Found

### Retrieve identity environment variables
- Extract the following from the container environment:
  - `IDENTITY_HEADER`
  - `IDENTITY_ENDPOINT`

### Request access token from managed identity
```bash
curl -H "X-IDENTITY-HEADER: $IDENTITY_HEADER" "$IDENTITY_ENDPOINT?resource=https://management.azure.com&api-version=2019-08-01"
```

### Use the token
- Authenticate with the new access token
- Query Azure resources

---

## SQL Enumeration

### Identify SQL resources
- Look for:
  - Azure SQL
  - Flexible SQL Servers

### Authenticate using discovered credentials
- Use previously obtained credentials to access SQL

### Basic SQL enumeration
```sql
-- List databases
SHOW DATABASES;

-- Select a database
USE <database_name>;

-- List tables
SHOW TABLES;

-- View table data
SELECT * FROM <table_name> LIMIT 10;
```

---

## Notes
- Secrets stored in Container Apps can expose credentials and tokens
- Managed identities can be abused to obtain access tokens
- SQL access often enables further data exfiltration and lateral movement
