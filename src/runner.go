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
	"runtime"
	"slices"
	"strings"
)

const PROTON_GE_VERSION_REQUEST_URL = "https://api.github.com/repos/GloriousEggroll/proton-ge-custom/releases?per_page=75"
const MGBA_VERSION_REQUEST_URL = "https://api.github.com/repos/mgba-emu/mgba/releases?per_page=75"

type Runner struct {
	DisplayName string
	Type        string
	System      string
	Run         string
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

	// Show installed runners in reverse alphabetical order
	slices.Reverse(dirEntries)

	for _, entry := range dirEntries {
		entryPath := filepath.Join(homeDir, ".local/share/game_disc_player/runners/windows", entry.Name())

		if b, err := os.ReadFile(filepath.Join(entryPath, "compatibilitytool.vdf")); err == nil {
			lines := strings.Split(strings.TrimSpace(string(b)), "\n")
			for _, line := range lines {
				if i := strings.Index(line, "\"display_name\""); i != -1 {
					displayName := strings.Trim(line[i+14:], " \n\"")
					runners = append(runners, Runner{DisplayName: displayName, Type: "proton", System: "windows", Run: entryPath})
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
			if runtime.GOARCH == "386" || strings.HasPrefix(runtime.GOARCH, "amd64") {
				return strings.HasSuffix(asset.Name, ".tar.gz") && !strings.HasSuffix(asset.Name, "aarch64.tar.gz")
			} else if strings.HasPrefix(runtime.GOARCH, "arm") {
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
			runners = append(runners, Runner{DisplayName: release.TagName, Type: "proton", System: "windows", Run: release.Assets[assetId].BrowserDownloadUrl})
		}
	}

	return
}

func GetGameboyAdvanceRunners() (runners []Runner, err error) {
	runners = make([]Runner, 0)

	// Get user home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return
	}

	// Get local runners
	if mgbaPath, err := exec.LookPath("mgba-qt"); err == nil {
		output, err := exec.Command(mgbaPath, "--version").Output()
		if err == nil {
			runners = append(runners, Runner{
				DisplayName: "mGBA " + strings.Split(string(output), " ")[1] + " (System)",
				Type:        "mgba",
				System:      "gba",
				Run:         mgbaPath,
			})
		}
	} else if mgbaPath, err := exec.LookPath("mgba"); err == nil {
		output, err := exec.Command(mgbaPath, "--version").Output()
		if err == nil {
			runners = append(runners, Runner{
				DisplayName: "mGBA " + strings.Split(string(output), " ")[1] + " (System)",
				Type:        "mgba",
				System:      "gba",
				Run:         mgbaPath,
			})
		}
	}
	if err = exec.Command("flatpak", "info", "io.mgba.mGBA").Run(); err == nil {
		output, err := exec.Command("flatpak", "run", "io.mgba.mGBA", "--version").Output()
		if err == nil {
			runners = append(runners, Runner{
				DisplayName: "mGBA " + strings.Split(string(output), " ")[1] + " (Flatpak)",
				Type:        "mgba",
				System:      "gba",
				Run:         "flatpak run io.mgba.mGBA",
			})
		}
	}

	dirEntries, err := os.ReadDir(filepath.Join(homeDir, ".local/share/game_disc_player/runners/gba"))
	if err == nil {
		// Show installed runners in reverse alphabetical order
		slices.Reverse(dirEntries)

		for _, entry := range dirEntries {
			entryPath := filepath.Join(homeDir, ".local/share/game_disc_player/runners/gba", entry.Name())

			// Check for mgba runners
			if strings.HasPrefix(entry.Name(), "mGBA-") && strings.HasSuffix(entry.Name(), ".appimage") {
				output, err := exec.Command(entryPath, "--version").Output()
				if err != nil {
					continue
				}
				runners = append(runners, Runner{DisplayName: "mGBA " + strings.Split(string(output), " ")[1], Type: "mgba", System: "gba", Run: entryPath})
			}
		}
	} else {
		err = nil
	}

	// Get downloadable runners
	req, err := http.NewRequest("GET", MGBA_VERSION_REQUEST_URL, nil)
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
			if runtime.GOARCH == "386" || strings.HasPrefix(runtime.GOARCH, "amd64") {
				return strings.HasSuffix(asset.Name, "-x64.appimage")
			} else if strings.HasPrefix(runtime.GOARCH, "arm") {
				return strings.HasSuffix(asset.Name, "-arm64.appimage ")
			}

			return false
		})
		if assetId == -1 {
			continue
		}

		if !slices.ContainsFunc(runners, func(runner Runner) bool {
			return runner.DisplayName == "mGBA "+release.TagName
		}) {
			runners = append(runners, Runner{DisplayName: "mGBA " + release.TagName, Type: "mgba", System: "gba", Run: release.Assets[assetId].BrowserDownloadUrl})
		}
	}

	return
}

func (runner *Runner) Download() error {
	if !strings.HasPrefix(runner.Run, "https://") && !strings.HasPrefix(runner.Run, "http://") {
		return nil
	}

	// Get user home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	// Create runner directory
	err = os.MkdirAll(filepath.Join(homeDir, ".local/share/game_disc_player/runners", runner.System), 0755)
	if err != nil {
		return err
	}

	// Download runner
	f, err := os.Create(filepath.Join(homeDir, ".cache", path.Base(runner.Run)))
	if err != nil {
		return err
	}
	defer f.Close()

	response, err := http.Get(runner.Run)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	fmt.Println("Downloading " + runner.DisplayName + "...")
	launcher.ProgressWindow.ResetProgressWindow64("Downloading "+runner.DisplayName+"...", 0, response.ContentLength)

	_, err = io.Copy(io.MultiWriter(f, &launcher.ProgressWindow), response.Body)
	if err != nil {
		return err
	}

	// Ensure file is closed
	f.Close()

	if strings.HasSuffix(runner.Run, ".tar") ||
		strings.HasSuffix(runner.Run, ".tar.gz") ||
		strings.HasSuffix(runner.Run, ".tar.xz") ||
		strings.HasSuffix(runner.Run, ".tar.zst") {
		launcher.ProgressWindow.SetStatus("Extracting " + runner.DisplayName + "...")

		// Setup extract command
		cmd := exec.Command("tar", "xf", filepath.Join(homeDir, ".cache", path.Base(runner.Run)))
		cmd.Stdin = response.Body
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Dir = filepath.Join(homeDir, ".local/share/game_disc_player/runners", runner.System)

		err = cmd.Run()
		if err != nil {
			return err
		}

		runner.Run = filepath.Join(homeDir, ".local/share/game_disc_player/runners", runner.System, runner.DisplayName)

		// Remove downloaded tarball
		os.Remove(filepath.Join(homeDir, ".cache", path.Base(runner.Run)))
	} else if strings.HasSuffix(runner.Run, ".appimage") || strings.HasSuffix(runner.Run, ".AppImage") {
		launcher.ProgressWindow.SetStatus("Moving " + runner.DisplayName + "...")

		src := filepath.Join(homeDir, ".cache", path.Base(runner.Run))
		dest := filepath.Join(homeDir, ".local/share/game_disc_player/runners", runner.System, path.Base(runner.Run))

		// Change .AppImage to all lower-case to make appimage discovery simpler
		dest = strings.ReplaceAll(dest, ".AppImage", ".appimage")

		err = Copy(src, dest)
		if err != nil {
			return err
		}

		err = os.Chmod(dest, 0755)
		if err != nil {
			return err
		}

		runner.Run = dest

		os.Remove(filepath.Join(homeDir, ".cache", path.Base(runner.Run)))
	}

	launcher.ProgressWindow.Hide()

	return nil
}
