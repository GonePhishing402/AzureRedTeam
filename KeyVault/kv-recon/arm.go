package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const (
	armBase     = "https://management.azure.com"
	armSubAPI   = "2022-12-01"
	armVaultAPI = "2023-07-01"
	armPermAPI  = "2022-04-01"
)

// ── Subscription ──────────────────────────────────────────────────────────────

// Subscription represents an Azure subscription.
type Subscription struct {
	SubscriptionID string `json:"subscriptionId"`
	DisplayName    string `json:"displayName"`
	TenantID       string `json:"tenantId"`
	State          string `json:"state"`
}

type subscriptionList struct {
	Value    []Subscription `json:"value"`
	NextLink string         `json:"nextLink"`
}

// GetSubscription returns the first enabled subscription visible to the token.
func GetSubscription(armToken string) (Subscription, error) {
	uri := fmt.Sprintf("%s/subscriptions?api-version=%s", armBase, armSubAPI)
	var list subscriptionList
	if err := armGet(armToken, uri, &list); err != nil {
		return Subscription{}, err
	}
	for _, s := range list.Value {
		if strings.EqualFold(s.State, "Enabled") {
			return s, nil
		}
	}
	if len(list.Value) > 0 {
		return list.Value[0], nil
	}
	return Subscription{}, fmt.Errorf("no subscriptions found")
}

// ── Key Vaults ────────────────────────────────────────────────────────────────

// VaultProperties holds Key Vault resource properties.
type VaultProperties struct {
	VaultURI              string `json:"vaultUri"`
	EnableSoftDelete      *bool  `json:"enableSoftDelete"`
	EnablePurgeProtection *bool  `json:"enablePurgeProtection"`
}

// Vault represents a Key Vault ARM resource.
type Vault struct {
	ID         string          `json:"id"`
	Name       string          `json:"name"`
	Location   string          `json:"location"`
	Properties VaultProperties `json:"properties"`
}

type vaultList struct {
	Value    []Vault `json:"value"`
	NextLink string  `json:"nextLink"`
}

// ListVaults returns all Key Vaults in the subscription, or just the named one.
func ListVaults(armToken, subID, vaultNameFilter string) ([]Vault, error) {
	vaults, err := listAllVaults(armToken, subID)
	if err != nil {
		return nil, err
	}
	if vaultNameFilter == "" {
		return vaults, nil
	}
	for _, v := range vaults {
		if strings.EqualFold(v.Name, vaultNameFilter) {
			return []Vault{v}, nil
		}
	}
	return nil, fmt.Errorf("vault %q not found in subscription", vaultNameFilter)
}

func listAllVaults(armToken, subID string) ([]Vault, error) {
	uri := fmt.Sprintf(
		"%s/subscriptions/%s/providers/Microsoft.KeyVault/vaults?api-version=%s",
		armBase, subID, armVaultAPI)
	var all []Vault
	for uri != "" {
		var page vaultList
		if err := armGet(armToken, uri, &page); err != nil {
			return nil, err
		}
		all = append(all, page.Value...)
		uri = page.NextLink
	}
	return all, nil
}

// ResourceGroupFromID extracts the resource group name from an ARM resource ID.
func ResourceGroupFromID(id string) string {
	parts := strings.Split(id, "/")
	for i, p := range parts {
		if strings.EqualFold(p, "resourceGroups") && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
}

// BoolStr returns "true"/"false" from a *bool, or "N/A" if nil.
func BoolStr(b *bool) string {
	if b == nil {
		return "N/A"
	}
	if *b {
		return "true"
	}
	return "false"
}

// ── RBAC Permissions ──────────────────────────────────────────────────────────

// Permission represents a set of RBAC permission actions on a resource.
type Permission struct {
	Actions        []string `json:"actions"`
	NotActions     []string `json:"notActions"`
	DataActions    []string `json:"dataActions"`
	NotDataActions []string `json:"notDataActions"`
}

type permissionList struct {
	Value []Permission `json:"value"`
}

// GetRBACPermissions returns the effective RBAC permissions for the vault.
func GetRBACPermissions(armToken, subID, resourceGroup, vaultName string) ([]Permission, error) {
	uri := fmt.Sprintf(
		"%s/subscriptions/%s/resourceGroups/%s/providers/Microsoft.KeyVault/vaults/%s"+
			"/providers/Microsoft.Authorization/permissions?api-version=%s",
		armBase, subID, resourceGroup, vaultName, armPermAPI)

	var list permissionList
	if err := armGet(armToken, uri, &list); err != nil {
		return nil, err
	}
	return list.Value, nil
}

// ── HTTP helper ───────────────────────────────────────────────────────────────

func armGet(token, uri string, out interface{}) error {
	req, err := http.NewRequest(http.MethodGet, uri, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}
	return json.Unmarshal(body, out)
}
