package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const (
	armBase    = "https://management.azure.com"
	armSubAPI  = "2022-12-01"
	armPermAPI = "2022-04-01"
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

// ── Helpers ───────────────────────────────────────────────────────────────────

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

// armGet performs an authenticated GET against the ARM API and JSON-decodes the result.
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
