package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
)

const UMU_LAUNCHER_VERSION_REQUEST_URL = "https://api.github.com/repos/Open-Wine-Components/umu-launcher/releases/latest"
const UMU_LAUNCHER_DOWNLOAD_URL = "https://github.com/Open-Wine-Components/umu-launcher/releases/download/$VERSION/umu-launcher-$VERSION-zipapp.tar"

func DownloadUmu() (err error) {
	// Get user home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	req, err := http.NewRequest("GET", UMU_LAUNCHER_VERSION_REQUEST_URL, nil)
	if err != nil {
		return
	}

	response, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}
	defer response.Body.Close()

	type GithubRelease struct {
		TagName string `json:"tag_name"`
	}

	decoder := json.NewDecoder(response.Body)

	var githubRelease GithubRelease

	err = decoder.Decode(&githubRelease)
	if err != nil {
		return
	}

	umuLauncherDownloadURL := strings.ReplaceAll(UMU_LAUNCHER_DOWNLOAD_URL, "$VERSION", githubRelease.TagName)

	// Create user bin directory
	err = os.MkdirAll(filepath.Join(homeDir, ".local/bin"), 0755)
	if err != nil {
		return
	}

	// Download umu tarball
	f, err := os.Create(filepath.Join(homeDir, ".cache", path.Base(umuLauncherDownloadURL)))
	if err != nil {
		return
	}
	defer f.Close()

	response, err = http.Get(umuLauncherDownloadURL)
	if err != nil {
		return
	}
	defer response.Body.Close()

	fmt.Println("Downloading umu-launcher " + githubRelease.TagName + "...")
	progressWindow := NewProgressWindow64("Downloading umu-launcher "+githubRelease.TagName+"...", 0, response.ContentLength)
	defer progressWindow.Close()

	_, err = io.Copy(io.MultiWriter(f, &progressWindow), response.Body)
	if err != nil {
		return
	}

	progressWindow.SetStatus("Extracting umu-launcher " + githubRelease.TagName + "...")

	// Setup extract command
	cmd := exec.Command("tar", "xf", filepath.Join(homeDir, ".cache", path.Base(umuLauncherDownloadURL)), "--strip-components=1", "umu/umu-run")
	cmd.Stdin = response.Body
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = filepath.Join(homeDir, ".local/bin")

	err = cmd.Run()
	if err != nil {
		return
	}

	// Remove tarball
	os.Remove(filepath.Join(homeDir, ".cache", path.Base(umuLauncherDownloadURL)))

	return
}
