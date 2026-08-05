package main

import (
	"github.com/diamondburned/gotk4/pkg/core/glib"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

type ProgressWindow struct {
	value       int64
	total       int64
	window      *gtk.ApplicationWindow
	statusLabel *gtk.Label
	progressBar *gtk.ProgressBar
}

func createProgressWindow() {
	window := gtk.NewApplicationWindow(launcher.App)
	window.SetTitle("...")
	window.SetDefaultSize(300, -1)
	window.SetResizable(false)
	window.SetHideOnClose(true)
	window.SetDeletable(false)

	// Create vertical box
	vbox := gtk.NewBox(gtk.Orientation(gtk.OrientationVertical), 10)
	vbox.SetMarginTop(10)
	vbox.SetMarginBottom(10)
	vbox.SetMarginStart(10)
	vbox.SetMarginEnd(10)
	// Create progress label
	statusLabel := gtk.NewLabel("...")
	statusLabel.SetHAlign(gtk.Align(gtk.AlignCenter))
	// Create progress bar
	progressBar := gtk.NewProgressBar()

	vbox.Append(statusLabel)
	vbox.Append(progressBar)
	window.SetChild(vbox)

	launcher.ProgressWindow = ProgressWindow{
		total:       0,
		window:      window,
		statusLabel: statusLabel,
		progressBar: progressBar,
	}
}

func (window *ProgressWindow) ResetProgressWindow(status string, progress, total int) {
	window.ResetProgressWindow64(status, int64(progress), int64(total))
}

func (window *ProgressWindow) ResetProgressWindow64(status string, progress, total int64) {
	window.SetStatus(status)
	window.Set64(progress)
	window.SetTotal64(total)
	window.Show()
}

func (window *ProgressWindow) Show() {
	glib.IdleAdd(func() {
		window.window.SetVisible(true)
	})
}

func (window *ProgressWindow) Hide() {
	glib.IdleAdd(func() {
		window.window.SetVisible(false)
	})
}

func (window *ProgressWindow) Pulse() {
	glib.IdleAdd(func() {
		window.progressBar.Pulse()
	})
}

func (window *ProgressWindow) SetStatus(status string) {
	glib.IdleAdd(func() {
		window.window.SetTitle(status)
		window.statusLabel.SetText(status)
	})
}

func (window *ProgressWindow) SetTotal64(total int64) {
	window.total = max(total, 0)
}

func (window *ProgressWindow) SetTotal(total int) {
	window.SetTotal64(int64(total))
}

func (window *ProgressWindow) Set64(progress int64) {
	window.value = min(progress, window.total)

	glib.IdleAdd(func() {
		window.progressBar.SetFraction(float64(window.value) / float64(window.total))
	})
}

func (window *ProgressWindow) Set(progress int) {
	window.Set64(int64(progress))
}

func (window *ProgressWindow) Add64(progress int64) {
	window.value = min(window.value+progress, window.total)

	glib.IdleAdd(func() {
		window.progressBar.SetFraction(float64(window.value) / float64(window.total))
	})
}

func (window *ProgressWindow) Add(progress int) {
	window.Add64(int64(progress))
}

func (window *ProgressWindow) Write(p []byte) (int, error) {
	window.Add(len(p))

	return len(p), nil
}
