package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

func main() {
	if appimage, ok := os.LookupEnv("APPIMAGE"); ok {
		// Change working directory to appimage file's location
		err := os.Chdir(filepath.Dir(appimage))
		if err != nil {
			log.Fatal(err)
		}
		// Fix python environment
		os.Unsetenv("PYTHONHOME")
		os.Unsetenv("PYTHONPATH")
	} else {
		// Change working directory to binary's location
		exe, err := os.Executable()
		if err != nil {
			log.Fatal(err)
		}
		err = os.Chdir(filepath.Dir(exe))
		if err != nil {
			log.Fatal(err)
		}
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}

	// Read game metadata
	launcher.Metadata, err = ReadMetadata()
	if err != nil {
		log.Fatalf("Game metadata not found: %s", err)
	}

	// Set data directory
	launcher.DataDir = filepath.Join(homeDir, "Games", launcher.Metadata.Name)

	// Parse options
	err = launcher.ParseOptions()
	if err != nil {
		log.Fatal(err)
	}

	launcher.App = gtk.NewApplication("dev.enumerated.GameDiscPlayer", gio.ApplicationFlags(gio.ApplicationNonUnique))
	launcher.App.ConnectActivate(activate)

	if code := launcher.App.Run(os.Args); code > 0 {
		os.Exit(code)
	}
}

func activate() {
	// Set launcher icon
	gtk.WindowSetDefaultIconName("GameDiscPlayer")

	createMainWindow()

	launcher.MainWindow.SetVisible(true)
}

func createMainWindow() {
	// Setup main launcher window
	launcher.MainWindow = gtk.NewApplicationWindow(launcher.App)
	launcher.MainWindow.SetTitle("Play " + launcher.Metadata.Name)
	launcher.MainWindow.SetDefaultSize(600, -1)
	launcher.MainWindow.SetResizable(false)
	launcher.MainWindow.ConnectCloseRequest(func() bool {
		launcher.App.Quit()
		return true
	})

	// Main vertical box
	mainBox := gtk.NewBox(gtk.Orientation(gtk.OrientationVertical), 10)
	mainBox.SetMarginTop(10)
	mainBox.SetMarginBottom(10)
	mainBox.SetMarginStart(10)
	mainBox.SetMarginEnd(10)
	// Horizontal metadata box
	metadataBox := gtk.NewBox(gtk.Orientation(gtk.OrientationHorizontal), 10)
	// Vertical label box
	labelBox := gtk.NewBox(gtk.Orientation(gtk.OrientationVertical), 10)
	// Game icon
	gameIcon := gtk.NewPictureForFilename("icon.png")
	gameIcon.SetSizeRequest(256, 256)
	gameIcon.SetHExpand(false)
	gameIcon.SetVExpand(false)
	// Labels
	versionLabel := gtk.NewLabel("Version: " + launcher.Metadata.Version)
	developerLabel := gtk.NewLabel("Developer: " + launcher.Metadata.Developer)
	publisherLabel := gtk.NewLabel("Publisher: " + launcher.Metadata.Publisher)
	systemLabel := gtk.NewLabel("System: " + systemsUserReadable[launcher.Metadata.System])

	// Description text view
	textBuffer := gtk.NewTextBuffer(nil)
	textBuffer.SetText(launcher.Metadata.Description)
	textView := gtk.NewTextViewWithBuffer(textBuffer)
	textView.SetWrapMode(gtk.WrapMode(gtk.WrapWord))
	textView.SetLeftMargin(10)
	textView.SetRightMargin(10)
	textView.SetEditable(false)
	scrolledWindow := gtk.NewScrolledWindow()
	scrolledWindow.SetHExpand(true)
	scrolledWindow.SetSizeRequest(-1, 200)
	scrolledWindow.SetPolicy(gtk.PolicyType(gtk.PolicyAutomatic), gtk.PolicyType(gtk.PolicyAutomatic))
	scrolledWindow.SetChild(textView)
	// Runner dropdown
	runnerDropdown := gtk.NewDropDownFromStrings(nil)
	setRunnerDropdownOptions := func() {
		var err error
		var runners []Runner

		runnerFetcher, ok := RunnerFetchers[launcher.Metadata.System]
		if ok {
			runners, err = runnerFetcher()
			if err != nil || len(runners) == 0 {
				runnerDropdown.SetSensitive(false)
				runnerDropdown.SetModel(gtk.NewStringList([]string{"No runners available"}))
			}
		}

		if len(runners) > 0 {
			options := make([]string, 0)
			selectedRunnerIndex := 0

			for i, runner := range runners {
				if runner.DisplayName == "" && runner.DisplayName == launcher.Options.Runner {
					selectedRunnerIndex = i
				}

				if strings.HasPrefix(runner.Run, "https://") || strings.HasPrefix(runner.Run, "http://") {
					options = append(options, fmt.Sprintf("%s (Downloadable)", runner.DisplayName))
				} else {
					options = append(options, runner.DisplayName)
				}
			}
			runnerDropdown.SetModel(gtk.NewStringList(options))

			runnerDropdown.SetSelected(uint(selectedRunnerIndex))
			launcher.SelectedRunner = &runners[selectedRunnerIndex]

			runnerDropdown.Connect("notify::selected", func() {
				launcher.SelectedRunner = &runners[runnerDropdown.Selected()]
				launcher.Options.Runner = launcher.SelectedRunner.DisplayName
			})
		}
	}
	setRunnerDropdownOptions()
	// Play buttons
	playButton := gtk.NewButtonWithLabel("Play")
	playButton.ConnectClicked(func() {
		if launcher.IsGameInstalled() {
			go launcher.Play()
		} else {
			go launcher.PlayFromDisc()
		}
	})
	// Install and uninstall buttons
	installButton := gtk.NewButtonWithLabel("Install")
	uninstallButton := gtk.NewButtonWithLabel("Uninstall")

	installButton.ConnectClicked(func() {
		go launcher.InstallGame()
	})
	uninstallButton.ConnectClicked(func() {

		launcher.MainWindow.SetSensitive(false)

		go launcher.UninstallGame()
	})

	if launcher.IsGameInstalled() {
		installButton.SetVisible(false)
		uninstallButton.SetVisible(true)
	} else {
		installButton.SetVisible(true)
		uninstallButton.SetVisible(false)

		if launcher.Metadata.RunFromDisc {
			playButton.SetLabel("Play from disc")
		} else {
			playButton.SetLabel("Play from disc (This game can't run from disc)")
			playButton.SetSensitive(false)
		}
	}

	if launcher.Metadata.System == "linux" {
		runnerDropdown.SetVisible(false)
	} else if launcher.SelectedRunner == nil {
		playButton.SetSensitive(false)
		installButton.SetSensitive(false)
	}

	labelBox.Append(gameIcon)
	labelBox.Append(versionLabel)
	labelBox.Append(developerLabel)
	labelBox.Append(publisherLabel)
	labelBox.Append(systemLabel)
	metadataBox.Append(labelBox)
	metadataBox.Append(scrolledWindow)
	mainBox.Append(metadataBox)
	mainBox.Append(runnerDropdown)

	mainBox.Append(playButton)
	mainBox.Append(installButton)
	mainBox.Append(uninstallButton)

	launcher.MainWindow.SetChild(mainBox)
}
