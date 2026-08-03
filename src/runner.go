package main

import (
	"os"
	"path/filepath"
	"strings"
)

func GetWindowsRunners() (runners map[string]string, err error) {
	runners = make(map[string]string)

	// Get local runners
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return
	}
	dirEntries, err := os.ReadDir(filepath.Join(homeDir, ".local/share/game_disc_player/runners/windows"))
	if err != nil {
		return
	}
	for _, entry := range dirEntries {
		entryPath := filepath.Join(homeDir, ".local/share/game_disc_player/runners/windows", entry.Name())

		if b, err := os.ReadFile(filepath.Join(entryPath, "compatibilitytool.vdf")); err == nil {
			lines := strings.Split(strings.TrimSpace(string(b)), "\n")
			for _, line := range lines {
				if i := strings.Index(line, "\"display_name\""); i != -1 {
					displayName := strings.Trim(line[i+14:], " \n\"")
					runners[displayName] = entryPath
				}
			}
		}
	}

	return
}
