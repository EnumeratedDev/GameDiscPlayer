package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
)

const PROTON_GE_VERSION_REQUEST_URL = "https://api.github.com/repos/GloriousEggroll/proton-ge-custom/releases?per_page=75"
const MGBA_VERSION_REQUEST_URL = "https://api.github.com/repos/mgba-emu/mgba/releases?per_page=75"
const DUCKSTATION_VERSION_REQUEST_URL = "https://api.github.com/repos/stenzek/duckstation/releases?per_page=75"
const PCSX2_VERSION_REQUEST_URL = "https://api.github.com/repos/PCSX2/pcsx2/releases?per_page=75"
const RPCS3_VERSION_REQUEST_URL = "https://api.github.com/repos/RPCS3/rpcs3/releases?per_page=75"

type Runner struct {
	DisplayName string
	RunnerID    string
	Type        string
	System      string
	Exec        string

	downloadFunc     func(runner *Runner) error
	runFunc          func(runner *Runner) error
	openSettingsFunc func(runner *Runner) error
}

var RunnerFetchers = map[string]func() ([]Runner, error){
	"linux":   GetLinuxRunners,
	"windows": GetWindowsRunners,
	"gb":      GetGameboyAdvanceRunners,
	"gbc":     GetGameboyAdvanceRunners,
	"gba":     GetGameboyAdvanceRunners,
	"ps1":     GetPlaystation1Runners,
	"ps2":     GetPlaystation2Runners,
}

func GetLinuxRunners() (runners []Runner, err error) {
	runners = make([]Runner, 0)

	// Get native runner
	runners = append(runners, Runner{
		DisplayName: "Native",
		RunnerID:    "native",
		Type:        "native",
		System:      "linux",
		runFunc:     runLinuxRunner})

	return
}

func GetWindowsRunners() (runners []Runner, err error) {
	runners = make([]Runner, 0)

	// Get local runners
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return
	}
	dirEntries, err := os.ReadDir(filepath.Join(homeDir, ".local/share/GameDiscPlayer/runners/windows"))
	if err != nil {
		return
	}

	// Show installed runners in reverse alphabetical order
	slices.Reverse(dirEntries)

	for _, entry := range dirEntries {
		entryPath := filepath.Join(homeDir, ".local/share/GameDiscPlayer/runners/windows", entry.Name())

		if b, err := os.ReadFile(filepath.Join(entryPath, "compatibilitytool.vdf")); err == nil {
			lines := strings.Split(strings.TrimSpace(string(b)), "\n")
			for _, line := range lines {
				if i := strings.Index(line, "\"display_name\""); i != -1 {
					displayName := strings.Trim(line[i+14:], " \n\"")
					runners = append(runners, Runner{
						DisplayName: displayName,
						RunnerID:    strings.ToLower(displayName),
						Type:        "proton",
						System:      "windows",
						Exec:        entryPath,
						runFunc:     runWindowsRunner})
				}
			}
		}
	}

	// Get downloadable runners
	if launcher.Offline {
		return
	}
	githubReleases, err := GetGithubReleases(PROTON_GE_VERSION_REQUEST_URL)
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
			runners = append(runners, Runner{
				DisplayName:  release.TagName,
				RunnerID:     strings.ToLower(release.TagName),
				Type:         "proton",
				System:       "windows",
				Exec:         release.Assets[assetId].BrowserDownloadUrl,
				downloadFunc: downloadWindowsRunner,
				runFunc:      runWindowsRunner})
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
			versionSplit := strings.Split(string(output), " ")
			version := versionSplit[len(versionSplit)-2]

			runners = append(runners, Runner{
				DisplayName:      "mGBA " + version + " (System)",
				RunnerID:         "mgba_system",
				Type:             "mgba",
				System:           "gba",
				Exec:             mgbaPath,
				runFunc:          runGameboyAdvanceRunner,
				openSettingsFunc: openSettingsGameboyAdvanceRunner,
			})
		}
	} else if mgbaPath, err := exec.LookPath("mgba"); err == nil {
		output, err := exec.Command(mgbaPath, "--version").Output()
		if err == nil {
			versionSplit := strings.Split(string(output), " ")
			version := versionSplit[len(versionSplit)-2]

			runners = append(runners, Runner{
				DisplayName:      "mGBA " + version + " (System)",
				RunnerID:         "mgba_system",
				Type:             "mgba",
				System:           "gba",
				Exec:             mgbaPath,
				runFunc:          runGameboyAdvanceRunner,
				openSettingsFunc: openSettingsGameboyAdvanceRunner,
			})
		}
	}
	if err = exec.Command("flatpak", "info", "io.mgba.mGBA").Run(); err == nil {
		output, err := exec.Command("flatpak", "run", "io.mgba.mGBA", "--version").Output()
		if err == nil {
			versionSplit := strings.Split(string(output), " ")
			version := versionSplit[len(versionSplit)-2]

			runners = append(runners, Runner{
				DisplayName:      "mGBA " + version + " (Flatpak)",
				RunnerID:         "mgba_flatpak",
				Type:             "mgba",
				System:           "gba",
				Exec:             "io.mgba.mGBA",
				runFunc:          runGameboyAdvanceRunner,
				openSettingsFunc: openSettingsGameboyAdvanceRunner,
			})
		}
	}

	dirEntries, err := os.ReadDir(filepath.Join(homeDir, ".local/share/GameDiscPlayer/runners/gba"))
	if err == nil {
		// Show installed runners in reverse alphabetical order
		slices.Reverse(dirEntries)

		for _, entry := range dirEntries {
			entryPath := filepath.Join(homeDir, ".local/share/GameDiscPlayer/runners/gba", entry.Name())

			// Check for mgba runners
			if _, err := os.Stat(filepath.Join(entryPath, "mgba")); err == nil {
				output, err := exec.Command(filepath.Join(entryPath, "mgba"), "--version").Output()
				if err != nil {
					continue
				}

				versionSplit := strings.Split(string(output), " ")
				version := versionSplit[len(versionSplit)-2]

				runners = append(runners, Runner{
					DisplayName:      "mGBA " + version,
					RunnerID:         "mgba_" + version,
					Type:             "mgba",
					System:           "gba",
					Exec:             filepath.Join(entryPath, "mgba"),
					runFunc:          runGameboyAdvanceRunner,
					openSettingsFunc: openSettingsGameboyAdvanceRunner,
				})
			}
		}
	} else {
		err = nil
	}

	// Get downloadable runners
	if launcher.Offline {
		return
	}
	githubReleases, err := GetGithubReleases(MGBA_VERSION_REQUEST_URL)
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
			return runner.RunnerID == "mgba_"+release.TagName
		}) {
			runners = append(runners, Runner{
				DisplayName:      "mGBA " + release.TagName,
				RunnerID:         "mgba_" + release.TagName,
				Type:             "mgba",
				System:           "gba",
				Exec:             release.Assets[assetId].BrowserDownloadUrl,
				downloadFunc:     downloadGameboyAdvanceRunner,
				runFunc:          runGameboyAdvanceRunner,
				openSettingsFunc: openSettingsGameboyAdvanceRunner,
			})
		}
	}

	return
}

func GetPlaystation1Runners() (runners []Runner, err error) {
	runners = make([]Runner, 0)

	// Get user home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return
	}

	// Get local runners
	if duckstationPath, err := exec.LookPath("duckstation"); err == nil {
		output, _ := exec.Command(duckstationPath, "-nogui", "-version").CombinedOutput()

		version := ""
		for _, line := range strings.Split(string(output), "\n") {
			if strings.HasPrefix(line, "DuckStation Version") {
				version = strings.Split(line, " ")[2]
				version = strings.Split(version, "-")[0] + "-" + strings.Split(version, "-")[1]
			}
		}
		if version != "" {
			runners = append(runners, Runner{
				DisplayName:      "DuckStation " + version + " (System)",
				RunnerID:         "duckstation_system",
				Type:             "duckstation",
				System:           "ps1",
				Exec:             duckstationPath,
				runFunc:          runPlaystation1Runner,
				openSettingsFunc: openSettingsPlaystation1Runner,
			})
		}
	}

	dirEntries, err := os.ReadDir(filepath.Join(homeDir, ".local/share/GameDiscPlayer/runners/ps1"))
	if err == nil {
		// Show installed runners in reverse alphabetical order
		slices.Reverse(dirEntries)

		for _, entry := range dirEntries {
			entryPath := filepath.Join(homeDir, ".local/share/GameDiscPlayer/runners/ps1", entry.Name())

			// Check for duckstation runners
			if _, err := os.Stat(filepath.Join(entryPath, "duckstation")); err == nil {
				output, _ := exec.Command(filepath.Join(entryPath, "duckstation"), "-nogui", "-version").CombinedOutput()

				version := ""
				for _, line := range strings.Split(string(output), "\n") {
					if strings.HasPrefix(line, "DuckStation Version") {
						version = strings.Split(line, " ")[2]
						version = strings.Split(version, "-")[0] + "-" + strings.Split(version, "-")[1]
					}
				}
				if version == "" {
					continue
				}

				runners = append(runners, Runner{
					DisplayName:      "DuckStation " + version,
					RunnerID:         "duckstation_" + version,
					Type:             "duckstation",
					System:           "ps1",
					Exec:             filepath.Join(entryPath, "duckstation"),
					runFunc:          runPlaystation1Runner,
					openSettingsFunc: openSettingsPlaystation1Runner,
				})
			}
		}
	} else {
		err = nil
	}

	// Get downloadable runners
	if launcher.Offline {
		return
	}
	githubReleases, err := GetGithubReleases(DUCKSTATION_VERSION_REQUEST_URL)
	if err != nil {
		return
	}

	for _, release := range githubReleases {
		if release.TagName[0] != 'v' {
			continue
		}
		release.TagName = release.TagName[1:]

		assetId := slices.IndexFunc(release.Assets, func(asset GithubAsset) bool {
			if strings.HasPrefix(runtime.GOARCH, "amd64") {
				return strings.HasSuffix(asset.Name, "-x64.AppImage")
			} else if strings.HasPrefix(runtime.GOARCH, "arm64") {
				return strings.HasSuffix(asset.Name, "-arm64.AppImage ")
			}

			return false
		})
		if assetId == -1 {
			continue
		}

		if !slices.ContainsFunc(runners, func(runner Runner) bool {
			return runner.RunnerID == "duckstation_"+release.TagName
		}) {
			runners = append(runners, Runner{
				DisplayName:      "DuckStation " + release.TagName,
				RunnerID:         "duckstation_" + release.TagName,
				Type:             "duckstation",
				System:           "ps1",
				Exec:             release.Assets[assetId].BrowserDownloadUrl,
				downloadFunc:     downloadPlaystation1Runner,
				runFunc:          runPlaystation1Runner,
				openSettingsFunc: openSettingsPlaystation1Runner,
			})
		}
	}

	return
}

func GetPlaystation2Runners() (runners []Runner, err error) {
	runners = make([]Runner, 0)

	// Get user home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return
	}

	// Get local runners
	if pcsx2Path, err := exec.LookPath("pcsx2"); err == nil {
		output, _ := exec.Command("flatpak", "run", "net.pcsx2.PCSX2", "-version").CombinedOutput()

		version := ""
		for _, line := range strings.Split(string(output), "\n") {
			if v, found := strings.CutPrefix(line, "PCSX2 v"); found {
				version = v
				break
			}
		}
		if version != "" {
			runners = append(runners, Runner{
				DisplayName:      "PCSX2 " + version + " (System)",
				RunnerID:         "pcsx2_system",
				Type:             "pcsx2",
				System:           "ps2",
				Exec:             pcsx2Path,
				runFunc:          runPlaystation2Runner,
				openSettingsFunc: openSettingsPlaystation2Runner,
			})
		}
	}
	if err = exec.Command("flatpak", "info", "net.pcsx2.PCSX2").Run(); err == nil {
		output, _ := exec.Command("flatpak", "run", "net.pcsx2.PCSX2", "-version").CombinedOutput()

		version := ""
		for _, line := range strings.Split(string(output), "\n") {
			if v, found := strings.CutPrefix(line, "PCSX2 v"); found {
				version = v
				break
			}
		}
		if version != "" {
			runners = append(runners, Runner{
				DisplayName:      "PCSX2 " + version + " (Flatpak)",
				RunnerID:         "pcsx2_flatpak",
				Type:             "pcsx2",
				System:           "ps2",
				Exec:             "net.pcsx2.PCSX2",
				runFunc:          runPlaystation2Runner,
				openSettingsFunc: openSettingsPlaystation2Runner,
			})
		}
	}

	dirEntries, err := os.ReadDir(filepath.Join(homeDir, ".local/share/GameDiscPlayer/runners/ps2"))
	if err == nil {
		// Show installed runners in reverse alphabetical order
		slices.Reverse(dirEntries)

		for _, entry := range dirEntries {
			entryPath := filepath.Join(homeDir, ".local/share/GameDiscPlayer/runners/ps2", entry.Name())

			// Check for PCSX2 runners
			if _, err = os.Stat(filepath.Join(entryPath, "pcsx2")); err == nil {
				output, _ := exec.Command(filepath.Join(entryPath, "pcsx2"), "-version").CombinedOutput()

				version := ""
				for _, line := range strings.Split(string(output), "\n") {
					if v, found := strings.CutPrefix(line, "PCSX2 v"); found {
						version = v
						break
					}
				}
				if version == "" {
					return
				}

				runners = append(runners, Runner{
					DisplayName:      "PCSX2 " + version,
					RunnerID:         "pcsx2_" + version,
					Type:             "pcsx2",
					System:           "ps2",
					Exec:             filepath.Join(entryPath, "pcsx2"),
					runFunc:          runPlaystation2Runner,
					openSettingsFunc: openSettingsPlaystation2Runner,
				})
			}
		}
	} else {
		err = nil
	}

	// Get downloadable runners
	if launcher.Offline {
		return
	}
	githubReleases, err := GetGithubReleases(PCSX2_VERSION_REQUEST_URL)
	if err != nil {
		return
	}

	for _, release := range githubReleases {
		if release.TagName[0] != 'v' {
			continue
		}
		release.TagName = release.TagName[1:]

		assetId := slices.IndexFunc(release.Assets, func(asset GithubAsset) bool {
			if strings.HasPrefix(runtime.GOARCH, "amd64") {
				return strings.HasSuffix(asset.Name, "x64-Qt.AppImage")
			}

			return false
		})
		if assetId == -1 {
			continue
		}

		if !slices.ContainsFunc(runners, func(runner Runner) bool {
			return runner.RunnerID == "pcsx2_"+release.TagName
		}) {
			runners = append(runners, Runner{
				DisplayName:      "PCSX2 " + release.TagName,
				RunnerID:         "pcsx2_" + release.TagName,
				Type:             "pcsx2",
				System:           "ps2",
				Exec:             release.Assets[assetId].BrowserDownloadUrl,
				downloadFunc:     downloadPlaystation2Runner,
				runFunc:          runPlaystation2Runner,
				openSettingsFunc: openSettingsPlaystation2Runner,
			})
		}
	}

	return
}

func downloadWindowsRunner(runner *Runner) (err error) {
	if !strings.HasPrefix(runner.Exec, "https://") && !strings.HasPrefix(runner.Exec, "http://") {
		return
	}

	// Get user home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return
	}

	// Download runner
	filename := filepath.Join(homeDir, ".local/share/GameDiscPlayer/runners", runner.System)
	err = DownloadFile(runner.Exec, filename, runner.DisplayName)
	if err != nil {
		return
	}

	fmt.Println("Extracting " + runner.DisplayName + "...")
	progressWindow := NewProgressWindow("Extracting "+runner.DisplayName+"...", 0, 1)
	progressWindow.Pulse()
	defer progressWindow.Close()

	// Setup extract command
	cmd := exec.Command("tar", "xf", filename)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = filepath.Join(homeDir, ".local/share/GameDiscPlayer/runners", runner.System)

	err = cmd.Run()
	if err != nil {
		return
	}

	// Remove downloaded file
	os.Remove(filename)

	runner.Exec = filepath.Join(homeDir, ".local/share/GameDiscPlayer/runners", runner.System, runner.DisplayName)

	return
}

func downloadGameboyAdvanceRunner(runner *Runner) (err error) {
	if !strings.HasPrefix(runner.Exec, "https://") && !strings.HasPrefix(runner.Exec, "http://") {
		return
	}

	// Get user home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return
	}

	// Download runner
	switch runner.Type {
	case "mgba":
		filename := filepath.Join(homeDir, ".local/share/GameDiscPlayer/runners", runner.System, runner.DisplayName, "mgba")
		err = DownloadFile(runner.Exec, filename, runner.DisplayName)
		if err != nil {
			return
		}

		// Make downloaded file executable
		err = os.Chmod(filename, 0755)
		if err != nil {
			return err
		}

		// Make installation portable to avoid interfering with system configurations
		var f *os.File
		f, err = os.Create(filepath.Join(filepath.Dir(filename), "portable.ini"))
		if err != nil && !os.IsExist(err) {
			return
		}
		f.Close()

		// Create saves directory
		err = os.MkdirAll(filepath.Join(filepath.Dir(filename), "saves"), 0755)
		if err != nil {
			return
		}
		// Create cheats directory
		err = os.MkdirAll(filepath.Join(filepath.Dir(filename), "cheats"), 0755)
		if err != nil {
			return
		}
		// Create screenshots directory
		err = os.MkdirAll(filepath.Join(filepath.Dir(filename), "screenshots"), 0755)
		if err != nil {
			return
		}
		// Create patches directory
		err = os.MkdirAll(filepath.Join(filepath.Dir(filename), "patches"), 0755)
		if err != nil {
			return
		}

		config := `[ports.qt]
savestatePath=states
cheatsPath=cheats
screenshotPath=screenshots
savegamePath=saves
patchPath=patches
gb.bios=../bios/gb_bios.bin
gbc.bios=../bios/gbc_bios.bin
sgb.bios=../bios/sgb_bios.bin
gba.bios=../bios/gba_bios.bin
`
		err = os.WriteFile(filepath.Join(filepath.Dir(filename), "config.ini"), []byte(config), 0644)
		if err != nil {
			return err
		}

		// Create bios directory
		err = os.MkdirAll(filepath.Join(homeDir, ".local/share/GameDiscPlayer/runners", runner.System, "bios"), 0755)
		if err != nil {
			return err
		}

		runner.Exec = filename
	default:
		return fmt.Errorf("unknown runner type")
	}

	return
}

func downloadPlaystation1Runner(runner *Runner) (err error) {
	if !strings.HasPrefix(runner.Exec, "https://") && !strings.HasPrefix(runner.Exec, "http://") {
		return
	}

	// Get user home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return
	}

	// Download runner
	switch runner.Type {
	case "duckstation":
		filename := filepath.Join(homeDir, ".local/share/GameDiscPlayer/runners", runner.System, runner.DisplayName, "duckstation")
		err = DownloadFile(runner.Exec, filename, runner.DisplayName)
		if err != nil {
			return
		}

		// Make downloaded file executable
		err = os.Chmod(filename, 0755)
		if err != nil {
			return err
		}

		// Make installation portable to avoid interfering with system configurations
		f, err := os.Create(filepath.Join(filepath.Dir(filename), "portable.txt"))
		if err != nil && !os.IsExist(err) {
			return err
		}
		f.Close()

		config := `[Main]
NoDesktopFile = true

[UI]
Theme =

[AutoUpdater]
CheckAtStartup = false

[BIOS]
SearchDirectory = ../bios

[Pad1]
Circle = Keyboard/L
Cross = Keyboard/K
Down = Keyboard/DownArrow
L1 = Keyboard/Q
L2 = Keyboard/1
L3 = Keyboard/2
LDown = Keyboard/S
LLeft = Keyboard/A
LRight = Keyboard/D
LUp = Keyboard/W
Left = Keyboard/LeftArrow
R1 = Keyboard/E
R2 = Keyboard/3
R3 = Keyboard/4
RDown = Keyboard/G
RLeft = Keyboard/F
RRight = Keyboard/H
RUp = Keyboard/T
Right = Keyboard/RightArrow
Select = Keyboard/Backspace
Square = Keyboard/J
Start = Keyboard/Enter
Triangle = Keyboard/I
Up = Keyboard/UpArrow

[Hotkeys]
OpenPauseMenu = Keyboard/Escape`
		err = os.WriteFile(filepath.Join(filepath.Dir(filename), "settings.ini"), []byte(config), 0644)
		if err != nil {
			return err
		}

		// Create bios directory
		err = os.MkdirAll(filepath.Join(homeDir, ".local/share/GameDiscPlayer/runners", runner.System, "bios"), 0755)
		if err != nil {
			return err
		}

		runner.Exec = filename
	default:
		return fmt.Errorf("unknown runner type")
	}

	return
}

func downloadPlaystation2Runner(runner *Runner) (err error) {
	if !strings.HasPrefix(runner.Exec, "https://") && !strings.HasPrefix(runner.Exec, "http://") {
		return
	}

	// Get user home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return
	}

	// Download runner
	switch runner.Type {
	case "pcsx2":
		filename := filepath.Join(homeDir, ".local/share/GameDiscPlayer/runners", runner.System, runner.DisplayName, "pcsx2")
		err = DownloadFile(runner.Exec, filename, runner.DisplayName)
		if err != nil {
			return
		}

		// Make downloaded file executable
		err = os.Chmod(filename, 0755)
		if err != nil {
			return err
		}

		config := `[UI]
SettingsVersion = 1
Theme =


[Folders]
Bios = ../../bios

[EmuCore]
EnableFastBoot = false
EnableFastBootFastForward = true

[InputSources]
Keyboard = true
Mouse = true
SDL = true
SDLControllerEnhancedMode = true
SDLPS5PlayerLED = true

[Hotkeys]
ToggleFullscreen = Keyboard/Alt & Keyboard/Return
CycleAspectRatio = Keyboard/F6
CycleInterlaceMode = Keyboard/F5
ToggleMipmapMode = Keyboard/Insert
GSDumpMultiFrame = Keyboard/Control & Keyboard/Shift & Keyboard/F8
Screenshot = Keyboard/F8
GSDumpSingleFrame = Keyboard/Shift & Keyboard/F8
ToggleSoftwareRendering = Keyboard/F9
ZoomIn = Keyboard/Control & Keyboard/Plus
ZoomOut = Keyboard/Control & Keyboard/Minus
InputRecToggleMode = Keyboard/Shift & Keyboard/R
LoadStateFromSlot = Keyboard/F3
SaveStateToSlot = Keyboard/F1
NextSaveStateSlot = Keyboard/F2
PreviousSaveStateSlot = Keyboard/Shift & Keyboard/F2
OpenPauseMenu = Keyboard/Escape
ToggleFrameLimit = Keyboard/F4
TogglePause = Keyboard/Space
ToggleSlowMotion = Keyboard/Shift & Keyboard/Backtab
ToggleTurbo = Keyboard/Tab
HoldTurbo = Keyboard/Period

[Pad1]
Type = DualShock2
InvertL = 0
InvertR = 0
Deadzone = 0
AxisScale = 1.33
LargeMotorScale = 1
SmallMotorScale = 1
ButtonDeadzone = 0
PressureModifier = 0.5
Up = Keyboard/Up
Right = Keyboard/Right
Down = Keyboard/Down
Left = Keyboard/Left
Triangle = Keyboard/I
Circle = Keyboard/L
Cross = Keyboard/K
Square = Keyboard/J
Select = Keyboard/Backspace
Start = Keyboard/Return
L1 = Keyboard/Q
L2 = Keyboard/1
R1 = Keyboard/E
R2 = Keyboard/3
L3 = Keyboard/2
R3 = Keyboard/4
LUp = Keyboard/W
LRight = Keyboard/D
LDown = Keyboard/S
LLeft = Keyboard/A
RUp = Keyboard/T
RRight = Keyboard/H
RDown = Keyboard/G
RLeft = Keyboard/F

[AutoUpdater]
CheckAtStartup = false
`
		err = os.MkdirAll(filepath.Join(filepath.Dir(filename), "PCSX2", "inis"), 0755)
		if err != nil {
			return
		}
		err = os.WriteFile(filepath.Join(filepath.Dir(filename), "PCSX2.ini"), []byte(config), 0644)
		if err != nil {
			return
		}

		// Create bios directory
		err = os.MkdirAll(filepath.Join(homeDir, ".local/share/GameDiscPlayer/runners", runner.System, "bios"), 0755)
		if err != nil {
			return err
		}

		runner.Exec = filename
	default:
		return fmt.Errorf("unknown runner type")
	}

	return
}

func runLinuxRunner(runner *Runner) (err error) {
	// Get working directory
	workDir, err := os.Getwd()
	if err != nil {
		return
	}

	// Run game
	var cmd *exec.Cmd
	if launcher.IsGameInstalled() {
		cmd = exec.Command(filepath.Join(launcher.DataDir, "files", launcher.Metadata.Run))
		cmd.Dir = filepath.Dir(cmd.Path)
	} else {
		cmd = exec.Command(filepath.Join(workDir, "files", launcher.Metadata.Run))
		cmd.Dir = filepath.Dir(cmd.Path)
	}
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Run with MangoHud
	mangohudPath, err := exec.LookPath("mangohud")
	if err == nil && launcher.Options.Mangohud {
		cmd.Args = slices.Insert(cmd.Args, 0, cmd.Path)
		cmd.Path = mangohudPath
	}

	// Setup environment
	cmd.Env = cmd.Environ()
	cmd.Env = append(cmd.Env, launcher.Options.Environment...)

	err = cmd.Start()
	if err != nil {
		return
	}
	launcher.GameProcess = cmd.Process
	EventEmit("game_state_changed", "running")
	cmd.Wait()
	launcher.GameProcess = nil
	EventEmit("game_state_changed", "idle")

	return
}

func runWindowsRunner(runner *Runner) (err error) {
	// Get working directory
	workDir, err := os.Getwd()
	if err != nil {
		return
	}

	// Get user home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return
	}

	// Check for umu
	umuPath, err := exec.LookPath("umu-run")
	if err != nil {
		umuPath = filepath.Join(homeDir, ".local/bin/umu-run")
		_, err = os.Stat(umuPath)
		if err != nil {
			err = DownloadUmu()
			if err != nil {
				return
			}
		}
	}

	// Download runner if required
	err = runner.Download()
	if err != nil {
		return
	}

	prefixDir := filepath.Join(launcher.DataDir, "prefix")

	// Run winetricks
	if _, err := os.Stat(prefixDir); err != nil && len(launcher.Metadata.WinetricksVerbs) > 0 {
		progressWindow := NewProgressWindow("Setting up prefix...", 0, 1)

		// Setup command
		cmd := exec.Command("umu-run", "winetricks")
		cmd.Args = append(cmd.Args, launcher.Metadata.WinetricksVerbs...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Dir = filepath.Join(workDir, "files")

		// Run with MangoHud
		mangohudPath, err := exec.LookPath("mangohud")
		if err == nil && launcher.Options.Mangohud {
			cmd.Args = slices.Insert(cmd.Args, 0, cmd.Path)
			cmd.Path = mangohudPath
		}

		// Setup environment
		cmd.Env = os.Environ()
		cmd.Env = append(cmd.Env, "PROTONPATH="+runner.Exec)
		cmd.Env = append(cmd.Env, "WINEPREFIX="+prefixDir)

		err = cmd.Run()
		if err != nil {
			log.Fatal(err)
		}

		progressWindow.Close()
	}

	// Run game
	cmd := exec.Command(umuPath, launcher.Metadata.Run)
	if launcher.IsGameInstalled() {
		cmd.Dir = filepath.Join(launcher.DataDir, "files")
	} else {
		cmd.Dir = filepath.Join(workDir, "files")
	}
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Run with MangoHud
	mangohudPath, err := exec.LookPath("mangohud")
	if err == nil && launcher.Options.Mangohud {
		cmd.Args = slices.Insert(cmd.Args, 0, cmd.Path)
		cmd.Path = mangohudPath
	}

	// Setup environment
	cmd.Env = cmd.Environ()
	cmd.Env = append(cmd.Env, launcher.Options.Environment...)
	cmd.Env = append(cmd.Env, "PROTONPATH="+runner.Exec)
	cmd.Env = append(cmd.Env, "WINEPREFIX="+prefixDir)

	err = cmd.Start()
	if err != nil {
		return
	}
	launcher.GameProcess = cmd.Process
	EventEmit("game_state_changed", "running")
	cmd.Wait()
	launcher.GameProcess = nil
	EventEmit("game_state_changed", "idle")

	return
}

func runGameboyAdvanceRunner(runner *Runner) (err error) {
	// Get working directory
	workDir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	// Download runner if required
	err = runner.Download()
	if err != nil {
		return
	}

	// Run game
	cmd := exec.Command(runner.Exec)
	if runner.IsFlatpak() {
		flatpakPath, err := exec.LookPath("flatpak")
		if err != nil {
			log.Fatal(err)
		}
		cmd.Path = flatpakPath
		cmd.Args = append(cmd.Args, "run", runner.Exec)
	}
	switch runner.Type {
	case "mgba":
		if runner.IsLocal() {
			cmd.Dir = filepath.Dir(runner.Exec)
		} else {
			err = os.MkdirAll(filepath.Join(launcher.DataDir, "saves"), 0755)
			if err != nil {
				return
			}
			cmd.Args = append(cmd.Args, "-CsavegamePath="+filepath.Join(launcher.DataDir, "saves"))
		}

		cmd.Args = append(cmd.Args, "--fullscreen")

		if launcher.IsGameInstalled() {
			cmd.Args = append(cmd.Args, filepath.Join(launcher.DataDir, "files", launcher.Metadata.Run))
		} else {
			cmd.Args = append(cmd.Args, filepath.Join(workDir, "files", launcher.Metadata.Run))
		}
	default:
		return fmt.Errorf("unknown runner type '%s'", runner.Type)
	}
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Run with MangoHud
	mangohudPath, err := exec.LookPath("mangohud")
	if err == nil && launcher.Options.Mangohud {
		cmd.Args = slices.Insert(cmd.Args, 0, cmd.Path)
		cmd.Path = mangohudPath
	}

	// Setup environment
	cmd.Env = cmd.Environ()
	cmd.Env = append(cmd.Env, launcher.Options.Environment...)

	err = cmd.Start()
	if err != nil {
		return
	}
	launcher.GameProcess = cmd.Process
	EventEmit("game_state_changed", "running")
	cmd.Wait()
	launcher.GameProcess = nil
	EventEmit("game_state_changed", "idle")

	return
}

func runPlaystation1Runner(runner *Runner) (err error) {
	// Get working directory
	workDir, err := os.Getwd()
	if err != nil {
		return
	}

	// Download runner if required
	err = runner.Download()
	if err != nil {
		return
	}

	// Run game
	cmd := exec.Command(runner.Exec)
	if runner.IsFlatpak() {
		flatpakPath, err := exec.LookPath("flatpak")
		if err != nil {
			return err
		}
		cmd.Path = flatpakPath
		cmd.Args = append(cmd.Args, "run", runner.Exec)
	}
	switch runner.Type {
	case "duckstation":
		if runner.IsLocal() {
			cmd.Dir = filepath.Dir(runner.Exec)
		}

		cmd.Args = append(cmd.Args, "-nogui", "-fullscreen")

		if launcher.IsGameInstalled() {
			cmd.Args = append(cmd.Args, filepath.Join(launcher.DataDir, "files", launcher.Metadata.Run))
		} else {
			cmd.Args = append(cmd.Args, filepath.Join(workDir, "files", launcher.Metadata.Run))
		}
	default:
		return fmt.Errorf("unknown runner type '%s'", runner.Type)
	}
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Run with MangoHud
	mangohudPath, err := exec.LookPath("mangohud")
	if err == nil && launcher.Options.Mangohud {
		cmd.Args = slices.Insert(cmd.Args, 0, cmd.Path)
		cmd.Path = mangohudPath
	}

	// Setup environment
	cmd.Env = cmd.Environ()
	cmd.Env = append(cmd.Env, launcher.Options.Environment...)

	err = cmd.Start()
	if err != nil {
		return
	}
	launcher.GameProcess = cmd.Process
	EventEmit("game_state_changed", "running")
	cmd.Wait()
	launcher.GameProcess = nil
	EventEmit("game_state_changed", "idle")

	return
}

func runPlaystation2Runner(runner *Runner) (err error) {
	// Get working directory
	workDir, err := os.Getwd()
	if err != nil {
		return
	}

	// Download runner if required
	err = runner.Download()
	if err != nil {
		return
	}

	// Run game
	cmd := exec.Command(runner.Exec)
	if runner.IsFlatpak() {
		flatpakPath, err := exec.LookPath("flatpak")
		if err != nil {
			return err
		}
		cmd.Path = flatpakPath
		cmd.Args = append(cmd.Args, "run", runner.Exec)
	}
	switch runner.Type {
	case "pcsx2":
		if runner.IsLocal() {
			cmd.Dir = filepath.Dir(runner.Exec)
			cmd.Args = append(cmd.Args, "-portable")
		}

		cmd.Args = append(cmd.Args, "-nogui", "-fullscreen")

		if launcher.IsGameInstalled() {
			cmd.Args = append(cmd.Args, filepath.Join(launcher.DataDir, "files", launcher.Metadata.Run))
		} else {
			cmd.Args = append(cmd.Args, filepath.Join(workDir, "files", launcher.Metadata.Run))
		}
	default:
		return fmt.Errorf("unknown runner type '%s'", runner.Type)
	}
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Run with MangoHud
	mangohudPath, err := exec.LookPath("mangohud")
	if err == nil && launcher.Options.Mangohud {
		cmd.Args = slices.Insert(cmd.Args, 0, cmd.Path)
		cmd.Path = mangohudPath
	}

	// Setup environment
	cmd.Env = cmd.Environ()
	cmd.Env = append(cmd.Env, launcher.Options.Environment...)

	err = cmd.Start()
	if err != nil {
		return
	}
	launcher.GameProcess = cmd.Process
	EventEmit("game_state_changed", "running")
	cmd.Wait()
	launcher.GameProcess = nil
	EventEmit("game_state_changed", "idle")

	return
}

func openSettingsGameboyAdvanceRunner(runner *Runner) (err error) {
	// Download runner if required
	err = runner.Download()
	if err != nil {
		return
	}

	// Open settings
	cmd := exec.Command(runner.Exec)
	if runner.IsFlatpak() {
		flatpakPath, err := exec.LookPath("flatpak")
		if err != nil {
			log.Fatal(err)
		}
		cmd.Path = flatpakPath
		cmd.Args = append(cmd.Args, "run", runner.Exec)
	}
	switch runner.Type {
	case "mgba":
		if runner.IsLocal() {
			cmd.Dir = filepath.Dir(runner.Exec)
		} else {
			err = os.MkdirAll(filepath.Join(launcher.DataDir, "saves"), 0755)
			if err != nil {
				return
			}
			cmd.Args = append(cmd.Args, "-CsavegamePath="+filepath.Join(launcher.DataDir, "saves"))
		}
	default:
		return fmt.Errorf("unknown runner type '%s'", runner.Type)
	}
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Run with MangoHud
	mangohudPath, err := exec.LookPath("mangohud")
	if err == nil && launcher.Options.Mangohud {
		cmd.Args = slices.Insert(cmd.Args, 0, cmd.Path)
		cmd.Path = mangohudPath
	}

	// Setup environment
	cmd.Env = cmd.Environ()
	cmd.Env = append(cmd.Env, launcher.Options.Environment...)

	err = cmd.Run()
	if err != nil {
		return
	}

	return
}

func openSettingsPlaystation1Runner(runner *Runner) (err error) {
	// Download runner if required
	err = runner.Download()
	if err != nil {
		return
	}

	// Run game
	cmd := exec.Command(runner.Exec)
	if runner.IsFlatpak() {
		flatpakPath, err := exec.LookPath("flatpak")
		if err != nil {
			log.Fatal(err)
		}
		cmd.Path = flatpakPath
		cmd.Args = append(cmd.Args, "run", runner.Exec)
	}
	switch runner.Type {
	case "duckstation":
		if runner.IsLocal() {
			cmd.Dir = filepath.Dir(runner.Exec)
		}
	default:
		return fmt.Errorf("unknown runner type '%s'", runner.Type)
	}
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Run with MangoHud
	mangohudPath, err := exec.LookPath("mangohud")
	if err == nil && launcher.Options.Mangohud {
		cmd.Args = slices.Insert(cmd.Args, 0, cmd.Path)
		cmd.Path = mangohudPath
	}

	// Setup environment
	cmd.Env = cmd.Environ()
	cmd.Env = append(cmd.Env, launcher.Options.Environment...)

	err = cmd.Run()
	if err != nil {
		return
	}

	return
}

func openSettingsPlaystation2Runner(runner *Runner) (err error) {
	// Download runner if required
	err = runner.Download()
	if err != nil {
		return
	}

	// Run game
	cmd := exec.Command(runner.Exec)
	if runner.IsFlatpak() {
		flatpakPath, err := exec.LookPath("flatpak")
		if err != nil {
			return err
		}
		cmd.Path = flatpakPath
		cmd.Args = append(cmd.Args, "run", runner.Exec)
	}
	switch runner.Type {
	case "pcsx2":
		if runner.IsLocal() {
			cmd.Dir = filepath.Dir(runner.Exec)
			cmd.Args = append(cmd.Args, "-portable")
		}
	default:
		return fmt.Errorf("unknown runner type '%s'", runner.Type)
	}
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Run with MangoHud
	mangohudPath, err := exec.LookPath("mangohud")
	if err == nil && launcher.Options.Mangohud {
		cmd.Args = slices.Insert(cmd.Args, 0, cmd.Path)
		cmd.Path = mangohudPath
	}

	// Setup environment
	cmd.Env = cmd.Environ()
	cmd.Env = append(cmd.Env, launcher.Options.Environment...)

	err = cmd.Run()
	if err != nil {
		return
	}
	cmd.Wait()

	return
}

func (runner *Runner) Download() error {
	if !strings.HasPrefix(runner.Exec, "http://") && !strings.HasPrefix(runner.Exec, "https://") {
		return nil
	}

	if runner.downloadFunc == nil {
		return fmt.Errorf("runner is missing a download function")
	}

	return runner.downloadFunc(runner)
}

func (runner *Runner) Run() error {
	if runner.runFunc == nil {
		return fmt.Errorf(runner.DisplayName + " runner is missing a play function")
	}

	return runner.runFunc(runner)
}

func (runner *Runner) OpenSettings() error {
	if runner.openSettingsFunc == nil {
		return fmt.Errorf(runner.DisplayName + " runner is missing an openSettings function")
	}

	return runner.openSettingsFunc(runner)
}

func (runner *Runner) IsSystem() bool {
	return strings.HasSuffix(runner.RunnerID, "_system")
}

func (runner *Runner) IsFlatpak() bool {
	return strings.HasSuffix(runner.RunnerID, "_flatpak")
}

func (runner *Runner) IsLocal() bool {
	// Get user home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return false
	}

	return strings.HasPrefix(runner.Exec, filepath.Join(homeDir, ".local/share/GameDiscPlayer/runners"))
}
