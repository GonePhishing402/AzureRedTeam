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
	listenAddr := addr + ":" + port
	baseURL := "http://" + listenAddr

	mux := http.NewServeMux()
	mux.HandleFunc("/", serveUI)
	mux.HandleFunc("/run", handleRun)
	mux.HandleFunc("/stream/", handleStream)

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
