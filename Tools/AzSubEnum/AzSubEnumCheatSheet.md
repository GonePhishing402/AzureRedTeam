# AzSubEnum Cheat Sheet

> Azure Service Subdomain Enumeration Tool  
> **Repo:** [https://github.com/yuyudhn/AzSubEnum](https://github.com/yuyudhn/AzSubEnum)

---

## Overview

AzSubEnum enumerates subdomains associated with Azure services using DNS resolution and permutation techniques. It covers App Services, Storage Accounts, Databases (MSSQL, Cosmos DB, Redis), Key Vaults, CDN, Email, SharePoint, Azure Container Registry, and more.

---

## Installation

```bash
git clone https://github.com/yuyudhn/AzSubEnum.git
cd AzSubEnum
pip3 install -r requirements.txt
```

### Dependencies

- Python 3
- `dnspython`
- `requests`

---

## Usage

```
python3 azsubenum.py [-h] -b BASE [-v] [-t THREADS] [-p PERMUTATIONS] [--blobsenum] [-bw WORDLIST] [-bt BLOB_THREADS]
```

### Arguments

| Flag | Description |
|---|---|
| `-b`, `--base` | **(Required)** Base name to enumerate (e.g., company name) |
| `-v`, `--verbose` | Show verbose output |
| `-t`, `--threads` | Number of threads for concurrent execution (default: 10) |
| `-p`, `--permutations` | File containing permutation words |
| `--blobsenum` | Enable blob container enumeration on discovered storage accounts |
| `-bw` | Path to custom wordlist for blob container enumeration |
| `-bt` | Number of threads for blob container enumeration |

---

## Examples

### Basic Enumeration

```bash
python3 azsubenum.py -b retailcorp -t 10
```

### With Permutation Wordlist

```bash
python3 azsubenum.py -b retailcorp -t 10 -p permutations.txt
```

### Verbose Output

```bash
python3 azsubenum.py -b retailcorp -t 10 -p permutations.txt -v
```

### Blob Container Enumeration

```bash
python3 azsubenum.py -b retailcorp -t 10 --blobsenum -bw permutations.txt -bt 10
```

---

## Azure Service Domains Checked

| Domain Suffix | Azure Service |
|---|---|
| `onmicrosoft.com` | Microsoft Hosted Domain |
| `azurewebsites.net` | App Services |
| `p.azurewebsites.net` | App Services |
| `scm.azurewebsites.net` | App Services - Management (Kudu) |
| `cloudapp.net` | App Services |
| `blob.core.windows.net` | Storage Accounts - Blobs |
| `file.core.windows.net` | Storage Accounts - Files |
| `queue.core.windows.net` | Storage Accounts - Queues |
| `table.core.windows.net` | Storage Accounts - Tables |
| `mail.protection.outlook.com` | Email |
| `sharepoint.com` | SharePoint |
| `redis.cache.windows.net` | Databases - Redis |
| `documents.azure.com` | Databases - Cosmos DB |
| `database.windows.net` | Databases - MSSQL |
| `vault.azure.net` | Key Vaults |
| `azureedge.net` | CDN |
| `search.windows.net` | Search Appliance |
| `azure-api.net` | API Services |
| `azurecr.io` | Azure Container Registry |

---

## Permutation Logic

When a permutation file is supplied (`-p`), AzSubEnum generates the following subdomain patterns for each word in the file:

```
<base>-<word>.<suffix>
<word>-<base>.<suffix>
<word><base>.<suffix>
<base><word>.<suffix>
```

This is applied across **all** Azure service domain suffixes listed above.

---

## Blob Container Enumeration

When `--blobsenum` is enabled, AzSubEnum will:

1. Identify discovered `blob.core.windows.net` subdomains
2. Brute-force container names from the provided wordlist (`-bw`)
3. Check each container for **public (anonymous) access** via the Azure Storage REST API
4. If accessible, list the blobs within the container
5. Generate an HTML report (`report.html`) with all findings

---

## Output

- **Console:** Discovered subdomains grouped by Azure service category
- **HTML Report:** `report.html` generated automatically with subdomain results and accessible blob container details

---

## Red Team Tips

- Run against target org names, abbreviations, and known project names
- Combine with a strong permutation wordlist for deeper coverage
- Discovered subdomains reveal the Azure service footprint — useful for identifying attack surface (exposed storage, Key Vaults, databases, Kudu/SCM endpoints, etc.)
- Publicly accessible blob containers may contain sensitive files — enumerate further with Azure Storage Explorer or `az storage blob list`
- Cross-reference discovered Key Vault names with token-based access attempts
- Use discovered `scm.azurewebsites.net` endpoints to check for exposed Kudu consoles

---

## Similar / Complementary Tools

- [Invoke-EnumerateAzureSubDomains](https://github.com/NetSPI/MicroBurst/blob/master/Misc/Invoke-EnumerateAzureSubDomains.ps1) — PowerShell equivalent from NetSPI MicroBurst
- [Sublert](https://github.com/yassineaboukir/sublert) — General subdomain monitoring
- [MicroBurst](https://github.com/NetSPI/MicroBurst) — Azure security assessment toolkit
