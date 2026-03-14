package main

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"
)

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
// This only requires management-plane access and works without data-plane RBAC roles.
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

// BlobContainer is a single container returned by the list-containers call.
type BlobContainer struct {
	Name         string `xml:"Name"`
	PublicAccess string `xml:"Properties>PublicAccess"`
	LastModified string `xml:"Properties>Last-Modified"`
}

type containerListResponse struct {
	XMLName    xml.Name        `xml:"EnumerationResults"`
	Containers []BlobContainer `xml:"Containers>Container"`
	NextMarker string          `xml:"NextMarker"`
}

// ListContainers returns all containers in a Blob Storage account (paginated).
func ListContainers(storageToken, accountName string) ([]BlobContainer, error) {
	base := fmt.Sprintf("https://%s.blob.core.windows.net/?comp=list", accountName)
	var all []BlobContainer
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
		var result containerListResponse
		if err := xml.Unmarshal(body, &result); err != nil {
			return all, fmt.Errorf("parse containers XML: %w", err)
		}
		all = append(all, result.Containers...)
		if result.NextMarker == "" {
			break
		}
		marker = result.NextMarker
	}
	return all, nil
}

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
		return nil, fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}
	return body, nil
}
