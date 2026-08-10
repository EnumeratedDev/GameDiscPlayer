package main

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Metadata struct {
	Name            string   `yaml:"name"`
	Description     string   `yaml:"description"`
	Version         string   `yaml:"version"`
	Developer       string   `yaml:"developer"`
	Publisher       string   `yaml:"publisher"`
	RunFromDisc     bool     `yaml:"run_from_disc"`
	Type            string   `yaml:"type"`
	Run             string   `yaml:"run"`
	WinetricksVerbs []string `yaml:"winetricks_verbs"`
}

var typeStrings map[string]string = map[string]string{
	"windows": "Windows executable",
	"linux":   "Linux application",
	"nes":     "NES/Famicom ROM",
	"snes":    "SNES/Super Famicom ROM",
	"n64":     "Nintendo 64 ROM",
	"gc":      "Nintendo Gamecube ROM",
	"wii":     "Nintendo Wii disc",
	"wii-u":   "Nintendo Wii U disc",
	"switch":  "Nintendo Switch ROM",
	"gb":      "Nintendo Gameboy ROM",
	"gbc":     "Nintendo Gameboy Color ROM",
	"gba":     "Nintendo Gameboy Advance ROM",
	"nds":     "Nintendo DS ROM",
	"ps1":     "Sony Playstation 1 disc",
	"ps2":     "Sony Playstation 2 disc",
	"ps3":     "Sony Playstation 3 disc",
	"ps4":     "Sony Playstation 4 disc",
	"xbox":    "Microsoft Xbox disc",
	"x360":    "Microsoft Xbox 360 disc",
	"xone":    "Microsoft Xbox One disc",
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
