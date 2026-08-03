package main

import (
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
	"windows": "Windows",
	"linux":   "Linux",
	"gba":     "Gameboy Advance",
}

func ReadMetadata() (Metadata, error) {
	data, err := os.ReadFile("metadata.yml")
	if err != nil {
		return Metadata{}, err
	}

	var metadata Metadata
	err = yaml.Unmarshal(data, &metadata)
	if err != nil {
		return Metadata{}, err
	}

	return metadata, nil
}
