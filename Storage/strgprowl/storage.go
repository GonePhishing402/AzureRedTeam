package main

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

// validateAccountName enforces Azure storage account naming rules.
// Names must be 3-24 lowercase alphanumeric characters.
func validateAccountName(name string) error {
	if l := len(name); l < 3 || l > 24 {
		return fmt.Errorf("storage account name must be 3-24 characters, got %d", l)
	}
	matched, _ := regexp.MatchString(`^[a-z0-9]+$`, name)
	if !matched {
		return fmt.Errorf("storage account name must contain only lowercase alphanumeric characters")
	}
	return nil
}

// validateContainerName enforces Azure container naming rules.
// Names must be 3-63 lowercase alphanumeric characters or hyphens.
func validateContainerName(name string) error {
	if l := len(name); l < 3 || l > 63 {
		return fmt.Errorf("container name must be 3-63 characters, got %d", l)
	}
	matched, _ := regexp.MatchString(`^[a-z0-9][a-z0-9\-]+[a-z0-9]$`, name)
	if !matched {
		return fmt.Errorf("container name must start/end with alphanumeric and contain only lowercase alphanumeric or hyphens")
	}
	return nil
}

// encodeBlobPath percent-encodes each path segment of a blob name.
func encodeBlobPath(blobName string) string {
	parts := strings.Split(blobName, "/")
	for i, p := range parts {
		parts[i] = url.PathEscape(p)
	}
	return strings.Join(parts, "/")
}

const (
	armStorageAPI  = "2023-01-01"
	blobAPIVersion = "2020-04-08"
)

// ── Storage Accounts (ARM) ────────────────────────────────────────────────────

type StorageEndpoints struct {
	Blob  string `json:"blob"`
	Table string `json:"table"`
	Queue string `json:"queue"`
	File  string `json:"file"`
}

type StorageAccountProps struct {
	PrimaryEndpoints      StorageEndpoints `json:"primaryEndpoints"`
	AllowBlobPublicAccess *bool            `json:"allowBlobPublicAccess"`
	HTTPSOnly             *bool            `json:"supportsHttpsTrafficOnly"`
}

type StorageAccount struct {
	ID         string              `json:"id"`
	Name       string              `json:"name"`
	Location   string              `json:"location"`
	Kind       string              `json:"kind"`
	Properties StorageAccountProps `json:"properties"`
}

type storageAccountList struct {
	Value    []StorageAccount `json:"value"`
	NextLink string           `json:"nextLink"`
}

// ListStorageAccounts returns all storage accounts in the subscription.
// If nameFilter is non-empty, only the matching account is returned.
func ListStorageAccounts(armToken, subID, nameFilter string) ([]StorageAccount, error) {
	uri := fmt.Sprintf(
		"%s/subscriptions/%s/providers/Microsoft.Storage/storageAccounts?api-version=%s",
		armBase, subID, armStorageAPI)
	var all []StorageAccount
	for uri != "" {
		var page storageAccountList
		if err := armGet(armToken, uri, &page); err != nil {
			return nil, err
		}
		all = append(all, page.Value...)
		uri = page.NextLink
	}
	if nameFilter == "" {
		return all, nil
	}
	for _, a := range all {
		if strings.EqualFold(a.Name, nameFilter) {
			return []StorageAccount{a}, nil
		}
	}
	return nil, fmt.Errorf("storage account %q not found in subscription", nameFilter)
}

// GetStorageRBACPermissions returns effective permissions on a storage account.
func GetStorageRBACPermissions(armToken, subID, resourceGroup, accountName string) ([]Permission, error) {
	uri := fmt.Sprintf(
		"%s/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Storage/storageAccounts/%s"+
			"/providers/Microsoft.Authorization/permissions?api-version=%s",
		armBase, subID, resourceGroup, accountName, armPermAPI)
	var list permissionList
	if err := armGet(armToken, uri, &list); err != nil {
		return nil, err
	}
	return list.Value, nil
}

// ── Blob Containers via ARM (management plane) ────────────────────────────────

// BlobContainerARM is a blob container returned by the ARM management API.
// Using ARM avoids the need for data-plane RBAC (Storage Blob Data Reader);
// only management-plane Reader access is required.
type BlobContainerARM struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Properties struct {
		PublicAccess     string `json:"publicAccess"`
		LastModifiedTime string `json:"lastModifiedTime"`
	} `json:"properties"`
}

type blobContainerARMList struct {
	Value    []BlobContainerARM `json:"value"`
	NextLink string             `json:"nextLink"`
}

// ListContainersViaARM lists blob containers using the ARM management API (paginated).
func ListContainersViaARM(armToken, subID, resourceGroup, accountName string) ([]BlobContainerARM, error) {
	uri := fmt.Sprintf(
		"%s/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Storage/storageAccounts/%s"+
			"/blobServices/default/containers?api-version=%s",
		armBase, subID, resourceGroup, accountName, armStorageAPI)
	var all []BlobContainerARM
	for uri != "" {
		var page blobContainerARMList
		if err := armGet(armToken, uri, &page); err != nil {
			return nil, err
		}
		all = append(all, page.Value...)
		uri = page.NextLink
	}
	return all, nil
}

// ── Blob Storage REST API (XML) ───────────────────────────────────────────────

// BlobItem is a single blob returned by the list-blobs call.
type BlobItem struct {
	Name          string `xml:"Name"`
	LastModified  string `xml:"Properties>Last-Modified"`
	ContentType   string `xml:"Properties>Content-Type"`
	ContentLength int64  `xml:"Properties>Content-Length"`
}

type blobListResponse struct {
	XMLName    xml.Name   `xml:"EnumerationResults"`
	Blobs      []BlobItem `xml:"Blobs>Blob"`
	NextMarker string     `xml:"NextMarker"`
}

// ListBlobs returns all blobs in a container (paginated).
func ListBlobs(storageToken, accountName, containerName string) ([]BlobItem, error) {
	base := fmt.Sprintf(
		"https://%s.blob.core.windows.net/%s?restype=container&comp=list",
		accountName, containerName)
	var all []BlobItem
	marker := ""
	for {
		uri := base
		if marker != "" {
			uri += "&marker=" + marker
		}
		body, err := blobGet(storageToken, uri)
		if err != nil {
			return all, err
		}
		var result blobListResponse
		if err := xml.Unmarshal(body, &result); err != nil {
			return all, fmt.Errorf("parse blobs XML: %w", err)
		}
		all = append(all, result.Blobs...)
		if result.NextMarker == "" {
			break
		}
		marker = result.NextMarker
	}
	return all, nil
}

// blobGet performs an authenticated GET to the Azure Blob Storage REST API.
func blobGet(storageToken, uri string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, uri, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+storageToken)
	req.Header.Set("x-ms-version", blobAPIVersion)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("storage request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading storage response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}
	return body, nil
}

// GetBlobContent downloads up to maxBytes of a blob, returning the raw data,
// the Content-Type header, whether the response was truncated, and any error.
func GetBlobContent(storageToken, accountName, containerName, blobName string, maxBytes int64) ([]byte, string, bool, error) {
	if err := validateAccountName(accountName); err != nil {
		return nil, "", false, err
	}
	if err := validateContainerName(containerName); err != nil {
		return nil, "", false, err
	}
	endpoint := fmt.Sprintf("https://%s.blob.core.windows.net/%s/%s",
		accountName, containerName, encodeBlobPath(blobName))

	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, "", false, err
	}
	req.Header.Set("Authorization", "Bearer "+storageToken)
	req.Header.Set("x-ms-version", blobAPIVersion)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, "", false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		errBody, _ := io.ReadAll(resp.Body)
		return nil, "", false, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(errBody))
	}

	contentType := resp.Header.Get("Content-Type")
	lr := io.LimitReader(resp.Body, maxBytes+1)
	data, err := io.ReadAll(lr)
	if err != nil {
		return nil, contentType, false, err
	}
	truncated := int64(len(data)) > maxBytes
	if truncated {
		data = data[:maxBytes]
	}
	return data, contentType, truncated, nil
}

// StreamBlobToWriter streams a blob directly into w, returning the Content-Type and
// total bytes written. The caller should set response headers before calling.
func StreamBlobToWriter(storageToken, accountName, containerName, blobName string, w io.Writer) (string, int64, error) {
	if err := validateAccountName(accountName); err != nil {
		return "", 0, err
	}
	if err := validateContainerName(containerName); err != nil {
		return "", 0, err
	}
	endpoint := fmt.Sprintf("https://%s.blob.core.windows.net/%s/%s",
		accountName, containerName, encodeBlobPath(blobName))

	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return "", 0, err
	}
	req.Header.Set("Authorization", "Bearer "+storageToken)
	req.Header.Set("x-ms-version", blobAPIVersion)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		errBody, _ := io.ReadAll(resp.Body)
		return "", 0, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(errBody))
	}

	contentType := resp.Header.Get("Content-Type")
	n, err := io.Copy(w, resp.Body)
	return contentType, n, err
}
