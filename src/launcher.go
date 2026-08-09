package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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
	// Hide and show main launcher window
	glib.IdleAdd(func() {
		launcher.MainWindow.SetSensitive(false)
		launcher.MainWindow.SetVisible(false)
	})
	defer glib.IdleAdd(func() {
		launcher.MainWindow.SetSensitive(true)
		launcher.MainWindow.SetVisible(true)
	})

	// Get user home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}

	switch launcher.Metadata.Type {
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

		prefixDir := filepath.Join(launcher.DataDir, "prefix")

		// Download runner if required
		launcher.SelectedRunner.Download()

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
		cmd.Env = append(cmd.Env, "PROTONPATH="+launcher.SelectedRunner.Run)
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
	case "linux":
		// Run Linux binary/script

		// Run game
		cmd := exec.Command(filepath.Join(launcher.DataDir, "files", launcher.Metadata.Run))
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
	case "gb", "gbc", "gba":
		// Run Gameboy Advance ROM

		// Download runner if required
		launcher.SelectedRunner.Download()

		// Set game options
		launcher.Options.Runner = launcher.SelectedRunner.DisplayName
		launcher.SaveOptions()

		// Run game
		cmd := exec.Command(launcher.SelectedRunner.Run)
		switch launcher.SelectedRunner.Type {
		case "mgba":
			// Create saves directory
			err = os.MkdirAll(filepath.Join(launcher.DataDir, "saves"), 0755)
			if err != nil {
				log.Fatal(err)
			}
			// Create cheats directory
			err = os.MkdirAll(filepath.Join(launcher.DataDir, "cheats"), 0755)
			if err != nil {
				log.Fatal(err)
			}
			// Create screenshots directory
			err = os.MkdirAll(filepath.Join(launcher.DataDir, "screenshots"), 0755)
			if err != nil {
				log.Fatal(err)
			}
			// Create patches directory
			err = os.MkdirAll(filepath.Join(launcher.DataDir, "patches"), 0755)
			if err != nil {
				log.Fatal(err)
			}

			cmd.Args = append(cmd.Args, "--fullscreen",
				"-CsavegamePath"+filepath.Join(launcher.DataDir, "saves"),
				"-CsavestatePath"+filepath.Join(launcher.DataDir, "saves"),
				"-CcheatsPath"+filepath.Join(launcher.DataDir, "cheats"),
				"-CscreenshotPath"+filepath.Join(launcher.DataDir, "screenshots"),
				"-CpatchPath"+filepath.Join(launcher.DataDir, "patches"),
				launcher.Metadata.Run)
		}
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

	err = launcher.SaveOptions()
	if err != nil {
		log.Fatal(err)
	}
}

func (launcher *Launcher) PlayFromDisc() {
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

	switch launcher.Metadata.Type {
	case "windows":
		// Run Windows executables with umu

		// Download runner if required
		launcher.SelectedRunner.Download()

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

		// Set game options
		launcher.Options.Runner = launcher.SelectedRunner.DisplayName
		launcher.SaveOptions()

		prefixDir := filepath.Join(launcher.DataDir, "prefix")

		// Run winetricks
		if _, err := os.Stat(prefixDir); err != nil && len(launcher.Metadata.WinetricksVerbs) > 0 {
			progressWindow := NewProgressWindow("Setting up prefix...", 0, 1)

			// Setup command
			cmd := exec.Command("umu-run", "winetricks")
			cmd.Args = append(cmd.Args, launcher.Metadata.WinetricksVerbs...)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			cmd.Dir = filepath.Join(workDir, "files")

			// Setup environment
			cmd.Env = os.Environ()
			cmd.Env = append(cmd.Env, "PROTONPATH="+launcher.SelectedRunner.Run)
			cmd.Env = append(cmd.Env, "WINEPREFIX="+prefixDir)

			err = cmd.Run()
			if err != nil {
				log.Fatal(err)
			}

			progressWindow.Close()
		}

		// Run game
		cmd := exec.Command(umuPath, launcher.Metadata.Run)
		cmd.Dir = filepath.Join(workDir, "files")
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		// Setup environment
		cmd.Env = cmd.Environ()
		cmd.Env = append(cmd.Env, launcher.Options.Environment...)
		cmd.Env = append(cmd.Env, "PROTONPATH="+launcher.SelectedRunner.Run)
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
	case "linux":
		// Run Linux binary/script

		// Run game
		cmd := exec.Command(filepath.Join(workDir, "files", launcher.Metadata.Run))
		cmd.Dir = filepath.Join(workDir, "files")
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
	case "gb", "gbc", "gba":
		// Run Gameboy Advance ROM

		// Download runner if required
		launcher.SelectedRunner.Download()

		// Set game options
		launcher.Options.Runner = launcher.SelectedRunner.DisplayName
		launcher.SaveOptions()

		// Run game
		runCmd := strings.Split(launcher.SelectedRunner.Run, " ")
		cmd := exec.Command(runCmd[0], runCmd[1:]...)
		if strings.HasPrefix(launcher.SelectedRunner.DisplayName, "mGBA") {
			// Create saves directory
			err = os.MkdirAll(filepath.Join(launcher.DataDir, "saves"), 0755)
			if err != nil {
				log.Fatal(err)
			}
			// Create cheats directory
			err = os.MkdirAll(filepath.Join(launcher.DataDir, "cheats"), 0755)
			if err != nil {
				log.Fatal(err)
			}
			// Create screenshots directory
			err = os.MkdirAll(filepath.Join(launcher.DataDir, "screenshots"), 0755)
			if err != nil {
				log.Fatal(err)
			}
			// Create patches directory
			err = os.MkdirAll(filepath.Join(launcher.DataDir, "patches"), 0755)
			if err != nil {
				log.Fatal(err)
			}

			cmd.Args = append(cmd.Args, "--fullscreen",
				"-CsavegamePath="+filepath.Join(launcher.DataDir, "saves"),
				"-CsavestatePath="+filepath.Join(launcher.DataDir, "saves"),
				"-CcheatsPath="+filepath.Join(launcher.DataDir, "cheats"),
				"-CscreenshotPath="+filepath.Join(launcher.DataDir, "screenshots"),
				"-CpatchPath="+filepath.Join(launcher.DataDir, "patches"),
				launcher.Metadata.Run)
		}
		cmd.Dir = filepath.Join(workDir, "files")
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
}

func (launcher *Launcher) InstallGame() {
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

	switch launcher.Metadata.Type {
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

			// Setup command
			cmd := exec.Command(umuPath, "winetricks")
			cmd.Args = append(cmd.Args, launcher.Metadata.WinetricksVerbs...)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			cmd.Dir = filepath.Join(launcher.DataDir, "files")

			// Setup environment
			cmd.Env = os.Environ()
			cmd.Env = append(cmd.Env, "PROTONPATH="+launcher.SelectedRunner.Run)
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

func (launcher *Launcher) UninstallGame() {
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
