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
	"gba":     "Gameboy Advance ROM",
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
