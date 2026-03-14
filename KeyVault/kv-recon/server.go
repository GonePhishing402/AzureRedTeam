package main

import (
	"crypto/rand"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
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

	// Token management
	mux.HandleFunc("/api/tokens", handleTokens)
	mux.HandleFunc("/api/tokens/", handleTokenByID)

	// Graph enumeration
	mux.HandleFunc("/api/graph/users", handleGraphUsers)
	mux.HandleFunc("/api/graph/devices", handleGraphDevices)
	mux.HandleFunc("/api/graph/applications", handleGraphApplications)
	mux.HandleFunc("/api/graph/approles", handleGraphAppRoles)
	mux.HandleFunc("/api/graph/caps", handleGraphCAPs)
	mux.HandleFunc("/api/graph/onedrive", handleGraphOneDrive)

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

func handleGraphAppRoles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	t, err := resolveToken(r)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}
	results, err := ListAppRoleAssignments(t.AccessToken)
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
