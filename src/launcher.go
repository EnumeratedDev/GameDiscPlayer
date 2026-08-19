package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"time"

	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

type Launcher struct {
	Metadata Metadata
	DataDir  string
	Options  Options

	App        *gtk.Application
	MainWindow *gtk.ApplicationWindow

	Runners        []Runner
	SelectedRunner string

	NoGUI   bool
	Offline bool

	GameProcess *os.Process
}

var launcher = Launcher{Options: Options{}}

func SetupLauncher() {
	launcher.App = gtk.NewApplication("dev.enumerated.GameDiscPlayer", gio.ApplicationFlags(gio.ApplicationNonUnique))

	launcher.App.AddMainOption("offline", 0, glib.OptionFlags(glib.OptionFlagNone), glib.OptionArg(glib.OptionArgNone), "Do not fetch downloadable runners from the internet", "")
	launcher.App.AddMainOption("runner", 0, glib.OptionFlags(glib.OptionFlagNone), glib.OptionArg(glib.OptionArgString), "Set the runner to use", "runner_id")
	launcher.App.AddMainOption("list-runners", 0, glib.OptionFlags(glib.OptionFlagNone), glib.OptionArg(glib.OptionArgNone), "List all available runners", "")
	launcher.App.AddMainOption("play", 0, glib.OptionFlags(glib.OptionFlagNone), glib.OptionArg(glib.OptionArgNone), "Skip the launcher and launch the game", "")
	launcher.App.AddMainOption("settings", 0, glib.OptionFlags(glib.OptionFlagNone), glib.OptionArg(glib.OptionArgNone), "Skip the launcher and open runner settings", "")
	launcher.App.AddMainOption("install", 0, glib.OptionFlags(glib.OptionFlagNone), glib.OptionArg(glib.OptionArgNone), "Skip the launcher and install game", "")
	launcher.App.AddMainOption("uninstall", 0, glib.OptionFlags(glib.OptionFlagNone), glib.OptionArg(glib.OptionArgNone), "Skip the launcher and uninstall game", "")

	launcher.App.ConnectStartup(handleStartup)
	launcher.App.ConnectHandleLocalOptions(handleLocalOptions)
	launcher.App.ConnectActivate(handleActivate)

	if code := launcher.App.Run(os.Args); code > 0 {
		os.Exit(code)
	}
}

func handleStartup() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}

	// Read game metadata
	launcher.Metadata, err = ReadMetadata()
	if err != nil {
		log.Fatalf("Game metadata not found: %s", err)
	}

	// Set data directory
	launcher.DataDir = filepath.Join(homeDir, "Games", launcher.Metadata.Name)

	// Parse options
	err = launcher.ParseOptions()
	if err != nil {
		log.Fatal(err)
	}

	// Get runners
	err = launcher.FetchAvailableRunners()
	if err != nil {
		log.Fatal(err)
	}

	if launcher.SelectedRunner != "" && !slices.ContainsFunc(launcher.Runners, func(r Runner) bool {
		return r.RunnerID == launcher.SelectedRunner
	}) {
		log.Fatalf("invalid runner_id")
	}
}

func handleLocalOptions(options *glib.VariantDict) (gint int) {
	launcher.Offline = options.Contains("offline")

	if v := options.LookupValue("runner", glib.NewVariantType("s")); v != nil {
		launcher.SelectedRunner = v.String()
	}
	if options.Contains("list-runners") {
		launcher.NoGUI = true
		handleStartup()

		fmt.Println("Runner ID\tDisplay Name\tType\tInstalled")
		for _, runner := range launcher.Runners {
			isInstalled := !strings.HasPrefix(runner.Exec, "http://") && !strings.HasPrefix(runner.Exec, "https://")

			fmt.Printf("%s\t%s\t%s\t%t\n", runner.RunnerID, runner.DisplayName, runner.Type, isInstalled)
		}
		return 0
	}
	if options.Contains("play") {
		launcher.NoGUI = true
		handleStartup()

		launcher.Play()

		return 0
	} else if options.Contains("settings") {
		launcher.NoGUI = true
		handleStartup()

		launcher.OpenRunnerSettings()

		return 0
	} else if options.Contains("install") {
		launcher.NoGUI = true
		handleStartup()

		launcher.Install()

		return 0
	} else if options.Contains("uninstall") {
		launcher.Offline = true
		launcher.NoGUI = true
		handleStartup()

		launcher.Uninstall()

		return 0
	}

	return -1
}

func handleActivate() {
	// Set launcher icon
	gtk.WindowSetDefaultIconName("GameDiscPlayer")

	createMainWindow()

	launcher.MainWindow.SetVisible(true)
}

func createMainWindow() {
	// Setup main launcher window
	launcher.MainWindow = gtk.NewApplicationWindow(launcher.App)
	launcher.MainWindow.SetTitle("Play " + launcher.Metadata.Name)
	launcher.MainWindow.SetDefaultSize(600, -1)
	launcher.MainWindow.SetResizable(false)
	launcher.MainWindow.ConnectCloseRequest(func() bool {
		launcher.App.Quit()
		return true
	})

	// Main vertical box
	mainBox := gtk.NewBox(gtk.Orientation(gtk.OrientationVertical), 10)
	mainBox.SetMarginTop(10)
	mainBox.SetMarginBottom(10)
	mainBox.SetMarginStart(10)
	mainBox.SetMarginEnd(10)
	// Horizontal metadata box
	metadataBox := gtk.NewBox(gtk.Orientation(gtk.OrientationHorizontal), 10)
	// Vertical label box
	labelBox := gtk.NewBox(gtk.Orientation(gtk.OrientationVertical), 10)
	// Game icon
	gameIcon := gtk.NewPictureForFilename("icon.png")
	gameIcon.SetSizeRequest(256, -1)
	gameIcon.SetCanShrink(true)
	gameIcon.SetContentFit(gtk.ContentFit(gtk.ContentFitScaleDown))
	gameIcon.SetHExpand(false)
	gameIcon.SetVExpand(false)
	// Game metadata text fields
	versionTextView := gtk.NewTextView()
	versionTextView.Buffer().SetText("Version: " + launcher.Metadata.Version)
	versionTextView.SetLeftMargin(10)
	versionTextView.SetRightMargin(10)
	versionTextView.SetEditable(false)
	versionTextView.SetCursorVisible(false)
	versionScrolledWindow := gtk.NewScrolledWindow()
	versionScrolledWindow.SetPolicy(gtk.PolicyType(gtk.PolicyAutomatic), gtk.PolicyType(gtk.PolicyNever))
	versionScrolledWindow.SetChild(versionTextView)

	developerTextView := gtk.NewTextView()
	developerTextView.Buffer().SetText("Developer: " + launcher.Metadata.Developer)
	developerTextView.SetLeftMargin(10)
	developerTextView.SetRightMargin(10)
	developerTextView.SetEditable(false)
	developerTextView.SetCursorVisible(false)
	developerScrolledWindow := gtk.NewScrolledWindow()
	developerScrolledWindow.SetPolicy(gtk.PolicyType(gtk.PolicyAutomatic), gtk.PolicyType(gtk.PolicyNever))
	developerScrolledWindow.SetChild(developerTextView)

	publisherTextView := gtk.NewTextView()
	publisherTextView.Buffer().SetText("Publisher: " + launcher.Metadata.Publisher)
	publisherTextView.SetLeftMargin(10)
	publisherTextView.SetRightMargin(10)
	publisherTextView.SetEditable(false)
	publisherTextView.SetCursorVisible(false)
	publisherScrolledWindow := gtk.NewScrolledWindow()
	publisherScrolledWindow.SetPolicy(gtk.PolicyType(gtk.PolicyAutomatic), gtk.PolicyType(gtk.PolicyNever))
	publisherScrolledWindow.SetChild(publisherTextView)

	systemTextView := gtk.NewTextView()
	systemTextView.Buffer().SetText("System: " + systemsUserReadable[launcher.Metadata.System])
	systemTextView.SetLeftMargin(10)
	systemTextView.SetRightMargin(10)
	systemTextView.SetEditable(false)
	systemTextView.SetCursorVisible(false)
	systemScrolledWindow := gtk.NewScrolledWindow()
	systemScrolledWindow.SetPolicy(gtk.PolicyType(gtk.PolicyAutomatic), gtk.PolicyType(gtk.PolicyNever))
	systemScrolledWindow.SetChild(systemTextView)

	// Description text view
	descriptionTextView := gtk.NewTextView()
	descriptionTextView.Buffer().InsertMarkup(descriptionTextView.Buffer().StartIter(), launcher.Metadata.Description)
	descriptionTextView.SetWrapMode(gtk.WrapMode(gtk.WrapWord))
	descriptionTextView.SetLeftMargin(10)
	descriptionTextView.SetRightMargin(10)
	descriptionTextView.SetEditable(false)
	descriptionTextView.SetCursorVisible(false)
	descriptionScrolledWindow := gtk.NewScrolledWindow()
	descriptionScrolledWindow.SetHExpand(true)
	descriptionScrolledWindow.SetSizeRequest(-1, 200)
	descriptionScrolledWindow.SetPolicy(gtk.PolicyType(gtk.PolicyNever), gtk.PolicyType(gtk.PolicyAutomatic))
	descriptionScrolledWindow.SetChild(descriptionTextView)
	// Runner settings box
	runnerSettingsBox := gtk.NewBox(gtk.Orientation(gtk.OrientationHorizontal), 10)
	// Runner settings button
	runnerSettingsButton := gtk.NewButtonFromIconName("preferences-other")
	runnerSettingsBox.SetTooltipText("Open runner settings")
	runnerSettingsButton.ConnectClicked(func() {
		go launcher.OpenRunnerSettings()
	})
	// Runner dropdown
	runnerDropdown := gtk.NewDropDownFromStrings(nil)
	runnerDropdown.SetHExpand(true)
	setRunnerDropdownOptions := func() {
		if len(launcher.Runners) == 0 {
			runnerDropdown.SetSensitive(false)
			runnerDropdown.SetModel(gtk.NewStringList([]string{"No runners available"}))
			return
		}

		options := make([]string, 0)
		selected := -1
		for i, runner := range launcher.Runners {
			if selected == -1 && runner.RunnerID == launcher.SelectedRunner {
				launcher.SelectedRunner = runner.RunnerID
				selected = i
			}
			if runner.NeedsDownload() {
				options = append(options, fmt.Sprintf("%s (Downloadable)", runner.DisplayName))
			} else {
				options = append(options, runner.DisplayName)
			}
		}
		runnerDropdown.SetModel(gtk.NewStringList(options))
		if selected >= 0 {
			runnerDropdown.SetSelected(uint(selected))
			runnerSettingsButton.SetVisible(launcher.GetSelectedRunner().openSettingsFunc != nil && !launcher.GetSelectedRunner().NeedsDownload())
		} else {
			launcher.SelectedRunner = launcher.Runners[0].RunnerID
		}

		runnerDropdown.Connect("notify::selected", func() {
			launcher.SelectedRunner = launcher.Runners[runnerDropdown.Selected()].RunnerID

			runnerSettingsButton.SetVisible(launcher.GetSelectedRunner().openSettingsFunc != nil && !launcher.GetSelectedRunner().NeedsDownload())
		})
	}

	setRunnerDropdownOptions()
	// Play button
	playButton := gtk.NewButtonWithLabel("Play")
	playButton.ConnectClicked(func() {
		go launcher.Play()

		if launcher.Options.ExitOnPlay {
			launcher.MainWindow.SetVisible(false)
		}
	})
	// Stop button
	stopButton := gtk.NewButtonWithLabel("Stop")
	stopButton.SetVisible(false)
	stopButton.ConnectClicked(func() {
		go launcher.Stop()
	})
	// Install and uninstall buttons
	installButton := gtk.NewButtonWithLabel("Install")
	uninstallButton := gtk.NewButtonWithLabel("Uninstall")

	installButton.ConnectClicked(func() {
		go launcher.Install()
	})
	uninstallButton.ConnectClicked(func() {

		launcher.MainWindow.SetSensitive(false)

		go launcher.Uninstall()
	})

	if launcher.IsGameInstalled() {
		installButton.SetVisible(false)
		uninstallButton.SetVisible(true)
	} else {
		installButton.SetVisible(true)
		uninstallButton.SetVisible(false)

		if launcher.Metadata.RunFromDisc {
			playButton.SetLabel("Play from disc")
		} else {
			playButton.SetLabel("Play from disc (This game can't run from disc)")
			playButton.SetSensitive(false)
		}
	}

	if launcher.GetSelectedRunner() == nil {
		playButton.SetSensitive(false)
		installButton.SetSensitive(false)
	}

	if len(launcher.Metadata.LaunchOptions) == 0 {
		playButton.SetLabel("Play (Game has no declared launch options)")
		playButton.SetSensitive(false)
	}

	// Setup events
	EventSubscribe("game_state_changed", func(args ...any) {
		state := args[0].(string)
		switch state {
		case "idle":
			if launcher.Options.ExitOnPlay {
				launcher.MainWindow.Close()
			}

			// Update dropdown options
			launcher.FetchAvailableRunners()
			setRunnerDropdownOptions()

			runnerDropdown.SetSensitive(true)
			runnerSettingsButton.SetSensitive(true)
			playButton.SetVisible(true)
			playButton.SetSensitive(true)
			stopButton.SetVisible(false)
			stopButton.SetSensitive(false)
			installButton.SetSensitive(true)
			uninstallButton.SetSensitive(true)

			if launcher.IsGameInstalled() {
				playButton.SetLabel("Play")
			} else if launcher.Metadata.RunFromDisc {
				playButton.SetLabel("Play from disc")
			} else {
				playButton.SetLabel("Play from disc (This game can't run from disc)")
				playButton.SetSensitive(false)
			}
		case "launching":
			runnerDropdown.SetSensitive(false)
			runnerSettingsButton.SetSensitive(false)
			playButton.SetVisible(true)
			playButton.SetSensitive(false)
			stopButton.SetVisible(false)
			stopButton.SetSensitive(false)
			installButton.SetSensitive(false)
			uninstallButton.SetSensitive(false)

			playButton.SetLabel("Launching...")
		case "running":
			runnerDropdown.SetSensitive(false)
			runnerSettingsButton.SetSensitive(false)
			playButton.SetVisible(false)
			playButton.SetSensitive(false)
			stopButton.SetVisible(true)
			stopButton.SetSensitive(true)
			installButton.SetSensitive(false)
			uninstallButton.SetSensitive(false)

			stopButton.SetLabel("Stop")
		case "stopping":
			runnerDropdown.SetSensitive(false)
			runnerSettingsButton.SetSensitive(false)
			playButton.SetVisible(false)
			playButton.SetSensitive(false)
			stopButton.SetVisible(true)
			stopButton.SetSensitive(false)
			installButton.SetSensitive(false)
			uninstallButton.SetSensitive(false)

			stopButton.SetLabel("Stopping...")
		}
	})

	labelBox.Append(gameIcon)
	labelBox.Append(versionScrolledWindow)
	labelBox.Append(developerScrolledWindow)
	labelBox.Append(publisherScrolledWindow)
	labelBox.Append(systemScrolledWindow)
	metadataBox.Append(labelBox)
	metadataBox.Append(descriptionScrolledWindow)
	runnerSettingsBox.Append(runnerDropdown)
	runnerSettingsBox.Append(runnerSettingsButton)
	mainBox.Append(metadataBox)
	mainBox.Append(runnerSettingsBox)

	mainBox.Append(playButton)
	mainBox.Append(stopButton)
	mainBox.Append(installButton)
	mainBox.Append(uninstallButton)

	launcher.MainWindow.SetChild(mainBox)
}

func createLaunchOptionsWindow() int {
	if len(launcher.Metadata.LaunchOptions) == 0 {
		return -1
	} else if len(launcher.Metadata.LaunchOptions) == 1 {
		return 0
	}

	ch := make(chan int, 1)

	glib.IdleAdd(func() {
		launcher.MainWindow.SetSensitive(false)

		// Setup launch options window
		window := gtk.NewApplicationWindow(launcher.App)
		window.SetTitle("Launch options")
		window.SetDefaultSize(300, 200)
		window.SetResizable(false)
		window.ConnectCloseRequest(func() bool {
			ch <- -1
			return false
		})

		// Vertical box
		box := gtk.NewBox(gtk.Orientation(gtk.OrientationVertical), 10)
		box.SetMarginTop(10)
		box.SetMarginBottom(10)
		box.SetMarginStart(10)
		box.SetMarginEnd(10)

		// Scrolled window
		scrolledWindow := gtk.NewScrolledWindow()
		scrolledWindow.SetPolicy(gtk.PolicyType(gtk.PolicyNever), gtk.PolicyType(gtk.PolicyAutomatic))

		// Launch option buttons
		for i, launchOption := range launcher.Metadata.LaunchOptions {
			button := gtk.NewButtonWithLabel("Launch " + launchOption.DisplayName)
			button.ConnectClicked(func() {
				ch <- i
				window.Close()
			})

			box.Append(button)
		}

		scrolledWindow.SetChild(box)
		window.SetChild(scrolledWindow)
		window.SetVisible(true)
	})

	ret := <-ch

	glib.IdleAdd(func() {
		launcher.MainWindow.SetSensitive(true)
	})

	return ret
}

func (launcher *Launcher) Play() {
	launchOptionIndex := createLaunchOptionsWindow()
	if launchOptionIndex < 0 {
		return
	}

	EventEmit("game_state_changed", "launching")

	// Get user home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		ShowErrorMessage(err)
		EventEmit("game_state_changed", "idle")
		return
	}

	if !launcher.IsGameInstalled() && !launcher.Metadata.RunFromDisc {
		ShowErrorMessage(fmt.Errorf("game cannot be run from disc"))
		EventEmit("game_state_changed", "idle")
		return
	}

	// Copy BIOS files
	biosSystems, err := os.ReadDir("bios")
	if err == nil {
		for _, entry := range biosSystems {
			if !entry.IsDir() {
				continue
			}

			CopyRecursivelyWithProgress(filepath.Join("bios", entry.Name()),
				filepath.Join(homeDir, ".local/share/GameDiscPlayer/runners", entry.Name(), "shared/bios"),
				true)
		}
	}

	if runner := launcher.GetSelectedRunner(); runner != nil {
		// Download runner if required
		if runner.NeedsDownload() {
			if !confirmDownload(runner) {
				EventEmit("game_state_changed", "idle")
				return
			}

			err = runner.Download()
			if err != nil {
				return
			}
		}

		// Set game options
		launcher.Options.Runner = launcher.SelectedRunner
		launcher.SaveOptions()

		fmt.Printf("Launching %s game with launch option '%s' using %s...\n", systemsUserReadable[launcher.Metadata.System], launcher.Metadata.LaunchOptions[launchOptionIndex].DisplayName, runner.DisplayName)
		err := runner.Run(launchOptionIndex)
		if err != nil {
			ShowErrorMessage(err)
			EventEmit("game_state_changed", "idle")
			return
		}
	} else {
		ShowErrorMessage(fmt.Errorf("invalid runner_id"))
		EventEmit("game_state_changed", "idle")
		return
	}

	if launcher.NoGUI || launcher.Options.ExitOnPlay {
		os.Exit(0)
	}
}

func (launcher *Launcher) Stop() {
	if launcher.GameProcess == nil {
		return
	}

	EventEmit("game_state_changed", "stopping")

	done := make(chan bool, 1)
	go func() {
		launcher.GameProcess.Wait()
		done <- true
	}()

	launcher.GameProcess.Signal(syscall.SIGTERM)
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		launcher.GameProcess.Kill()
	}

	EventEmit("game_state_changed", "idle")
}

func (launcher *Launcher) FetchAvailableRunners() (err error) {
	runnerFetcher, ok := RunnerFetchers[launcher.Metadata.System]
	if ok {
		launcher.Runners, err = runnerFetcher()
		if err != nil {
			return
		}

		if len(launcher.Runners) > 0 && launcher.SelectedRunner == "" {
			launcher.SelectedRunner = launcher.Runners[0].RunnerID
		}
	}

	return
}

func (launcher *Launcher) GetSelectedRunner() *Runner {
	for _, runner := range launcher.Runners {
		if runner.RunnerID == launcher.SelectedRunner {
			return &runner
		}
	}

	return nil
}

func (launcher *Launcher) OpenRunnerSettings() {
	if runner := launcher.GetSelectedRunner(); runner != nil {
		// Download runner if required
		if runner.NeedsDownload() {
			ShowErrorMessage(fmt.Errorf("runner is not installed"))
			EventEmit("game_state_changed", "idle")
			return
		}

		err := runner.OpenSettings()
		if err != nil {
			ShowErrorMessage(err)
			EventEmit("game_state_changed", "idle")
			return
		}
	} else {
		ShowErrorMessage(fmt.Errorf("invalid runner_id"))
		EventEmit("game_state_changed", "idle")
		return
	}
}

func (launcher *Launcher) Install() {
	if _, err := os.Stat(".installed"); err == nil {
		ShowErrorMessage(fmt.Errorf("cannot reinstall game from installation directory"))
		EventEmit("game_state_changed", "idle")
		return
	}

	// Get working directory
	workDir, err := os.Getwd()
	if err != nil {
		ShowErrorMessage(err)
		EventEmit("game_state_changed", "idle")
		return
	}

	// Get user home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		ShowErrorMessage(err)
		EventEmit("game_state_changed", "idle")
		return
	}

	// Create game data directory
	err = os.MkdirAll(launcher.DataDir, 0755)
	if err != nil {
		ShowErrorMessage(err)
		EventEmit("game_state_changed", "idle")
		return
	}

	err = CopyRecursivelyWithProgress(filepath.Join(workDir, "files"), filepath.Join(launcher.DataDir, "files"), false)
	if err != nil {
		ShowErrorMessage(err)
		EventEmit("game_state_changed", "idle")
		return
	}

	// Copy icon
	err = Copy(filepath.Join(workDir, "icon.png"), filepath.Join(launcher.DataDir, "icon.png"))
	if err != nil {
		ShowErrorMessage(err)
		EventEmit("game_state_changed", "idle")
		return
	}

	// Copy metadata
	err = Copy(filepath.Join(workDir, "metadata.yml"), filepath.Join(launcher.DataDir, "metadata.yml"))
	if err != nil {
		ShowErrorMessage(err)
		EventEmit("game_state_changed", "idle")
		return
	}

	// Copy launcher into game files
	exe, err := os.Executable()
	if err != nil {
		ShowErrorMessage(err)
		EventEmit("game_state_changed", "idle")
		return
	}
	err = Copy(exe, filepath.Join(launcher.DataDir, "launcher"))
	if err != nil {
		ShowErrorMessage(err)
		EventEmit("game_state_changed", "idle")
		return
	}

	// Create .desktop file
	toWrite := fmt.Sprintf(`[Desktop Entry]
Name=%s
Comment=Play using GameDiscPlayer
Exec="%s/launcher"
Path=%s
Icon=%s/icon.png
Terminal=false
Type=Application
Categories=Game;
`, launcher.Metadata.Name, launcher.DataDir, launcher.DataDir, launcher.DataDir)
	err = os.MkdirAll(filepath.Join(homeDir, ".local/share/applications/games"), 0755)
	if err != nil {
		ShowErrorMessage(err)
		EventEmit("game_state_changed", "idle")
		return
	}
	err = os.WriteFile(filepath.Join(homeDir, ".local/share/applications/games", launcher.Metadata.Name+".desktop"), []byte(toWrite), 0755)
	if err != nil {
		ShowErrorMessage(err)
		EventEmit("game_state_changed", "idle")
		return
	}

	// Create .installed file
	f, err := os.Create(filepath.Join(launcher.DataDir, ".installed"))
	if err != nil {
		ShowErrorMessage(err)
		EventEmit("game_state_changed", "idle")
		return
	}
	f.Close()

	// Set game options
	if launcher.GetSelectedRunner() != nil {
		launcher.Options.Runner = launcher.SelectedRunner
	}
	launcher.SaveOptions()

	// Copy BIOS files
	biosSystems, err := os.ReadDir("bios")
	if err == nil {
		for _, entry := range biosSystems {
			if !entry.IsDir() {
				continue
			}

			CopyRecursivelyWithProgress(filepath.Join("bios", entry.Name()),
				filepath.Join(homeDir, ".local/share/GameDiscPlayer/runners", entry.Name(), "shared/bios"),
				true)
		}
	}

	// Restart launcher
	if !launcher.NoGUI {
		syscall.Exec(exe, os.Args[1:], os.Environ())
	}
}

func (launcher *Launcher) Uninstall() {
	shouldQuit := launcher.IsRunningFromInstallationDirectory() || launcher.NoGUI

	if _, err := os.Stat(launcher.DataDir); os.IsNotExist(err) {
		return
	} else if err != nil {
		ShowErrorMessage(err)
		EventEmit("game_state_changed", "idle")
		return
	}

	// Get user home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		ShowErrorMessage(err)
		EventEmit("game_state_changed", "idle")
		return
	}

	// Removing files
	RemoveDirectoryRecursively(launcher.DataDir)

	err = os.Remove(filepath.Join(homeDir, ".local/share/applications/games", launcher.Metadata.Name+".desktop"))
	if err != nil && !os.IsNotExist(err) {
		ShowErrorMessage(err)
		EventEmit("game_state_changed", "idle")
		return
	}

	if shouldQuit {
		// Exit launcher
		launcher.App.Quit()
	} else {
		// Restart launcher
		exe, err := os.Executable()
		if err != nil {
			ShowErrorMessage(err)
			EventEmit("game_state_changed", "idle")
			return
		}
		syscall.Exec(exe, os.Args[1:], os.Environ())
	}
}

func (launcher *Launcher) IsRunningFromInstallationDirectory() bool {
	_, err := os.Stat(".installed")
	return err == nil
}

func (launcher *Launcher) IsGameInstalled() bool {
	_, err := os.Stat(filepath.Join(launcher.DataDir, ".installed"))
	return err == nil
}

func ShowErrorMessage(err error) {
	if launcher.NoGUI {
		log.Fatal(err)
		return
	}

	ch := make(chan bool, 1)

	glib.IdleAdd(func() {
		dialog := gtk.NewMessageDialog(&launcher.MainWindow.Window, gtk.DialogFlags(gtk.DialogModal)|gtk.DialogDestroyWithParent, gtk.MessageType(gtk.MessageInfo), gtk.ButtonsType(gtk.ButtonsOK))
		dialog.SetTitle("Launcher error")
		dialog.SetMarkup("Error: " + err.Error())
		dialog.ConnectResponse(func(responseId int) {
			ch <- true

			dialog.Destroy()
		})

		dialog.SetVisible(true)
	})

	// Wait for response
	<-ch
}
