package main

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Options struct {
	Runner      string   `yaml:"runner,omitempty"`
	ExitOnPlay  bool     `yaml:"exit_on_play,omitempty"`
	Mangohud    bool     `yaml:"mangohud,omitempty"`
	Environment []string `yaml:"environment,omitempty"`
}

func (launcher *Launcher) ParseOptions() (err error) {
	var options Options

	if launcher.DataDir == "" {
		return fmt.Errorf("game not installed")
	}

	f, err := os.Open(filepath.Join(launcher.DataDir, "options.yml"))
	if os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return
	}

	decoder := yaml.NewDecoder(f)
	err = decoder.Decode(&options)
	if err != nil {
		return
	}

	launcher.Options = options

	// Set selected runner
	if launcher.SelectedRunner == "" && launcher.Options.Runner != "" {
		launcher.SelectedRunner = launcher.Options.Runner
	}

	return
}

func (launcher *Launcher) SaveOptions() (err error) {
	if launcher.DataDir == "" {
		return fmt.Errorf("data directory is not set")
	}

	err = os.MkdirAll(launcher.DataDir, 0755)
	if err != nil {
		return
	}

	f, err := os.OpenFile(filepath.Join(launcher.DataDir, "options.yml"), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return
	}

	encoder := yaml.NewEncoder(f)
	err = encoder.Encode(&launcher.Options)
	if err != nil {
		return
	}

	return
}
