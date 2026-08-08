package main

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Options struct {
	Runner      string   `yaml:"runner,omitempty"`
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

	return
}

func (launcher *Launcher) SaveOptions() (err error) {

	if launcher.DataDir == "" {
		return fmt.Errorf("game not installed")
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
