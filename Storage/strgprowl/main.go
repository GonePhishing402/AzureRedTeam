package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	// Web UI flags
	addr   := flag.String("addr", "127.0.0.1", "Address for the web UI to listen on")
	port   := flag.String("port", "8080", "Port for the web UI to listen on")
	noOpen := flag.Bool("no-open", false, "Do not auto-open the browser on startup")

	// Headless CLI flags
	refreshToken := flag.String("refresh-token", "", "OAuth refresh token (headless mode)")
	clientID     := flag.String("client-id", "", "Azure AD Application (Client) ID (headless mode)")
	tenantID     := flag.String("tenant-id", "", "Azure AD Tenant ID (headless mode)")
	accountName  := flag.String("account", "", "Limit recon to a single storage account (optional)")
	outputPath   := flag.String("output", "./StorageReconOutput", "Directory for saved output files")
	listBlobs    := flag.Bool("blobs", true, "Enumerate blobs inside each container")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: strgprowl [flags]\n\n")
		fmt.Fprintf(os.Stderr, "  No credential flags → start web UI\n")
		fmt.Fprintf(os.Stderr, "  With credential flags → headless CLI mode\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  strgprowl\n")
		fmt.Fprintf(os.Stderr, "  strgprowl -refresh-token <token> -client-id <id> -tenant-id <tenant>\n")
	}

	flag.Parse()

	// If any credential flag is provided, run headless CLI mode.
	if *refreshToken != "" || *clientID != "" || *tenantID != "" {
		missing := false
		if *refreshToken == "" {
			fmt.Fprintln(os.Stderr, "error: -refresh-token is required")
			missing = true
		}
		if *clientID == "" {
			fmt.Fprintln(os.Stderr, "error: -client-id is required")
			missing = true
		}
		if *tenantID == "" {
			fmt.Fprintln(os.Stderr, "error: -tenant-id is required")
			missing = true
		}
		if missing {
			fmt.Fprintln(os.Stderr, "")
			flag.Usage()
			os.Exit(1)
		}

		cfg := StorageReconConfig{
			RefreshToken:       *refreshToken,
			ClientID:           *clientID,
			TenantID:           *tenantID,
			StorageAccountName: *accountName,
			OutputPath:         *outputPath,
			ListBlobs:          *listBlobs,
		}

		logCh := make(chan string, 512)
		go func() {
			RunStorageRecon(cfg, logCh)
			close(logCh)
		}()

		for line := range logCh {
			fmt.Println(line)
		}
		return
	}

	// No credential flags → start web UI.
	StartServer(*addr, *port, !*noOpen)
}
