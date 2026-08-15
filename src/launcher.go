package main

import (
	"fmt"
	"log"
	"os"
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
		ShowErrorMessage(err)
		EventEmit("game_state_changed", "idle")
		return
	}

	if !launcher.IsGameInstalled() && !launcher.Metadata.RunFromDisc {
		ShowErrorMessage(fmt.Errorf("game cannot be run from disc"))
		EventEmit("game_state_changed", "idle")
		return
	}

	// Copy BIOS files
	biosSystems, err := os.ReadDir("bios")
	if err == nil {
		for _, entry := range biosSystems {
			if !entry.IsDir() {
				continue
			}

			CopyRecursivelyWithProgress(filepath.Join("bios", entry.Name()),
				filepath.Join(homeDir, ".local/share/GameDiscPlayer/runners", entry.Name(), "shared/bios"),
				true)
		}
	}

	if runner := launcher.GetSelectedRunner(); runner != nil {
		// Download runner if required
		if runner.NeedsDownload() {
			if !confirmDownload(runner) {
				EventEmit("game_state_changed", "idle")
				return
			}

			err = runner.Download()
			if err != nil {
				return
			}
		}

		// Set game options
		launcher.Options.Runner = launcher.SelectedRunner
		launcher.SaveOptions()

		fmt.Printf("Launching %s game using %s...\n", systemsUserReadable[launcher.Metadata.System], runner.DisplayName)
		err := runner.Run()
		if err != nil {
			ShowErrorMessage(err)
			EventEmit("game_state_changed", "idle")
			return
		}
	} else {
		ShowErrorMessage(fmt.Errorf("invalid runner_id"))
		EventEmit("game_state_changed", "idle")
		return
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
		// Download runner if required
		if runner.NeedsDownload() {
			ShowErrorMessage(fmt.Errorf("runner is not installed"))
			EventEmit("game_state_changed", "idle")
			return
		}

		err := runner.OpenSettings()
		if err != nil {
			ShowErrorMessage(err)
			EventEmit("game_state_changed", "idle")
			return
		}
	} else {
		ShowErrorMessage(fmt.Errorf("invalid runner_id"))
		EventEmit("game_state_changed", "idle")
		return
	}
}

func (launcher *Launcher) Install() {
	if _, err := os.Stat(".installed"); err == nil {
		ShowErrorMessage(fmt.Errorf("cannot reinstall game from installation directory"))
		EventEmit("game_state_changed", "idle")
		return
	}

	// Get working directory
	workDir, err := os.Getwd()
	if err != nil {
		ShowErrorMessage(err)
		EventEmit("game_state_changed", "idle")
		return
	}

	// Get user home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		ShowErrorMessage(err)
		EventEmit("game_state_changed", "idle")
		return
	}

	// Create game data directory
	err = os.MkdirAll(launcher.DataDir, 0755)
	if err != nil {
		ShowErrorMessage(err)
		EventEmit("game_state_changed", "idle")
		return
	}

	err = CopyRecursivelyWithProgress(filepath.Join(workDir, "files"), filepath.Join(launcher.DataDir, "files"), false)
	if err != nil {
		ShowErrorMessage(err)
		EventEmit("game_state_changed", "idle")
		return
	}

	// Copy icon
	err = Copy(filepath.Join(workDir, "icon.png"), filepath.Join(launcher.DataDir, "icon.png"))
	if err != nil {
		ShowErrorMessage(err)
		EventEmit("game_state_changed", "idle")
		return
	}

	// Copy metadata
	err = Copy(filepath.Join(workDir, "metadata.yml"), filepath.Join(launcher.DataDir, "metadata.yml"))
	if err != nil {
		ShowErrorMessage(err)
		EventEmit("game_state_changed", "idle")
		return
	}

	// Copy launcher into game files
	exe, err := os.Executable()
	if err != nil {
		ShowErrorMessage(err)
		EventEmit("game_state_changed", "idle")
		return
	}
	err = Copy(exe, filepath.Join(launcher.DataDir, "launcher"))
	if err != nil {
		ShowErrorMessage(err)
		EventEmit("game_state_changed", "idle")
		return
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
		ShowErrorMessage(err)
		EventEmit("game_state_changed", "idle")
		return
	}
	err = os.WriteFile(filepath.Join(homeDir, ".local/share/applications/games", launcher.Metadata.Name+".desktop"), []byte(toWrite), 0755)
	if err != nil {
		ShowErrorMessage(err)
		EventEmit("game_state_changed", "idle")
		return
	}

	// Create .installed file
	f, err := os.Create(filepath.Join(launcher.DataDir, ".installed"))
	if err != nil {
		ShowErrorMessage(err)
		EventEmit("game_state_changed", "idle")
		return
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
				filepath.Join(homeDir, ".local/share/GameDiscPlayer/runners", entry.Name(), "shared/bios"),
				true)
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
		ShowErrorMessage(err)
		EventEmit("game_state_changed", "idle")
		return
	}

	// Get user home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		ShowErrorMessage(err)
		EventEmit("game_state_changed", "idle")
		return
	}

	// Removing files
	RemoveDirectoryRecursively(launcher.DataDir)

	err = os.Remove(filepath.Join(homeDir, ".local/share/applications/games", launcher.Metadata.Name+".desktop"))
	if err != nil && !os.IsNotExist(err) {
		ShowErrorMessage(err)
		EventEmit("game_state_changed", "idle")
		return
	}

	if shouldQuit {
		// Exit launcher
		launcher.App.Quit()
	} else {
		// Restart launcher
		exe, err := os.Executable()
		if err != nil {
			ShowErrorMessage(err)
			EventEmit("game_state_changed", "idle")
			return
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

func ShowErrorMessage(err error) {
	if launcher.NoGUI {
		log.Fatal(err)
		return
	}

	dialog := gtk.NewMessageDialog(&launcher.MainWindow.Window, gtk.DialogFlags(gtk.DialogModal)|gtk.DialogDestroyWithParent, gtk.MessageType(gtk.MessageInfo), gtk.ButtonsType(gtk.ButtonsOK))
	dialog.SetTitle("Launcher error")
	dialog.SetMarkup("Error: " + err.Error())
	dialog.SetVisible(true)

	ch := make(chan bool, 1)

	dialog.ConnectResponse(func(responseId int) {
		ch <- true

		dialog.Destroy()
	})

	// Wait for response
	<-ch
}
