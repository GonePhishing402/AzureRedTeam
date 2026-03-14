package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const graphBase = "https://graph.microsoft.com/v1.0"

// ── Generic paged GET ─────────────────────────────────────────────────────────

// graphGet fetches a single page from the Graph API and unmarshals into out.
// Returns the nextLink (empty string when done).
func graphGet(accessToken, url string, out interface{}) (nextLink string, err error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("graph request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("graph error %d: %s", resp.StatusCode, string(body))
	}

	// Peek at nextLink before unmarshalling into the caller's struct.
	var paging struct {
		NextLink string `json:"@odata.nextLink"`
	}
	_ = json.Unmarshal(body, &paging)

	if err := json.Unmarshal(body, out); err != nil {
		return "", fmt.Errorf("unmarshal response: %w", err)
	}
	return paging.NextLink, nil
}

// ── Users ─────────────────────────────────────────────────────────────────────

type GraphUser struct {
	ID                string `json:"id"`
	DisplayName       string `json:"displayName"`
	UserPrincipalName string `json:"userPrincipalName"`
	JobTitle          string `json:"jobTitle"`
	Department        string `json:"department"`
	Mail              string `json:"mail"`
	AccountEnabled    *bool  `json:"accountEnabled"`
	CreatedDateTime   string `json:"createdDateTime"`
}

type userPage struct {
	Value    []GraphUser `json:"value"`
	NextLink string      `json:"@odata.nextLink"`
}

func ListUsers(accessToken string) ([]GraphUser, error) {
	url := graphBase + "/users?$select=id,displayName,userPrincipalName,jobTitle,department,mail,accountEnabled,createdDateTime&$top=999"
	var all []GraphUser
	for url != "" {
		var page userPage
		next, err := graphGet(accessToken, url, &page)
		if err != nil {
			return all, err
		}
		all = append(all, page.Value...)
		url = next
	}
	return all, nil
}

// ── Devices ───────────────────────────────────────────────────────────────────

type GraphDevice struct {
	ID                      string `json:"id"`
	DisplayName             string `json:"displayName"`
	OperatingSystem         string `json:"operatingSystem"`
	OperatingSystemVersion  string `json:"operatingSystemVersion"`
	IsCompliant             *bool  `json:"isCompliant"`
	IsManaged               *bool  `json:"isManaged"`
	TrustType               string `json:"trustType"`
	ApproximateLastSignIn   string `json:"approximateLastSignInDateTime"`
}

type devicePage struct {
	Value    []GraphDevice `json:"value"`
	NextLink string        `json:"@odata.nextLink"`
}

func ListDevices(accessToken string) ([]GraphDevice, error) {
	url := graphBase + "/devices?$select=id,displayName,operatingSystem,operatingSystemVersion,isCompliant,isManaged,trustType,approximateLastSignInDateTime&$top=999"
	var all []GraphDevice
	for url != "" {
		var page devicePage
		next, err := graphGet(accessToken, url, &page)
		if err != nil {
			return all, err
		}
		all = append(all, page.Value...)
		url = next
	}
	return all, nil
}

// ── Applications ──────────────────────────────────────────────────────────────

type GraphApplication struct {
	ID               string `json:"id"`
	AppID            string `json:"appId"`
	DisplayName      string `json:"displayName"`
	CreatedDateTime  string `json:"createdDateTime"`
	SignInAudience   string `json:"signInAudience"`
	PublisherDomain  string `json:"publisherDomain"`
}

type appPage struct {
	Value    []GraphApplication `json:"value"`
	NextLink string             `json:"@odata.nextLink"`
}

func ListApplications(accessToken string) ([]GraphApplication, error) {
	url := graphBase + "/applications?$select=id,appId,displayName,createdDateTime,signInAudience,publisherDomain&$top=999"
	var all []GraphApplication
	for url != "" {
		var page appPage
		next, err := graphGet(accessToken, url, &page)
		if err != nil {
			return all, err
		}
		all = append(all, page.Value...)
		url = next
	}
	return all, nil
}

// ── Conditional Access Policies ───────────────────────────────────────────────


// ── Conditional Access Policies ───────────────────────────────────────────────

type CAPConditions struct {
	Users struct {
		IncludeUsers  []string `json:"includeUsers"`
		ExcludeUsers  []string `json:"excludeUsers"`
		IncludeGroups []string `json:"includeGroups"`
	} `json:"users"`
	Applications struct {
		IncludeApplications []string `json:"includeApplications"`
	} `json:"applications"`
}

type CAPGrantControls struct {
	Operator        string   `json:"operator"`
	BuiltInControls []string `json:"builtInControls"`
}

type GraphCAP struct {
	ID           string           `json:"id"`
	DisplayName  string           `json:"displayName"`
	State        string           `json:"state"`
	CreatedAt    string           `json:"createdDateTime"`
	ModifiedAt   string           `json:"modifiedDateTime"`
	Conditions   CAPConditions    `json:"conditions"`
	GrantControls *CAPGrantControls `json:"grantControls"`
}

type capPage struct {
	Value    []GraphCAP `json:"value"`
	NextLink string     `json:"@odata.nextLink"`
}

func ListCAPs(accessToken string) ([]GraphCAP, error) {
	url := graphBase + "/identity/conditionalAccess/policies"
	var all []GraphCAP
	for url != "" {
		var page capPage
		next, err := graphGet(accessToken, url, &page)
		if err != nil {
			return all, err
		}
		all = append(all, page.Value...)
		url = next
	}
	return all, nil
}

// ── OneDrive ──────────────────────────────────────────────────────────────────

type DriveItemSize struct {
	Total int64 `json:"total"`
	Used  int64 `json:"used"`
}

type DriveItemFolder struct {
	ChildCount int `json:"childCount"`
}

type DriveItem struct {
	ID                  string           `json:"id"`
	Name                string           `json:"name"`
	Size                int64            `json:"size"`
	LastModifiedDateTime string          `json:"lastModifiedDateTime"`
	Folder              *DriveItemFolder `json:"folder"`
	DownloadURL         string           `json:"@microsoft.graph.downloadUrl"`
	WebURL              string           `json:"webUrl"`
}

type driveItemPage struct {
	Value    []DriveItem `json:"value"`
	NextLink string      `json:"@odata.nextLink"`
}

func ListOwnDrive(accessToken string) ([]DriveItem, error) {
	// Try /me/drive first; fall back to first available drive on 404.
	root := graphBase + "/me/drive/root/children?$select=id,name,size,lastModifiedDateTime,folder,webUrl&$top=200"
	all, err := fetchDriveItems(accessToken, root)
	if err != nil && strings.Contains(err.Error(), "404") {
		// User may have OneDrive under /me/drives instead.
		type driveRef struct {
			ID string `json:"id"`
		}
		type drivesPage struct {
			Value []driveRef `json:"value"`
		}
		var dp drivesPage
		_, rerr := graphGet(accessToken, graphBase+"/me/drives", &dp)
		if rerr != nil || len(dp.Value) == 0 {
			return nil, err // return original error
		}
		fallback := fmt.Sprintf("%s/drives/%s/root/children?$select=id,name,size,lastModifiedDateTime,folder,webUrl&$top=200",
			graphBase, dp.Value[0].ID)
		return fetchDriveItems(accessToken, fallback)
	}
	return all, err
}

func fetchDriveItems(accessToken, startURL string) ([]DriveItem, error) {
	var all []DriveItem
	url := startURL
	for url != "" {
		var page driveItemPage
		next, err := graphGet(accessToken, url, &page)
		if err != nil {
			return all, err
		}
		all = append(all, page.Value...)
		url = next
	}
	return all, nil
}
