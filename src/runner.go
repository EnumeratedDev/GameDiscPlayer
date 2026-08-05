package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
)

const PROTON_GE_VERSION_REQUEST_URL = "https://api.github.com/repos/GloriousEggroll/proton-ge-custom/releases?per_page=75"

type Runner struct {
	DisplayName string
	Type        string
	Path        string
}

func GetWindowsRunners() (runners []Runner, err error) {
	runners = make([]Runner, 0)

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
					runners = append(runners, Runner{DisplayName: displayName, Type: "windows", Path: entryPath})
				}
			}
		}
	}

	// Get downloadable runners
	req, err := http.NewRequest("GET", PROTON_GE_VERSION_REQUEST_URL, nil)
	if err != nil {
		return
	}

	response, err := http.DefaultClient.Do(req)
	if err != nil {
		return runners, nil
	}
	defer response.Body.Close()

	decoder := json.NewDecoder(response.Body)

	type GithubAsset struct {
		Name               string `json:"name"`
		BrowserDownloadUrl string `json:"browser_download_url"`
	}
	type GithubRelease struct {
		TagName string        `json:"tag_name"`
		Assets  []GithubAsset `json:"assets"`
	}

	var githubReleases []GithubRelease

	err = decoder.Decode(&githubReleases)
	if err != nil {
		return
	}

	for _, release := range githubReleases {
		assetId := slices.IndexFunc(release.Assets, func(asset GithubAsset) bool {
			if runtime.GOARCH == "amd64" || runtime.GOARCH == "386" {
				return strings.HasSuffix(asset.Name, ".tar.gz") && !strings.HasSuffix(asset.Name, "aarch64.tar.gz")
			} else if runtime.GOARCH == "arm" {
				return strings.HasSuffix(asset.Name, "aarch64.tar.gz")
			}

			return false
		})
		if assetId == -1 {
			continue
		}

		if !slices.ContainsFunc(runners, func(runner Runner) bool {
			return runner.DisplayName == release.TagName
		}) {
			runners = append(runners, Runner{DisplayName: release.TagName, Type: "windows", Path: release.Assets[assetId].BrowserDownloadUrl})
		}
	}

	return
}

func (runner *Runner) Download() error {
	if !strings.HasPrefix(runner.Path, "https://") && !strings.HasPrefix(runner.Path, "http://") {
		return nil
	}

	fmt.Printf("Downloading runner (%s)...\n", runner.DisplayName)

	// Get user home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}

	// Create runner directory
	err = os.MkdirAll(filepath.Join(homeDir, ".local/share/game_disc_player/runners", runner.Type), 0755)
	if err != nil {
		log.Fatal(err)
	}

	// Download runner tarball
	f, err := os.Create(filepath.Join(homeDir, ".cache", path.Base(runner.Path)))
	if err != nil {
		return err
	}
	defer f.Close()

	response, err := http.Get(runner.Path)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	launcher.ProgressWindow.ResetProgressWindow64("Downloading "+runner.DisplayName+"...", 0, response.ContentLength)

	_, err = io.Copy(io.MultiWriter(f, &launcher.ProgressWindow), response.Body)
	if err != nil {
		return err
	}

	launcher.ProgressWindow.SetStatus("Extracting " + runner.DisplayName + "...")

	// Setup extract command
	cmd := exec.Command("tar", "xf", filepath.Join(homeDir, ".cache", path.Base(runner.Path)))
	cmd.Stdin = response.Body
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = filepath.Join(homeDir, ".local/share/game_disc_player/runners", runner.Type)

	err = cmd.Run()
	if err != nil {
		return err
	}

	runner.Path = filepath.Join(homeDir, ".local/share/game_disc_player/runners", runner.Type, runner.DisplayName)

	os.Remove(filepath.Join(homeDir, ".cache", path.Base(runner.Path)))

	return nil
}
