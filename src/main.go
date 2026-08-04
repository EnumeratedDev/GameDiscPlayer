package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

var app *gtk.Application
var metadata Metadata
var selectedRunner *Runner

func main() {
	var err error

	// Read game metadata
	metadata, err = ReadMetadata()
	if err != nil {
		log.Fatalf("Game metadata not found: %s", err)
	}

	app = gtk.NewApplication("com.github.diamondburned.gotk4-examples.gtk4.simple", gio.ApplicationFlags(gio.ApplicationFlagsNone))
	app.ConnectActivate(activate)

	if code := app.Run(os.Args); code > 0 {
		os.Exit(code)
	}
}

func activate() {
	// Get user home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}

	window := gtk.NewApplicationWindow(app)
	window.SetTitle("Play " + metadata.Name)
	window.SetDefaultSize(600, -1)
	window.SetResizable(false)

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
	gameIcon.SetSizeRequest(64, 64)
	gameIcon.SetHExpand(false)
	gameIcon.SetVExpand(false)
	// Labels
	versionLabel := gtk.NewLabel("Version: " + metadata.Version)
	developerLabel := gtk.NewLabel("Developer: " + metadata.Developer)
	publisherLabel := gtk.NewLabel("Developer: " + metadata.Publisher)
	gameTypeLabel := gtk.NewLabel("Type: " + typeStrings[metadata.Type])

	// Description text view
	textBuffer := gtk.NewTextBuffer(nil)
	textBuffer.SetText(metadata.Description)
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
		switch metadata.Type {
		case "windows":
			runners, err := GetWindowsRunners()
			if err != nil || len(runners) == 0 {
				runnerDropdown.SetSensitive(false)
				runnerDropdown.SetModel(gtk.NewStringList([]string{"No runners available"}))
			} else {
				options := make([]string, 0)
				for _, runner := range runners {
					if strings.HasPrefix(runner.Path, "https://") || strings.HasPrefix(runner.Path, "http://") {
						options = append(options, fmt.Sprintf("%s (Downloadable)", runner.DisplayName))
					} else {
						options = append(options, runner.DisplayName)
					}
				}
				selectedRunner = &runners[0]
				runnerDropdown.SetModel(gtk.NewStringList(options))
				runnerDropdown.Connect("notify::selected", func() {
					selectedRunner = &runners[runnerDropdown.Selected()]
				})
			}
		}
	}
	setRunnerDropdownOptions()
	// Play buttons
	playButton := gtk.NewButtonWithLabel("Play")
	playButton.ConnectClicked(func() {
		if _, err := os.Stat(filepath.Join(homeDir, "Games", metadata.Name)); err == nil || !metadata.RunFromDisc {
			window.SetSensitive(false)

			progressWindow := NewProgressWindow("Running "+metadata.Name+"...", 0)

			go Play(progressWindow)
		} else {
			window.SetSensitive(false)

			progressWindow := NewProgressWindow("Running "+metadata.Name+" from disc...", 0)

			go PlayFromDisc(progressWindow)
		}
	})
	// Install and uninstall buttons
	installButton := gtk.NewButtonWithLabel("Install")
	uninstallButton := gtk.NewButtonWithLabel("Uninstall")

	installButton.ConnectClicked(func() {
		window.SetSensitive(false)

		progressWindow := NewProgressWindow("Installing "+metadata.Name+"...", 0)

		onFinish := func() {
			setRunnerDropdownOptions()
			installButton.SetVisible(false)
			uninstallButton.SetVisible(true)

			playButton.SetLabel("Play")
			playButton.SetSensitive(true)

			window.SetSensitive(true)
		}

		go InstallGame(progressWindow, onFinish)

	})
	uninstallButton.ConnectClicked(func() {
		window.SetSensitive(false)

		progressWindow := NewProgressWindow("Running uninstall script...", 0)

		onFinish := func() {
			installButton.SetVisible(true)
			uninstallButton.SetVisible(false)

			if metadata.RunFromDisc {
				playButton.SetLabel("Play from disc")
				playButton.SetSensitive(true)
			} else {
				playButton.SetLabel("Play from disc (This game can't run from disc)")
				playButton.SetSensitive(false)
			}

			window.SetSensitive(true)
		}

		go UninstallGame(progressWindow, onFinish)
	})

	if _, err := os.Stat(filepath.Join(homeDir, "Games", metadata.Name)); err == nil {
		installButton.SetVisible(false)
		uninstallButton.SetVisible(true)
	} else {
		installButton.SetVisible(true)
		uninstallButton.SetVisible(false)

		if metadata.RunFromDisc {
			playButton.SetLabel("Play from disc")
		} else {
			playButton.SetLabel("Play from disc (This game can't run from disc)")
			playButton.SetSensitive(false)
		}
	}

	if selectedRunner == nil {
		playButton.SetSensitive(false)
		installButton.SetSensitive(false)
	}

	labelBox.Append(gameIcon)
	labelBox.Append(versionLabel)
	labelBox.Append(developerLabel)
	labelBox.Append(publisherLabel)
	labelBox.Append(gameTypeLabel)
	metadataBox.Append(labelBox)
	metadataBox.Append(scrolledWindow)
	mainBox.Append(metadataBox)
	if metadata.Type != "bin" || metadata.Type != "script" {
		mainBox.Append(runnerDropdown)
	}
	mainBox.Append(playButton)
	mainBox.Append(installButton)
	mainBox.Append(uninstallButton)

	window.SetChild(mainBox)

	window.SetVisible(true)
}

func Play(progressWindow ProgressWindow) {
	// Get user home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}

	// Get game data directory
	gameDataDir := filepath.Join(homeDir, "Games", metadata.Name)

	switch metadata.Type {
	case "windows":

		prefixDir := filepath.Join(gameDataDir, "prefix")

		progressWindow := NewProgressWindow("Downloading "+selectedRunner.DisplayName, 0)

		// Download runner if required
		selectedRunner.Download(progressWindow)

		// Update run script to use selected runner
		toWrite := fmt.Sprintf(`#!/bin/sh
cd "%s/files"

# Set environment variables for umu-launcher
[ -z "$PROTONPATH" ] && export PROTONPATH="%s"
export WINEPREFIX="%s"

# Run game
umu-run "%s"
`, strings.ReplaceAll(gameDataDir, homeDir, "$HOME"), strings.ReplaceAll(selectedRunner.Path, homeDir, "$HOME"), strings.ReplaceAll(prefixDir, homeDir, "$HOME"), metadata.Run)

		err = os.WriteFile(filepath.Join(gameDataDir, "run.sh"), []byte(toWrite), 0755)
		if err != nil {
			log.Fatal(err)
		}

		glib.IdleAdd(func() {
			// Close launcher windows
			for _, window := range app.Windows() {
				window.Close()
			}

			// Run game
			err = os.Chdir(gameDataDir)
			if err != nil {
				log.Fatal(err)
			}
			err = syscall.Exec("run.sh", []string{}, os.Environ())
			if err != nil {
				log.Fatal(err)
			}
		})
	case "bin", "script":
		// Run Linux binary/script

		// Close launcher windows
		glib.IdleAdd(func() {
			// Close launcher windows
			for _, window := range app.Windows() {
				window.Close()
			}

			// Run game
			err = os.Chdir(filepath.Join(gameDataDir, "files"))
			if err != nil {
				log.Fatal(err)
			}
			err = syscall.Exec(metadata.Run, []string{}, os.Environ())
			if err != nil {
				log.Fatal(err)
			}
		})
	}
}

func PlayFromDisc(progressWindow ProgressWindow) {
	if !metadata.RunFromDisc {
		return
	}

	// Get working directory
	workDir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	// Get user home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}

	// Create game data directory
	gameDataDir := filepath.Join(homeDir, "Games", metadata.Name)
	err = os.MkdirAll(gameDataDir, 0755)
	if err != nil {
		log.Fatal(err)
	}

	switch metadata.Type {
	case "windows":
		// Run Windows executables with umu

		// Download runner if required
		selectedRunner.Download(progressWindow)

		// Check for umu
		umuPath, err := exec.LookPath("umu-run")
		if err != nil {
			log.Fatalf("umu-run not found in PATH")
		}

		prefixDir := filepath.Join(gameDataDir, "prefix")

		// Run winetricks
		if _, err := os.Stat(prefixDir); err != nil && len(metadata.WinetricksVerbs) > 0 {
			progressWindow.SetStatus("Setting up prefix...")
			progressWindow.Set(0)
			progressWindow.SetTotal(1)

			// Setup command
			cmd := exec.Command("umu-run", "winetricks")
			cmd.Args = append(cmd.Args, metadata.WinetricksVerbs...)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			cmd.Dir = filepath.Join(workDir, "files")

			// Setup environment
			cmd.Env = os.Environ()
			cmd.Env = append(cmd.Env, "PROTONPATH="+selectedRunner.Path)
			cmd.Env = append(cmd.Env, "WINEPREFIX="+prefixDir)

			err = cmd.Run()
			if err != nil {
				log.Fatal(err)
			}
		}

		// Setup environment
		env := os.Environ()
		env = append(env, "PROTONPATH="+selectedRunner.Path)
		env = append(env, "WINEPREFIX="+filepath.Join(homeDir, "Games", metadata.Name, "prefix"))

		glib.IdleAdd(func() {
			// Close launcher windows
			for _, window := range app.Windows() {
				window.Close()
			}

			// Run game
			err = os.Chdir(filepath.Join(workDir, "files"))
			if err != nil {
				log.Fatal(err)
			}
			err = syscall.Exec(umuPath, []string{metadata.Run}, env)
			if err != nil {
				log.Fatal(err)
			}
		})
	case "bin", "script":
		// Run Linux binary/script

		glib.IdleAdd(func() {
			// Close launcher windows
			for _, window := range app.Windows() {
				window.Close()
			}

			// Run game
			err = os.Chdir(filepath.Join(workDir, "files"))
			if err != nil {
				log.Fatal(err)
			}
			err = syscall.Exec(metadata.Run, []string{}, os.Environ())
			if err != nil {
				log.Fatal(err)
			}
		})
	}
}

func InstallGame(progressWindow ProgressWindow, onFinish func()) {
	// Get working directory
	workDir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	// Get user home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}

	// Create game data directory
	gameDataDir := filepath.Join(homeDir, "Games", metadata.Name)
	err = os.MkdirAll(gameDataDir, 0755)
	if err != nil {
		log.Fatal(err)
	}

	err = CopyRecursivelyWithProgress(filepath.Join(workDir, "files"), filepath.Join(gameDataDir, "files"), progressWindow)
	if err != nil {
		log.Fatal(err)
	}

	// Copy icon
	err = Copy(filepath.Join(workDir, "icon.png"), filepath.Join(gameDataDir, "icon.png"))
	if err != nil {
		log.Fatal(err)
	}

	// Copy metadata
	err = Copy(filepath.Join(workDir, "metadata.yml"), filepath.Join(gameDataDir, "metadata.yml"))
	if err != nil {
		log.Fatal(err)
	}

	prefixDir := filepath.Join(gameDataDir, "prefix")

	// Create run script and desktop file
	switch metadata.Type {
	case "windows":
		toWrite := fmt.Sprintf(`#!/bin/sh
cd "%s/files"

# Set environment variables for umu-launcher
[ -z "$PROTONPATH" ] && export PROTONPATH="%s"
export WINEPREFIX="%s"

# Run game
umu-run "%s"
`, strings.ReplaceAll(gameDataDir, homeDir, "$HOME"), strings.ReplaceAll(selectedRunner.Path, homeDir, "$HOME"), strings.ReplaceAll(prefixDir, homeDir, "$HOME"), metadata.Run)

		err = os.WriteFile(filepath.Join(gameDataDir, "run.sh"), []byte(toWrite), 0755)
		if err != nil {
			log.Fatal(err)
		}

		toWrite = fmt.Sprintf(`[Desktop Entry]
Name=%s
Comment=Game Disc Player
Exec="%s/run.sh"
Icon=%s/icon.png
Terminal=false
Type=Application
Categories=Game;
`, metadata.Name, gameDataDir, gameDataDir)
		err = os.MkdirAll(filepath.Join(homeDir, ".local/share/applications/games"), 0755)
		if err != nil {
			log.Fatal(err)
		}
		err = os.WriteFile(filepath.Join(homeDir, ".local/share/applications/games", metadata.Name+".desktop"), []byte(toWrite), 0755)
		if err != nil {
			log.Fatal(err)
		}

		// Create uninstall script
		toWrite = fmt.Sprintf(`#!/bin/sh
# Remove desktop file
rm -f "%s"

# Remove game data directory
rm -rf "%s"
`, strings.ReplaceAll(filepath.Join(homeDir, ".local/share/applications/games", metadata.Name+".desktop"), homeDir, "$HOME"), strings.ReplaceAll(gameDataDir, homeDir, "$HOME"))
		err = os.WriteFile(filepath.Join(gameDataDir, "uninstall.sh"), []byte(toWrite), 0755)
		if err != nil {
			log.Fatal(err)
		}

		// Download runner if required
		selectedRunner.Download(progressWindow)

		// Setup prefix
		if _, err := os.Stat(prefixDir); err != nil && len(metadata.WinetricksVerbs) > 0 {
			progressWindow.SetStatus("Setting up prefix...")
			progressWindow.Set(0)
			progressWindow.SetTotal(1)

			// Setup command
			cmd := exec.Command("umu-run", "winetricks")
			cmd.Args = append(cmd.Args, metadata.WinetricksVerbs...)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			cmd.Dir = filepath.Join(gameDataDir, "files")

			// Setup environment
			cmd.Env = os.Environ()
			cmd.Env = append(cmd.Env, "PROTONPATH="+selectedRunner.Path)
			cmd.Env = append(cmd.Env, "WINEPREFIX="+prefixDir)

			err = cmd.Run()
			if err != nil {
				log.Fatal(err)
			}
		}
	}

	progressWindow.CloseWindow()

	glib.IdleAdd(func() {
		onFinish()
	})
}

func UninstallGame(progressWindow ProgressWindow, onFinish func()) {
	progressWindow.Pulse()

	// Get user home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}

	// Get game data directory
	if metadata.Name == "" {
		// Just to be sure I don't accidentally delete everything in the Games directory
		log.Fatalf("game name is empty")
	}
	gameDataDir := filepath.Join(homeDir, "Games", metadata.Name)

	// Removing files
	RemoveDirectoryRecursively(gameDataDir, progressWindow)

	err = os.Remove(filepath.Join(homeDir, ".local/share/applications/games", metadata.Name+".desktop"))
	if err != nil && !os.IsNotExist(err) {
		log.Fatal(err)
	}

	progressWindow.CloseWindow()

	glib.IdleAdd(func() {
		onFinish()
	})
}
