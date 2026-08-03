package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

var app *gtk.Application
var metadata Metadata
var selectedRunner string

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
	if metadata.Type == "windows" {
		runners, err := GetWindowsRunners()
		if err != nil || len(runners) == 0 {
			runnerDropdown.SetSensitive(false)
			runnerDropdown.SetModel(gtk.NewStringList([]string{"No runners available"}))
		} else {
			options := make([]string, 0)
			for k, v := range runners {
				if selectedRunner == "" {
					selectedRunner = v
				}

				if strings.HasPrefix(v, "https://") || strings.HasPrefix(v, "http://") {
					options = append(options, fmt.Sprintf("%s (Downloadable)", k))
				} else {
					options = append(options, k)
				}
			}
			runnerDropdown.SetModel(gtk.NewStringList(options))
			runnerDropdown.Connect("notify::selected", func() {
				obj := runnerDropdown.SelectedItem().Cast().(*gtk.StringObject)
				selectedRunner = strings.TrimSpace(strings.SplitN(obj.String(), "(", 2)[0])
			})
		}
	}
	// Play button
	playButton := gtk.NewButtonWithLabel("Play from disc")
	playButton.ConnectClicked(PlayFromDisc)
	if !metadata.RunFromDisc {
		playButton.SetLabel("Play from disc (This game can't run from disc)")
		playButton.SetSensitive(false)
	}
	// Install and uninstall buttons
	installButton := gtk.NewButtonWithLabel("Install")
	uninstallButton := gtk.NewButtonWithLabel("Uninstall")

	installButton.ConnectClicked(func() { InstallGame(); installButton.SetVisible(false); uninstallButton.SetVisible(true) })
	uninstallButton.ConnectClicked(func() { UninstallGame(); installButton.SetVisible(true); uninstallButton.SetVisible(false) })

	if _, err := os.Stat(filepath.Join(homeDir, "Games", metadata.Name)); err == nil {
		installButton.SetVisible(false)
		uninstallButton.SetVisible(true)
	} else {
		installButton.SetVisible(true)
		uninstallButton.SetVisible(false)
	}

	labelBox.Append(gameIcon)
	labelBox.Append(versionLabel)
	labelBox.Append(developerLabel)
	labelBox.Append(publisherLabel)
	labelBox.Append(gameTypeLabel)
	metadataBox.Append(labelBox)
	metadataBox.Append(scrolledWindow)
	mainBox.Append(metadataBox)
	mainBox.Append(runnerDropdown)
	mainBox.Append(playButton)
	mainBox.Append(installButton)
	mainBox.Append(uninstallButton)

	window.SetChild(mainBox)

	window.SetVisible(true)
}

func PlayFromDisc() {
	if !metadata.RunFromDisc {
		return
	}

	// Close launcher windows
	for _, window := range app.Windows() {
		window.Close()
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

		// Check for umu
		if _, err := exec.LookPath("umu-run"); err != nil {
			log.Fatalf("umu-run not found in PATH")
		}

		prefixPath := filepath.Join(gameDataDir, "prefix")

		// Run winetricks
		if _, err := os.Stat(prefixPath); err != nil && len(metadata.WinetricksVerbs) > 0 {
			// Setup command
			cmd := exec.Command("umu-run", "winetricks")
			cmd.Args = append(cmd.Args, metadata.WinetricksVerbs...)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			cmd.Dir = filepath.Join(workDir, "files")

			// Setup environment
			cmd.Env = os.Environ()
			cmd.Env = append(cmd.Env, "PROTONPATH="+selectedRunner)
			cmd.Env = append(cmd.Env, "WINEPREFIX="+prefixPath)

			err = cmd.Run()
			if err != nil {
				log.Fatal(err)
			}
		}

		// Setup command
		cmd := exec.Command("umu-run", metadata.Run)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Dir = filepath.Join(workDir, "files")

		// Setup environment
		cmd.Env = os.Environ()
		cmd.Env = append(cmd.Env, "PROTONPATH="+selectedRunner)
		cmd.Env = append(cmd.Env, "WINEPREFIX="+filepath.Join(homeDir, "Games", metadata.Name, "prefix"))

		err = cmd.Run()
		if err != nil {
			log.Fatal(err)
		}
	case "bin", "script":
		// Run Linux binary/script

		// Setup command
		cmd := exec.Command(metadata.Run)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Dir = filepath.Join(workDir, "files")

		// Setup environment
		cmd.Env = os.Environ()

		err := cmd.Run()
		if err != nil {
			log.Fatal(err)
		}
	}
}

func InstallGame() {
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

	fsDir := os.DirFS(filepath.Join(workDir, "files"))
	err = os.CopyFS(filepath.Join(gameDataDir, "files"), fsDir)
	if err != nil {
		log.Fatal(err)
	}

	// Copy icon
	b, err := os.ReadFile(filepath.Join(workDir, "icon.png"))
	if err != nil {
		log.Fatal(err)
	}
	err = os.WriteFile(filepath.Join(gameDataDir, "icon.png"), b, 0644)
	if err != nil {
		log.Fatal(err)
	}

	// Copy metadata
	b, err = os.ReadFile(filepath.Join(workDir, "metadata.yml"))
	if err != nil {
		log.Fatal(err)
	}
	err = os.WriteFile(filepath.Join(gameDataDir, "metadata.yml"), b, 0644)
	if err != nil {
		log.Fatal(err)
	}

	prefixPath := filepath.Join(gameDataDir, "prefix")

	// Create run script and desktop file
	if metadata.Type == "windows" {
		toWrite := fmt.Sprintf(`#!/bin/sh
cd "%s/files"

# Set environment variables for umu-launcher
export PROTONPATH="%s"
export WINEPREFIX="%s"

# Run game
umu-run "%s"
`, strings.ReplaceAll(gameDataDir, homeDir, "$HOME"), strings.ReplaceAll(selectedRunner, homeDir, "$HOME"), strings.ReplaceAll(prefixPath, homeDir, "$HOME"), metadata.Run)

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
	}

	// Create uninstall script
	toWrite := fmt.Sprintf(`#!/bin/sh
# Remove desktop file
rm -f "%s"

# Remove game data directory
rm -rf "%s"
`, strings.ReplaceAll(filepath.Join(homeDir, ".local/share/applications/games", metadata.Name+".desktop"), homeDir, "$HOME"), strings.ReplaceAll(gameDataDir, homeDir, "$HOME"))
	err = os.WriteFile(filepath.Join(gameDataDir, "uninstall.sh"), []byte(toWrite), 0755)
	if err != nil {
		log.Fatal(err)
	}

	// Setup prefix
	if _, err := os.Stat(prefixPath); err != nil && len(metadata.WinetricksVerbs) > 0 {
		// Setup command
		cmd := exec.Command("umu-run", "winetricks")
		cmd.Args = append(cmd.Args, metadata.WinetricksVerbs...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Dir = filepath.Join(gameDataDir, "files")

		// Setup environment
		cmd.Env = os.Environ()
		cmd.Env = append(cmd.Env, "PROTONPATH="+selectedRunner)
		cmd.Env = append(cmd.Env, "WINEPREFIX="+prefixPath)

		err = cmd.Run()
		if err != nil {
			log.Fatal(err)
		}
	}
}

func UninstallGame() {
	// Get user home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}

	// Get game data directory
	gameDataDir := filepath.Join(homeDir, "Games", metadata.Name)

	// Setup command
	cmd := exec.Command(filepath.Join(gameDataDir, "uninstall.sh"))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
}
