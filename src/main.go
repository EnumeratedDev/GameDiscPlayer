package main

import (
	"log"
	"os"
	"path/filepath"
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

	// Update local data
	err := updateLocalData()
	if err != nil {
		log.Fatal(err)
	}

	SetupLauncher()
}

func updateLocalData() error {
	// Get user home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	// Rename old 'game_disc_player' directory to 'GameDiscPlayer'
	err = os.Rename(filepath.Join(homeDir, ".local/share/game_disc_player"), filepath.Join(homeDir, ".local/share/GameDiscPlayer"))
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	return nil
}
