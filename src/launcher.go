package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

type Launcher struct {
	Metadata Metadata
	DataDir  string
	Options  Options

	App        *gtk.Application
	MainWindow *gtk.ApplicationWindow

	SelectedRunner *Runner

	GameProcess *os.Process
}

var launcher = Launcher{Options: Options{}}

func (launcher *Launcher) Play() {
	// Hide main launcher window
	glib.IdleAdd(func() {
		launcher.MainWindow.SetSensitive(false)
		launcher.MainWindow.SetVisible(false)
	})

	if launcher.Metadata.System == "linux" {
		// Run Linux binary/script

		// Get working directory
		workDir, err := os.Getwd()
		if err != nil {
			log.Fatal(err)
		}

		// Run game
		var cmd *exec.Cmd
		if launcher.IsGameInstalled() {
			cmd = exec.Command(filepath.Join(launcher.DataDir, "files", launcher.Metadata.Run))
			cmd.Dir = filepath.Join(launcher.DataDir, "files")
		} else {
			cmd = exec.Command(filepath.Join(workDir, "files", launcher.Metadata.Run))
			cmd.Dir = filepath.Join(workDir, "files")
		}
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		// Setup environment
		cmd.Env = cmd.Environ()
		cmd.Env = append(cmd.Env, launcher.Options.Environment...)

		err = cmd.Start()
		if err != nil {
			log.Fatal(err)
		}
		launcher.GameProcess = cmd.Process
		err = cmd.Wait()
		if err != nil {
			log.Fatal(err)
		}
		launcher.GameProcess = nil
	} else {
		err := launcher.SelectedRunner.Run()
		if err != nil {
			log.Fatal(err)
		}
	}

	// Show main launcher window
	glib.IdleAdd(func() {
		launcher.MainWindow.SetSensitive(true)
		launcher.MainWindow.SetVisible(true)
	})
}

func (launcher *Launcher) Install() {
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
	launcher.Options.Runner = launcher.SelectedRunner.RunnerID
	launcher.SaveOptions()

	// Copy launcher into game files
	exe, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}
	err = Copy(exe, filepath.Join(launcher.DataDir, "launcher"))
	if err != nil {
		log.Fatal(err)
	}

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

	// Create .installed file
	f, err := os.Create(filepath.Join(launcher.DataDir, ".installed"))
	if err != nil {
		log.Fatal(err)
	}
	f.Close()

	switch launcher.Metadata.System {
	case "windows":
		// Check for umu
		umuPath, err := exec.LookPath("umu-run")
		if err != nil {
			umuPath = filepath.Join(homeDir, ".local/bin/umu-run")
			_, err = os.Stat(umuPath)
			if err != nil {
				err = DownloadUmu()
				if err != nil {
					log.Fatal(err)
				}
			}
		}

		// Download runner if required
		launcher.SelectedRunner.Download()

		prefixDir := filepath.Join(launcher.DataDir, "prefix")

		// Setup prefix
		if _, err := os.Stat(prefixDir); err != nil && len(launcher.Metadata.WinetricksVerbs) > 0 {
			progressWindow := NewProgressWindow("Setting up prefix...", 0, 1)
			progressWindow.Pulse()

			// Setup command
			cmd := exec.Command(umuPath, "winetricks")
			cmd.Args = append(cmd.Args, launcher.Metadata.WinetricksVerbs...)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			cmd.Dir = filepath.Join(launcher.DataDir, "files")

			// Setup environment
			cmd.Env = os.Environ()
			cmd.Env = append(cmd.Env, "PROTONPATH="+launcher.SelectedRunner.Exec)
			cmd.Env = append(cmd.Env, "WINEPREFIX="+prefixDir)

			err = cmd.Run()
			if err != nil {
				log.Fatal(err)
			}

			progressWindow.Close()
		}
	}

	// Restart launcher
	syscall.Exec(exe, nil, os.Environ())
}

func (launcher *Launcher) Uninstall() {
	shouldQuit := launcher.IsRunningFromInstallationDirectory()

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

	if shouldQuit {
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

func (launcher *Launcher) IsRunningFromInstallationDirectory() bool {
	_, err := os.Stat(".installed")
	return err == nil
}

func (launcher *Launcher) IsGameInstalled() bool {
	_, err := os.Stat(filepath.Join(launcher.DataDir, ".installed"))
	return err == nil
}
