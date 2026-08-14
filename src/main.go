package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

func main() {
	if appimage, ok := os.LookupEnv("APPIMAGE"); ok {
		// Change working directory to appimage file's location
		err := os.Chdir(filepath.Dir(appimage))
		if err != nil {
			log.Fatal(err)
		}
		// Fix python environment
		os.Unsetenv("PYTHONHOME")
		os.Unsetenv("PYTHONPATH")
	} else {
		// Change working directory to binary's location
		exe, err := os.Executable()
		if err != nil {
			log.Fatal(err)
		}
		err = os.Chdir(filepath.Dir(exe))
		if err != nil {
			log.Fatal(err)
		}
	}

	// Update local data
	err := updateLocalData()
	if err != nil {
		log.Fatal(err)
	}

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
	descriptionTextView.Buffer().SetText(launcher.Metadata.Description)
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
			if strings.HasPrefix(runner.Exec, "https://") || strings.HasPrefix(runner.Exec, "http://") {
				options = append(options, fmt.Sprintf("%s (Downloadable)", runner.DisplayName))
			} else {
				options = append(options, runner.DisplayName)
			}
		}
		runnerDropdown.SetModel(gtk.NewStringList(options))
		if selected >= 0 {
			runnerDropdown.SetSelected(uint(selected))
			runnerSettingsButton.SetVisible(launcher.GetSelectedRunner().openSettingsFunc != nil)
		} else {
			launcher.SelectedRunner = launcher.Runners[0].RunnerID
		}

		runnerDropdown.Connect("notify::selected", func() {
			launcher.SelectedRunner = launcher.Runners[runnerDropdown.Selected()].RunnerID

			runnerSettingsButton.SetVisible(launcher.GetSelectedRunner().openSettingsFunc != nil)
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

	// Setup events
	EventSubscribe("game_state_changed", func(args ...any) {
		state := args[0].(string)
		switch state {
		case "idle":
			if launcher.Options.ExitOnPlay {
				launcher.MainWindow.Close()
			}

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

func updateLocalData() error {
	// Get user home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	// Rename old 'game_disc_player' directory to 'GameDiscPlayer'
	err = os.Rename(filepath.Join(homeDir, ".local/share/game_disc_player"), filepath.Join(homeDir, ".local/share/GameDiscPlayer"))
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	return nil
}
