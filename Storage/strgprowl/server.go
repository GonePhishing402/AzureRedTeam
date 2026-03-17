package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"mime"
	"net"
	"net/http"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"

	_ "embed"
)

//go:embed ui/index.html
var uiHTML []byte

// ── Session store ─────────────────────────────────────────────────────────────

type session struct {
	ArmToken     string
	StorageToken string
	TenantID     string
	ClientID     string
	RefreshToken string
	SubID        string
	SubName      string
	ExpiresAt    time.Time
}

var (
	sessionsMu sync.Mutex
	sessions   = map[string]*session{}
)

func newSessionID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func getSession(r *http.Request) (*session, bool) {
	sid := r.Header.Get("X-Session-Id")
	if sid == "" {
		return nil, false
	}
	sessionsMu.Lock()
	defer sessionsMu.Unlock()
	s, ok := sessions[sid]
	if !ok || time.Now().After(s.ExpiresAt) {
		delete(sessions, sid)
		return nil, false
	}
	return s, true
}

// ── SSE run records ───────────────────────────────────────────────────────────

type runRecord struct {
	logCh chan string
	done  chan struct{}
}

var (
	runsMu sync.Mutex
	runs   = map[string]*runRecord{}
)

func newRunID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// ── HTTP helpers ──────────────────────────────────────────────────────────────

func respondJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func jsonError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func requireSession(w http.ResponseWriter, r *http.Request) (*session, bool) {
	s, ok := getSession(r)
	if !ok {
		jsonError(w, "invalid or expired session — please reconnect", http.StatusUnauthorized)
	}
	return s, ok
}

func decodeJSON(r *http.Request, out interface{}) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(out)
}

// ── Handlers ──────────────────────────────────────────────────────────────────

// POST /api/connect
// Body: { tenantId, clientId, refreshToken }
// Response: { sessionId, subId, subName }
func handleConnect(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TenantID     string `json:"tenantId"`
		ClientID     string `json:"clientId"`
		RefreshToken string `json:"refreshToken"`
	}
	if err := decodeJSON(r, &req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.TenantID == "" || req.ClientID == "" || req.RefreshToken == "" {
		jsonError(w, "tenantId, clientId, and refreshToken are required", http.StatusBadRequest)
		return
	}

	armToken, err := exchangeToken(req.TenantID, req.ClientID, req.RefreshToken,
		"https://management.azure.com/.default")
	if err != nil {
		jsonError(w, "failed to exchange ARM token: "+err.Error(), http.StatusBadRequest)
		return
	}
	storageToken, err := exchangeToken(req.TenantID, req.ClientID, req.RefreshToken,
		"https://storage.azure.com/.default")
	if err != nil {
		jsonError(w, "failed to exchange Storage token: "+err.Error(), http.StatusBadRequest)
		return
	}

	sub, err := GetSubscription(armToken)
	if err != nil {
		jsonError(w, "failed to get subscription: "+err.Error(), http.StatusBadRequest)
		return
	}

	s := &session{
		ArmToken:     armToken,
		StorageToken: storageToken,
		TenantID:     req.TenantID,
		ClientID:     req.ClientID,
		RefreshToken: req.RefreshToken,
		SubID:        sub.SubscriptionID,
		SubName:      sub.DisplayName,
		ExpiresAt:    time.Now().Add(60 * time.Minute),
	}
	sid := newSessionID()
	sessionsMu.Lock()
	sessions[sid] = s
	sessionsMu.Unlock()

	respondJSON(w, map[string]string{
		"sessionId": sid,
		"subId":     s.SubID,
		"subName":   s.SubName,
	})
}

// POST /api/accounts — list storage accounts
func handleAccounts(w http.ResponseWriter, r *http.Request) {
	s, ok := requireSession(w, r)
	if !ok {
		return
	}
	accounts, err := ListStorageAccounts(s.ArmToken, s.SubID, "")
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	type row struct {
		Name      string `json:"name"`
		Location  string `json:"location"`
		BlobEP    string `json:"blobEndpoint"`
		PublicAcc string `json:"publicAccess"`
		HTTPSOnly string `json:"httpsOnly"`
	}
	var rows []row
	for _, a := range accounts {
		rows = append(rows, row{
			Name:      a.Name,
			Location:  a.Location,
			BlobEP:    a.Properties.PrimaryEndpoints.Blob,
			PublicAcc: BoolStr(a.Properties.AllowBlobPublicAccess),
			HTTPSOnly: BoolStr(a.Properties.HTTPSOnly),
		})
	}
	if rows == nil {
		rows = []row{}
	}
	respondJSON(w, rows)
}

// POST /api/containers — list containers in a storage account
// Body: { accountName }
func handleContainers(w http.ResponseWriter, r *http.Request) {
	s, ok := requireSession(w, r)
	if !ok {
		return
	}
	var req struct {
		AccountName string `json:"accountName"`
	}
	if err := decodeJSON(r, &req); err != nil || req.AccountName == "" {
		jsonError(w, "accountName is required", http.StatusBadRequest)
		return
	}
	if err := validateAccountName(req.AccountName); err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}
	containers, err := ListContainersViaARM(s.ArmToken, s.SubID, req.AccountName)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	type row struct {
		Name      string `json:"name"`
		PublicAcc string `json:"publicAccess"`
	}
	var rows []row
	for _, c := range containers {
		rows = append(rows, row{
			Name:      c.Name,
			PublicAcc: c.Properties.PublicAccess,
		})
	}
	if rows == nil {
		rows = []row{}
	}
	respondJSON(w, rows)
}

// POST /api/blobs — list blobs in a container
// Body: { accountName, containerName }
func handleBlobs(w http.ResponseWriter, r *http.Request) {
	s, ok := requireSession(w, r)
	if !ok {
		return
	}
	var req struct {
		AccountName   string `json:"accountName"`
		ContainerName string `json:"containerName"`
	}
	if err := decodeJSON(r, &req); err != nil || req.AccountName == "" || req.ContainerName == "" {
		jsonError(w, "accountName and containerName are required", http.StatusBadRequest)
		return
	}
	if err := validateAccountName(req.AccountName); err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := validateContainerName(req.ContainerName); err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}
	blobs, err := ListBlobs(s.StorageToken, req.AccountName, req.ContainerName)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	type row struct {
		Name         string `json:"name"`
		Size         int64  `json:"size"`
		LastModified string `json:"lastModified"`
		ContentType  string `json:"contentType"`
	}
	var rows []row
	for _, b := range blobs {
		rows = append(rows, row{
			Name:         b.Name,
			Size:         b.Properties.ContentLength,
			LastModified: b.Properties.LastModified,
			ContentType:  b.Properties.ContentType,
		})
	}
	if rows == nil {
		rows = []row{}
	}
	respondJSON(w, rows)
}

const maxViewBytes = 512 * 1024 // 512 KB

// POST /api/blob/content — preview blob contents
// Body: { accountName, containerName, blobName }
// Response: { contentType, data (base64 for binary / string for text), truncated, encoding }
func handleBlobContent(w http.ResponseWriter, r *http.Request) {
	s, ok := requireSession(w, r)
	if !ok {
		return
	}
	var req struct {
		AccountName   string `json:"accountName"`
		ContainerName string `json:"containerName"`
		BlobName      string `json:"blobName"`
	}
	if err := decodeJSON(r, &req); err != nil || req.AccountName == "" || req.ContainerName == "" || req.BlobName == "" {
		jsonError(w, "accountName, containerName, and blobName are required", http.StatusBadRequest)
		return
	}
	if err := validateAccountName(req.AccountName); err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := validateContainerName(req.ContainerName); err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	data, ct, truncated, err := GetBlobContent(s.StorageToken, req.AccountName, req.ContainerName, req.BlobName, maxViewBytes)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Determine whether content is text-safe.
	mediaType, _, _ := mime.ParseMediaType(ct)
	isText := strings.HasPrefix(mediaType, "text/") ||
		mediaType == "application/json" ||
		mediaType == "application/xml" ||
		mediaType == "application/javascript" ||
		mediaType == "" // unknown — try as text

	if isText {
		respondJSON(w, map[string]interface{}{
			"contentType": ct,
			"data":        string(data),
			"truncated":   truncated,
			"encoding":    "text",
		})
	} else {
		respondJSON(w, map[string]interface{}{
			"contentType": ct,
			"data":        data, // JSON will base64-encode []byte automatically
			"truncated":   truncated,
			"encoding":    "base64",
		})
	}
}

// POST /api/blob/download — stream blob to browser as a download
// Body: { accountName, containerName, blobName }
func handleBlobDownload(w http.ResponseWriter, r *http.Request) {
	s, ok := requireSession(w, r)
	if !ok {
		return
	}
	var req struct {
		AccountName   string `json:"accountName"`
		ContainerName string `json:"containerName"`
		BlobName      string `json:"blobName"`
	}
	if err := decodeJSON(r, &req); err != nil || req.AccountName == "" || req.ContainerName == "" || req.BlobName == "" {
		jsonError(w, "accountName, containerName, and blobName are required", http.StatusBadRequest)
		return
	}
	if err := validateAccountName(req.AccountName); err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := validateContainerName(req.ContainerName); err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Derive a safe filename from the blob name (last path segment only).
	parts := strings.Split(req.BlobName, "/")
	filename := parts[len(parts)-1]
	if filename == "" {
		filename = "blob"
	}

	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename=%q`, filename))
	ct, _, err := StreamBlobToWriter(s.StorageToken, req.AccountName, req.ContainerName, req.BlobName, w)
	if err != nil {
		// Headers already sent; nothing useful we can do.
		return
	}
	if ct != "" {
		// Content-Type was set before any write — set it speculatively (may be no-op).
		w.Header().Set("Content-Type", ct)
	}
}

// ── Recon SSE endpoints ───────────────────────────────────────────────────────

// POST /api/run — start a background recon job
// Body: { tenantId, clientId, refreshToken, account?, output?, blobs? }
// Response: { runId }
func handleRun(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TenantID     string `json:"tenantId"`
		ClientID     string `json:"clientId"`
		RefreshToken string `json:"refreshToken"`
		Account      string `json:"account"`
		Output       string `json:"output"`
		Blobs        *bool  `json:"blobs"`
	}
	if err := decodeJSON(r, &req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.TenantID == "" || req.ClientID == "" || req.RefreshToken == "" {
		jsonError(w, "tenantId, clientId, and refreshToken are required", http.StatusBadRequest)
		return
	}
	outputPath := req.Output
	if outputPath == "" {
		outputPath = "./StorageReconOutput"
	}
	listBlobs := true
	if req.Blobs != nil {
		listBlobs = *req.Blobs
	}

	cfg := StorageReconConfig{
		RefreshToken:       req.RefreshToken,
		ClientID:           req.ClientID,
		TenantID:           req.TenantID,
		StorageAccountName: req.Account,
		OutputPath:         outputPath,
		ListBlobs:          listBlobs,
	}

	id := newRunID()
	rec := &runRecord{
		logCh: make(chan string, 512),
		done:  make(chan struct{}),
	}
	runsMu.Lock()
	runs[id] = rec
	runsMu.Unlock()

	go func() {
		RunStorageRecon(cfg, rec.logCh)
		close(rec.logCh)
		close(rec.done)
	}()

	respondJSON(w, map[string]string{"runId": id})
}

// GET /api/stream/{id} — SSE stream for a run
func handleStream(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/stream/")
	runsMu.Lock()
	rec, ok := runs[id]
	runsMu.Unlock()
	if !ok {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Accel-Buffering", "no")
	flusher, _ := w.(http.Flusher)

	for {
		select {
		case line, more := <-rec.logCh:
			if !more {
				fmt.Fprintf(w, "event: done\ndata: {}\n\n")
				if flusher != nil {
					flusher.Flush()
				}
				return
			}
			fmt.Fprintf(w, "data: %s\n\n", jsonStr(line))
			if flusher != nil {
				flusher.Flush()
			}
		case <-r.Context().Done():
			return
		}
	}
}

func jsonStr(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

// ── Server startup ────────────────────────────────────────────────────────────

func serveUI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(uiHTML)
}

func StartServer(addr, port string, openBrowser bool) {
	mux := http.NewServeMux()

	// UI
	mux.HandleFunc("/", serveUI)

	// Auth / session
	mux.HandleFunc("/api/connect", handleConnect)

	// Blob browser
	mux.HandleFunc("/api/accounts", handleAccounts)
	mux.HandleFunc("/api/containers", handleContainers)
	mux.HandleFunc("/api/blobs", handleBlobs)
	mux.HandleFunc("/api/blob/content", handleBlobContent)
	mux.HandleFunc("/api/blob/download", handleBlobDownload)

	// Recon + SSE
	mux.HandleFunc("/api/run", handleRun)
	mux.HandleFunc("/api/stream/", handleStream)

	listenAddr := net.JoinHostPort(addr, port)
	fmt.Printf("[strgprowl] starting web UI at http://%s\n", listenAddr)

	if openBrowser {
		go func() {
			time.Sleep(300 * time.Millisecond)
			openURL("http://" + listenAddr)
		}()
	}

	if err := http.ListenAndServe(listenAddr, mux); err != nil {
		fmt.Printf("[strgprowl] server error: %v\n", err)
	}
}

func openURL(url string) {
	switch runtime.GOOS {
	case "windows":
		exec.Command("cmd", "/c", "start", url).Start()
	case "darwin":
		exec.Command("open", url).Start()
	default:
		exec.Command("xdg-open", url).Start()
	}
}
