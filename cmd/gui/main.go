package main

import (
	"os"
	"path/filepath"
	"strings"

	"youtube-downloader/internal/downloader"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

func main() {
	a := app.New()
	w := a.NewWindow("YouTube Downloader")
	w.Resize(fyne.NewSize(520, 360))
	w.SetFixedSize(true)

	// --- URL ---
	urlEntry := widget.NewEntry()
	urlEntry.SetPlaceHolder("https://youtube.com/watch?v=...")

	// --- Filename ---
	filenameEntry := widget.NewEntry()
	filenameEntry.SetPlaceHolder("Leave blank to use video title")

	// --- Output folder ---
	home, _ := os.UserHomeDir()
	outDir := filepath.Join(home, "Downloads")
	outDirLabel := widget.NewLabel(outDir)
	outDirLabel.Truncation = fyne.TextTruncateEllipsis
	browseBtn := widget.NewButtonWithIcon("Browse", theme.FolderOpenIcon(), func() {
		dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
			if err != nil || uri == nil {
				return
			}
			outDir = uri.Path()
			outDirLabel.SetText(outDir)
		}, w)
	})

	// --- Format ---
	formatSelect := widget.NewSelect([]string{"mp3", "mp4", "wav"}, nil)
	formatSelect.SetSelected("mp3")

	// --- Quality ---
	qualitySelect := widget.NewSelect([]string{"best", "medium", "worst"}, nil)
	qualitySelect.SetSelected("best")

	// --- Status / progress ---
	status := widget.NewLabel("Ready.")
	status.Wrapping = fyne.TextWrapWord
	progressBar := widget.NewProgressBar()
	progressBar.Hide()

	// --- Download button ---
	var downloadBtn *widget.Button
	downloadBtn = widget.NewButtonWithIcon("Download", theme.DownloadIcon(), func() {
		url := strings.TrimSpace(urlEntry.Text)
		if url == "" {
			dialog.ShowError(errorf("Please enter a YouTube URL."), w)
			return
		}

		selectedFormat := formatSelect.Selected
		selectedQuality := qualitySelect.Selected

		downloadBtn.Disable()
		progressBar.Show()
		progressBar.SetValue(0)
		status.SetText("Fetching video info...")

		go func() {
			title, _, err := downloader.VideoInfo(url)
			if err != nil {
				showErr(w, downloadBtn, progressBar, status, "Could not fetch video: "+err.Error())
				return
			}

			customName := strings.TrimSpace(filenameEntry.Text)
			base := title
			if customName != "" {
				base = customName
			}
			outPath := filepath.Join(outDir, downloader.SanitizeFilename(base)+"."+selectedFormat)

			status.SetText("Downloading: " + title)

			err = downloader.Download(url, outPath, selectedQuality, selectedFormat, func(p float64) {
				progressBar.SetValue(p)
				status.SetText("Downloading...")
			})
			if err != nil {
				showErr(w, downloadBtn, progressBar, status, err.Error())
				return
			}

			progressBar.SetValue(1)
			msg := "Saved: " + outPath
			if info, err := downloader.ProbeFile(outPath, selectedFormat); err == nil {
				msg += "\n" + info.String()
			}
			status.SetText(msg)
			downloadBtn.Enable()
		}()
	})
	downloadBtn.Importance = widget.HighImportance

	// --- Layout ---
	form := widget.NewForm(
		widget.NewFormItem("URL", urlEntry),
		widget.NewFormItem("Filename", filenameEntry),
		widget.NewFormItem("Save to", container.NewBorder(nil, nil, nil, browseBtn, outDirLabel)),
		widget.NewFormItem("Format", formatSelect),
		widget.NewFormItem("Quality", qualitySelect),
	)

	w.SetContent(container.NewPadded(container.NewVBox(
		form,
		downloadBtn,
		progressBar,
		status,
	)))
	w.ShowAndRun()
}

func showErr(w fyne.Window, btn *widget.Button, bar *widget.ProgressBar, lbl *widget.Label, msg string) {
	bar.Hide()
	lbl.SetText("Error: " + msg)
	btn.Enable()
	dialog.ShowError(errorf(msg), w)
}

type simpleError string

func errorf(s string) error        { return simpleError(s) }
func (e simpleError) Error() string { return string(e) }
