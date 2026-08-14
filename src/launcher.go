package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

type Launcher struct {
	Metadata Metadata
	DataDir  string
	Options  Options

	App        *gtk.Application
	MainWindow *gtk.ApplicationWindow

	Runners        []Runner
	SelectedRunner string

	NoGUI   bool
	Offline bool

	GameProcess *os.Process
}

var launcher = Launcher{Options: Options{}}

func (launcher *Launcher) Play() {
	EventEmit("game_state_changed", "launching")

	// Get user home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}

	if !launcher.IsGameInstalled() && !launcher.Metadata.RunFromDisc {
		log.Fatalf("game cannot be run from disc")
	}

	// Copy BIOS files
	biosSystems, err := os.ReadDir("bios")
	if err == nil {
		for _, entry := range biosSystems {
			if !entry.IsDir() {
				continue
			}

			CopyRecursivelyWithProgress(filepath.Join("bios", entry.Name()),
				filepath.Join(homeDir, ".local/share/GameDiscPlayer/runners", entry.Name(), "bios"),
				true)
		}
	}

	if runner := launcher.GetSelectedRunner(); runner != nil {
		// Set game options
		launcher.Options.Runner = launcher.SelectedRunner
		launcher.SaveOptions()

		fmt.Printf("Launching %s game using %s...\n", systemsUserReadable[launcher.Metadata.System], runner.DisplayName)
		err := runner.Run()
		if err != nil {
			log.Fatal(err)
		}
	} else {
		log.Fatalf("invalid runner_id")
	}

	if launcher.NoGUI || launcher.Options.ExitOnPlay {
		os.Exit(0)
	}
}

func (launcher *Launcher) Stop() {
	if launcher.GameProcess == nil {
		return
	}

	EventEmit("game_state_changed", "stopping")

	done := make(chan bool, 1)
	go func() {
		launcher.GameProcess.Wait()
		done <- true
	}()

	launcher.GameProcess.Signal(syscall.SIGTERM)
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		launcher.GameProcess.Kill()
	}

	EventEmit("game_state_changed", "idle")
}

func (launcher *Launcher) FetchAvailableRunners() (err error) {
	runnerFetcher, ok := RunnerFetchers[launcher.Metadata.System]
	if ok {
		launcher.Runners, err = runnerFetcher()
		if err != nil {
			return
		}

		if len(launcher.Runners) > 0 && launcher.SelectedRunner == "" {
			launcher.SelectedRunner = launcher.Runners[0].RunnerID
		}
	}

	return
}

func (launcher *Launcher) GetSelectedRunner() *Runner {
	for _, runner := range launcher.Runners {
		if runner.RunnerID == launcher.SelectedRunner {
			return &runner
		}
	}

	return nil
}

func (launcher *Launcher) OpenRunnerSettings() {
	if runner := launcher.GetSelectedRunner(); runner != nil {
		err := runner.OpenSettings()
		if err != nil {
			log.Fatal(err)
		}
	} else {
		log.Fatalf("invalid runner_id")
	}
}

func (launcher *Launcher) Install() {
	if _, err := os.Stat(".installed"); err == nil {
		log.Fatalf("cannot reinstall game from installation directory")
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
	err = os.MkdirAll(launcher.DataDir, 0755)
	if err != nil {
		log.Fatal(err)
	}

	err = CopyRecursivelyWithProgress(filepath.Join(workDir, "files"), filepath.Join(launcher.DataDir, "files"), false)
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

	// Set game options
	if launcher.GetSelectedRunner() != nil {
		launcher.Options.Runner = launcher.SelectedRunner
	}
	launcher.SaveOptions()

	// Copy BIOS files
	biosSystems, err := os.ReadDir("bios")
	if err == nil {
		for _, entry := range biosSystems {
			if !entry.IsDir() {
				continue
			}

			CopyRecursivelyWithProgress(filepath.Join("bios", entry.Name()),
				filepath.Join(homeDir, ".local/share/GameDiscPlayer/runners", entry.Name(), "bios"),
				true)
		}
	}

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
		if launcher.GetSelectedRunner().NeedsDownload() {
			if !confirmDownload(launcher.GetSelectedRunner()) {
				EventEmit("game_state_changed", "idle")
				return
			}

			err = launcher.GetSelectedRunner().Download()
			if err != nil {
				log.Fatal(err)
			}
		}

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
			cmd.Env = append(cmd.Env, "PROTONPATH="+launcher.GetSelectedRunner().Exec)
			cmd.Env = append(cmd.Env, "WINEPREFIX="+prefixDir)

			err = cmd.Run()
			if err != nil {
				log.Fatal(err)
			}

			progressWindow.Close()
		}
	}

	// Restart launcher
	if !launcher.NoGUI {
		syscall.Exec(exe, os.Args[1:], os.Environ())
	}
}

func (launcher *Launcher) Uninstall() {
	shouldQuit := launcher.IsRunningFromInstallationDirectory() || launcher.NoGUI

	if _, err := os.Stat(launcher.DataDir); os.IsNotExist(err) {
		return
	} else if err != nil {
		log.Fatal(err)
	}

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
		syscall.Exec(exe, os.Args[1:], os.Environ())
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
