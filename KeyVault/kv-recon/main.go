package main

import (
	"os/exec"
	"runtime"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

func main() {
	a := app.New()
	w := a.NewWindow("Azure Key Vault Recon")
	w.Resize(fyne.NewSize(860, 720))

	// ── Form fields ───────────────────────────────────────────────
	refreshTokenEntry := widget.NewPasswordEntry()
	refreshTokenEntry.SetPlaceHolder("Required")

	clientIDEntry := widget.NewEntry()
	clientIDEntry.SetPlaceHolder("Required")

	tenantIDEntry := widget.NewEntry()
	tenantIDEntry.SetPlaceHolder("Required")

	vaultNameEntry := widget.NewEntry()
	vaultNameEntry.SetPlaceHolder("Optional — leave blank to enumerate all vaults")

	outputPathEntry := widget.NewEntry()
	outputPathEntry.SetText("./KVReconOutput")

	form := widget.NewForm(
		widget.NewFormItem("Refresh Token", refreshTokenEntry),
		widget.NewFormItem("Client ID", clientIDEntry),
		widget.NewFormItem("Tenant ID", tenantIDEntry),
		widget.NewFormItem("Vault Name", vaultNameEntry),
		widget.NewFormItem("Output Path", outputPathEntry),
	)

	// ── Log panel ─────────────────────────────────────────────────
	logEntry := widget.NewMultiLineEntry()
	logEntry.Disable()
	logEntry.Wrapping = fyne.TextWrapWord
	logScroll := container.NewScroll(logEntry)
	logScroll.SetMinSize(fyne.NewSize(840, 380))

	logCh := make(chan string, 512)

	// Drain the log channel and append lines to the UI entry.
	go func() {
		for msg := range logCh {
			msg := msg // capture for closure
			fyne.Do(func() {
				cur := logEntry.Text
				if cur == "" {
					logEntry.SetText(msg)
				} else {
					logEntry.SetText(cur + "\n" + msg)
				}
				logScroll.ScrollToBottom()
			})
		}
	}()

	// ── Buttons ───────────────────────────────────────────────────
	var runBtn *widget.Button

	clearBtn := widget.NewButton("Clear Log", func() {
		logEntry.SetText("")
	})

	openDirBtn := widget.NewButton("Open Output Folder", func() {
		path := outputPathEntry.Text
		if path == "" {
			path = "./KVReconOutput"
		}
		var cmd *exec.Cmd
		switch runtime.GOOS {
		case "windows":
			cmd = exec.Command("explorer", path)
		case "darwin":
			cmd = exec.Command("open", path)
		default:
			cmd = exec.Command("xdg-open", path)
		}
		if err := cmd.Start(); err != nil {
			dialog.ShowError(err, w)
		}
	})

	runBtn = widget.NewButton("▶  Run Recon", func() {
		if refreshTokenEntry.Text == "" || clientIDEntry.Text == "" || tenantIDEntry.Text == "" {
			dialog.ShowInformation("Missing Fields",
				"Refresh Token, Client ID, and Tenant ID are all required.", w)
			return
		}

		runBtn.Disable()
		logEntry.SetText("")

		cfg := ReconConfig{
			RefreshToken: refreshTokenEntry.Text,
			ClientID:     clientIDEntry.Text,
			TenantID:     tenantIDEntry.Text,
			VaultName:    vaultNameEntry.Text,
			OutputPath:   outputPathEntry.Text,
			KVAPIVersion: "7.4",
		}
		if cfg.OutputPath == "" {
			cfg.OutputPath = "./KVReconOutput"
		}

		go func() {
			RunRecon(cfg, logCh)
			fyne.Do(func() {
				runBtn.Enable()
			})
		}()
	})

	// ── Layout ────────────────────────────────────────────────────
	title := widget.NewLabelWithStyle(
		"Azure Key Vault Recon",
		fyne.TextAlignCenter,
		fyne.TextStyle{Bold: true},
	)

	buttonBar := container.NewHBox(runBtn, clearBtn, openDirBtn)

	content := container.NewVBox(
		title,
		form,
		widget.NewSeparator(),
		widget.NewLabel("Output Log"),
		logScroll,
		buttonBar,
	)

	w.SetContent(container.NewPadded(content))
	w.ShowAndRun()
}

// ReconConfig holds all parameters collected from the GUI form.
type ReconConfig struct {
	RefreshToken string
	ClientID     string
	TenantID     string
	VaultName    string
	OutputPath   string
	KVAPIVersion string
}
