package main

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Metadata struct {
	Name          string         `yaml:"name"`
	Description   string         `yaml:"description"`
	Version       string         `yaml:"version"`
	Developer     string         `yaml:"developer"`
	Publisher     string         `yaml:"publisher"`
	RunFromDisc   bool           `yaml:"run_from_disc"`
	System        string         `yaml:"system"`
	LaunchOptions []LaunchOption `yaml:"launch_options"`
}

type LaunchOption struct {
	DisplayName     string   `yaml:"display_name"`
	Exec            string   `yaml:"exec"`
	Environment     []string `yaml:"environment"`
	WinetricksVerbs []string `yaml:"winetricks_verbs"`
}

var systemsUserReadable map[string]string = map[string]string{
	"windows": "Windows",
	"linux":   "Linux",
	"nes":     "NES/Famicom",
	"snes":    "SNES/Super Famicom",
	"n64":     "Nintendo 64",
	"gc":      "Nintendo Gamecube",
	"wii":     "Nintendo Wii",
	"wii-u":   "Nintendo Wii U",
	"switch":  "Nintendo Switch",
	"gb":      "Nintendo Gameboy",
	"gbc":     "Nintendo Gameboy Color",
	"gba":     "Nintendo Gameboy Advance",
	"nds":     "Nintendo DS",
	"ps1":     "Sony Playstation 1",
	"ps2":     "Sony Playstation 2",
	"ps3":     "Sony Playstation 3",
	"ps4":     "Sony Playstation 4",
	"xbox":    "Microsoft Xbox",
	"x360":    "Microsoft Xbox 360",
	"xone":    "Microsoft Xbox One",
}

func ReadMetadata() (metadata Metadata, err error) {
	data, err := os.ReadFile("metadata.yml")
	if err != nil {
		return
	}

	err = yaml.Unmarshal(data, &metadata)
	if err != nil {
		return
	}

	// Validate metadata
	if metadata.Name == "" {
		return metadata, fmt.Errorf("game name cannot be empty")
	}

	return metadata, nil
}
