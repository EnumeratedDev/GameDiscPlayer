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

type Launcher struct {
	Metadata Metadata
	DataDir  string
	Options  Options

	App            *gtk.Application
	MainWindow     *gtk.ApplicationWindow
	ProgressWindow ProgressWindow

	SelectedRunner *Runner

	GameProcess *os.Process
}

var launcher = Launcher{Options: Options{}}

func main() {
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
	fmt.Println(launcher.DataDir)

	// Parse options
	err = launcher.ParseOptions()
	if err != nil {
		log.Fatal(err)
	}

	launcher.App = gtk.NewApplication("com.github.diamondburned.gotk4-examples.gtk4.simple", gio.ApplicationFlags(gio.ApplicationFlagsNone))
	launcher.App.ConnectActivate(activate)

	if code := launcher.App.Run(os.Args); code > 0 {
		os.Exit(code)
	}
}

func activate() {
	createLauncherWindow()
	createProgressWindow()

	launcher.MainWindow.SetVisible(true)
}

func createLauncherWindow() {
	// Get user home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}

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
	gameIcon.SetSizeRequest(64, 64)
	gameIcon.SetHExpand(false)
	gameIcon.SetVExpand(false)
	// Labels
	versionLabel := gtk.NewLabel("Version: " + launcher.Metadata.Version)
	developerLabel := gtk.NewLabel("Developer: " + launcher.Metadata.Developer)
	publisherLabel := gtk.NewLabel("Developer: " + launcher.Metadata.Publisher)
	gameTypeLabel := gtk.NewLabel("Type: " + typeStrings[launcher.Metadata.Type])

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
		switch launcher.Metadata.Type {
		case "windows":
			runners, err := GetWindowsRunners()
			if err != nil || len(runners) == 0 {
				runnerDropdown.SetSensitive(false)
				runnerDropdown.SetModel(gtk.NewStringList([]string{"No runners available"}))
			} else {
				options := make([]string, 0)
				selectedRunnerIndex := 0

				for i, runner := range runners {
					if runner.DisplayName == launcher.Options.Runner {
						selectedRunnerIndex = i
					}

					if strings.HasPrefix(runner.Path, "https://") || strings.HasPrefix(runner.Path, "http://") {
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
	}
	setRunnerDropdownOptions()
	// Play buttons
	playButton := gtk.NewButtonWithLabel("Play")
	playButton.ConnectClicked(func() {
		if _, err := os.Stat(filepath.Join(homeDir, "Games", launcher.Metadata.Name)); err == nil || !launcher.Metadata.RunFromDisc {
			go Play()
		} else {
			go PlayFromDisc()
		}
	})
	// Install and uninstall buttons
	installButton := gtk.NewButtonWithLabel("Install")
	uninstallButton := gtk.NewButtonWithLabel("Uninstall")

	installButton.ConnectClicked(func() {
		go InstallGame()
	})
	uninstallButton.ConnectClicked(func() {

		launcher.MainWindow.SetSensitive(false)

		go UninstallGame()
	})

	if IsGameInstalled() {
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

	if launcher.SelectedRunner == nil {
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
	if launcher.Metadata.Type != "bin" || launcher.Metadata.Type != "script" {
		mainBox.Append(runnerDropdown)
	}
	mainBox.Append(playButton)
	mainBox.Append(installButton)
	mainBox.Append(uninstallButton)

	launcher.MainWindow.SetChild(mainBox)
}

func Play() {
	// Hide and show main launcher window
	glib.IdleAdd(func() {
		launcher.MainWindow.SetSensitive(false)
		launcher.MainWindow.SetVisible(false)
	})
	defer glib.IdleAdd(func() {
		launcher.MainWindow.SetSensitive(true)
		launcher.MainWindow.SetVisible(true)
	})

	switch launcher.Metadata.Type {
	case "windows":
		prefixDir := filepath.Join(launcher.DataDir, "prefix")

		// Download runner if required
		launcher.SelectedRunner.Download()

		// Check for umu
		umuPath, err := exec.LookPath("umu-run")
		if err != nil {
			log.Fatalf("umu-run not found in PATH")
		}

		// Set game options
		launcher.Options.Runner = launcher.SelectedRunner.DisplayName
		launcher.SaveOptions()

		// Run game
		cmd := exec.Command(umuPath, launcher.Metadata.Run)
		cmd.Dir = filepath.Join(launcher.DataDir, "files")
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		// Setup environment
		cmd.Env = cmd.Environ()
		cmd.Env = append(cmd.Env, launcher.Options.Environment...)
		cmd.Env = append(cmd.Env, "PROTONPATH="+launcher.SelectedRunner.Path)
		cmd.Env = append(cmd.Env, "WINEPREFIX="+prefixDir)

		err = cmd.Start()
		if err != nil {
			log.Fatal(err)
		}
		launcher.GameProcess = cmd.Process
		err = cmd.Wait()
		if err != nil {
			log.Fatal(err)
		}
	case "bin", "script":
		// Run Linux binary/script

		// Run game
		cmd := exec.Command(launcher.Metadata.Run)
		cmd.Dir = filepath.Join(launcher.DataDir, "files")
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		// Setup environment
		cmd.Env = cmd.Environ()
		cmd.Env = append(cmd.Env, launcher.Options.Environment...)

		err := cmd.Start()
		if err != nil {
			log.Fatal(err)
		}
		launcher.GameProcess = cmd.Process
		err = cmd.Wait()
		if err != nil {
			log.Fatal(err)
		}
	}

	err := launcher.SaveOptions()
	if err != nil {
		log.Fatal(err)
	}
}

func PlayFromDisc() {
	if !launcher.Metadata.RunFromDisc {
		return
	}

	// Hide and show main launcher window
	glib.IdleAdd(func() {
		launcher.MainWindow.SetSensitive(false)
		launcher.MainWindow.SetVisible(false)
	})
	defer glib.IdleAdd(func() {
		launcher.MainWindow.SetSensitive(true)
		launcher.MainWindow.SetVisible(true)
	})

	// Get working directory
	workDir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	// Create game data directory
	err = os.MkdirAll(launcher.DataDir, 0755)
	if err != nil {
		log.Fatal(err)
	}

	switch launcher.Metadata.Type {
	case "windows":
		// Run Windows executables with umu

		// Download runner if required
		launcher.SelectedRunner.Download()

		// Check for umu
		umuPath, err := exec.LookPath("umu-run")
		if err != nil {
			log.Fatalf("umu-run not found in PATH")
		}

		// Set game options
		launcher.Options.Runner = launcher.SelectedRunner.DisplayName
		launcher.SaveOptions()

		prefixDir := filepath.Join(launcher.DataDir, "prefix")

		// Run winetricks
		if _, err := os.Stat(prefixDir); err != nil && len(launcher.Metadata.WinetricksVerbs) > 0 {
			launcher.ProgressWindow.ResetProgressWindow("Setting up prefix...", 0, 1)

			// Setup command
			cmd := exec.Command("umu-run", "winetricks")
			cmd.Args = append(cmd.Args, launcher.Metadata.WinetricksVerbs...)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			cmd.Dir = filepath.Join(workDir, "files")

			// Setup environment
			cmd.Env = os.Environ()
			cmd.Env = append(cmd.Env, "PROTONPATH="+launcher.SelectedRunner.Path)
			cmd.Env = append(cmd.Env, "WINEPREFIX="+prefixDir)

			err = cmd.Run()
			if err != nil {
				log.Fatal(err)
			}
		}

		// Run game
		cmd := exec.Command(umuPath, launcher.Metadata.Run)
		cmd.Dir = filepath.Join(workDir, "files")
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		// Setup environment
		cmd.Env = cmd.Environ()
		cmd.Env = append(cmd.Env, "PROTONPATH="+launcher.SelectedRunner.Path)
		cmd.Env = append(cmd.Env, "WINEPREFIX="+prefixDir)

		err = cmd.Start()
		if err != nil {
			log.Fatal(err)
		}
		launcher.GameProcess = cmd.Process
		err = cmd.Wait()
		if err != nil {
			log.Fatal(err)
		}
	case "bin", "script":
		// Run Linux binary/script

		// Run game
		cmd := exec.Command(launcher.Metadata.Run)
		cmd.Dir = filepath.Join(workDir, "files")
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		err = cmd.Start()
		if err != nil {
			log.Fatal(err)
		}
		launcher.GameProcess = cmd.Process
		err = cmd.Wait()
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
	err = os.MkdirAll(launcher.DataDir, 0755)
	if err != nil {
		log.Fatal(err)
	}

	err = CopyRecursivelyWithProgress(filepath.Join(workDir, "files"), filepath.Join(launcher.DataDir, "files"))
	if err != nil {
		log.Fatal(err)
	}

	// Copy icon
	err = Copy(filepath.Join(workDir, "icon.png"), filepath.Join(launcher.DataDir, "icon.png"))
	if err != nil {
		log.Fatal(err)
	}

	// Copy metadata
	err = Copy(filepath.Join(workDir, "metadata.yml"), filepath.Join(launcher.DataDir, "metadata.yml"))
	if err != nil {
		log.Fatal(err)
	}

	// Set game options
	launcher.Options.Runner = launcher.SelectedRunner.DisplayName
	launcher.SaveOptions()

	// Create .installed file
	f, err := os.Create(filepath.Join(launcher.DataDir, ".installed"))
	if err != nil {
		log.Fatal(err)
	}
	f.Close()

	switch launcher.Metadata.Type {
	case "windows":
		// Create .desktop file
		toWrite := fmt.Sprintf(`[Desktop Entry]
Name=%s
Comment=Play using GameDiscPlayer
Exec="%s/launcher"
Path=%s
Icon=%s/icon.png
Terminal=false
Type=Application
Categories=Game;
`, launcher.Metadata.Name, launcher.DataDir, launcher.DataDir, launcher.DataDir)
		err = os.MkdirAll(filepath.Join(homeDir, ".local/share/applications/games"), 0755)
		if err != nil {
			log.Fatal(err)
		}
		err = os.WriteFile(filepath.Join(homeDir, ".local/share/applications/games", launcher.Metadata.Name+".desktop"), []byte(toWrite), 0755)
		if err != nil {
			log.Fatal(err)
		}

		// Copy launcher into game files
		exe, err := os.Executable()
		if err != nil {
			log.Fatal(err)
		}
		err = Copy(exe, filepath.Join(launcher.DataDir, "launcher"))
		if err != nil {
			log.Fatal(err)
		}

		// Download runner if required
		launcher.SelectedRunner.Download()

		prefixDir := filepath.Join(launcher.DataDir, "prefix")

		// Setup prefix
		if _, err := os.Stat(prefixDir); err != nil && len(launcher.Metadata.WinetricksVerbs) > 0 {
			launcher.ProgressWindow.ResetProgressWindow64("Setting up prefix...", 0, 1)

			// Setup command
			cmd := exec.Command("umu-run", "winetricks")
			cmd.Args = append(cmd.Args, launcher.Metadata.WinetricksVerbs...)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			cmd.Dir = filepath.Join(launcher.DataDir, "files")

			// Setup environment
			cmd.Env = os.Environ()
			cmd.Env = append(cmd.Env, "PROTONPATH="+launcher.SelectedRunner.Path)
			cmd.Env = append(cmd.Env, "WINEPREFIX="+prefixDir)

			err = cmd.Run()
			if err != nil {
				log.Fatal(err)
			}
		}
	}

	launcher.ProgressWindow.Hide()

	// Restart launcher
	exe, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}
	syscall.Exec(exe, nil, os.Environ())
}

func UninstallGame() {
	isLauncherInstalled := IsLauncherInstalled()

	// Get user home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}

	// Removing files
	RemoveDirectoryRecursively(launcher.DataDir)

	err = os.Remove(filepath.Join(homeDir, ".local/share/applications/games", launcher.Metadata.Name+".desktop"))
	if err != nil && !os.IsNotExist(err) {
		log.Fatal(err)
	}

	launcher.ProgressWindow.Hide()

	if isLauncherInstalled {
		// Exit launcher
		launcher.App.Quit()
	} else {
		// Restart launcher
		exe, err := os.Executable()
		if err != nil {
			log.Fatal(err)
		}
		syscall.Exec(exe, nil, os.Environ())
	}
}

func IsLauncherInstalled() bool {
	_, err := os.Stat(".installed")
	return err == nil
}

func IsGameInstalled() bool {
	_, err := os.Stat(filepath.Join(launcher.DataDir, ".installed"))
	return err == nil
}
