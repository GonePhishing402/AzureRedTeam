$headers = @{
    Authorization = "Bearer $StorageAccessToken"
    "x-ms-version" = "2023-11-03"
}

Invoke-RestMethod `
    -Method GET `
    -Uri "https://blobhunt.blob.core.windows.net/?comp=list" `
    -Headers $headers
